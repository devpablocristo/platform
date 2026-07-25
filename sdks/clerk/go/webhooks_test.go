package clerk

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	svix "github.com/svix/svix-webhooks/go"
)

const testWebhookSecret = "whsec_MfKQ9r8GKYqrTwjUPD8ILPZIo2LaLaSw"

func TestWebhookVerifierVerifiesAndDecodesUserFixture(t *testing.T) {
	payload := webhookFixture(t, "user.created.json")
	headers := signedWebhookHeaders(t, payload, time.Now().UTC())
	verifier := mustWebhookVerifier(t)

	event, err := verifier.VerifyAndDecode(payload, headers)
	if err != nil {
		t.Fatalf("VerifyAndDecode: %v", err)
	}
	if event.ID != "msg_test_123" || event.Type != WebhookUserCreated || event.InstanceID != "ins_123" {
		t.Fatalf("unexpected event envelope %+v", event)
	}
	user, ok := event.Data.(*User)
	if !ok {
		t.Fatalf("expected *User data, got %T", event.Data)
	}
	if user.ID != "user_123" || user.Email != "user@example.com" || !user.EmailVerified {
		t.Fatalf("unexpected user data %+v", user)
	}
	if user.FirstName != "Ada" || user.LastName != "Lovelace" || user.CreatedAt.IsZero() {
		t.Fatalf("unexpected user profile %+v", user)
	}
}

func TestWebhookVerifierDecodesMembershipFixture(t *testing.T) {
	payload := webhookFixture(t, "organizationMembership.updated.json")
	event, err := mustWebhookVerifier(t).VerifyAndDecode(payload, signedWebhookHeaders(t, payload, time.Now().UTC()))
	if err != nil {
		t.Fatalf("VerifyAndDecode: %v", err)
	}
	membership, ok := event.Data.(*OrganizationMembership)
	if !ok {
		t.Fatalf("expected *OrganizationMembership, got %T", event.Data)
	}
	if membership.ID != "orgmem_123" || membership.OrganizationID != "org_123" ||
		membership.User.ID != "user_123" || membership.Role != "org:admin" {
		t.Fatalf("unexpected membership %+v", membership)
	}
}

func TestWebhookVerifierDecodesOrganizationMetadataFixture(t *testing.T) {
	payload := webhookFixture(t, "organization.created.json")
	event, err := mustWebhookVerifier(t).VerifyAndDecode(payload, signedWebhookHeaders(t, payload, time.Now().UTC()))
	if err != nil {
		t.Fatalf("VerifyAndDecode: %v", err)
	}
	organization, ok := event.Data.(*Organization)
	if !ok {
		t.Fatalf("expected *Organization, got %T", event.Data)
	}
	if organization.ID != "org_123" ||
		organization.PrivateMetadata["operation_id"] != "op_123" ||
		organization.PrivateMetadata["attempt"] != json.Number("9007199254740993") {
		t.Fatalf("unexpected organization metadata %+v", organization)
	}
}

func TestWebhookVerifierDecodesInvitationMetadataFixture(t *testing.T) {
	payload := webhookFixture(t, "organizationInvitation.created.json")
	event, err := mustWebhookVerifier(t).VerifyAndDecode(payload, signedWebhookHeaders(t, payload, time.Now().UTC()))
	if err != nil {
		t.Fatalf("VerifyAndDecode: %v", err)
	}
	invitation, ok := event.Data.(*Invitation)
	if !ok {
		t.Fatalf("expected *Invitation, got %T", event.Data)
	}
	if invitation.ID != "orginv_123" ||
		invitation.URL != "https://accounts.example/invitations/orginv_123" ||
		invitation.PrivateMetadata["operation_id"] != "op_123" ||
		invitation.PublicMetadata["source"] != "admin" {
		t.Fatalf("unexpected invitation metadata %+v", invitation)
	}
}

func TestWebhookVerifierRejectsTamperingAndOldMessages(t *testing.T) {
	payload := webhookFixture(t, "user.created.json")
	verifier := mustWebhookVerifier(t)

	headers := signedWebhookHeaders(t, payload, time.Now().UTC())
	if _, err := verifier.VerifyAndDecode(append(payload, ' '), headers); !errors.Is(err, ErrInvalidWebhookSignature) {
		t.Fatalf("expected signature error for tampered payload, got %v", err)
	}

	oldHeaders := signedWebhookHeaders(t, payload, time.Now().UTC().Add(-10*time.Minute))
	if _, err := verifier.VerifyAndDecode(payload, oldHeaders); !errors.Is(err, ErrInvalidWebhookSignature) {
		t.Fatalf("expected signature error for stale message, got %v", err)
	}
}

func TestWebhookDecoderPreservesUnknownVerifiedEvent(t *testing.T) {
	payload := webhookFixture(t, "unknown.created.json")
	event, err := mustWebhookVerifier(t).VerifyAndDecode(payload, signedWebhookHeaders(t, payload, time.Now().UTC()))
	if err != nil {
		t.Fatalf("VerifyAndDecode: %v", err)
	}
	raw, ok := event.Data.(*RawWebhookData)
	if !ok || len(raw.Raw) == 0 || len(event.RawData) == 0 {
		t.Fatalf("expected raw unknown data, got %T %+v", event.Data, event)
	}
}

func TestWebhookDecoderRejectsMalformedKnownPayload(t *testing.T) {
	payload := []byte(`{
		"data":{},
		"object":"event",
		"type":"user.updated",
		"timestamp":1784818800000,
		"instance_id":"ins_123"
	}`)
	headers := signedWebhookHeaders(t, payload, time.Now().UTC())
	_, err := mustWebhookVerifier(t).VerifyAndDecode(payload, headers)
	if !errors.Is(err, ErrInvalidWebhookPayload) {
		t.Fatalf("expected payload error, got %v", err)
	}
}

func TestNewWebhookVerifierRejectsMissingSecret(t *testing.T) {
	if _, err := NewWebhookVerifier(" "); err == nil {
		t.Fatal("expected configuration error")
	}
}

func mustWebhookVerifier(t *testing.T) *WebhookVerifier {
	t.Helper()
	verifier, err := NewWebhookVerifier(testWebhookSecret)
	if err != nil {
		t.Fatalf("NewWebhookVerifier: %v", err)
	}
	return verifier
}

func signedWebhookHeaders(t *testing.T, payload []byte, timestamp time.Time) http.Header {
	t.Helper()
	verifier, err := svix.NewWebhook(testWebhookSecret)
	if err != nil {
		t.Fatalf("create Svix verifier: %v", err)
	}
	signature, err := verifier.Sign("msg_test_123", timestamp, payload)
	if err != nil {
		t.Fatalf("sign webhook: %v", err)
	}
	headers := http.Header{}
	headers.Set("svix-id", "msg_test_123")
	headers.Set("svix-timestamp", fmt.Sprint(timestamp.Unix()))
	headers.Set("svix-signature", signature)
	return headers
}

func webhookFixture(t *testing.T, name string) []byte {
	t.Helper()
	payload, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return payload
}
