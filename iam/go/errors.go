package iam

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	// ErrInvalidArgument reports malformed domain or store input.
	ErrInvalidArgument = errors.New("iam: invalid argument")
	// ErrNotFound reports that the requested IAM record does not exist.
	ErrNotFound = errors.New("iam: record not found")
	// ErrConflict reports a persistence uniqueness or consistency conflict.
	ErrConflict = errors.New("iam: conflict")
	// ErrInvalidVerifiedSession reports a missing, inconsistent or expired
	// identity-provider session projection.
	ErrInvalidVerifiedSession = errors.New("iam: invalid verified session")
	// ErrActiveMembershipRequired intentionally hides whether identity,
	// organization or membership state caused fail-closed resolution.
	ErrActiveMembershipRequired = errors.New("iam: active membership required")
)

func storeError(operation string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: %w", operation, ErrNotFound)
	}
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) {
		switch postgresError.Code {
		case "23503", "23505":
			return fmt.Errorf("%s: %w: %s", operation, ErrConflict, postgresError.ConstraintName)
		case "23502", "23514", "22P02":
			return fmt.Errorf("%s: %w", operation, ErrInvalidArgument)
		}
	}
	return fmt.Errorf("%s: %w", operation, err)
}
