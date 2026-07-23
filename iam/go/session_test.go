package iam

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	postgres "github.com/devpablocristo/platform/databases/postgres/go"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestVerifiedSessionValidationIsFailClosed(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 23, 15, 0, 0, 0, time.UTC)
	valid := validVerifiedSession(now)
	if err := valid.ValidateAt(now); err != nil {
		t.Fatalf("valid session: %v", err)
	}

	tests := []struct {
		name    string
		mutate  func(*VerifiedSession)
		at      time.Time
		wantErr error
	}{
		{
			name:   "provider missing",
			mutate: func(session *VerifiedSession) { session.Provider = " " },
			at:     now, wantErr: ErrInvalidVerifiedSession,
		},
		{
			name:   "organization missing",
			mutate: func(session *VerifiedSession) { session.ExternalOrganizationID = "" },
			at:     now, wantErr: ErrInvalidVerifiedSession,
		},
		{
			name:   "not active yet",
			mutate: func(*VerifiedSession) {},
			at:     valid.IssuedAt.Add(-time.Nanosecond), wantErr: ErrInvalidVerifiedSession,
		},
		{
			name:   "expired",
			mutate: func(*VerifiedSession) {},
			at:     valid.ExpiresAt, wantErr: ErrInvalidVerifiedSession,
		},
		{
			name: "invalid interval",
			mutate: func(session *VerifiedSession) {
				session.ExpiresAt = session.IssuedAt
			},
			at: now, wantErr: ErrInvalidVerifiedSession,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			session := valid
			test.mutate(&session)
			if err := session.ValidateAt(test.at); !errors.Is(err, test.wantErr) {
				t.Fatalf("ValidateAt error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestWithinSessionTxResolvesScopesCommitsAndRunsAfterCommit(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 23, 15, 0, 0, 0, time.UTC)
	tx := &sessionTestTx{}
	beginner := &sessionTestBeginner{tx: tx}
	active := activeMembershipFixture()
	var resolvedSession VerifiedSession
	resolver := MembershipResolverFunc(func(
		_ context.Context,
		db DBTX,
		session VerifiedSession,
	) (ActiveMembership, error) {
		if db != tx {
			t.Fatalf("resolver database = %T, want transaction", db)
		}
		resolvedSession = session
		return active, nil
	})
	transactor, err := NewSessionTransactor(beginner, resolver, SessionTransactorConfig{
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewSessionTransactor: %v", err)
	}

	afterCommit := false
	callbackCalled := false
	session := validVerifiedSession(now)
	session.ProviderPermissions = []string{" read ", "", "write", "read"}
	err = transactor.WithinSessionTx(t.Context(), session, func(
		ctx context.Context,
		callbackTx pgx.Tx,
		got ActiveMembership,
	) error {
		callbackCalled = true
		if callbackTx != tx || got != active {
			t.Fatalf("callback tx=%T active=%#v", callbackTx, got)
		}
		return postgres.AfterCommit(ctx, func(context.Context) error {
			afterCommit = true
			return nil
		})
	})
	if err != nil {
		t.Fatalf("WithinSessionTx: %v", err)
	}
	if !callbackCalled || !afterCommit || tx.commits != 1 || tx.rollbacks != 0 {
		t.Fatalf(
			"callback=%v afterCommit=%v commits=%d rollbacks=%d",
			callbackCalled,
			afterCommit,
			tx.commits,
			tx.rollbacks,
		)
	}
	if len(resolvedSession.ProviderPermissions) != 2 ||
		resolvedSession.ProviderPermissions[0] != "read" ||
		resolvedSession.ProviderPermissions[1] != "write" {
		t.Fatalf("permissions were not normalized: %#v", resolvedSession.ProviderPermissions)
	}
	if len(tx.execSQL) != 1 || !strings.Contains(tx.execSQL[0], "app.user_id") ||
		!strings.Contains(tx.execSQL[0], "app.org_id") {
		t.Fatalf("transaction context SQL = %#v", tx.execSQL)
	}
	if len(tx.execArgs[0]) != 2 || tx.execArgs[0][0] != active.UserID ||
		tx.execArgs[0][1] != active.OrganizationID {
		t.Fatalf("transaction context args = %#v", tx.execArgs)
	}
}

func TestWithinSessionTxRollsBackBeforeCallbackWhenAdmissionFails(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 23, 15, 0, 0, 0, time.UTC)
	tx := &sessionTestTx{}
	beginner := &sessionTestBeginner{tx: tx}
	transactor, err := NewSessionTransactor(
		beginner,
		MembershipResolverFunc(func(context.Context, DBTX, VerifiedSession) (ActiveMembership, error) {
			return ActiveMembership{}, ErrActiveMembershipRequired
		}),
		SessionTransactorConfig{Now: func() time.Time { return now }},
	)
	if err != nil {
		t.Fatal(err)
	}
	callbackCalled := false
	err = transactor.WithinSessionTx(t.Context(), validVerifiedSession(now), func(
		context.Context,
		pgx.Tx,
		ActiveMembership,
	) error {
		callbackCalled = true
		return nil
	})
	if !errors.Is(err, ErrActiveMembershipRequired) {
		t.Fatalf("error = %v, want ErrActiveMembershipRequired", err)
	}
	if callbackCalled || tx.commits != 0 || tx.rollbacks != 1 || len(tx.execSQL) != 0 {
		t.Fatalf(
			"callback=%v commits=%d rollbacks=%d execs=%d",
			callbackCalled,
			tx.commits,
			tx.rollbacks,
			len(tx.execSQL),
		)
	}
}

func TestWithinSessionTxRejectsExpiredSessionBeforeBegin(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 23, 15, 0, 0, 0, time.UTC)
	beginner := &sessionTestBeginner{tx: &sessionTestTx{}}
	transactor, err := NewSessionTransactor(
		beginner,
		NewPostgresMembershipResolver(),
		SessionTransactorConfig{Now: func() time.Time { return now }},
	)
	if err != nil {
		t.Fatal(err)
	}
	session := validVerifiedSession(now)
	session.ExpiresAt = now
	err = transactor.WithinSessionTx(
		t.Context(),
		session,
		func(context.Context, pgx.Tx, ActiveMembership) error { return nil },
	)
	if !errors.Is(err, ErrInvalidVerifiedSession) {
		t.Fatalf("error = %v, want ErrInvalidVerifiedSession", err)
	}
	if beginner.begins != 0 {
		t.Fatalf("expired session began %d transactions", beginner.begins)
	}
}

func TestWithinSessionTxRollsBackCallbackErrorAndPanic(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 23, 15, 0, 0, 0, time.UTC)
	callbackError := errors.New("business failure")

	t.Run("error", func(t *testing.T) {
		tx := &sessionTestTx{}
		transactor := sessionTestTransactor(t, tx, now)
		err := transactor.WithinSessionTx(t.Context(), validVerifiedSession(now), func(
			context.Context,
			pgx.Tx,
			ActiveMembership,
		) error {
			return callbackError
		})
		if !errors.Is(err, callbackError) || tx.commits != 0 || tx.rollbacks != 1 {
			t.Fatalf("error=%v commits=%d rollbacks=%d", err, tx.commits, tx.rollbacks)
		}
	})

	t.Run("panic", func(t *testing.T) {
		tx := &sessionTestTx{}
		transactor := sessionTestTransactor(t, tx, now)
		err := transactor.WithinSessionTx(t.Context(), validVerifiedSession(now), func(
			context.Context,
			pgx.Tx,
			ActiveMembership,
		) error {
			panic("boom")
		})
		if err == nil || !strings.Contains(err.Error(), "panic in transaction: boom") {
			t.Fatalf("panic error = %v", err)
		}
		if tx.commits != 0 || tx.rollbacks != 1 {
			t.Fatalf("commits=%d rollbacks=%d", tx.commits, tx.rollbacks)
		}
	})
}

func TestPostgresMembershipResolverUsesExactActiveTuple(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 23, 15, 0, 0, 0, time.UTC)
	active := activeMembershipFixture()
	tx := &sessionTestTx{
		queryRow: sessionTestRowFunc(func(dest ...any) error {
			*dest[0].(*string) = active.MembershipID
			*dest[1].(*string) = active.OrganizationID
			*dest[2].(*string) = active.UserID
			*dest[3].(*string) = active.Role
			return nil
		}),
	}
	session := validVerifiedSession(now)
	resolved, err := NewPostgresMembershipResolver().ResolveActiveMembership(
		t.Context(),
		tx,
		session,
	)
	if err != nil {
		t.Fatalf("ResolveActiveMembership: %v", err)
	}
	if resolved != active {
		t.Fatalf("resolved = %#v, want %#v", resolved, active)
	}
	if len(tx.queryArgs) != 3 || tx.queryArgs[0] != session.Provider ||
		tx.queryArgs[1] != session.Subject ||
		tx.queryArgs[2] != session.ExternalOrganizationID {
		t.Fatalf("resolver args = %#v", tx.queryArgs)
	}
	if !strings.Contains(tx.querySQL, "m.status = 'active'") ||
		!strings.Contains(tx.querySQL, "o.status = 'active'") ||
		!strings.Contains(tx.querySQL, "u.status = 'active'") {
		t.Fatalf("resolver query is not fail-closed: %s", tx.querySQL)
	}
}

