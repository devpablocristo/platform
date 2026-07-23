package clerk

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func (c *Client) ListOrgInvitations(ctx context.Context, providerOrgID string, input OrgInvitationListInput) ([]Invitation, error) {
	providerOrgID = strings.TrimSpace(providerOrgID)
	if providerOrgID == "" {
		return nil, nil
	}
	limit := boundedLimit(input.Limit)
	out := make([]Invitation, 0)
	for offset := max(input.Offset, 0); ; offset += limit {
		values := url.Values{}
		values.Set("limit", strconv.Itoa(limit))
		values.Set("offset", strconv.Itoa(offset))
		for _, status := range input.Statuses {
			if status = strings.TrimSpace(status); status != "" {
				values.Add("status", status)
			}
		}
		if email := strings.TrimSpace(strings.ToLower(input.Email)); email != "" {
			values.Set("email_address", email)
		}
		if orderBy := strings.TrimSpace(input.OrderBy); orderBy != "" {
			values.Set("order_by", orderBy)
		}

		path := "/organizations/" + url.PathEscape(providerOrgID) + "/invitations?" + values.Encode()
		var payload invitationListResponse
		if err := c.json(ctx, http.MethodGet, path, nil, &payload); err != nil {
			return nil, err
		}
		for _, item := range payload.Data {
			invitation := invitationFromPayload(item)
			if invitation.OrganizationID == "" {
				invitation.OrganizationID = providerOrgID
			}
			out = append(out, invitation)
		}
		if len(payload.Data) < limit || payload.TotalCount <= offset+len(payload.Data) {
			return out, nil
		}
	}
}

func (c *Client) GetOrgInvitation(ctx context.Context, providerOrgID, invitationID string) (Invitation, error) {
	providerOrgID = strings.TrimSpace(providerOrgID)
	invitationID = strings.TrimSpace(invitationID)
	if providerOrgID == "" || invitationID == "" {
		return Invitation{}, fmt.Errorf("clerk: organization id and invitation id are required")
	}
	path := "/organizations/" + url.PathEscape(providerOrgID) + "/invitations/" + url.PathEscape(invitationID)
	var payload clerkInvitation
	if err := c.json(ctx, http.MethodGet, path, nil, &payload); err != nil {
		return Invitation{}, err
	}
	invitation := invitationFromPayload(payload)
	if invitation.OrganizationID == "" {
		invitation.OrganizationID = providerOrgID
	}
	return invitation, nil
}

func (c *Client) RevokeOrgInvitation(ctx context.Context, providerOrgID, invitationID string) error {
	providerOrgID = strings.TrimSpace(providerOrgID)
	invitationID = strings.TrimSpace(invitationID)
	if providerOrgID == "" || invitationID == "" {
		return fmt.Errorf("clerk: organization id and invitation id are required")
	}
	path := "/organizations/" + url.PathEscape(providerOrgID) + "/invitations/" + url.PathEscape(invitationID) + "/revoke"
	err := c.json(ctx, http.MethodPost, path, nil, nil)
	if IsNotFound(err) {
		return nil
	}
	return err
}

// GetOrgMembership resolves a membership by organization and provider user ID.
// Clerk exposes this operation as a filtered list, so the bool reports whether
// the exact membership exists.
func (c *Client) GetOrgMembership(ctx context.Context, providerOrgID, providerUserID string) (OrganizationMembership, bool, error) {
	providerOrgID = strings.TrimSpace(providerOrgID)
	providerUserID = strings.TrimSpace(providerUserID)
	if providerOrgID == "" || providerUserID == "" {
		return OrganizationMembership{}, false, nil
	}
	values := url.Values{}
	values.Set("limit", "1")
	values.Set("offset", "0")
	values.Set("user_id", providerUserID)
	path := "/organizations/" + url.PathEscape(providerOrgID) + "/memberships?" + values.Encode()
	var payload orgMembershipListResponse
	if err := c.json(ctx, http.MethodGet, path, nil, &payload); err != nil {
		return OrganizationMembership{}, false, err
	}
	if len(payload.Data) == 0 {
		return OrganizationMembership{}, false, nil
	}
	membership := orgMembershipFromPayload(payload.Data[0])
	if membership.OrganizationID == "" {
		membership.OrganizationID = providerOrgID
	}
	if membership.User.ID != "" && membership.User.ID != providerUserID {
		return OrganizationMembership{}, false, nil
	}
	return membership, true, nil
}

// RevokeOrgMembership is an idempotent alias for removing a Clerk
// organization membership.
func (c *Client) RevokeOrgMembership(ctx context.Context, providerOrgID, providerUserID string) error {
	return c.DeleteOrgMembership(ctx, providerOrgID, providerUserID)
}

func (c *Client) ListSessions(ctx context.Context, input SessionListInput) ([]Session, error) {
	limit := boundedLimit(input.Limit)
	out := make([]Session, 0)
	for offset := max(input.Offset, 0); ; offset += limit {
		values := url.Values{}
		values.Set("paginated", "true")
		values.Set("limit", strconv.Itoa(limit))
		values.Set("offset", strconv.Itoa(offset))
		if userID := strings.TrimSpace(input.ProviderUserID); userID != "" {
			values.Set("user_id", userID)
		}
		if clientID := strings.TrimSpace(input.ClientID); clientID != "" {
			values.Set("client_id", clientID)
		}
		if status := strings.TrimSpace(input.Status); status != "" {
			values.Set("status", status)
		}

		var payload sessionListResponse
		if err := c.json(ctx, http.MethodGet, "/sessions?"+values.Encode(), nil, &payload); err != nil {
			return nil, err
		}
		for _, item := range payload.Data {
			out = append(out, sessionFromPayload(item))
		}
		if len(payload.Data) < limit || payload.TotalCount <= offset+len(payload.Data) {
			return out, nil
		}
	}
}

func (c *Client) GetSession(ctx context.Context, sessionID string) (Session, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return Session{}, fmt.Errorf("clerk: session id is required")
	}
	var payload clerkSession
	if err := c.json(ctx, http.MethodGet, "/sessions/"+url.PathEscape(sessionID), nil, &payload); err != nil {
		return Session{}, err
	}
	return sessionFromPayload(payload), nil
}

func (c *Client) RevokeSession(ctx context.Context, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("clerk: session id is required")
	}
	err := c.json(ctx, http.MethodPost, "/sessions/"+url.PathEscape(sessionID)+"/revoke", nil, nil)
	if IsNotFound(err) {
		return nil
	}
	return err
}

func boundedLimit(limit int) int {
	if limit <= 0 {
		return 100
	}
	if limit > 500 {
		return 500
	}
	return limit
}
