package clerk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.clerk.com/v1"

type Config struct {
	SecretKey string
	BaseURL   string
	Client    *http.Client
}

type Client struct {
	secretKey string
	baseURL   string
	client    *http.Client
}

type APIError struct {
	StatusCode int
	Body       string
	Headers    http.Header
	receivedAt time.Time
}

func (e *APIError) Error() string {
	return fmt.Sprintf("clerk request failed with status %d", e.StatusCode)
}

func (e *APIError) Message(fallback string) string {
	var payload struct {
		Errors []struct {
			LongMessage string `json:"long_message"`
			Message     string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal([]byte(e.Body), &payload); err == nil {
		for _, item := range payload.Errors {
			if message := strings.TrimSpace(firstNonEmpty(item.LongMessage, item.Message)); message != "" {
				return message
			}
		}
	}
	return fallback
}

func StatusCode(err error) int {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode
	}
	return 0
}

func IsNotFound(err error) bool {
	return StatusCode(err) == http.StatusNotFound
}

func IsRateLimited(err error) bool {
	return StatusCode(err) == http.StatusTooManyRequests
}

// RetryAfter returns Clerk's requested retry delay for a rate-limited
// response. Both delta-seconds and HTTP-date values are supported.
func RetryAfter(err error) (time.Duration, bool) {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return 0, false
	}
	return apiErr.RetryAfter()
}

func (e *APIError) RetryAfter() (time.Duration, bool) {
	if e == nil {
		return 0, false
	}
	raw := strings.TrimSpace(e.Headers.Get("Retry-After"))
	if raw == "" {
		return 0, false
	}
	if seconds, err := strconv.ParseInt(raw, 10, 64); err == nil {
		if seconds < 0 {
			return 0, false
		}
		return time.Duration(seconds) * time.Second, true
	}
	at, err := http.ParseTime(raw)
	if err != nil {
		return 0, false
	}
	receivedAt := e.receivedAt
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}
	delay := at.Sub(receivedAt)
	if delay < 0 {
		delay = 0
	}
	return delay, true
}

func New(config Config) *Client {
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	client := config.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{
		secretKey: strings.TrimSpace(config.SecretKey),
		baseURL:   baseURL,
		client:    client,
	}
}

func (c *Client) FindUserByEmail(ctx context.Context, email string) (User, bool, error) {
	values := url.Values{}
	values.Add("email_address", strings.TrimSpace(strings.ToLower(email)))
	var payload userListResponse
	if err := c.json(ctx, http.MethodGet, "/users?"+values.Encode(), nil, &payload); err != nil {
		return User{}, false, err
	}
	if len(payload.Data) == 0 {
		return User{}, false, nil
	}
	return userFromPayload(payload.Data[0]), true, nil
}

func (c *Client) CreateUser(ctx context.Context, input CreateUserInput) (User, error) {
	body := map[string]any{
		"email_address":             []string{strings.TrimSpace(strings.ToLower(input.Email))},
		"skip_password_requirement": true,
		"skip_password_checks":      true,
	}
	var payload clerkUser
	if err := c.json(ctx, http.MethodPost, "/users", body, &payload); err != nil {
		return User{}, err
	}
	return userFromPayload(payload), nil
}

func (c *Client) UpdateUserEmail(ctx context.Context, providerUserID string, email string) (User, error) {
	body := map[string]any{
		"email_address": strings.TrimSpace(strings.ToLower(email)),
	}
	path := "/users/" + url.PathEscape(strings.TrimSpace(providerUserID)) + "/email_address"
	if err := c.json(ctx, http.MethodPut, path, body, nil); err != nil {
		return User{}, err
	}
	return c.GetUser(ctx, providerUserID)
}

func (c *Client) DeleteUser(ctx context.Context, providerUserID string) error {
	err := c.json(ctx, http.MethodDelete, "/users/"+url.PathEscape(strings.TrimSpace(providerUserID)), nil, nil)
	if IsNotFound(err) {
		return nil
	}
	return err
}

