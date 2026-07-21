package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type rowFunc func(...any) error

func (row rowFunc) Scan(destinations ...any) error {
	return row(destinations...)
}

type execResult struct {
	tag pgconn.CommandTag
	err error
}

type scriptedDB struct {
	rows        []pgx.Row
	execs       []execResult
	querySQL    []string
	queryArgs   [][]any
	execSQL     []string
	execArgs    [][]any
	queryCursor int
	execCursor  int
}

func (db *scriptedDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	db.querySQL = append(db.querySQL, query)
	db.queryArgs = append(db.queryArgs, append([]any(nil), args...))
	if db.queryCursor >= len(db.rows) {
		return rowFunc(func(...any) error { return errors.New("unexpected query") })
	}
	row := db.rows[db.queryCursor]
	db.queryCursor++
	return row
}

func (db *scriptedDB) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	db.execSQL = append(db.execSQL, query)
	db.execArgs = append(db.execArgs, append([]any(nil), args...))
	if db.execCursor >= len(db.execs) {
		return pgconn.CommandTag{}, errors.New("unexpected exec")
	}
	result := db.execs[db.execCursor]
	db.execCursor++
	return result.tag, result.err
}

func fixedStore(t *testing.T, db DBTX) *PostgresStore {
	t.Helper()
	store, err := NewPostgresStore(db, StoreConfig{
		LeaseDuration: time.Minute,
		RecordTTL:     time.Hour,
		Now: func() time.Time {
			return time.Date(2026, 7, 21, 12, 0, 0, 0, time.FixedZone("ART", -3*60*60))
		},
		NewToken: func() (string, error) { return "lease-token", nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func acquiredRow(token string) pgx.Row {
	return rowFunc(func(destinations ...any) error {
		*destinations[0].(*string) = token
		return nil
	})
}

func noRow() pgx.Row {
	return rowFunc(func(...any) error { return pgx.ErrNoRows })
}

func existingRow(fingerprint, state string, status pgtype.Int4, header http.Header, body []byte) pgx.Row {
	return rowFunc(func(destinations ...any) error {
		*destinations[0].(*string) = fingerprint
		*destinations[1].(*string) = state
		*destinations[2].(*pgtype.Int4) = status
		if header != nil {
			encoded, err := json.Marshal(header)
			if err != nil {
				return err
			}
			*destinations[3].(*[]byte) = encoded
		}
		*destinations[4].(*[]byte) = append([]byte(nil), body...)
		return nil
	})
}

func TestPostgresStoreClaimAcquiresAtomically(t *testing.T) {
	db := &scriptedDB{rows: []pgx.Row{acquiredRow("lease-token")}}
	store := fixedStore(t, db)

	result, err := store.Claim(context.Background(), ClaimRequest{Scope: "scope-a", Key: "key-a", Fingerprint: "fingerprint-a"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != OutcomeAcquired || result.Lease.Token != "lease-token" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if len(db.queryArgs) != 1 || len(db.queryArgs[0]) != 7 {
		t.Fatalf("claim should use seven bound arguments, got %#v", db.queryArgs)
	}
	if got := db.queryArgs[0][0]; got != "scope-a" {
		t.Fatalf("scope was not bound: %v", got)
	}
}

func TestPostgresStoreClaimRequiresScope(t *testing.T) {
	store := fixedStore(t, &scriptedDB{})

	_, err := store.Claim(context.Background(), ClaimRequest{
		Scope:       "  ",
		Key:         "key-a",
		Fingerprint: "fingerprint-a",
	})
	if !errors.Is(err, ErrScopeRequired) {
		t.Fatalf("scope-less claim returned %v", err)
	}
}

func TestPostgresStoreClaimReplaysCompletedResponse(t *testing.T) {
	header := http.Header{"Content-Type": {"application/json"}, "X-Result": {"stable"}}
	db := &scriptedDB{rows: []pgx.Row{
		noRow(),
		existingRow("fingerprint-a", "completed", pgtype.Int4{Int32: 201, Valid: true}, header, []byte(`{"id":"one"}`)),
	}}
	store := fixedStore(t, db)

	result, err := store.Claim(context.Background(), ClaimRequest{Scope: "scope-a", Key: "key-a", Fingerprint: "fingerprint-a"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != OutcomeReplay || result.Response.StatusCode != http.StatusCreated {
		t.Fatalf("unexpected replay: %#v", result)
	}
	if got := result.Response.Header.Get("X-Result"); got != "stable" {
		t.Fatalf("unexpected replay header %q", got)
	}
	if got := string(result.Response.Body); got != `{"id":"one"}` {
		t.Fatalf("unexpected replay body %q", got)
	}
}

func TestPostgresStoreClaimReportsConflictAndInProgress(t *testing.T) {
	tests := []struct {
		name        string
		fingerprint string
		state       string
		want        error
	}{
		{name: "mismatch", fingerprint: "other", state: "completed", want: ErrFingerprintMismatch},
		{name: "processing", fingerprint: "fingerprint-a", state: "processing", want: ErrInProgress},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := &scriptedDB{rows: []pgx.Row{
				noRow(),
				existingRow(test.fingerprint, test.state, pgtype.Int4{}, nil, nil),
			}}
			store := fixedStore(t, db)
			_, err := store.Claim(context.Background(), ClaimRequest{Scope: "scope-a", Key: "key-a", Fingerprint: "fingerprint-a"})
			if !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v", test.want, err)
			}
		})
	}
}

func TestPostgresStoreCompleteAndAbandonRequireCurrentLease(t *testing.T) {
	lease := Lease{Scope: "scope-a", Key: "key-a", Fingerprint: "fingerprint-a", Token: "lease-token"}
	t.Run("complete", func(t *testing.T) {
		db := &scriptedDB{execs: []execResult{{tag: pgconn.NewCommandTag("UPDATE 1")}}}
		store := fixedStore(t, db)
		err := store.Complete(context.Background(), lease, Response{StatusCode: http.StatusNoContent})
		if err != nil {
			t.Fatal(err)
		}
		if len(db.execArgs) != 1 || len(db.execArgs[0]) != 9 {
			t.Fatalf("complete should use nine bound arguments, got %#v", db.execArgs)
		}
		body, ok := db.execArgs[0][7].([]byte)
		if !ok || body == nil || len(body) != 0 {
			t.Fatalf("empty response must be persisted as non-null bytea: %#v", db.execArgs[0][7])
		}
	})

	t.Run("lease lost", func(t *testing.T) {
		db := &scriptedDB{execs: []execResult{{tag: pgconn.NewCommandTag("DELETE 0")}}}
		store := fixedStore(t, db)
		err := store.Abandon(context.Background(), lease)
		if err != ErrLeaseLost {
			t.Fatalf("expected ErrLeaseLost, got %v", err)
		}
	})
}

func TestPostgresStorePurgeExpiredIsBounded(t *testing.T) {
	db := &scriptedDB{execs: []execResult{{tag: pgconn.NewCommandTag("DELETE 3")}}}
	store := fixedStore(t, db)
	deleted, err := store.PurgeExpired(context.Background(), 3)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 3 {
		t.Fatalf("expected three deleted records, got %d", deleted)
	}
	if got := db.execArgs[0][1]; got != 3 {
		t.Fatalf("purge limit was not bound: %v", got)
	}
}

func TestNewPostgresStoreValidatesDurations(t *testing.T) {
	db := &scriptedDB{}
	_, err := NewPostgresStore(db, StoreConfig{LeaseDuration: time.Minute, RecordTTL: time.Second})
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got %v", err)
	}
}
