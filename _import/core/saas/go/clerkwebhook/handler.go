package clerkwebhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/devpablocristo/core/http/go/httperr"
	"github.com/devpablocristo/core/saas/go/notifications"
	userdomain "github.com/devpablocristo/core/saas/go/users/usecases/domain"
)

const (
	headerSvixID        = "svix-id"
	headerSvixTimestamp = "svix-timestamp"
	headerSvixSignature = "svix-signature"

	maxWebhookBodyBytes = 2 * 1024 * 1024
	maxClockSkew        = 5 * time.Minute

	webhookRateLimit  = 60
	webhookRateWindow = time.Minute
)

var sigV1Regexp = regexp.MustCompile(`v1,([A-Za-z0-9+/=_-]+)`)

type UserSyncer interface {
	SyncUser(ctx context.Context, externalID, email, name string, avatarURL *string) (userdomain.User, error)
	SyncOrganization(ctx context.Context, orgExternalID, orgName string) (string, error)
	SyncMembership(ctx context.Context, orgID, userExternalID, email, name string, avatarURL *string, role string) (userdomain.OrgMember, error)
	SoftDeleteUser(ctx context.Context, externalID string) error
	RemoveMembership(ctx context.Context, userExternalID, orgExternalID, orgName string) error
}

type Config struct {
	ClerkWebhookSecret string
	ConsoleBaseURL     string
}

type Handler struct {
	syncer         UserSyncer
	notifications  notifications.NotificationPort
	consoleBaseURL string
	webhookSecret  string
	now            func() time.Time
	logger         *slog.Logger

	rateMu    sync.Mutex
	rateCount int
	rateReset time.Time
}

func NewHandler(cfg Config, syncer UserSyncer, notif notifications.NotificationPort, logger *slog.Logger) *Handler {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.ConsoleBaseURL), "/")
	if baseURL == "" {
		baseURL = "http://localhost:5173"
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		syncer:         syncer,
		notifications:  notif,
		consoleBaseURL: baseURL,
		webhookSecret:  strings.TrimSpace(cfg.ClerkWebhookSecret),
		now: func() time.Time {
			return time.Now().UTC()
		},
		logger: logger,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /webhooks/clerk", h.handle)
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request) {
	if h.webhookSecret == "" {
		httperr.Write(w, http.StatusServiceUnavailable, httperr.CodeInternal, "CLERK_WEBHOOK_SECRET not configured")
		return
	}
	if !h.checkRateLimit() {
		httperr.Write(w, http.StatusTooManyRequests, httperr.CodeRateLimited, "webhook rate limit exceeded")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBodyBytes))
	if err != nil {
		httperr.BadRequest(w, "invalid body")
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	if err := verifySvix(
		h.webhookSecret,
		r.Header.Get(headerSvixID),
		r.Header.Get(headerSvixTimestamp),
		r.Header.Get(headerSvixSignature),
		body,
		h.now,
	); err != nil {
		httperr.Unauthorized(w, "invalid webhook signature")
		return
	}

	var event clerkEventEnvelope
	if err := json.Unmarshal(body, &event); err != nil {
		httperr.BadRequest(w, "invalid webhook payload")
		return
	}

	if err := h.dispatch(r.Context(), event); err != nil {
		h.logger.Error("failed processing clerk webhook", "type", event.Type, "error", err)
		httperr.Write(w, http.StatusInternalServerError, httperr.CodeInternal, "failed processing webhook")
		return
	}

	httperr.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) checkRateLimit() bool {
	h.rateMu.Lock()
	defer h.rateMu.Unlock()
	now := h.now()
	if now.After(h.rateReset) {
		h.rateCount = 0
		h.rateReset = now.Add(webhookRateWindow)
	}
	h.rateCount++
	return h.rateCount <= webhookRateLimit
}

func (h *Handler) dispatch(ctx context.Context, event clerkEventEnvelope) error {
	switch strings.TrimSpace(event.Type) {
	case "user.created":
		return h.onUserUpsert(ctx, event.Data, true)
	case "user.updated":
		return h.onUserUpsert(ctx, event.Data, false)
	case "user.deleted":
		return h.onUserDeleted(ctx, event.Data)
	case "organization.created":
		return h.onOrganizationCreated(ctx, event.Data)
	case "organizationMembership.created":
		return h.onOrganizationMembershipCreated(ctx, event.Data)
	case "organizationMembership.deleted":
		return h.onOrganizationMembershipDeleted(ctx, event.Data)
	default:
		return nil
	}
}

