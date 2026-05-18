package entitlement

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/devpablocristo/core/errors/go/domainerr"
)

type ClientConfig struct {
	BaseURL     string
	InternalKey string
	TimeoutMS   int
}

type Client struct {
	baseURL     string
	internalKey string
	client      *http.Client
	logger      *slog.Logger
}

func NewClient(cfg ClientConfig, logger *slog.Logger) *Client {
	timeoutMS := cfg.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = 300
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		baseURL:     strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		internalKey: strings.TrimSpace(cfg.InternalKey),
		client:      &http.Client{Timeout: time.Duration(timeoutMS) * time.Millisecond},
		logger:      logger,
	}
}

type entitlementsResponse struct {
	TenantID   string         `json:"tenant_id,omitempty"`
	OrgID      string         `json:"org_id,omitempty"`
	PlanCode   string         `json:"plan_code"`
	Status     string         `json:"status"`
	HardLimits map[string]any `json:"hard_limits"`
}

func (c *Client) GetRunRPM(ctx context.Context, tenantID string) (int, error) {
	if c.baseURL == "" || c.internalKey == "" {
		return 0, nil
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return 0, domainerr.Validation("tenant id is required")
	}

	url := fmt.Sprintf("%s/internal/entitlements/%s", c.baseURL, tenantID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, nil
	}
	req.Header.Set("X-Internal-Key", c.internalKey)

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Warn("saas entitlements request failed", "error", err)
		return 0, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		c.logger.Warn("saas entitlements non-success response", "status_code", resp.StatusCode)
		return 0, nil
	}

	var out entitlementsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		c.logger.Warn("saas entitlements decode failed", "error", err)
		return 0, nil
	}
	if status := strings.TrimSpace(out.Status); status != "" && !strings.EqualFold(status, "active") {
		return 0, domainerr.Forbidden("tenant suspended/deleted")
	}
	if out.HardLimits == nil {
		return 0, nil
	}

	switch value := out.HardLimits["run_rpm"].(type) {
	case float64:
		return int(value), nil
	case int:
		return value, nil
	case int64:
		return int(value), nil
	default:
		return 0, nil
	}
}
