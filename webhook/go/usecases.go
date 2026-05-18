package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"time"

	domain "github.com/devpablocristo/core/webhook/go/usecases/domain"
)

const (
	defaultMaxAttempts = 5
	defaultBackoffBase = time.Minute
	defaultBackoffMax  = 30 * time.Minute
)

type Usecases struct {
	repo        Repository
	now         func() time.Time
	baseBackoff time.Duration
	maxBackoff  time.Duration
	maxAttempts int
}

func NewUsecases(repo Repository) *Usecases {
	return &Usecases{
		repo: repo,
		now: func() time.Time {
			return time.Now().UTC()
		},
		baseBackoff: defaultBackoffBase,
		maxBackoff:  defaultBackoffMax,
		maxAttempts: defaultMaxAttempts,
	}
}

func (u *Usecases) CreateEndpoint(ctx context.Context, endpoint domain.Endpoint) (domain.Endpoint, error) {
	if err := validateEndpointURL(endpoint.URL); err != nil {
		return domain.Endpoint{}, err
	}
	if strings.TrimSpace(endpoint.TenantID) == "" {
		return domain.Endpoint{}, fmt.Errorf("tenant_id is required")
	}
	endpoint.ID = firstNonEmpty(strings.TrimSpace(endpoint.ID), newID())
	endpoint.TenantID = strings.TrimSpace(endpoint.TenantID)
	endpoint.URL = strings.TrimSpace(endpoint.URL)
	endpoint.Secret = firstNonEmpty(strings.TrimSpace(endpoint.Secret), NewSecret())
	endpoint.Events = normalizeEvents(endpoint.Events)
	endpoint.CreatedAt = normalizeTime(endpoint.CreatedAt, u.now)
	endpoint.UpdatedAt = normalizeTime(endpoint.UpdatedAt, func() time.Time { return endpoint.CreatedAt })
	return u.repo.CreateEndpoint(ctx, endpoint)
}

func (u *Usecases) UpdateEndpoint(ctx context.Context, endpoint domain.Endpoint) (domain.Endpoint, error) {
	if strings.TrimSpace(endpoint.TenantID) == "" || strings.TrimSpace(endpoint.ID) == "" {
		return domain.Endpoint{}, fmt.Errorf("tenant_id and id are required")
	}
	if err := validateEndpointURL(endpoint.URL); err != nil {
		return domain.Endpoint{}, err
	}
	current, err := u.repo.GetEndpoint(ctx, strings.TrimSpace(endpoint.TenantID), strings.TrimSpace(endpoint.ID))
	if err != nil {
		return domain.Endpoint{}, fmt.Errorf("get current endpoint: %w", err)
	}
	if strings.TrimSpace(endpoint.Secret) == "" {
		endpoint.Secret = current.Secret
	}
	if endpoint.CreatedAt.IsZero() {
		endpoint.CreatedAt = current.CreatedAt
	}
	endpoint.TenantID = strings.TrimSpace(endpoint.TenantID)
	endpoint.URL = strings.TrimSpace(endpoint.URL)
	endpoint.Events = normalizeEvents(endpoint.Events)
	endpoint.UpdatedAt = u.now().UTC()
	return u.repo.UpdateEndpoint(ctx, endpoint)
}

