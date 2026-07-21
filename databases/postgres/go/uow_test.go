package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

type fakeTx struct {
	commits   int
	rollbacks int
}

func TestUnitOfWorkCommitsAndRunsAfterCommit(t *testing.T) {
	tx := &fakeTx{}
	afterCommit := 0
	afterRollback := 0

	uow, err := NewUnitOfWork(
		func(context.Context) (*fakeTx, error) { return tx, nil },
		func(context.Context, *fakeTx) error {
			tx.commits++
			return nil
		},
		func(context.Context, *fakeTx) error {
			tx.rollbacks++
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	err = uow.Do(context.Background(), func(ctx context.Context, txCtx *TxContext[*fakeTx]) error {
		if txCtx.Tx() != tx {
			t.Fatal("unexpected transaction")
		}
		txCtx.AfterCommit(func(context.Context) error {
			afterCommit++
			return nil
		})
		txCtx.AfterRollback(func(context.Context) error {
			afterRollback++
			return nil
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if tx.commits != 1 || tx.rollbacks != 0 {
		t.Fatalf("commits=%d rollbacks=%d", tx.commits, tx.rollbacks)
	}
	if afterCommit != 1 || afterRollback != 0 {
		t.Fatalf("afterCommit=%d afterRollback=%d", afterCommit, afterRollback)
	}
}

func TestUnitOfWorkRollsBackOnError(t *testing.T) {
	tx := &fakeTx{}
	expected := errors.New("business failed")
	afterRollback := 0

	uow, _ := NewUnitOfWork(
		func(context.Context) (*fakeTx, error) { return tx, nil },
		func(context.Context, *fakeTx) error {
			tx.commits++
			return nil
		},
		func(context.Context, *fakeTx) error {
			tx.rollbacks++
			return nil
		},
	)

	err := uow.Do(context.Background(), func(ctx context.Context, txCtx *TxContext[*fakeTx]) error {
		txCtx.AfterRollback(func(context.Context) error {
			afterRollback++
			return nil
		})
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected business error, got %v", err)
	}
	if tx.commits != 0 || tx.rollbacks != 1 || afterRollback != 1 {
		t.Fatalf("commits=%d rollbacks=%d afterRollback=%d", tx.commits, tx.rollbacks, afterRollback)
	}
}

func TestUnitOfWorkRollsBackOnCommitError(t *testing.T) {
	tx := &fakeTx{}
	commitErr := errors.New("commit failed")

	uow, _ := NewUnitOfWork(
		func(context.Context) (*fakeTx, error) { return tx, nil },
		func(context.Context, *fakeTx) error {
			tx.commits++
			return commitErr
		},
		func(context.Context, *fakeTx) error {
			tx.rollbacks++
			return nil
		},
	)

	err := uow.Do(context.Background(), func(context.Context, *TxContext[*fakeTx]) error {
		return nil
	})
	if !errors.Is(err, commitErr) {
		t.Fatalf("expected commit error, got %v", err)
	}
	if tx.commits != 1 || tx.rollbacks != 1 {
		t.Fatalf("commits=%d rollbacks=%d", tx.commits, tx.rollbacks)
	}
}

func TestUnitOfWorkRollsBackOnPanic(t *testing.T) {
	tx := &fakeTx{}
	uow, _ := NewUnitOfWork(
		func(context.Context) (*fakeTx, error) { return tx, nil },
		func(context.Context, *fakeTx) error {
			tx.commits++
			return nil
		},
		func(context.Context, *fakeTx) error {
			tx.rollbacks++
			return nil
		},
	)

	err := uow.Do(context.Background(), func(context.Context, *TxContext[*fakeTx]) error {
		panic("boom")
	})
	if err == nil || !strings.Contains(err.Error(), "panic in transaction: boom") {
		t.Fatalf("expected panic error, got %v", err)
	}
	if tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatalf("commits=%d rollbacks=%d", tx.commits, tx.rollbacks)
	}
}

func TestNewUnitOfWorkRequiresFunctions(t *testing.T) {
	if _, err := NewUnitOfWork[*fakeTx](nil, nil, nil); !errors.Is(err, ErrNilBeginTx) {
		t.Fatalf("expected ErrNilBeginTx, got %v", err)
	}
}

func TestWithinTxExposesTransactionAndRunsAfterCommit(t *testing.T) {
	tx := &fakeTx{}
	afterCommit := 0
	uow, err := NewUnitOfWork(
		func(context.Context) (*fakeTx, error) { return tx, nil },
		func(context.Context, *fakeTx) error {
			tx.commits++
			return nil
		},
		func(context.Context, *fakeTx) error {
			tx.rollbacks++
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	err = uow.WithinTx(context.Background(), func(ctx context.Context) error {
		activeTx, err := Tx[*fakeTx](ctx)
		if err != nil {
			return err
		}
		if activeTx != tx {
			t.Fatal("unexpected transaction")
		}
		return AfterCommit(ctx, func(callbackCtx context.Context) error {
			afterCommit++
			if _, err := Tx[*fakeTx](callbackCtx); !errors.Is(err, ErrNoActiveTransaction) {
				t.Fatalf("expected committed transaction to be absent from callback context, got %v", err)
			}
			return nil
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if tx.commits != 1 || tx.rollbacks != 0 {
		t.Fatalf("commits=%d rollbacks=%d", tx.commits, tx.rollbacks)
	}
	if afterCommit != 1 {
		t.Fatalf("afterCommit=%d", afterCommit)
	}
}

func TestContextTransactionHelpersFailOutsideTransaction(t *testing.T) {
	if _, err := Tx[*fakeTx](context.Background()); !errors.Is(err, ErrNoActiveTransaction) {
		t.Fatalf("expected ErrNoActiveTransaction from Tx, got %v", err)
	}
	if _, err := Tx[*fakeTx](nil); !errors.Is(err, ErrNoActiveTransaction) {
		t.Fatalf("expected ErrNoActiveTransaction from nil context, got %v", err)
	}
	if err := AfterCommit(context.Background(), func(context.Context) error { return nil }); !errors.Is(err, ErrNoActiveTransaction) {
		t.Fatalf("expected ErrNoActiveTransaction from AfterCommit, got %v", err)
	}
}

func TestContextTransactionHelpersValidateTypeAndCallback(t *testing.T) {
	tx := &fakeTx{}
	uow, _ := NewUnitOfWork(
		func(context.Context) (*fakeTx, error) { return tx, nil },
		func(context.Context, *fakeTx) error { return nil },
		func(context.Context, *fakeTx) error {
			tx.rollbacks++
			return nil
		},
	)

	err := uow.WithinTx(context.Background(), func(ctx context.Context) error {
		if _, err := Tx[string](ctx); !errors.Is(err, ErrTransactionTypeMismatch) {
			t.Fatalf("expected ErrTransactionTypeMismatch, got %v", err)
		}
		return AfterCommit(ctx, nil)
	})
	if !errors.Is(err, ErrNilTransactionCallback) {
		t.Fatalf("expected ErrNilTransactionCallback, got %v", err)
	}
	if tx.rollbacks != 1 {
		t.Fatalf("rollbacks=%d", tx.rollbacks)
	}
}

func TestWithinTxRollsBackOnErrorAndPanic(t *testing.T) {
	transactionErr := errors.New("transaction failed")
	tests := []struct {
		name string
		fn   func(context.Context) error
		want error
	}{
		{
			name: "error",
			fn:   func(context.Context) error { return transactionErr },
			want: transactionErr,
		},
		{
			name: "panic",
			fn: func(context.Context) error {
				panic("context transaction panic")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := &fakeTx{}
			uow, _ := NewUnitOfWork(
				func(context.Context) (*fakeTx, error) { return tx, nil },
				func(context.Context, *fakeTx) error {
					tx.commits++
					return nil
				},
				func(context.Context, *fakeTx) error {
					tx.rollbacks++
					return nil
				},
			)

			err := uow.WithinTx(context.Background(), tt.fn)
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Fatalf("expected transaction error, got %v", err)
			}
			if tt.want == nil && (err == nil || !strings.Contains(err.Error(), "panic in transaction")) {
				t.Fatalf("expected panic error, got %v", err)
			}
			if tx.commits != 0 || tx.rollbacks != 1 {
				t.Fatalf("commits=%d rollbacks=%d", tx.commits, tx.rollbacks)
			}
		})
	}
}

func TestWithinTxReturnsAfterCommitCallbackError(t *testing.T) {
	tx := &fakeTx{}
	callbackErr := errors.New("callback failed")
	uow, _ := NewUnitOfWork(
		func(context.Context) (*fakeTx, error) { return tx, nil },
		func(context.Context, *fakeTx) error {
			tx.commits++
			return nil
		},
		func(context.Context, *fakeTx) error {
			tx.rollbacks++
			return nil
		},
	)

	err := uow.WithinTx(context.Background(), func(ctx context.Context) error {
		return AfterCommit(ctx, func(context.Context) error { return callbackErr })
	})
	if !errors.Is(err, callbackErr) {
		t.Fatalf("expected callback error, got %v", err)
	}
	if tx.commits != 1 || tx.rollbacks != 0 {
		t.Fatalf("commits=%d rollbacks=%d", tx.commits, tx.rollbacks)
	}
}

type fakePgxBeginner struct {
	tx pgx.Tx
}

func (beginner fakePgxBeginner) Begin(context.Context) (pgx.Tx, error) {
	return beginner.tx, nil
}

type fakePgxTx struct {
	pgx.Tx
	commitErr   error
	rollbackErr error
	commits     int
	rollbacks   int
}

func (tx *fakePgxTx) Commit(context.Context) error {
	tx.commits++
	return tx.commitErr
}

func (tx *fakePgxTx) Rollback(context.Context) error {
	tx.rollbacks++
	return tx.rollbackErr
}

func TestPgxUnitOfWorkTreatsClosedRollbackAsIdempotent(t *testing.T) {
	commitErr := errors.New("commit failed")
	tx := &fakePgxTx{commitErr: commitErr, rollbackErr: pgx.ErrTxClosed}
	uow, err := NewPgxUnitOfWork(fakePgxBeginner{tx: tx})
	if err != nil {
		t.Fatal(err)
	}

	err = uow.WithinTx(context.Background(), func(context.Context) error { return nil })
	if !errors.Is(err, commitErr) {
		t.Fatalf("expected commit error, got %v", err)
	}
	if errors.Is(err, pgx.ErrTxClosed) {
		t.Fatalf("closed rollback must be treated as idempotent: %v", err)
	}
	if tx.commits != 1 || tx.rollbacks != 1 {
		t.Fatalf("commits=%d rollbacks=%d", tx.commits, tx.rollbacks)
	}
}
