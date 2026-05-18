package billing

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	maxStripeWebhookBodyBytes = 2 * 1024 * 1024
	stripeWebhookRateLimit    = 120
	stripeWebhookRateWindow   = time.Minute
)

type WebhookHandler struct {
	runtime *Runtime

	whRateMu    sync.Mutex
	whRateCount int
	whRateReset time.Time
}

func NewWebhookHandler(runtime *Runtime) *WebhookHandler {
	return &WebhookHandler{runtime: runtime}
}

func (h *WebhookHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/webhooks/stripe", h.handleStripeWebhook)
}

func (h *WebhookHandler) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	if h.runtime == nil || !h.runtime.Enabled() {
		writeError(w, http.StatusServiceUnavailable, ErrCodeInternal, "stripe billing is not configured")
		return
	}
	if !h.checkWebhookRateLimit() {
		writeError(w, http.StatusTooManyRequests, ErrCodeRateLimit, "stripe webhook rate limit exceeded")
		return
	}
	secret := h.runtime.WebhookSecret()
	if secret == "" {
		writeError(w, http.StatusServiceUnavailable, ErrCodeInternal, "stripe webhook secret is not configured")
		return
	}

	payload, err := io.ReadAll(io.LimitReader(r.Body, maxStripeWebhookBodyBytes))
	if err != nil {
		writeBadRequest(w, "invalid webhook payload")
		return
	}
	event, err := h.runtime.ConstructAndVerifyWebhook(payload, r.Header.Get("Stripe-Signature"), secret)
	if err != nil {
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "invalid stripe signature")
		return
	}
	h.runtime.RecordWebhookReceived("stripe", strings.TrimSpace(string(event.Type)))
	if err := h.runtime.HandleWebhookEvent(r.Context(), event); err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternal, "failed processing stripe webhook")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *WebhookHandler) checkWebhookRateLimit() bool {
	h.whRateMu.Lock()
	defer h.whRateMu.Unlock()
	now := time.Now().UTC()
	if now.After(h.whRateReset) {
		h.whRateCount = 0
		h.whRateReset = now.Add(stripeWebhookRateWindow)
	}
	h.whRateCount++
	return h.whRateCount <= stripeWebhookRateLimit
}
