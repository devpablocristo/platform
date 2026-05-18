package users

import (
	"context"
	"fmt"
	"testing"

	userdomain "github.com/devpablocristo/core/saas/go/users/usecases/domain"
)

func TestGetMeLoadsUserWhenPresent(t *testing.T) {
	t.Parallel()

	repo := &usersRepo{
		user: userdomain.User{ID: "user-1", ExternalID: "ext-1", Email: "a@example.com"},
	}
	usecases := NewUseCases(repo)

	me, err := usecases.GetMe(context.Background(), "org-1", "ext-1", "admin", []string{"tools:read"})
	if err != nil {
		t.Fatalf("GetMe returned error: %v", err)
	}
	if me.User == nil || me.User.ID != "user-1" {
		t.Fatalf("unexpected me profile: %#v", me)
	}
}

func TestEnsureOrgMatch(t *testing.T) {
	t.Parallel()

	if err := EnsureOrgMatch("org-1", "org-1"); err != nil {
		t.Fatalf("did not expect error: %v", err)
	}
	if err := EnsureOrgMatch("org-1", "org-2"); err == nil {
		t.Fatal("expected cross-org error")
	}
}

type usersRepo struct {
	user userdomain.User
}

func (r *usersRepo) FindUserByExternalID(context.Context, string) (userdomain.User, bool, error) {
	return r.user, true, nil
}
func (r *usersRepo) UpsertUser(context.Context, string, string, string, *string) (userdomain.User, error) {
	return r.user, nil
}
func (r *usersRepo) UpsertOrg(context.Context, string, string) (string, error) { return "org-1", nil }
func (r *usersRepo) UpsertOrgMember(context.Context, string, string, string) (userdomain.OrgMember, error) {
	return userdomain.OrgMember{ID: "mem-1", UserID: r.user.ID}, nil
}
func (r *usersRepo) ListOrgMembers(context.Context, string) ([]userdomain.OrgMember, error) {
	return nil, nil
}
func (r *usersRepo) ListAPIKeys(context.Context, string) ([]userdomain.APIKey, error) {
	return nil, nil
}
func (r *usersRepo) CreateAPIKey(context.Context, string, string, []string) (userdomain.CreatedAPIKey, error) {
	return userdomain.CreatedAPIKey{}, nil
}
func (r *usersRepo) DeleteAPIKey(context.Context, string, string) error { return nil }
func (r *usersRepo) RotateAPIKey(context.Context, string, string) (userdomain.RotatedAPIKey, error) {
	return userdomain.RotatedAPIKey{}, nil
}
func (r *usersRepo) SoftDeleteUser(context.Context, string) error                   { return nil }
func (r *usersRepo) RemoveMembership(context.Context, string, string, string) error { return nil }

var _ Repository = (*usersRepo)(nil)
var _ = fmt.Sprintf
