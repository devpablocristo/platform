package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

var (
	ErrNilUnitOfWork           = errors.New("postgres: unit of work is nil")
	ErrNilBeginTx              = errors.New("postgres: begin transaction function is nil")
	ErrNilTransactionFunc      = errors.New("postgres: unit of work function is nil")
	ErrNoActiveTransaction     = errors.New("postgres: no active transaction in context")
	ErrTransactionTypeMismatch = errors.New("postgres: active transaction has unexpected type")
	ErrNilTransactionCallback  = errors.New("postgres: transaction callback is nil")
)

type transactionContextKey struct{}

type activeTransaction struct {
	tx          any
	afterCommit func(TxCallback)
}

// TxBeginFunc starts a transaction of type T.
type TxBeginFunc[T any] func(context.Context) (T, error)

// TxEndFunc commits or rolls back a transaction of type T.
type TxEndFunc[T any] func(context.Context, T) error

// TxCallback runs after commit or rollback.
type TxCallback func(context.Context) error

// UnitOfWork wraps transactional execution for any concrete transaction type.
type UnitOfWork[T any] struct {
	begin    TxBeginFunc[T]
	commit   TxEndFunc[T]
	rollback TxEndFunc[T]
}

// NewUnitOfWork creates a generic transactional runner.
func NewUnitOfWork[T any](begin TxBeginFunc[T], commit TxEndFunc[T], rollback TxEndFunc[T]) (*UnitOfWork[T], error) {
	if begin == nil {
		return nil, ErrNilBeginTx
	}
	if commit == nil {
		return nil, errors.New("postgres: commit transaction function is nil")
	}
	if rollback == nil {
		return nil, errors.New("postgres: rollback transaction function is nil")
	}
	return &UnitOfWork[T]{begin: begin, commit: commit, rollback: rollback}, nil
}

// TxContext exposes the active transaction and transactional callbacks.
type TxContext[T any] struct {
	tx            T
	afterCommit   []TxCallback
	afterRollback []TxCallback
}

// Tx returns the active transaction.
func (ctx *TxContext[T]) Tx() T {
	return ctx.tx
}

// AfterCommit registers a callback that runs only after a successful commit.
func (ctx *TxContext[T]) AfterCommit(callback TxCallback) {
	if callback != nil {
		ctx.afterCommit = append(ctx.afterCommit, callback)
	}
}

// AfterRollback registers a callback that runs after rollback.
func (ctx *TxContext[T]) AfterRollback(callback TxCallback) {
	if callback != nil {
		ctx.afterRollback = append(ctx.afterRollback, callback)
	}
}

// WithinTx runs fn inside a transaction and exposes it through the callback context.
func (uow *UnitOfWork[T]) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	if uow == nil {
		return ErrNilUnitOfWork
	}
	if fn == nil {
		return ErrNilTransactionFunc
	}

	return uow.Do(ctx, func(ctx context.Context, txCtx *TxContext[T]) error {
		ctx = context.WithValue(ctx, transactionContextKey{}, &activeTransaction{
			tx:          txCtx.Tx(),
			afterCommit: txCtx.AfterCommit,
		})
		return fn(ctx)
	})
}

// Tx returns the active transaction carried by ctx.
func Tx[T any](ctx context.Context) (T, error) {
	var zero T
	active, err := activeTransactionFromContext(ctx)
	if err != nil {
		return zero, err
	}
	tx, ok := active.tx.(T)
	if !ok {
		return zero, ErrTransactionTypeMismatch
	}
	return tx, nil
}

// AfterCommit registers a callback on the active transaction carried by ctx.
func AfterCommit(ctx context.Context, callback TxCallback) error {
	if callback == nil {
		return ErrNilTransactionCallback
	}
	active, err := activeTransactionFromContext(ctx)
	if err != nil {
		return err
	}
	active.afterCommit(callback)
	return nil
}

func activeTransactionFromContext(ctx context.Context) (*activeTransaction, error) {
	if ctx == nil {
		return nil, ErrNoActiveTransaction
	}
	active, ok := ctx.Value(transactionContextKey{}).(*activeTransaction)
	if !ok || active == nil {
		return nil, ErrNoActiveTransaction
	}
	return active, nil
}

// Do runs fn inside a transaction.
func (uow *UnitOfWork[T]) Do(ctx context.Context, fn func(context.Context, *TxContext[T]) error) (err error) {
	if uow == nil {
		return ErrNilUnitOfWork
	}
	if fn == nil {
		return ErrNilTransactionFunc
	}

	tx, err := uow.begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	txCtx := &TxContext[T]{tx: tx}

	defer func() {
		if recovered := recover(); recovered != nil {
			rollbackErr := uow.rollback(ctx, tx)
			callbackErr := runCallbacks(ctx, txCtx.afterRollback)
			err = errors.Join(fmt.Errorf("panic in transaction: %v", recovered), rollbackErr, callbackErr)
		}
	}()

	if err := fn(ctx, txCtx); err != nil {
		rollbackErr := uow.rollback(ctx, tx)
		callbackErr := runCallbacks(ctx, txCtx.afterRollback)
		return errors.Join(err, rollbackErr, callbackErr)
	}
	if err := uow.commit(ctx, tx); err != nil {
		rollbackErr := uow.rollback(ctx, tx)
		callbackErr := runCallbacks(ctx, txCtx.afterRollback)
		return errors.Join(fmt.Errorf("commit transaction: %w", err), rollbackErr, callbackErr)
	}
	if err := runCallbacks(ctx, txCtx.afterCommit); err != nil {
		return fmt.Errorf("after commit callback: %w", err)
	}
	return nil
}

// PgxBeginner is implemented by pgxpool.Pool and pgx.Conn.
type PgxBeginner interface {
	Begin(context.Context) (pgx.Tx, error)
}

// NewPgxUnitOfWork creates a UnitOfWork for pgx transactions.
func NewPgxUnitOfWork(beginner PgxBeginner) (*UnitOfWork[pgx.Tx], error) {
	if beginner == nil {
		return nil, ErrNilBeginTx
	}
	return NewUnitOfWork(
		beginner.Begin,
		func(ctx context.Context, tx pgx.Tx) error {
			return tx.Commit(ctx)
		},
		func(ctx context.Context, tx pgx.Tx) error {
			err := tx.Rollback(ctx)
			if errors.Is(err, pgx.ErrTxClosed) {
				return nil
			}
			return err
		},
	)
}

func runCallbacks(ctx context.Context, callbacks []TxCallback) error {
	var err error
	for _, callback := range callbacks {
		err = errors.Join(err, callback(ctx))
	}
	return err
}