func TestPostgresMembershipResolverHidesMissingTuple(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 23, 15, 0, 0, 0, time.UTC)
	tx := &sessionTestTx{
		queryRow: sessionTestRowFunc(func(...any) error { return pgx.ErrNoRows }),
	}
	_, err := NewPostgresMembershipResolver().ResolveActiveMembership(
		t.Context(),
		tx,
		validVerifiedSession(now),
	)
	if !errors.Is(err, ErrActiveMembershipRequired) {
		t.Fatalf("error = %v, want ErrActiveMembershipRequired", err)
	}
}

func sessionTestTransactor(
	t *testing.T,
	tx *sessionTestTx,
	now time.Time,
) *SessionTransactor {
	t.Helper()
	transactor, err := NewSessionTransactor(
		&sessionTestBeginner{tx: tx},
		MembershipResolverFunc(func(context.Context, DBTX, VerifiedSession) (ActiveMembership, error) {
			return activeMembershipFixture(), nil
		}),
		SessionTransactorConfig{Now: func() time.Time { return now }},
	)
	if err != nil {
		t.Fatal(err)
	}
	return transactor
}

func validVerifiedSession(now time.Time) VerifiedSession {
	return VerifiedSession{
		Provider:               "identity-provider",
		Subject:                "external-user",
		SessionID:              "session",
		ExternalOrganizationID: "external-organization",
		ProviderRole:           "provider-role",
		ProviderPermissions:    []string{"provider:read"},
		IssuedAt:               now.Add(-time.Minute),
		ExpiresAt:              now.Add(time.Minute),
	}
}

