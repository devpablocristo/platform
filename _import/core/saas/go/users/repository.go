package users

import (
	"context"

	userdomain "github.com/devpablocristo/core/saas/go/users/usecases/domain"
)

// Repository define el puerto de persistencia del contexto users.
type Repository interface {
	FindUserByExternalID(context.Context, string) (userdomain.User, bool, error)
	UpsertUser(context.Context, string, string, string, *string) (userdomain.User, error)
	UpsertOrg(context.Context, string, string) (string, error)
	UpsertOrgMember(context.Context, string, string, string) (userdomain.OrgMember, error)
	ListOrgMembers(context.Context, string) ([]userdomain.OrgMember, error)
	ListAPIKeys(context.Context, string) ([]userdomain.APIKey, error)
	CreateAPIKey(context.Context, string, string, []string) (userdomain.CreatedAPIKey, error)
	DeleteAPIKey(context.Context, string, string) error
	RotateAPIKey(context.Context, string, string) (userdomain.RotatedAPIKey, error)
	SoftDeleteUser(context.Context, string) error
	RemoveMembership(context.Context, string, string, string) error
}