func (h *Handler) onUserUpsert(ctx context.Context, raw json.RawMessage, sendWelcome bool) error {
	var data clerkUserData
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}
	email, err := buildWebhookUserEmail(data.primaryEmail(), data.allEmails())
	if err != nil {
		return err
	}
	name := formatUserName(data.FirstName, data.LastName, email)
	user, err := h.syncer.SyncUser(ctx, data.ID, email, name, nullable(data.ImageURL))
	if err != nil {
		return err
	}
	if sendWelcome && h.notifications != nil {
		payload := map[string]string{
			"recipient_name":  name,
			"action_url":      h.consoleBaseURL + "/tools",
			"preferences_url": h.consoleBaseURL + "/settings/notifications",
		}
		go func(tenantID string, data map[string]string) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if notifyErr := h.notifications.Notify(ctx, tenantID, "welcome", data); notifyErr != nil {
				h.logger.Error("failed async welcome notification", "tenant_id", tenantID, "error", notifyErr)
			}
		}(user.ID, payload)
	}
	return nil
}

func (h *Handler) onUserDeleted(ctx context.Context, raw json.RawMessage) error {
	var data struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}
	if strings.TrimSpace(data.ID) == "" {
		return errors.New("user.deleted: missing user id")
	}
	return h.syncer.SoftDeleteUser(ctx, data.ID)
}

func (h *Handler) onOrganizationCreated(ctx context.Context, raw json.RawMessage) error {
	var data clerkOrganizationData
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}
	orgName := firstNonEmpty(data.Name, data.Slug, data.ID)
	if orgName == "" {
		return errors.New("organization name is empty")
	}
	_, err := h.syncer.SyncOrganization(ctx, data.ID, orgName)
	return err
}

func (h *Handler) onOrganizationMembershipCreated(ctx context.Context, raw json.RawMessage) error {
	var data clerkMembershipData
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}

	orgName := firstNonEmpty(data.Organization.Name, data.Organization.Slug, data.Organization.ID)
	if orgName == "" {
		return errors.New("organization name is empty")
	}
	orgID, err := h.syncer.SyncOrganization(ctx, data.Organization.ID, orgName)
	if err != nil {
		return err
	}

	userID := firstNonEmpty(data.PublicUserData.UserID, data.User.ID)
	if userID == "" {
		return errors.New("membership user_id missing")
	}
	email, err := buildWebhookUserEmail(data.User.primaryEmail(), []string{data.PublicUserData.Identifier})
	if err != nil {
		return err
	}
	name := formatUserName(
		firstNonEmpty(data.PublicUserData.FirstName, data.User.FirstName),
		firstNonEmpty(data.PublicUserData.LastName, data.User.LastName),
		email,
	)
	_, err = h.syncer.SyncMembership(
		ctx,
		orgID,
		userID,
		email,
		name,
		nullable(firstNonEmpty(data.PublicUserData.ImageURL, data.User.ImageURL)),
		data.Role,
	)
	return err
}

func (h *Handler) onOrganizationMembershipDeleted(ctx context.Context, raw json.RawMessage) error {
	var data clerkMembershipData
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}
	userID := firstNonEmpty(data.PublicUserData.UserID, data.User.ID)
	orgName := firstNonEmpty(data.Organization.Name, data.Organization.Slug, data.Organization.ID)
	if userID == "" || orgName == "" {
		return errors.New("organizationMembership.deleted: missing user_id or org")
	}
	return h.syncer.RemoveMembership(ctx, userID, data.Organization.ID, orgName)
}

type clerkEventEnvelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type clerkEmailAddress struct {
	ID           string `json:"id"`
	EmailAddress string `json:"email_address"`
}

