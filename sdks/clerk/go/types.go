package clerk

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"
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
	User           User
	Permissions    []string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Invitation struct {
	ID             string
	OrganizationID string
	Email          string
	Role           string
	Status         string
	ExpiresAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type SessionActivity struct {
	DeviceType     string
	IsMobile       bool
	BrowserName    string
	BrowserVersion string
	IPAddress      string
	City           string
	Country        string
}

type Session struct {
	ID                       string
	UserID                   string
	ClientID                 string
	Status                   string
	LastActiveOrganizationID string
	LatestActivity           *SessionActivity
	CreatedAt                time.Time
	UpdatedAt                time.Time
	LastActiveAt             time.Time
	ExpiresAt                time.Time
	AbandonAt                time.Time
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
	ExpiresInDays         int
}

type OrgInvitationListInput struct {
	ListInput
	Statuses []string
	Email    string
	OrderBy  string
}

type SessionListInput struct {
	ListInput
	ProviderUserID string
	ClientID       string
	Status         string
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

type invitationListResponse struct {
	Data       []clerkInvitation `json:"data"`
	TotalCount int               `json:"total_count"`
}

func (r *invitationListResponse) UnmarshalJSON(raw []byte) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil
	}
	if raw[0] == '[' {
		return json.Unmarshal(raw, &r.Data)
	}
	type response invitationListResponse
	return json.Unmarshal(raw, (*response)(r))
}

type sessionListResponse struct {
	Data       []clerkSession `json:"data"`
	TotalCount int            `json:"total_count"`
}

func (r *sessionListResponse) UnmarshalJSON(raw []byte) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil
	}
	if raw[0] == '[' {
		return json.Unmarshal(raw, &r.Data)
	}
	type response sessionListResponse
	return json.Unmarshal(raw, (*response)(r))
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
	PublicUserData clerkPublicUser   `json:"public_user_data"`
	Permissions    []string          `json:"permissions"`
	CreatedAt      int64             `json:"created_at"`
	UpdatedAt      int64             `json:"updated_at"`
}

type clerkPublicUser struct {
	UserID     string `json:"user_id"`
	Identifier string `json:"identifier"`
}

type clerkInvitation struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	EmailAddress   string `json:"email_address"`
	Role           string `json:"role"`
	Status         string `json:"status"`
	ExpiresAt      *int64 `json:"expires_at"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

type clerkSessionActivity struct {
	DeviceType     *string `json:"device_type"`
	IsMobile       bool    `json:"is_mobile"`
	BrowserName    *string `json:"browser_name"`
	BrowserVersion *string `json:"browser_version"`
	IPAddress      *string `json:"ip_address"`
	City           *string `json:"city"`
	Country        *string `json:"country"`
}

type clerkSession struct {
	ID                       string                `json:"id"`
	UserID                   string                `json:"user_id"`
	ClientID                 string                `json:"client_id"`
	Status                   string                `json:"status"`
	LastActiveOrganizationID string                `json:"last_active_organization_id"`
	LatestActivity           *clerkSessionActivity `json:"latest_activity"`
	CreatedAt                int64                 `json:"created_at"`
	UpdatedAt                int64                 `json:"updated_at"`
	LastActiveAt             int64                 `json:"last_active_at"`
	ExpireAt                 int64                 `json:"expire_at"`
	AbandonAt                int64                 `json:"abandon_at"`
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
		User: User{
			ID:    strings.TrimSpace(item.PublicUserData.UserID),
			Email: strings.TrimSpace(strings.ToLower(item.PublicUserData.Identifier)),
		},
		Permissions: append([]string(nil), item.Permissions...),
		CreatedAt:   millisToTime(item.CreatedAt),
		UpdatedAt:   millisToTime(item.UpdatedAt),
	}
}

func invitationFromPayload(item clerkInvitation) Invitation {
	return Invitation{
		ID:             strings.TrimSpace(item.ID),
		OrganizationID: strings.TrimSpace(item.OrganizationID),
		Email:          strings.TrimSpace(strings.ToLower(item.EmailAddress)),
		Role:           strings.TrimSpace(item.Role),
		Status:         strings.TrimSpace(item.Status),
		ExpiresAt:      millisToTimePtr(item.ExpiresAt),
		CreatedAt:      millisToTime(item.CreatedAt),
		UpdatedAt:      millisToTime(item.UpdatedAt),
	}
}

func sessionFromPayload(item clerkSession) Session {
	session := Session{
		ID:                       strings.TrimSpace(item.ID),
		UserID:                   strings.TrimSpace(item.UserID),
		ClientID:                 strings.TrimSpace(item.ClientID),
		Status:                   strings.TrimSpace(item.Status),
		LastActiveOrganizationID: strings.TrimSpace(item.LastActiveOrganizationID),
		CreatedAt:                millisToTime(item.CreatedAt),
		UpdatedAt:                millisToTime(item.UpdatedAt),
		LastActiveAt:             millisToTime(item.LastActiveAt),
		ExpiresAt:                millisToTime(item.ExpireAt),
		AbandonAt:                millisToTime(item.AbandonAt),
	}
	if item.LatestActivity != nil {
		session.LatestActivity = &SessionActivity{
			DeviceType:     stringValue(item.LatestActivity.DeviceType),
			IsMobile:       item.LatestActivity.IsMobile,
			BrowserName:    stringValue(item.LatestActivity.BrowserName),
			BrowserVersion: stringValue(item.LatestActivity.BrowserVersion),
			IPAddress:      stringValue(item.LatestActivity.IPAddress),
			City:           stringValue(item.LatestActivity.City),
			Country:        stringValue(item.LatestActivity.Country),
		}
	}
	return session
}

func millisToTime(value int64) time.Time {
	if value <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(value).UTC()
}

func millisToTimePtr(value *int64) *time.Time {
	if value == nil || *value <= 0 {
		return nil
	}
	result := time.UnixMilli(*value).UTC()
	return &result
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
