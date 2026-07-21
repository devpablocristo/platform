package idempotency

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	DefaultLeaseDuration = 30 * time.Second
	DefaultRecordTTL     = 24 * time.Hour

	maxScopeRunes       = 255
	maxKeyRunes         = 512
	maxFingerprintRunes = 128
)

const claimSQL = `
INSERT INTO platform_idempotency_records (
    scope,
    idempotency_key,
    fingerprint,
    state,
    claim_token,
    lease_expires_at,
    record_expires_at,
    response_status,
    response_headers,
    response_body,
    created_at,
    updated_at
) VALUES ($1, $2, $3, 'processing', $4, $5, $6, NULL, NULL, NULL, $7, $7)
ON CONFLICT (scope, idempotency_key) DO UPDATE SET
    fingerprint = EXCLUDED.fingerprint,
    state = 'processing',
    claim_token = EXCLUDED.claim_token,
    lease_expires_at = EXCLUDED.lease_expires_at,
    record_expires_at = EXCLUDED.record_expires_at,
    response_status = NULL,
    response_headers = NULL,
    response_body = NULL,
    created_at = CASE
        WHEN platform_idempotency_records.record_expires_at <= EXCLUDED.updated_at
            THEN EXCLUDED.created_at
        ELSE platform_idempotency_records.created_at
    END,
    updated_at = EXCLUDED.updated_at
WHERE
    platform_idempotency_records.record_expires_at <= EXCLUDED.updated_at
    OR (
        platform_idempotency_records.state = 'processing'
        AND platform_idempotency_records.fingerprint = EXCLUDED.fingerprint
        AND platform_idempotency_records.lease_expires_at <= EXCLUDED.updated_at
    )
RETURNING claim_token`

const selectSQL = `
SELECT fingerprint, state, response_status, response_headers, response_body
FROM platform_idempotency_records
WHERE scope = $1 AND idempotency_key = $2`

const completeSQL = `
UPDATE platform_idempotency_records
SET
    state = 'completed',
    claim_token = NULL,
    lease_expires_at = NULL,
    record_expires_at = $5,
    response_status = $6,
    response_headers = $7::jsonb,
    response_body = $8,
    updated_at = $9
WHERE
    scope = $1
    AND idempotency_key = $2
    AND fingerprint = $3
    AND state = 'processing'
    AND claim_token = $4`

const abandonSQL = `
DELETE FROM platform_idempotency_records
WHERE
    scope = $1
    AND idempotency_key = $2
    AND fingerprint = $3
    AND state = 'processing'
    AND claim_token = $4`

const purgeSQL = `
WITH expired AS (
    SELECT ctid
    FROM platform_idempotency_records
    WHERE record_expires_at <= $1
    ORDER BY record_expires_at
    LIMIT $2
)
DELETE FROM platform_idempotency_records AS records
USING expired
WHERE records.ctid = expired.ctid`

// DBTX is implemented by pgxpool.Pool, pgx.Conn, and pgx.Tx.
type DBTX interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// StoreConfig configures PostgreSQL lease and retention behavior.
type StoreConfig struct {
	LeaseDuration time.Duration
	RecordTTL     time.Duration
	Now           func() time.Time
	NewToken      func() (string, error)
}

// DefaultStoreConfig returns production-safe defaults with a cryptographic token source.
func DefaultStoreConfig() StoreConfig {
	return StoreConfig{
		LeaseDuration: DefaultLeaseDuration,
		RecordTTL:     DefaultRecordTTL,
		Now:           time.Now,
		NewToken:      randomToken,
	}
}

// PostgresStore persists idempotency claims and completed HTTP responses.
type PostgresStore struct {
	db            DBTX
	leaseDuration time.Duration
	recordTTL     time.Duration
	now           func() time.Time
	newToken      func() (string, error)
}

