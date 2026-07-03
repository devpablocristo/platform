package clerk

import (
	"bytes"
	"encoding/json"
	"strings"
)

type User struct {
	ID    string
	Email string
}

type Organization struct {
	ID   string
	Name string
	Slug string
}

type OrganizationMembership struct {
	ID             string
	Role           string
	OrganizationID string
	Organization   Organization
}

type Invitation struct {
	ID     string
	Status string
}

type CreateUserInput struct {
	Email string
}

type OrganizationInput struct {
	Name string
	Slug string
}

type ListInput struct {
	Limit  int
	Offset int
}

type OrgMembershipInput struct {
	ProviderOrgID  string
	ProviderUserID string
	Role           string
}

type OrgInvitationInput struct {
	ProviderOrgID         string
	Email                 string
	Role                  string
	InviterProviderUserID string
	RedirectURL           string
}

type userListResponse struct {
	Data       []clerkUser `json:"data"`
	TotalCount int         `json:"total_count"`
}

func (r *userListResponse) UnmarshalJSON(raw []byte) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil
	}
	if raw[0] == '[' {
		return json.Unmarshal(raw, &r.Data)
	}
	type response userListResponse
	return json.Unmarshal(raw, (*response)(r))
}

type organizationListResponse struct {
	Data       []clerkOrganization `json:"data"`
	TotalCount int                 `json:"total_count"`
}

func (r *organizationListResponse) UnmarshalJSON(raw []byte) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil
	}
	if raw[0] == '[' {
		return json.Unmarshal(raw, &r.Data)
	}
	type response organizationListResponse
	return json.Unmarshal(raw, (*response)(r))
}

type orgMembershipListResponse struct {
	Data       []clerkOrgMembership `json:"data"`
	TotalCount int                  `json:"total_count"`
}

func (r *orgMembershipListResponse) UnmarshalJSON(raw []byte) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil
	}
	if raw[0] == '[' {
		return json.Unmarshal(raw, &r.Data)
	}
	type response orgMembershipListResponse
	return json.Unmarshal(raw, (*response)(r))
}

type clerkUser struct {
	ID                    string              `json:"id"`
	EmailAddress          string              `json:"email_address"`
	PrimaryEmailAddressID string              `json:"primary_email_address_id"`
	EmailAddresses        []clerkEmailAddress `json:"email_addresses"`
}

type clerkEmailAddress struct {
	ID           string `json:"id"`
	EmailAddress string `json:"email_address"`
}

type clerkOrganization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type clerkOrgMembership struct {
	ID             string            `json:"id"`
	Role           string            `json:"role"`
	OrganizationID string            `json:"organization_id"`
	Organization   clerkOrganization `json:"organization"`
}

type clerkInvitation struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func userFromPayload(user clerkUser) User {
	return User{
		ID:    strings.TrimSpace(user.ID),
		Email: user.primaryEmail(),
	}
}

func (u clerkUser) primaryEmail() string {
	if email := strings.TrimSpace(strings.ToLower(u.EmailAddress)); email != "" {
		return email
	}
	if u.PrimaryEmailAddressID != "" {
		for _, item := range u.EmailAddresses {
			if strings.TrimSpace(item.ID) == strings.TrimSpace(u.PrimaryEmailAddressID) {
				return strings.TrimSpace(strings.ToLower(item.EmailAddress))
			}
		}
	}
	for _, item := range u.EmailAddresses {
		if email := strings.TrimSpace(strings.ToLower(item.EmailAddress)); email != "" {
			return email
		}
	}
	return strings.TrimSpace(strings.ToLower(u.ID))
}

func organizationFromPayload(org clerkOrganization) Organization {
	return Organization{
		ID:   strings.TrimSpace(org.ID),
		Name: strings.TrimSpace(org.Name),
		Slug: strings.TrimSpace(org.Slug),
	}
}

func orgMembershipFromPayload(item clerkOrgMembership) OrganizationMembership {
	org := organizationFromPayload(item.Organization)
	if org.ID == "" {
		org.ID = strings.TrimSpace(item.OrganizationID)
	}
	return OrganizationMembership{
		ID:             strings.TrimSpace(item.ID),
		Role:           strings.TrimSpace(item.Role),
		OrganizationID: firstNonEmpty(org.ID, item.OrganizationID),
		Organization:   org,
	}
}
