package postgres

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

func TestIsUniqueViolation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "gorm duplicated key", err: gorm.ErrDuplicatedKey, want: true},
		{name: "pgx unique", err: &pgconn.PgError{Code: "23505"}, want: true},
		{name: "pq unique", err: &pq.Error{Code: "23505"}, want: true},
		{name: "wrapped pgx unique", err: fmt.Errorf("create row: %w", &pgconn.PgError{Code: "23505"}), want: true},
		{name: "string fallback duplicate", err: errors.New("duplicate key value violates unique constraint"), want: true},
		{name: "other error", err: errors.New("boom"), want: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := IsUniqueViolation(tc.err)
			if got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