type clerkUserData struct {
	ID                    string              `json:"id"`
	FirstName             string              `json:"first_name"`
	LastName              string              `json:"last_name"`
	ImageURL              string              `json:"image_url"`
	PrimaryEmailAddressID string              `json:"primary_email_address_id"`
	EmailAddresses        []clerkEmailAddress `json:"email_addresses"`
}

func (u clerkUserData) primaryEmail() string {
	if u.PrimaryEmailAddressID != "" {
		for _, email := range u.EmailAddresses {
			if strings.TrimSpace(email.ID) == strings.TrimSpace(u.PrimaryEmailAddressID) {
				return strings.TrimSpace(email.EmailAddress)
			}
		}
	}
	for _, email := range u.EmailAddresses {
		if item := strings.TrimSpace(email.EmailAddress); item != "" {
			return item
		}
	}
	return ""
}

func (u clerkUserData) allEmails() []string {
	out := make([]string, 0, len(u.EmailAddresses))
	for _, email := range u.EmailAddresses {
		out = append(out, strings.TrimSpace(email.EmailAddress))
	}
	return out
}

type clerkOrganizationData struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type clerkMembershipData struct {
	ID           string `json:"id"`
	Role         string `json:"role"`
	Organization struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"organization"`
	PublicUserData struct {
		UserID     string `json:"user_id"`
		FirstName  string `json:"first_name"`
		LastName   string `json:"last_name"`
		Identifier string `json:"identifier"`
		ImageURL   string `json:"image_url"`
	} `json:"public_user_data"`
	User clerkUserData `json:"user"`
}

func verifySvix(secret, id, ts, sigHeader string, payload []byte, now func() time.Time) error {
	secret = strings.TrimSpace(secret)
	id = strings.TrimSpace(id)
	ts = strings.TrimSpace(ts)
	sigHeader = strings.TrimSpace(sigHeader)
	if secret == "" || id == "" || ts == "" || sigHeader == "" {
		return errors.New("missing svix headers")
	}

	timestamp, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return errors.New("invalid svix timestamp")
	}
	delta := now().UTC().Sub(time.Unix(timestamp, 0).UTC())
	if delta < 0 {
		delta = -delta
	}
	if delta > maxClockSkew {
		return errors.New("svix timestamp expired")
	}

	secretBytes, err := decodeSvixSecret(secret)
	if err != nil {
		return err
	}

	message := id + "." + ts + "." + string(payload)
	mac := hmac.New(sha256.New, secretBytes)
	_, _ = mac.Write([]byte(message))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	signatures := extractV1Signatures(sigHeader)
	if len(signatures) == 0 {
		return errors.New("missing v1 signatures")
	}
	for _, candidate := range signatures {
		if hmac.Equal([]byte(candidate), []byte(expected)) {
			return nil
		}
	}
	return errors.New("signature mismatch")
}

func decodeSvixSecret(secret string) ([]byte, error) {
	secret = strings.TrimPrefix(strings.TrimSpace(secret), "whsec_")
	if secret == "" {
		return nil, errors.New("invalid svix secret")
	}
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	for _, encoding := range encodings {
		decoded, err := encoding.DecodeString(secret)
		if err == nil && len(decoded) > 0 {
			return decoded, nil
		}
	}
	return nil, errors.New("invalid svix secret encoding")
}

func extractV1Signatures(raw string) []string {
	matches := sigV1Regexp.FindAllStringSubmatch(raw, -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) == 2 {
			out = append(out, strings.TrimSpace(match[1]))
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func nullable(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func formatUserName(firstName, lastName, email string) string {
	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)
	email = strings.TrimSpace(email)
	switch {
	case firstName != "" && lastName != "":
		return firstName + " " + lastName
	case firstName != "":
		return firstName
	case lastName != "":
		return lastName
	case email != "":
		return email
	default:
		return "Unknown User"
	}
}

func buildWebhookUserEmail(primaryEmail string, all []string) (string, error) {
	primaryEmail = strings.TrimSpace(primaryEmail)
	if primaryEmail != "" {
		return primaryEmail, nil
	}
	for _, email := range all {
		email = strings.TrimSpace(email)
		if email != "" {
			return email, nil
		}
	}
	return "", fmt.Errorf("email not found in webhook payload")
}
