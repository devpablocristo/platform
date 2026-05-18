package domain

import kerneldomain "github.com/devpablocristo/core/saas/go/kernel/usecases/domain"

type PlanCode = kerneldomain.PlanCode
type BillingStatus = kerneldomain.BillingStatus
type HardLimits = kerneldomain.HardLimits
type UsageCounters = kerneldomain.UsageCounters
type UsageSummary = kerneldomain.UsageSummary
type TenantBilling = kerneldomain.TenantBilling

const (
	PlanStarter    = kerneldomain.PlanStarter
	PlanGrowth     = kerneldomain.PlanGrowth
	PlanEnterprise = kerneldomain.PlanEnterprise

	BillingTrialing = kerneldomain.BillingTrialing
	BillingActive   = kerneldomain.BillingActive
	BillingPastDue  = kerneldomain.BillingPastDue
	BillingCanceled = kerneldomain.BillingCanceled
	BillingUnpaid   = kerneldomain.BillingUnpaid
)
