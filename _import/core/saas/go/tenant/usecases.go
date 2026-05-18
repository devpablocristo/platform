package tenant

import tenantdomain "github.com/devpablocristo/core/saas/go/tenant/usecases/domain"

// UseCases agrupa helpers de tenancy y memberships.
type UseCases struct{}

func NewUseCases() *UseCases {
	return &UseCases{}
}

func (u *UseCases) NormalizeSlug(raw string) string {
	return NormalizeSlug(raw)
}

func (u *UseCases) NormalizeRole(raw string) string {
	return NormalizeRole(raw)
}

func (u *UseCases) NewMembership(tenantID, userID, role string) tenantdomain.Membership {
	return NewMembership(tenantID, userID, role)
}