func (u *Usecases) BuildDispatch(_ context.Context, endpoint domain.Endpoint, eventType string, payload map[string]any, attempt int) (domain.DispatchRequest, domain.Delivery, error) {
	if err := validateEndpointURL(endpoint.URL); err != nil {
		return domain.DispatchRequest{}, domain.Delivery{}, err
	}
	if strings.TrimSpace(eventType) == "" {
		return domain.DispatchRequest{}, domain.Delivery{}, fmt.Errorf("event_type is required")
	}
	if attempt <= 0 {
		attempt = 1
	}
	body, err := json.Marshal(clonePayload(payload))
	if err != nil {
		return domain.DispatchRequest{}, domain.Delivery{}, fmt.Errorf("marshal payload: %w", err)
	}

	now := u.now().UTC()
	deliveryID := newID()
	timestamp := now.Format(time.RFC3339)
	headers := map[string]string{
		"Content-Type":        "application/json",
		"X-Webhook-Id":        deliveryID,
		"X-Webhook-Event":     strings.TrimSpace(eventType),
		"X-Webhook-Timestamp": timestamp,
		"X-Webhook-Signature": Sign(endpoint.Secret, deliveryID, timestamp, body),
		"X-Webhook-Attempt":   fmt.Sprintf("%d", attempt),
		"User-Agent":          "core-webhook/1.0",
	}

	delivery := domain.Delivery{
		ID:         deliveryID,
		EndpointID: endpoint.ID,
		EventType:  strings.TrimSpace(eventType),
		Payload:    clonePayload(payload),
		Attempts:   attempt,
		CreatedAt:  now,
	}
	return domain.DispatchRequest{
		Method:  "POST",
		URL:     endpoint.URL,
		Body:    body,
		Headers: headers,
	}, delivery, nil
}

func (u *Usecases) RegisterSuccess(ctx context.Context, delivery domain.Delivery, statusCode int, responseBody string) (domain.Delivery, error) {
	now := u.now().UTC()
	delivery.StatusCode = &statusCode
	delivery.ResponseBody = responseBody
	delivery.DeliveredAt = &now
	delivery.NextRetry = nil
	return u.repo.UpdateDelivery(ctx, delivery)
}

func (u *Usecases) RegisterFailure(ctx context.Context, delivery domain.Delivery, statusCode int, responseBody string) (domain.Delivery, error) {
	now := u.now().UTC()
	delivery.StatusCode = &statusCode
	delivery.ResponseBody = responseBody
	if next, ok := u.NextRetry(now, delivery.Attempts); ok {
		delivery.NextRetry = &next
	} else {
		delivery.NextRetry = nil
	}
	return u.repo.UpdateDelivery(ctx, delivery)
}

func (u *Usecases) NextRetry(now time.Time, attempts int) (time.Time, bool) {
	if attempts >= u.maxAttempts {
		return time.Time{}, false
	}
	backoff := u.baseBackoff
	for i := 1; i < attempts; i++ {
		backoff *= 2
		if backoff >= u.maxBackoff {
			backoff = u.maxBackoff
			break
		}
	}
	return now.UTC().Add(backoff), true
}

func NewSecret() string {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("secret-%d", time.Now().UTC().UnixNano())
	}
	return hex.EncodeToString(raw[:])
}

func Sign(secret, webhookID, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(secret)))
	mac.Write([]byte(strings.TrimSpace(webhookID)))
	mac.Write([]byte("."))
	mac.Write([]byte(strings.TrimSpace(timestamp)))
	mac.Write([]byte("."))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func clonePayload(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = value
	}
	return out
}

func normalizeEvents(events []string) []string {
	if len(events) == 0 {
		return []string{"*"}
	}
	out := make([]string, 0, len(events))
	seen := map[string]struct{}{}
	for _, event := range events {
		event = strings.TrimSpace(strings.ToLower(event))
		if event == "" {
			continue
		}
		if _, ok := seen[event]; ok {
			continue
		}
		seen[event] = struct{}{}
		out = append(out, event)
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}

func normalizeTime(value time.Time, now func() time.Time) time.Time {
	if value.IsZero() {
		return now().UTC()
	}
	return value.UTC()
}

func validateEndpointURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("invalid endpoint url: %w", err)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("webhook url must use http or https")
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return fmt.Errorf("webhook host is required")
	}
	if parsed.Scheme == "http" && host != "localhost" {
		return fmt.Errorf("plain http is only allowed for localhost")
	}
	ip := net.ParseIP(host)
	if ip != nil {
		if ip.IsLoopback() {
			return nil
		}
		if addr, ok := netip.AddrFromSlice(ip); ok {
			if addr.IsPrivate() || addr.IsLoopback() || addr.IsLinkLocalUnicast() {
				return fmt.Errorf("private network addresses are not allowed")
			}
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func newID() string {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return fmt.Sprintf("wh_%d", time.Now().UTC().UnixNano())
	}
	return hex.EncodeToString(data[:])
}
