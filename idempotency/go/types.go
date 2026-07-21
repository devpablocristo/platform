package idempotency

import (
	"context"
	"net/http"
)

// Outcome describes whether a claim may execute or must replay a stored result.
type Outcome uint8

const (
	OutcomeAcquired Outcome = iota + 1
	OutcomeReplay
)

// ClaimRequest identifies one idempotent operation.
type ClaimRequest struct {
	Scope       string
	Key         string
	Fingerprint string
}

// Lease is the opaque ownership proof returned by a successful claim.
// Consumers must pass it back unchanged to Complete or Abandon.
type Lease struct {
	Scope       string
	Key         string
	Fingerprint string
	Token       string
}

// Response is the HTTP result persisted for future replays.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// ClaimResult is either an acquired Lease or a completed Response.
type ClaimResult struct {
	Outcome  Outcome
	Lease    Lease
	Response Response
}

// Store is the durable state machine used by Middleware.
type Store interface {
	Claim(context.Context, ClaimRequest) (ClaimResult, error)
	Complete(context.Context, Lease, Response) error
	Abandon(context.Context, Lease) error
}
