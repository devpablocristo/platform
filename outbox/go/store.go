package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StoreConfig controls persistence defaults and deterministic dependencies.
type StoreConfig struct {
	Table               pgx.Identifier
	DefaultMaxAttempts  int
	Clock               Clock
	MessageIDGenerator  TokenGenerator
	LeaseTokenGenerator TokenGenerator
}

// Store is a PostgreSQL transactional outbox store.
type Store struct {
	pool               *pgxpool.Pool
	tableSQL           string
	defaultMaxAttempts int
	clock              Clock
	messageIDs         TokenGenerator
	leaseTokens        TokenGenerator
}

// NewStore validates config and creates a PostgreSQL store.
func NewStore(pool *pgxpool.Pool, config StoreConfig) (*Store, error) {
	if pool == nil {
		return nil, ErrNilPool
	}
	table, err := normalizeTable(config.Table)
	if err != nil {
		return nil, err
	}
	if config.DefaultMaxAttempts <= 0 {
		return nil, fmt.Errorf("%w: default max attempts must be positive", ErrInvalidMessage)
	}
	if config.Clock == nil {
		config.Clock = systemClock{}
	}
	if config.MessageIDGenerator == nil {
		config.MessageIDGenerator = cryptoTokenGenerator{}
	}
	if config.LeaseTokenGenerator == nil {
		config.LeaseTokenGenerator = cryptoTokenGenerator{}
	}
	return &Store{
		pool:               pool,
		tableSQL:           table.Sanitize(),
		defaultMaxAttempts: config.DefaultMaxAttempts,
		clock:              config.Clock,
		messageIDs:         config.MessageIDGenerator,
		leaseTokens:        config.LeaseTokenGenerator,
	}, nil
}

// Append writes a message through the caller-provided transaction. The row is
// visible to workers only if that business transaction commits.
func (store *Store) Append(ctx context.Context, tx pgx.Tx, input MessageInput) (Message, error) {
	if tx == nil {
		return Message{}, ErrNilTransaction
	}
	if store == nil {
		return Message{}, ErrNilPool
	}

	input.Topic = strings.TrimSpace(input.Topic)
	if input.Topic == "" {
		return Message{}, fmt.Errorf("%w: topic is required", ErrInvalidMessage)
	}
	input.ID = strings.TrimSpace(input.ID)
	if input.ID == "" {
		generated, err := generatedToken(store.messageIDs)
		if err != nil {
			return Message{}, fmt.Errorf("outbox: generate message ID: %w", err)
		}
		input.ID = generated
	}
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.IdempotencyKey == "" {
		input.IdempotencyKey = input.ID
	}
	if input.MaxAttempts == 0 {
		input.MaxAttempts = store.defaultMaxAttempts
	}
	if input.MaxAttempts < 1 {
		return Message{}, fmt.Errorf("%w: max attempts must be positive", ErrInvalidMessage)
	}

	now, err := store.now()
	if err != nil {
		return Message{}, err
	}
	if input.AvailableAt.IsZero() {
		input.AvailableAt = now
	} else {
		input.AvailableAt = input.AvailableAt.UTC()
	}
	headers, err := json.Marshal(copyHeaders(input.Headers))
	if err != nil {
		return Message{}, fmt.Errorf("outbox: encode headers: %w", err)
	}
	payload := copyBytes(input.Payload)
	if payload == nil {
		payload = []byte{}
	}

	query := `INSERT INTO ` + store.tableSQL + ` (
            id, idempotency_key, topic, payload, headers, available_at,
            attempts, max_attempts, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5::jsonb, $6, 0, $7, $8, $8)
        RETURNING ` + messageColumns
	message, err := scanMessage(tx.QueryRow(
		ctx,
		query,
		input.ID,
		input.IdempotencyKey,
		input.Topic,
		payload,
		headers,
		input.AvailableAt,
		input.MaxAttempts,
		now,
	))
	if err != nil {
		return Message{}, fmt.Errorf("outbox: append message: %w", err)
	}
	return message, nil
}

