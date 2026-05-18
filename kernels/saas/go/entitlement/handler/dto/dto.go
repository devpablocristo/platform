package dto

import entitlementdomain "github.com/devpablocristo/core/saas/go/entitlement/usecases/domain"

type NormalizePlanRequest struct {
	Plan entitlementdomain.PlanCode `json:"plan"`
}

type NormalizePlanResponse struct {
	Plan entitlementdomain.PlanCode `json:"plan"`
}

type DefaultHardLimitsResponse struct {
	Limits entitlementdomain.HardLimits `json:"limits"`
}

type CanUseRequest struct {
	Current int `json:"current"`
	Limit   int `json:"limit"`
}

type BoolResponse struct {
	Allowed bool `json:"allowed"`
}