func (c *Client) GetUser(ctx context.Context, providerUserID string) (User, error) {
	var payload clerkUser
	if err := c.json(ctx, http.MethodGet, "/users/"+url.PathEscape(strings.TrimSpace(providerUserID)), nil, &payload); err != nil {
		return User{}, err
	}
	return userFromPayload(payload), nil
}

func (c *Client) CreateOrganization(ctx context.Context, input OrganizationInput) (Organization, error) {
	body := map[string]any{
		"name": strings.TrimSpace(input.Name),
	}
	if slug := strings.TrimSpace(input.Slug); slug != "" {
		body["slug"] = slug
	}
	if err := addMetadata(body, "private_metadata", input.PrivateMetadata); err != nil {
		return Organization{}, err
	}
	var payload clerkOrganization
	if err := c.json(ctx, http.MethodPost, "/organizations", body, &payload); err != nil {
		return Organization{}, err
	}
	return organizationFromPayload(payload), nil
}

func (c *Client) GetOrganization(ctx context.Context, providerOrgID string) (Organization, error) {
	var payload clerkOrganization
	if err := c.json(ctx, http.MethodGet, "/organizations/"+url.PathEscape(strings.TrimSpace(providerOrgID)), nil, &payload); err != nil {
		return Organization{}, err
	}
	return organizationFromPayload(payload), nil
}

func (c *Client) UpdateOrganization(ctx context.Context, providerOrgID string, input OrganizationInput) (Organization, error) {
	body := map[string]any{}
	if name := strings.TrimSpace(input.Name); name != "" {
		body["name"] = name
	}
	if slug := strings.TrimSpace(input.Slug); slug != "" {
		body["slug"] = slug
	}
	if err := addMetadata(body, "private_metadata", input.PrivateMetadata); err != nil {
		return Organization{}, err
	}
	var payload clerkOrganization
	if err := c.json(ctx, http.MethodPatch, "/organizations/"+url.PathEscape(strings.TrimSpace(providerOrgID)), body, &payload); err != nil {
		return Organization{}, err
	}
	return organizationFromPayload(payload), nil
}

func (c *Client) DeleteOrganization(ctx context.Context, providerOrgID string) error {
	err := c.json(ctx, http.MethodDelete, "/organizations/"+url.PathEscape(strings.TrimSpace(providerOrgID)), nil, nil)
	if IsNotFound(err) {
		return nil
	}
	return err
}

func (c *Client) ListOrganizations(ctx context.Context, input ListInput) ([]Organization, error) {
	limit := input.Limit
	if limit <= 0 {
		limit = 100
	}
	out := make([]Organization, 0)
	for offset := input.Offset; ; offset += limit {
		path := fmt.Sprintf("/organizations?limit=%d&offset=%d", limit, offset)
		var payload organizationListResponse
		if err := c.json(ctx, http.MethodGet, path, nil, &payload); err != nil {
			return nil, err
		}
		for _, item := range payload.Data {
			out = append(out, organizationFromPayload(item))
		}
		if len(payload.Data) < limit || payload.TotalCount <= offset+len(payload.Data) {
			return out, nil
		}
	}
}

func (c *Client) ListUserOrgMemberships(ctx context.Context, providerUserID string) ([]OrganizationMembership, error) {
	providerUserID = strings.TrimSpace(providerUserID)
	if providerUserID == "" {
		return nil, nil
	}
	const limit = 100
	out := make([]OrganizationMembership, 0)
	for offset := 0; ; offset += limit {
		path := fmt.Sprintf(
			"/users/%s/organization_memberships?limit=%d&offset=%d",
			url.PathEscape(providerUserID),
			limit,
			offset,
		)
		var payload orgMembershipListResponse
		if err := c.json(ctx, http.MethodGet, path, nil, &payload); err != nil {
			return nil, err
		}
		for _, item := range payload.Data {
			out = append(out, orgMembershipFromPayload(item))
		}
		if len(payload.Data) < limit || payload.TotalCount <= offset+len(payload.Data) {
			return out, nil
		}
	}
}

