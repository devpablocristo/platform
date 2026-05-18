package users

import (
	"context"
	"fmt"
	"strings"

	userdomain "github.com/devpablocristo/core/saas/go/users/usecases/domain"
)

type UseCases struct {
	repo Repository
}

func NewUseCases(repo Repository) *UseCases {
	return &UseCases{repo: repo}
}

func (u *UseCases) GetMe(ctx context.Context, orgID, externalID, role string, scopes []string) (userdomain.MeProfile, error) {
	me := userdomain.MeProfile{
		OrgID:      strings.TrimSpace(orgID),
		ExternalID: strings.TrimSpace(externalID),
		Role:       strings.TrimSpace(role),
		Scopes:     append([]string(nil), scopes...),
	}
	if me.ExternalID == "" {
		return me, nil
	}
	user, ok, err := u.repo.FindUserByExternalID(ctx, me.ExternalID)
	if err != nil {
		return userdomain.MeProfile{}, err
	}
	if ok {
		me.User = &user
	}
	return me, nil
}

func (u *UseCases) ListOrgMembers(ctx context.Context, orgID string) ([]userdomain.OrgMember, error) {
	return u.repo.ListOrgMembers(ctx, strings.TrimSpace(orgID))
}

func (u *UseCases) ListAPIKeys(ctx context.Context, orgID string) ([]userdomain.APIKey, error) {
	return u.repo.ListAPIKeys(ctx, strings.TrimSpace(orgID))
}

func (u *UseCases) CreateAPIKey(ctx context.Context, orgID, name string, scopes []string) (userdomain.CreatedAPIKey, error) {
	return u.repo.CreateAPIKey(ctx, strings.TrimSpace(orgID), strings.TrimSpace(name), append([]string(nil), scopes...))
}

func (u *UseCases) DeleteAPIKey(ctx context.Context, orgID, keyID string) error {
	return u.repo.DeleteAPIKey(ctx, strings.TrimSpace(orgID), strings.TrimSpace(keyID))
}

func (u *UseCases) RotateAPIKey(ctx context.Context, orgID, keyID string) (userdomain.RotatedAPIKey, error) {
	return u.repo.RotateAPIKey(ctx, strings.TrimSpace(orgID), strings.TrimSpace(keyID))
}

func (u *UseCases) SyncUser(ctx context.Context, externalID, email, name string, avatarURL *string) (userdomain.User, error) {
	return u.repo.UpsertUser(ctx, strings.TrimSpace(externalID), strings.TrimSpace(email), strings.TrimSpace(name), avatarURL)
}

func (u *UseCases) SyncOrganization(ctx context.Context, externalID, orgName string) (string, error) {
	return u.repo.UpsertOrg(ctx, strings.TrimSpace(externalID), strings.TrimSpace(orgName))
}

func (u *UseCases) SyncMembership(ctx context.Context, orgID, userExternalID, email, name string, avatarURL *string, role string) (userdomain.OrgMember, error) {
	user, err := u.repo.UpsertUser(ctx, strings.TrimSpace(userExternalID), strings.TrimSpace(email), strings.TrimSpace(name), avatarURL)
	if err != nil {
		return userdomain.OrgMember{}, err
	}
	member, err := u.repo.UpsertOrgMember(ctx, strings.TrimSpace(orgID), user.ID, strings.TrimSpace(role))
	if err != nil {
		return userdomain.OrgMember{}, err
	}
	member.User = user
	return member, nil
}

func (u *UseCases) SoftDeleteUser(ctx context.Context, externalID string) error {
	return u.repo.SoftDeleteUser(ctx, strings.TrimSpace(externalID))
}

func (u *UseCases) RemoveMembership(ctx context.Context, userExternalID, orgExternalID, orgName string) error {
	return u.repo.RemoveMembership(ctx, strings.TrimSpace(userExternalID), strings.TrimSpace(orgExternalID), strings.TrimSpace(orgName))
}

func EnsureOrgMatch(pathOrgID, tokenOrgID string) error {
	if strings.TrimSpace(pathOrgID) == "" {
		return fmt.Errorf("invalid org id")
	}
	if strings.TrimSpace(tokenOrgID) == "" {
		return fmt.Errorf("missing org context")
	}
	if strings.TrimSpace(pathOrgID) != strings.TrimSpace(tokenOrgID) {
		return fmt.Errorf("cross-org access denied")
	}
	return nil
}