// NewPostgresStore creates a store backed by a pgx database handle.
func NewPostgresStore(db DBTX, config StoreConfig) (*PostgresStore, error) {
	if db == nil {
		return nil, fmt.Errorf("%w: database is nil", ErrInvalidConfig)
	}

	defaults := DefaultStoreConfig()
	if config.LeaseDuration == 0 {
		config.LeaseDuration = defaults.LeaseDuration
	}
	if config.RecordTTL == 0 {
		config.RecordTTL = defaults.RecordTTL
	}
	if config.Now == nil {
		config.Now = defaults.Now
	}
	if config.NewToken == nil {
		config.NewToken = defaults.NewToken
	}
	if config.LeaseDuration < 0 {
		return nil, fmt.Errorf("%w: lease duration must be positive", ErrInvalidConfig)
	}
	if config.RecordTTL < config.LeaseDuration {
		return nil, fmt.Errorf("%w: record TTL must be at least the lease duration", ErrInvalidConfig)
	}

	return &PostgresStore{
		db:            db,
		leaseDuration: config.LeaseDuration,
		recordTTL:     config.RecordTTL,
		now:           config.Now,
		newToken:      config.NewToken,
	}, nil
}

// Claim atomically acquires a new or expired record, or returns a stored replay.
func (s *PostgresStore) Claim(ctx context.Context, request ClaimRequest) (ClaimResult, error) {
	if s == nil || s.db == nil {
		return ClaimResult{}, fmt.Errorf("%w: postgres store is nil", ErrInvalidConfig)
	}
	if err := validateClaimRequest(request); err != nil {
		return ClaimResult{}, err
	}

	token, err := s.newToken()
	if err != nil {
		return ClaimResult{}, fmt.Errorf("generate lease token: %w", err)
	}
	if strings.TrimSpace(token) == "" {
		return ClaimResult{}, fmt.Errorf("%w: token source returned an empty token", ErrInvalidConfig)
	}

	now := s.now().UTC()
	leaseUntil := now.Add(s.leaseDuration)
	expiresAt := now.Add(s.recordTTL)

	for attempt := 0; attempt < 2; attempt++ {
		var storedToken string
		err = s.db.QueryRow(
			ctx,
			claimSQL,
			request.Scope,
			request.Key,
			request.Fingerprint,
			token,
			leaseUntil,
			expiresAt,
			now,
		).Scan(&storedToken)
		if err == nil {
			return ClaimResult{
				Outcome: OutcomeAcquired,
				Lease: Lease{
					Scope:       request.Scope,
					Key:         request.Key,
					Fingerprint: request.Fingerprint,
					Token:       storedToken,
				},
			}, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return ClaimResult{}, fmt.Errorf("claim idempotency key: %w", err)
		}

		result, found, selectErr := s.readExisting(ctx, request)
		if selectErr != nil {
			return ClaimResult{}, selectErr
		}
		if found {
			return result, nil
		}
	}

	return ClaimResult{}, errors.New("idempotency: record disappeared during claim")
}

func (s *PostgresStore) readExisting(ctx context.Context, request ClaimRequest) (ClaimResult, bool, error) {
	var (
		fingerprint    string
		state          string
		responseStatus pgtype.Int4
		responseHeader []byte
		responseBody   []byte
	)

	err := s.db.QueryRow(ctx, selectSQL, request.Scope, request.Key).Scan(
		&fingerprint,
		&state,
		&responseStatus,
		&responseHeader,
		&responseBody,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ClaimResult{}, false, nil
	}
	if err != nil {
		return ClaimResult{}, false, fmt.Errorf("read idempotency key: %w", err)
	}
	if fingerprint != request.Fingerprint {
		return ClaimResult{}, true, ErrFingerprintMismatch
	}

	switch state {
	case "processing":
		return ClaimResult{}, true, ErrInProgress
	case "completed":
		if !responseStatus.Valid {
			return ClaimResult{}, true, errors.New("idempotency: completed record has no status")
		}
		var header http.Header
		if err := json.Unmarshal(responseHeader, &header); err != nil {
			return ClaimResult{}, true, fmt.Errorf("decode replay headers: %w", err)
		}
		if header == nil {
			header = make(http.Header)
		}
		return ClaimResult{
			Outcome: OutcomeReplay,
			Response: Response{
				StatusCode: int(responseStatus.Int32),
				Header:     header,
				Body:       append([]byte(nil), responseBody...),
			},
		}, true, nil
	default:
		return ClaimResult{}, true, fmt.Errorf("idempotency: unsupported record state %q", state)
	}
}

