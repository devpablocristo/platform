package clerk

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	svix "github.com/svix/svix-webhooks/go"
)

var (
	ErrInvalidWebhookSignature = errors.New("clerk: invalid webhook signature")
	ErrInvalidWebhookPayload   = errors.New("clerk: invalid webhook payload")
)

type WebhookEventType string

const (
	WebhookUserCreated                    WebhookEventType = "user.created"
	WebhookUserUpdated                    WebhookEventType = "user.updated"
	WebhookUserDeleted                    WebhookEventType = "user.deleted"
	WebhookOrganizationCreated            WebhookEventType = "organization.created"
	WebhookOrganizationUpdated            WebhookEventType = "organization.updated"
	WebhookOrganizationDeleted            WebhookEventType = "organization.deleted"
	WebhookOrganizationMembershipCreated  WebhookEventType = "organizationMembership.created"
	WebhookOrganizationMembershipUpdated  WebhookEventType = "organizationMembership.updated"
	WebhookOrganizationMembershipDeleted  WebhookEventType = "organizationMembership.deleted"
	WebhookOrganizationInvitationCreated  WebhookEventType = "organizationInvitation.created"
	WebhookOrganizationInvitationAccepted WebhookEventType = "organizationInvitation.accepted"
	WebhookOrganizationInvitationRevoked  WebhookEventType = "organizationInvitation.revoked"
	WebhookSessionCreated                 WebhookEventType = "session.created"
	WebhookSessionEnded                   WebhookEventType = "session.ended"
	WebhookSessionRemoved                 WebhookEventType = "session.removed"
	WebhookSessionRevoked                 WebhookEventType = "session.revoked"
)

// WebhookData is a closed union of the provider resources decoded by this
// package. Unknown event families are represented by RawWebhookData.
type WebhookData interface {
	isWebhookData()
}

func (*User) isWebhookData()                   {}
func (*Organization) isWebhookData()           {}
func (*OrganizationMembership) isWebhookData() {}
func (*Invitation) isWebhookData()             {}
func (*Session) isWebhookData()                {}
func (*DeletedResource) isWebhookData()        {}
func (*RawWebhookData) isWebhookData()         {}

type DeletedResource struct {
	ID      string
	Object  string
	Deleted bool
}

type RawWebhookData struct {
	Raw json.RawMessage
}

type WebhookEvent struct {
	ID              string
	Type            WebhookEventType
	InstanceID      string
	Timestamp       time.Time
	Data            WebhookData
	RawData         json.RawMessage
	EventAttributes json.RawMessage
}

type WebhookVerifier struct {
	verifier *svix.Webhook
}

type webhookEnvelope struct {
	Data            json.RawMessage  `json:"data"`
	Object          string           `json:"object"`
	Type            WebhookEventType `json:"type"`
	Timestamp       int64            `json:"timestamp"`
	InstanceID      string           `json:"instance_id"`
	EventAttributes json.RawMessage  `json:"event_attributes"`
}

type deletedResourcePayload struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Deleted bool   `json:"deleted"`
}

func NewWebhookVerifier(signingSecret string) (*WebhookVerifier, error) {
	signingSecret = strings.TrimSpace(signingSecret)
	if signingSecret == "" {
		return nil, fmt.Errorf("clerk: webhook signing secret is required")
	}
	verifier, err := svix.NewWebhook(signingSecret)
	if err != nil {
		return nil, fmt.Errorf("clerk: parse webhook signing secret: %w", err)
	}
	return &WebhookVerifier{verifier: verifier}, nil
}

// VerifyAndDecode verifies the Svix signature over the unmodified payload
// before decoding the Clerk event.
func (v *WebhookVerifier) VerifyAndDecode(payload []byte, headers http.Header) (WebhookEvent, error) {
	if v == nil || v.verifier == nil {
		return WebhookEvent{}, fmt.Errorf("%w: verifier is not configured", ErrInvalidWebhookSignature)
	}
	if err := v.verifier.Verify(payload, headers); err != nil {
		return WebhookEvent{}, fmt.Errorf("%w: %v", ErrInvalidWebhookSignature, err)
	}
	event, err := DecodeWebhookEvent(payload)
	if err != nil {
		return WebhookEvent{}, err
	}
	event.ID = firstNonEmpty(headers.Get("svix-id"), headers.Get("webhook-id"))
	if event.ID == "" {
		return WebhookEvent{}, fmt.Errorf("%w: message id is missing", ErrInvalidWebhookPayload)
	}
	return event, nil
}

