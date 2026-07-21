package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
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