func (c *Client) ListOrganizationMemberships(ctx context.Context, providerOrgID string) ([]OrganizationMembership, error) {
	providerOrgID = strings.TrimSpace(providerOrgID)
	if providerOrgID == "" {
		return nil, nil
	}
	const limit = 100
	out := make([]OrganizationMembership, 0)
	for offset := 0; ; offset += limit {
		values := url.Values{}
		values.Set("organization_id", providerOrgID)
		values.Set("limit", fmt.Sprintf("%d", limit))
		values.Set("offset", fmt.Sprintf("%d", offset))
		var payload orgMembershipListResponse
		if err := c.json(ctx, http.MethodGet, "/organization_memberships?"+values.Encode(), nil, &payload); err != nil {
			return nil, err
		}
		for _, item := range payload.Data {
			out = append(out, orgMembershipFromPayload(item))
		}
		if len(payload.Data) < limit || payload.TotalCount <= offset+len(payload.Data) {
			return out, nil
		}
	}
}

func (c *Client) CreateOrgMembership(ctx context.Context, input OrgMembershipInput) error {
	body := map[string]any{
		"user_id": strings.TrimSpace(input.ProviderUserID),
		"role":    strings.TrimSpace(input.Role),
	}
	path := "/organizations/" + url.PathEscape(strings.TrimSpace(input.ProviderOrgID)) + "/memberships"
	return c.json(ctx, http.MethodPost, path, body, nil)
}

func (c *Client) UpdateOrgMembership(ctx context.Context, input OrgMembershipInput) error {
	body := map[string]any{"role": strings.TrimSpace(input.Role)}
	path := "/organizations/" + url.PathEscape(strings.TrimSpace(input.ProviderOrgID)) + "/memberships/" + url.PathEscape(strings.TrimSpace(input.ProviderUserID))
	return c.json(ctx, http.MethodPatch, path, body, nil)
}

func (c *Client) DeleteOrgMembership(ctx context.Context, providerOrgID, providerUserID string) error {
	path := "/organizations/" + url.PathEscape(strings.TrimSpace(providerOrgID)) + "/memberships/" + url.PathEscape(strings.TrimSpace(providerUserID))
	err := c.json(ctx, http.MethodDelete, path, nil, nil)
	if IsNotFound(err) {
		return nil
	}
	return err
}

func (c *Client) CreateOrgInvitation(ctx context.Context, input OrgInvitationInput) (Invitation, error) {
	body := map[string]any{
		"email_address": strings.TrimSpace(strings.ToLower(input.Email)),
		"role":          strings.TrimSpace(input.Role),
	}
	if inviter := strings.TrimSpace(input.InviterProviderUserID); inviter != "" {
		body["inviter_user_id"] = inviter
	}
	if redirectURL := strings.TrimSpace(input.RedirectURL); redirectURL != "" {
		body["redirect_url"] = redirectURL
	}
	if input.ExpiresInDays > 0 {
		body["expires_in_days"] = input.ExpiresInDays
	}
	if err := addMetadata(body, "private_metadata", input.PrivateMetadata); err != nil {
		return Invitation{}, err
	}
	if err := addMetadata(body, "public_metadata", input.PublicMetadata); err != nil {
		return Invitation{}, err
	}
	var payload clerkInvitation
	path := "/organizations/" + url.PathEscape(strings.TrimSpace(input.ProviderOrgID)) + "/invitations"
	if err := c.json(ctx, http.MethodPost, path, body, &payload); err != nil {
		return Invitation{}, err
	}
	invitation := invitationFromPayload(payload)
	if invitation.OrganizationID == "" {
		invitation.OrganizationID = strings.TrimSpace(input.ProviderOrgID)
	}
	return invitation, nil
}

func (c *Client) json(ctx context.Context, method string, path string, body any, out any) error {
	if strings.TrimSpace(c.secretKey) == "" {
		return &APIError{StatusCode: http.StatusUnauthorized, Body: `{"errors":[{"message":"clerk secret key is not configured"}]}`}
	}
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(raw),
			Headers:    resp.Header.Clone(),
			receivedAt: time.Now().UTC(),
		}
	}
	if out == nil || len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
