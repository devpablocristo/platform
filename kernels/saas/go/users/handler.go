package users

import (
	"context"

	"github.com/devpablocristo/core/saas/go/users/handler/dto"
)

// Handler expone un adapter de aplicación listo para transporte externo.
type Handler struct {
	usecases *UseCases
}

func NewHandler(usecases *UseCases) *Handler {
	return &Handler{usecases: usecases}
}

func (h *Handler) GetMe(ctx context.Context, input dto.GetMeRequest) (dto.GetMeResponse, error) {
	profile, err := h.usecases.GetMe(ctx, input.OrgID, input.ExternalID, input.Role, input.Scopes)
	if err != nil {
		return dto.GetMeResponse{}, err
	}
	return dto.GetMeResponse{Profile: profile}, nil
}

func (h *Handler) ListOrgMembers(ctx context.Context, input dto.OrgRequest) (dto.ListOrgMembersResponse, error) {
	items, err := h.usecases.ListOrgMembers(ctx, input.OrgID)
	if err != nil {
		return dto.ListOrgMembersResponse{}, err
	}
	return dto.ListOrgMembersResponse{Members: items}, nil
}

func (h *Handler) ListAPIKeys(ctx context.Context, input dto.OrgRequest) (dto.ListAPIKeysResponse, error) {
	items, err := h.usecases.ListAPIKeys(ctx, input.OrgID)
	if err != nil {
		return dto.ListAPIKeysResponse{}, err
	}
	return dto.ListAPIKeysResponse{APIKeys: items}, nil
}

func (h *Handler) CreateAPIKey(ctx context.Context, input dto.CreateAPIKeyRequest) (dto.CreateAPIKeyResponse, error) {
	item, err := h.usecases.CreateAPIKey(ctx, input.OrgID, input.Name, input.Scopes)
	if err != nil {
		return dto.CreateAPIKeyResponse{}, err
	}
	return dto.CreateAPIKeyResponse{Created: item}, nil
}

func (h *Handler) DeleteAPIKey(ctx context.Context, input dto.DeleteAPIKeyRequest) error {
	return h.usecases.DeleteAPIKey(ctx, input.OrgID, input.KeyID)
}

func (h *Handler) RotateAPIKey(ctx context.Context, input dto.DeleteAPIKeyRequest) (dto.RotateAPIKeyResponse, error) {
	item, err := h.usecases.RotateAPIKey(ctx, input.OrgID, input.KeyID)
	if err != nil {
		return dto.RotateAPIKeyResponse{}, err
	}
	return dto.RotateAPIKeyResponse{Rotated: item}, nil
}

func (h *Handler) SyncUser(ctx context.Context, input dto.SyncUserRequest) (dto.SyncUserResponse, error) {
	item, err := h.usecases.SyncUser(ctx, input.ExternalID, input.Email, input.Name, input.AvatarURL)
	if err != nil {
		return dto.SyncUserResponse{}, err
	}
	return dto.SyncUserResponse{User: item}, nil
}

func (h *Handler) SyncOrganization(ctx context.Context, input dto.SyncOrganizationRequest) (dto.SyncOrganizationResponse, error) {
	orgID, err := h.usecases.SyncOrganization(ctx, input.ExternalID, input.Name)
	if err != nil {
		return dto.SyncOrganizationResponse{}, err
	}
	return dto.SyncOrganizationResponse{OrgID: orgID}, nil
}

func (h *Handler) SyncMembership(ctx context.Context, input dto.SyncMembershipRequest) (dto.SyncMembershipResponse, error) {
	item, err := h.usecases.SyncMembership(ctx, input.OrgID, input.UserExternalID, input.Email, input.Name, input.AvatarURL, input.Role)
	if err != nil {
		return dto.SyncMembershipResponse{}, err
	}
	return dto.SyncMembershipResponse{Member: item}, nil
}

func (h *Handler) SoftDeleteUser(ctx context.Context, input dto.SoftDeleteUserRequest) error {
	return h.usecases.SoftDeleteUser(ctx, input.ExternalID)
}

func (h *Handler) RemoveMembership(ctx context.Context, input dto.RemoveMembershipRequest) error {
	return h.usecases.RemoveMembership(ctx, input.UserExternalID, input.OrgExternalID, input.OrgName)
}