// DecodeWebhookEvent decodes a Clerk event without verifying its signature.
// Callers handling HTTP requests should use VerifyAndDecode instead.
func DecodeWebhookEvent(payload []byte) (WebhookEvent, error) {
	if len(bytes.TrimSpace(payload)) == 0 {
		return WebhookEvent{}, fmt.Errorf("%w: body is empty", ErrInvalidWebhookPayload)
	}
	var envelope webhookEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return WebhookEvent{}, fmt.Errorf("%w: %v", ErrInvalidWebhookPayload, err)
	}
	if strings.TrimSpace(envelope.Object) != "event" {
		return WebhookEvent{}, fmt.Errorf("%w: object must be event", ErrInvalidWebhookPayload)
	}
	if strings.TrimSpace(string(envelope.Type)) == "" {
		return WebhookEvent{}, fmt.Errorf("%w: event type is missing", ErrInvalidWebhookPayload)
	}
	if strings.TrimSpace(envelope.InstanceID) == "" {
		return WebhookEvent{}, fmt.Errorf("%w: instance id is missing", ErrInvalidWebhookPayload)
	}
	if envelope.Timestamp <= 0 {
		return WebhookEvent{}, fmt.Errorf("%w: timestamp is missing", ErrInvalidWebhookPayload)
	}
	if len(bytes.TrimSpace(envelope.Data)) == 0 || bytes.Equal(bytes.TrimSpace(envelope.Data), []byte("null")) {
		return WebhookEvent{}, fmt.Errorf("%w: data is missing", ErrInvalidWebhookPayload)
	}

	data, err := decodeWebhookData(envelope.Type, envelope.Data)
	if err != nil {
		return WebhookEvent{}, err
	}
	return WebhookEvent{
		Type:            envelope.Type,
		InstanceID:      strings.TrimSpace(envelope.InstanceID),
		Timestamp:       time.UnixMilli(envelope.Timestamp).UTC(),
		Data:            data,
		RawData:         append(json.RawMessage(nil), envelope.Data...),
		EventAttributes: append(json.RawMessage(nil), envelope.EventAttributes...),
	}, nil
}

func decodeWebhookData(eventType WebhookEventType, raw json.RawMessage) (WebhookData, error) {
	eventName := string(eventType)
	if strings.HasSuffix(eventName, ".deleted") &&
		(strings.HasPrefix(eventName, "user.") || strings.HasPrefix(eventName, "organization.")) {
		var payload deletedResourcePayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, invalidWebhookData(eventType, err)
		}
		if strings.TrimSpace(payload.ID) == "" {
			return nil, invalidWebhookData(eventType, errors.New("resource id is missing"))
		}
		return &DeletedResource{
			ID:      strings.TrimSpace(payload.ID),
			Object:  strings.TrimSpace(payload.Object),
			Deleted: payload.Deleted,
		}, nil
	}

	switch {
	case strings.HasPrefix(eventName, "user."):
		var payload clerkUser
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, invalidWebhookData(eventType, err)
		}
		data := userFromPayload(payload)
		if data.ID == "" {
			return nil, invalidWebhookData(eventType, errors.New("user id is missing"))
		}
		return &data, nil
	case strings.HasPrefix(eventName, "organizationMembership."):
		var payload clerkOrgMembership
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, invalidWebhookData(eventType, err)
		}
		data := orgMembershipFromPayload(payload)
		if data.ID == "" {
			return nil, invalidWebhookData(eventType, errors.New("membership id is missing"))
		}
		return &data, nil
	case strings.HasPrefix(eventName, "organizationInvitation."):
		var payload clerkInvitation
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, invalidWebhookData(eventType, err)
		}
		data := invitationFromPayload(payload)
		if data.ID == "" {
			return nil, invalidWebhookData(eventType, errors.New("invitation id is missing"))
		}
		return &data, nil
	case strings.HasPrefix(eventName, "organization."):
		var payload clerkOrganization
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, invalidWebhookData(eventType, err)
		}
		data := organizationFromPayload(payload)
		if data.ID == "" {
			return nil, invalidWebhookData(eventType, errors.New("organization id is missing"))
		}
		return &data, nil
	case strings.HasPrefix(eventName, "session."):
		var payload clerkSession
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, invalidWebhookData(eventType, err)
		}
		data := sessionFromPayload(payload)
		if data.ID == "" {
			return nil, invalidWebhookData(eventType, errors.New("session id is missing"))
		}
		return &data, nil
	default:
		return &RawWebhookData{Raw: append(json.RawMessage(nil), raw...)}, nil
	}
}

func invalidWebhookData(eventType WebhookEventType, err error) error {
	return fmt.Errorf("%w: decode %s data: %v", ErrInvalidWebhookPayload, eventType, err)
}