// Complete stores a response only while lease ownership still matches.
func (s *PostgresStore) Complete(ctx context.Context, lease Lease, response Response) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("%w: postgres store is nil", ErrInvalidConfig)
	}
	if err := validateLease(lease); err != nil {
		return err
	}
	if response.StatusCode < 100 || response.StatusCode > 599 {
		return fmt.Errorf("%w: response status must be between 100 and 599", ErrInvalidConfig)
	}

	headerJSON, err := json.Marshal(cloneHeader(response.Header))
	if err != nil {
		return fmt.Errorf("encode replay headers: %w", err)
	}
	body := response.Body
	if body == nil {
		body = []byte{}
	}
	now := s.now().UTC()
	tag, err := s.db.Exec(
		ctx,
		completeSQL,
		lease.Scope,
		lease.Key,
		lease.Fingerprint,
		lease.Token,
		now.Add(s.recordTTL),
		response.StatusCode,
		string(headerJSON),
		body,
		now,
	)
	if err != nil {
		return fmt.Errorf("complete idempotency key: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrLeaseLost
	}
	return nil
}

// Abandon removes a processing record owned by lease so a retry can run immediately.
func (s *PostgresStore) Abandon(ctx context.Context, lease Lease) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("%w: postgres store is nil", ErrInvalidConfig)
	}
	if err := validateLease(lease); err != nil {
		return err
	}
	tag, err := s.db.Exec(ctx, abandonSQL, lease.Scope, lease.Key, lease.Fingerprint, lease.Token)
	if err != nil {
		return fmt.Errorf("abandon idempotency key: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrLeaseLost
	}
	return nil
}

// PurgeExpired deletes at most limit expired records and returns the deleted count.
func (s *PostgresStore) PurgeExpired(ctx context.Context, limit int) (int64, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("%w: postgres store is nil", ErrInvalidConfig)
	}
	if limit < 1 || limit > 10_000 {
		return 0, fmt.Errorf("%w: purge limit must be between 1 and 10000", ErrInvalidConfig)
	}
	tag, err := s.db.Exec(ctx, purgeSQL, s.now().UTC(), limit)
	if err != nil {
		return 0, fmt.Errorf("purge expired idempotency keys: %w", err)
	}
	return tag.RowsAffected(), nil
}

func validateClaimRequest(request ClaimRequest) error {
	if strings.TrimSpace(request.Scope) == "" {
		return ErrScopeRequired
	}
	if strings.TrimSpace(request.Key) == "" {
		return ErrKeyRequired
	}
	if utf8.RuneCountInString(request.Scope) > maxScopeRunes {
		return fmt.Errorf("%w: scope exceeds %d characters", ErrInvalidKey, maxScopeRunes)
	}
	if utf8.RuneCountInString(request.Key) > maxKeyRunes {
		return fmt.Errorf("%w: key exceeds %d characters", ErrInvalidKey, maxKeyRunes)
	}
	if strings.TrimSpace(request.Fingerprint) == "" {
		return ErrFingerprintRequired
	}
	if utf8.RuneCountInString(request.Fingerprint) > maxFingerprintRunes {
		return fmt.Errorf("%w: fingerprint exceeds %d characters", ErrInvalidKey, maxFingerprintRunes)
	}
	return nil
}

func validateLease(lease Lease) error {
	if err := validateClaimRequest(ClaimRequest{
		Scope:       lease.Scope,
		Key:         lease.Key,
		Fingerprint: lease.Fingerprint,
	}); err != nil {
		return err
	}
	if strings.TrimSpace(lease.Token) == "" {
		return fmt.Errorf("%w: lease token is required", ErrInvalidKey)
	}
	return nil
}

func randomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func cloneHeader(header http.Header) http.Header {
	if header == nil {
		return make(http.Header)
	}
	return header.Clone()
}
