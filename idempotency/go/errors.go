package idempotency

import "errors"

var (
	// ErrNilStore indicates that the middleware has no durable store.
	ErrNilStore = errors.New("idempotency: store is nil")
	// ErrInvalidConfig indicates an invalid store or middleware configuration.
	ErrInvalidConfig = errors.New("idempotency: invalid configuration")
	// ErrKeyRequired indicates a missing Idempotency-Key value.
	ErrKeyRequired = errors.New("idempotency: key is required")
	// ErrInvalidKey indicates a key or scope that cannot be stored safely.
	ErrInvalidKey = errors.New("idempotency: invalid key")
	// ErrScopeRequired indicates that an HTTP key has no trusted namespace.
	ErrScopeRequired = errors.New("idempotency: scope is required")
	// ErrFingerprintRequired indicates a missing request fingerprint.
	ErrFingerprintRequired = errors.New("idempotency: fingerprint is required")
	// ErrFingerprintMismatch indicates that a live key was reused for a different request.
	ErrFingerprintMismatch = errors.New("idempotency: fingerprint mismatch")
	// ErrInProgress indicates that another owner holds a live processing lease.
	ErrInProgress = errors.New("idempotency: request is in progress")
	// ErrLeaseLost indicates that a lease was reclaimed or no longer exists.
	ErrLeaseLost = errors.New("idempotency: lease lost")
	// ErrRequestTooLarge indicates that the request body exceeds the configured limit.
	ErrRequestTooLarge = errors.New("idempotency: request body too large")
	// ErrResponseTooLarge indicates that the response body exceeds the configured limit.
	ErrResponseTooLarge = errors.New("idempotency: response body too large")
)
