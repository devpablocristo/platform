package dto

import userdomain "github.com/devpablocristo/core/saas/go/users/usecases/domain"

type OrgRequest struct {
	OrgID string `json:"org_id"`
}

type GetMeRequest struct {
	OrgID      string   `json:"org_id"`
	ExternalID string   `json:"external_id"`
	Role       string   `json:"role"`
	Scopes     []string `json:"scopes,omitempty"`
}

type GetMeResponse struct {
	Profile userdomain.MeProfile `json:"profile"`
}

type ListOrgMembersResponse struct {
	Members []userdomain.OrgMember `json:"members"`
}

type ListAPIKeysResponse struct {
	APIKeys []userdomain.APIKey `json:"api_keys"`
}

type CreateAPIKeyRequest struct {
	OrgID  string   `json:"org_id"`
	Name   string   `json:"name"`
	Scopes []string `json:"scopes,omitempty"`
}

type CreateAPIKeyResponse struct {
	Created userdomain.CreatedAPIKey `json:"created"`
}

type DeleteAPIKeyRequest struct {
	OrgID string `json:"org_id"`
	KeyID string `json:"key_id"`
}

type RotateAPIKeyResponse struct {
	Rotated userdomain.RotatedAPIKey `json:"rotated"`
}

type SyncUserRequest struct {
	ExternalID string  `json:"external_id"`
	Email      string  `json:"email"`
	Name       string  `json:"name"`
	AvatarURL  *string `json:"avatar_url,omitempty"`
}

type SyncUserResponse struct {
	User userdomain.User `json:"user"`
}

type SyncOrganizationRequest struct {
	ExternalID string `json:"external_id"`
	Name       string `json:"name"`
}

type SyncOrganizationResponse struct {
	OrgID string `json:"org_id"`
}

type SyncMembershipRequest struct {
	OrgID          string  `json:"org_id"`
	UserExternalID string  `json:"user_external_id"`
	Email          string  `json:"email"`
	Name           string  `json:"name"`
	AvatarURL      *string `json:"avatar_url,omitempty"`
	Role           string  `json:"role"`
}

type SyncMembershipResponse struct {
	Member userdomain.OrgMember `json:"member"`
}

type SoftDeleteUserRequest struct {
	ExternalID string `json:"external_id"`
}

type RemoveMembershipRequest struct {
	UserExternalID string `json:"user_external_id"`
	OrgExternalID  string `json:"org_external_id"`
	OrgName        string `json:"org_name"`
}
