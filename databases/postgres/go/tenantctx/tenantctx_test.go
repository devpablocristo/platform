package tenantctx

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/devpablocristo/platform/security/go/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type execCall struct {
	query string
	args  []any
}

type recordingTx struct {
	pgx.Tx
	calls []execCall
	err   error
}

func (tx *recordingTx) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	tx.calls = append(tx.calls, execCall{query: query, args: args})
	if err := ctx.Err(); err != nil {
		return pgconn.CommandTag{}, err
	}
	if tx.err != nil {
		return pgconn.CommandTag{}, tx.err
	}
	return pgconn.NewCommandTag("SELECT 1"), nil
}

func TestSetLocalFailsClosedWithoutTenant(t *testing.T) {
	t.Parallel()

	tx := &recordingTx{}
	if err := SetLocal(context.Background(), tx); err == nil {
		t.Fatal("expected missing tenant error")
	}
	if len(tx.calls) != 0 {
		t.Fatalf("executed %d statements without a tenant", len(tx.calls))
	}
}

func TestSetLocalRejectsNilTransaction(t *testing.T) {
	t.Parallel()

	ctx := tenant.WithID(context.Background(), tenant.FromString("org-a"))
	if err := SetLocal(ctx, nil); !errors.Is(err, ErrNilTransaction) {
		t.Fatalf("error = %v, want ErrNilTransaction", err)
	}
}

func TestSetLocalUsesParameterizedTransactionLocalSetting(t *testing.T) {
	t.Parallel()

	tx := &recordingTx{}
	ctx := tenant.WithID(context.Background(), tenant.FromString("org-'quoted"))
	if err := SetLocal(ctx, tx); err != nil {
		t.Fatalf("SetLocal: %v", err)
	}

	if len(tx.calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(tx.calls))
	}
	call := tx.calls[0]
	if call.query != setLocalSQL {
		t.Fatalf("query = %q, want %q", call.query, setLocalSQL)
	}
	wantArgs := []any{Setting, "org-'quoted"}
	if !reflect.DeepEqual(call.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", call.args, wantArgs)
	}
}

func TestSetLocalPropagatesPostgresError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("postgres unavailable")
	tx := &recordingTx{err: wantErr}
	ctx := tenant.WithID(context.Background(), tenant.FromString("org-a"))
	if err := SetLocal(ctx, tx); !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped %v", err, wantErr)
	}
}

func TestSetLocalPropagatesCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ctx = tenant.WithID(ctx, tenant.FromString("org-a"))
	if err := SetLocal(ctx, &recordingTx{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
