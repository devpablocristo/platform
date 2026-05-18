package entitlement

import entitlementdomain "github.com/devpablocristo/core/saas/go/entitlement/usecases/domain"

// UseCases expone reglas reutilizables de plan y límites.
type UseCases struct{}

func NewUseCases() *UseCases {
	return &UseCases{}
}

func (u *UseCases) NormalizePlan(raw string) entitlementdomain.PlanCode {
	return NormalizePlan(raw)
}

func (u *UseCases) DefaultHardLimits(plan entitlementdomain.PlanCode) entitlementdomain.HardLimits {
	return DefaultHardLimits(plan)
}

func (u *UseCases) CanUse(current, limit int) bool {
	return CanUse(current, limit)
}