func activeMembershipFixture() ActiveMembership {
	return ActiveMembership{
		MembershipID:   "f2b31195-b6e5-4e79-8fbf-f166578c67d3",
		OrganizationID: "7c2d141b-7d3d-458d-86f8-0acb24fbde6b",
		UserID:         "231858ec-c314-41bd-a8d1-e52567da8639",
		Role:           "consumer-role",
	}
}

type sessionTestBeginner struct {
	tx     pgx.Tx
	err    error
	begins int
}

func (beginner *sessionTestBeginner) Begin(context.Context) (pgx.Tx, error) {
	beginner.begins++
	return beginner.tx, beginner.err
}

type sessionTestTx struct {
	pgx.Tx
	execSQL     []string
	execArgs    [][]any
	execErr     error
	querySQL    string
	queryArgs   []any
	queryRow    pgx.Row
	commitErr   error
	rollbackErr error
	commits     int
	rollbacks   int
}

func (tx *sessionTestTx) Exec(
	_ context.Context,
	sql string,
	arguments ...any,
) (pgconn.CommandTag, error) {
	tx.execSQL = append(tx.execSQL, sql)
	tx.execArgs = append(tx.execArgs, append([]any(nil), arguments...))
	if tx.execErr != nil {
		return pgconn.CommandTag{}, tx.execErr
	}
	return pgconn.NewCommandTag("SELECT 1"), nil
}

func (tx *sessionTestTx) QueryRow(
	_ context.Context,
	sql string,
	arguments ...any,
) pgx.Row {
	tx.querySQL = sql
	tx.queryArgs = append([]any(nil), arguments...)
	if tx.queryRow == nil {
		return sessionTestRowFunc(func(...any) error { return pgx.ErrNoRows })
	}
	return tx.queryRow
}

func (tx *sessionTestTx) Commit(context.Context) error {
	tx.commits++
	return tx.commitErr
}

func (tx *sessionTestTx) Rollback(context.Context) error {
	tx.rollbacks++
	return tx.rollbackErr
}

type sessionTestRowFunc func(...any) error

func (function sessionTestRowFunc) Scan(destinations ...any) error {
	return function(destinations...)
}
