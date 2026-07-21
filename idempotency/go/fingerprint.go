package idempotency

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// FingerprintRequest hashes the request method, target, representation headers,
// and body. It restores Body before returning so the handler can read it.
func FingerprintRequest(request *http.Request, maxBodyBytes int64) (string, error) {
	if request == nil {
		return "", fmt.Errorf("%w: request is nil", ErrInvalidConfig)
	}
	if maxBodyBytes < 1 {
		return "", fmt.Errorf("%w: request body limit must be positive", ErrInvalidConfig)
	}

	var body []byte
	if request.Body != nil {
		limited := io.LimitReader(request.Body, maxBodyBytes+1)
		var err error
		body, err = io.ReadAll(limited)
		closeErr := request.Body.Close()
		request.Body = io.NopCloser(bytes.NewReader(body))
		if err != nil {
			return "", fmt.Errorf("read request body: %w", err)
		}
		if closeErr != nil {
			return "", fmt.Errorf("close request body: %w", closeErr)
		}
		if int64(len(body)) > maxBodyBytes {
			return "", ErrRequestTooLarge
		}
	}

	path := request.URL.EscapedPath()
	if path == "" {
		path = "/"
	}
	canonical := strings.Join([]string{
		strings.ToUpper(request.Method),
		path,
		request.URL.RawQuery,
		strings.ToLower(strings.TrimSpace(request.Header.Get("Content-Type"))),
		strings.ToLower(strings.TrimSpace(request.Header.Get("Content-Encoding"))),
	}, "\n")

	digest := sha256.New()
	_, _ = io.WriteString(digest, canonical)
	_, _ = digest.Write([]byte{'\n'})
	_, _ = digest.Write(body)
	return hex.EncodeToString(digest.Sum(nil)), nil
}