// Lease atomically claims ready messages with FOR UPDATE SKIP LOCKED. Attempts
// are incremented when the lease is acquired.
func (store *Store) Lease(ctx context.Context, request LeaseRequest) ([]LeasedMessage, error) {
	if store == nil || store.pool == nil {
		return nil, ErrNilPool
	}
	if request.Limit < 1 || request.Duration <= 0 {
		return nil, ErrInvalidLease
	}

	now, err := store.now()
	if err != nil {
		return nil, err
	}
	leaseToken, err := generatedToken(store.leaseTokens)
	if err != nil {
		return nil, fmt.Errorf("outbox: generate lease token: %w", err)
	}
	leaseExpiresAt := now.Add(request.Duration)

	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("outbox: begin lease transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	query := `WITH candidates AS (
            SELECT id
            FROM ` + store.tableSQL + `
            WHERE published_at IS NULL
              AND failed_at IS NULL
              AND attempts < max_attempts
              AND available_at <= $1
              AND (lease_expires_at IS NULL OR lease_expires_at <= $1)
            ORDER BY available_at, created_at, id
            FOR UPDATE SKIP LOCKED
            LIMIT $2
        )
        UPDATE ` + store.tableSQL + ` AS message
        SET lease_token = $3,
            leased_at = $1,
            lease_expires_at = $4,
            attempts = message.attempts + 1,
            last_attempt_at = $1,
            updated_at = $1
        FROM candidates
        WHERE message.id = candidates.id
        RETURNING ` + leasedMessageColumns("message")
	rows, err := tx.Query(ctx, query, now, request.Limit, leaseToken, leaseExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("outbox: lease messages: %w", err)
	}
	defer rows.Close()

	messages := make([]LeasedMessage, 0, request.Limit)
	for rows.Next() {
		message, scanErr := scanLeasedMessage(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("outbox: scan leased message: %w", scanErr)
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("outbox: iterate leased messages: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("outbox: commit leases: %w", err)
	}
	return messages, nil
}

// MarkPublished finalizes a message only while its lease is current. Calling it
// again after publication is an idempotent no-op.
func (store *Store) MarkPublished(ctx context.Context, leased LeasedMessage) error {
	if store == nil || store.pool == nil {
		return ErrNilPool
	}
	if err := validateLeasedMessage(leased); err != nil {
		return err
	}
	now, err := store.now()
	if err != nil {
		return err
	}

	command, err := store.pool.Exec(ctx, `UPDATE `+store.tableSQL+`
        SET published_at = $3,
            lease_token = NULL,
            leased_at = NULL,
            lease_expires_at = NULL,
            updated_at = $3
        WHERE id = $1
          AND lease_token = $2
          AND lease_expires_at > $3
          AND published_at IS NULL
          AND failed_at IS NULL`, leased.ID, leased.LeaseToken, now)
	if err != nil {
		return fmt.Errorf("outbox: mark published: %w", err)
	}
	if command.RowsAffected() == 1 {
		return nil
	}

	var publishedAt *time.Time
	err = store.pool.QueryRow(ctx, `SELECT published_at FROM `+store.tableSQL+` WHERE id = $1`, leased.ID).Scan(&publishedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrMessageNotFound
	}
	if err != nil {
		return fmt.Errorf("outbox: inspect published state: %w", err)
	}
	if publishedAt != nil {
		return nil
	}
	return ErrLeaseLost
}

// FailureDisposition reports whether a delivery was scheduled again or became
// permanently failed.
type FailureDisposition uint8

const (
	FailureRetryScheduled FailureDisposition = iota + 1
	FailureAttemptsExhausted
)

// MarkFailed records a publisher failure under the active lease. retryDelay is
// ignored once MaxAttempts is reached.
func (store *Store) MarkFailed(
	ctx context.Context,
	leased LeasedMessage,
	failure error,
	retryDelay time.Duration,
) (FailureDisposition, error) {
	if store == nil || store.pool == nil {
		return 0, ErrNilPool
	}
	if err := validateLeasedMessage(leased); err != nil {
		return 0, err
	}
	if failure == nil {
		return 0, ErrNilFailure
	}
	if retryDelay < 0 {
		return 0, fmt.Errorf("%w: retry delay is negative", ErrInvalidMessage)
	}
	now, err := store.now()
	if err != nil {
		return 0, err
	}

	disposition := FailureRetryScheduled
	availableAt := now.Add(retryDelay)
	var failedAt *time.Time
	if leased.Attempts >= leased.MaxAttempts {
		disposition = FailureAttemptsExhausted
		availableAt = leased.AvailableAt
		failedAt = &now
	}
	command, err := store.pool.Exec(ctx, `UPDATE `+store.tableSQL+`
        SET available_at = $4,
            last_error = $5,
            last_error_at = $3,
            failed_at = $6,
            lease_token = NULL,
            leased_at = NULL,
            lease_expires_at = NULL,
            updated_at = $3
        WHERE id = $1
          AND lease_token = $2
          AND lease_expires_at > $3
          AND published_at IS NULL
          AND failed_at IS NULL`,
		leased.ID,
		leased.LeaseToken,
		now,
		availableAt,
		failure.Error(),
		failedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("outbox: mark failed: %w", err)
	}
	if command.RowsAffected() != 1 {
		return 0, ErrLeaseLost
	}
	return disposition, nil
}

func (store *Store) now() (time.Time, error) {
	now := store.clock.Now()
	if now.IsZero() {
		return time.Time{}, ErrClockReturnedZero
	}
	return now.UTC(), nil
}

func validateLeasedMessage(message LeasedMessage) error {
	if strings.TrimSpace(message.ID) == "" || strings.TrimSpace(message.LeaseToken) == "" ||
		message.Attempts < 1 || message.MaxAttempts < 1 || message.LeaseExpiresAt.IsZero() {
		return ErrInvalidLease
	}
	return nil
}

const messageColumns = `id, idempotency_key, topic, payload, headers, available_at,
    attempts, max_attempts, last_attempt_at, last_error, last_error_at,
    created_at, updated_at, published_at, failed_at`

func leasedMessageColumns(alias string) string {
	return alias + `.id, ` + alias + `.idempotency_key, ` + alias + `.topic, ` +
		alias + `.payload, ` + alias + `.headers, ` + alias + `.available_at, ` +
		alias + `.attempts, ` + alias + `.max_attempts, ` + alias + `.last_attempt_at, ` +
		alias + `.last_error, ` + alias + `.last_error_at, ` + alias + `.created_at, ` +
		alias + `.updated_at, ` + alias + `.published_at, ` + alias + `.failed_at, ` +
		alias + `.lease_token, ` + alias + `.leased_at, ` + alias + `.lease_expires_at`
}

type rowScanner interface {
	Scan(...any) error
}

func scanMessage(row rowScanner) (Message, error) {
	var message Message
	var headers []byte
	var lastError *string
	if err := row.Scan(
		&message.ID,
		&message.IdempotencyKey,
		&message.Topic,
		&message.Payload,
		&headers,
		&message.AvailableAt,
		&message.Attempts,
		&message.MaxAttempts,
		&message.LastAttemptAt,
		&lastError,
		&message.LastErrorAt,
		&message.CreatedAt,
		&message.UpdatedAt,
		&message.PublishedAt,
		&message.FailedAt,
	); err != nil {
		return Message{}, err
	}
	if err := json.Unmarshal(headers, &message.Headers); err != nil {
		return Message{}, fmt.Errorf("decode headers: %w", err)
	}
	if lastError != nil {
		message.LastError = *lastError
	}
	message.Payload = copyBytes(message.Payload)
	message.Headers = copyHeaders(message.Headers)
	return message, nil
}

func scanLeasedMessage(row rowScanner) (LeasedMessage, error) {
	var message LeasedMessage
	var headers []byte
	var lastError *string
	if err := row.Scan(
		&message.ID,
		&message.IdempotencyKey,
		&message.Topic,
		&message.Payload,
		&headers,
		&message.AvailableAt,
		&message.Attempts,
		&message.MaxAttempts,
		&message.LastAttemptAt,
		&lastError,
		&message.LastErrorAt,
		&message.CreatedAt,
		&message.UpdatedAt,
		&message.PublishedAt,
		&message.FailedAt,
		&message.LeaseToken,
		&message.LeasedAt,
		&message.LeaseExpiresAt,
	); err != nil {
		return LeasedMessage{}, err
	}
	if err := json.Unmarshal(headers, &message.Headers); err != nil {
		return LeasedMessage{}, fmt.Errorf("decode headers: %w", err)
	}
	if lastError != nil {
		message.LastError = *lastError
	}
	message.Payload = copyBytes(message.Payload)
	message.Headers = copyHeaders(message.Headers)
	return message, nil
}
