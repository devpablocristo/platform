package event

import (
	"encoding/json"
	"strings"
	"time"
)

// Metadata agrupa datos comunes de eventos asíncronos.
type Metadata struct {
	ID       string            `json:"id,omitempty"`
	Kind     string            `json:"kind"`
	Source   string            `json:"source,omitempty"`
	TenantID string            `json:"tenant_id,omitempty"`
	Version  int               `json:"version,omitempty"`
	SentAt   time.Time         `json:"sent_at"`
	Extra    map[string]string `json:"extra,omitempty"`
}

// Envelope representa un evento tipado reusable.
type Envelope[T any] struct {
	Metadata Metadata `json:"metadata"`
	Payload  T        `json:"payload"`
}

// New construye un envelope con defaults razonables.
func New[T any](kind string, payload T, metadata Metadata, now time.Time) Envelope[T] {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	metadata.Kind = strings.TrimSpace(kind)
	if metadata.Version <= 0 {
		metadata.Version = 1
	}
	if metadata.SentAt.IsZero() {
		metadata.SentAt = now
	}
	return Envelope[T]{
		Metadata: Metadata{
			ID:       strings.TrimSpace(metadata.ID),
			Kind:     metadata.Kind,
			Source:   strings.TrimSpace(metadata.Source),
			TenantID: strings.TrimSpace(metadata.TenantID),
			Version:  metadata.Version,
			SentAt:   metadata.SentAt,
			Extra:    cloneMap(metadata.Extra),
		},
		Payload: payload,
	}
}

// Encode serializa un envelope a JSON.
func Encode[T any](value Envelope[T]) ([]byte, error) {
	return json.Marshal(value)
}

// Decode deserializa un envelope tipado desde JSON.
func Decode[T any](body []byte) (Envelope[T], error) {
	var value Envelope[T]
	if err := json.Unmarshal(body, &value); err != nil {
		return Envelope[T]{}, err
	}
	return value, nil
}

func cloneMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
