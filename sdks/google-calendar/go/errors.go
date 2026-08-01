package google

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// ValidationError reports invalid local input before a request is sent.
type ValidationError struct {
	Operation string
	Field     string
	Message   string
}

func (e *ValidationError) Error() string {
	prefix := "google calendar"
	if strings.TrimSpace(e.Operation) != "" {
		prefix = "google " + strings.TrimSpace(e.Operation)
	}
	return fmt.Sprintf("%s: %s %s", prefix, e.Field, e.Message)
}

func validationError(operation, field, message string) error {
	return &ValidationError{
		Operation: operation,
		Field:     field,
		Message:   message,
	}
}

// TransportError wraps failures that happened before an HTTP response was
// received, including context and net/http timeouts.
type TransportError struct {
	Operation string
	Err       error
}

func (e *TransportError) Error() string {
	return fmt.Sprintf("google calendar: %s: %v", e.Operation, e.Err)
}

func (e *TransportError) Unwrap() error {
	return e.Err
}

// Timeout reports whether the transport error was caused by a deadline.
func (e *TransportError) Timeout() bool {
	return IsTimeout(e.Err)
}

// ResponseError wraps a malformed successful response.
type ResponseError struct {
	Operation string
	Err       error
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("google calendar: %s response: %v", e.Operation, e.Err)
}

func (e *ResponseError) Unwrap() error {
	return e.Err
}

// ErrorDetail is one item from the Google JSON error envelope.
type ErrorDetail struct {
	Domain       string `json:"domain,omitempty"`
	Reason       string `json:"reason,omitempty"`
	Message      string `json:"message,omitempty"`
	LocationType string `json:"locationType,omitempty"`
	Location     string `json:"location,omitempty"`
}

// APIError is returned when Google Calendar responds with a non-2xx status.
// Body is retained (with a size limit) for diagnostics.
type APIError struct {
	StatusCode int
	Status     string
	Message    string
	Details    []ErrorDetail
	Body       string
	Headers    http.Header
}

func (e *APIError) Error() string {
	if strings.TrimSpace(e.Message) != "" {
		return fmt.Sprintf("google calendar request failed with status %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("google calendar request failed with status %d", e.StatusCode)
}

// OAuthError is returned when a Google OAuth endpoint rejects a request.
type OAuthError struct {
	StatusCode  int
	Code        string
	Description string
	Body        string
	Headers     http.Header
}

func (e *OAuthError) Error() string {
	switch {
	case e.Code != "" && e.Description != "":
		return fmt.Sprintf("google oauth: %s: %s", e.Code, e.Description)
	case e.Code != "":
		return fmt.Sprintf("google oauth: %s", e.Code)
	default:
		return fmt.Sprintf("google oauth request failed with status %d", e.StatusCode)
	}
}

// StatusCode extracts an HTTP status from APIError or OAuthError.
func StatusCode(err error) int {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode
	}
	var oauthErr *OAuthError
	if errors.As(err, &oauthErr) {
		return oauthErr.StatusCode
	}
	return 0
}

func IsNotFound(err error) bool {
	return StatusCode(err) == http.StatusNotFound
}

func IsConflict(err error) bool {
	return StatusCode(err) == http.StatusConflict
}

func IsPreconditionFailed(err error) bool {
	return StatusCode(err) == http.StatusPreconditionFailed
}

// IsTimeout reports context deadlines and transport errors implementing
// Timeout() bool. Context cancellation without a deadline is not a timeout.
func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var timeout interface{ Timeout() bool }
	return errors.As(err, &timeout) && timeout.Timeout()
}

func decodeAPIError(statusCode int, headers http.Header, body []byte) *APIError {
	var envelope struct {
		Error struct {
			Code    int           `json:"code"`
			Message string        `json:"message"`
			Errors  []ErrorDetail `json:"errors"`
			Status  string        `json:"status"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &envelope)
	return &APIError{
		StatusCode: statusCode,
		Status:     envelope.Error.Status,
		Message:    envelope.Error.Message,
		Details:    envelope.Error.Errors,
		Body:       string(body),
		Headers:    headers.Clone(),
	}
}
