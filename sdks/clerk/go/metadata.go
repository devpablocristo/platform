package clerk

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// ErrInvalidMetadata reports metadata that cannot be represented as a JSON
// object. Metadata is validated before a request is sent to Clerk.
var ErrInvalidMetadata = errors.New("clerk: invalid metadata")

// Metadata is a provider-neutral JSON object. Numbers decoded from Clerk are
// preserved as json.Number instead of being coerced to float64.
type Metadata map[string]any

func (m *Metadata) UnmarshalJSON(raw []byte) error {
	if m == nil {
		return fmt.Errorf("%w: destination is nil", ErrInvalidMetadata)
	}
	raw = bytes.TrimSpace(raw)
	if bytes.Equal(raw, []byte("null")) {
		*m = nil
		return nil
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value map[string]any
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidMetadata, err)
	}
	if value == nil {
		return fmt.Errorf("%w: expected a JSON object", ErrInvalidMetadata)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: unexpected trailing JSON value", ErrInvalidMetadata)
	}
	*m = Metadata(value)
	return nil
}

func normalizedMetadata(field string, value Metadata) (Metadata, error) {
	if value == nil {
		return nil, nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrInvalidMetadata, field, err)
	}
	var normalized Metadata
	if err := json.Unmarshal(raw, &normalized); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrInvalidMetadata, field, err)
	}
	return normalized, nil
}

func addMetadata(body map[string]any, field string, value Metadata) error {
	if value == nil {
		return nil
	}
	normalized, err := normalizedMetadata(field, value)
	if err != nil {
		return err
	}
	body[field] = normalized
	return nil
}
