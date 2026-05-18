package entitlement

import (
	"strings"

	kerneldomain "github.com/devpablocristo/core/saas/go/kernel/usecases/domain"
)

func NormalizePlan(raw string) kerneldomain.PlanCode {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case string(kerneldomain.PlanGrowth):
		return kerneldomain.PlanGrowth
	case string(kerneldomain.PlanEnterprise):
		return kerneldomain.PlanEnterprise
	default:
		return kerneldomain.PlanStarter
	}
}

func DefaultHardLimits(plan kerneldomain.PlanCode) kerneldomain.HardLimits {
	switch NormalizePlan(string(plan)) {
	case kerneldomain.PlanGrowth:
		return kerneldomain.HardLimits{
			ToolsMax:           50,
			RunRPM:             120,
			AuditRetentionDays: 90,
		}
	case kerneldomain.PlanEnterprise:
		return kerneldomain.HardLimits{
			ToolsMax:           250,
			RunRPM:             600,
			AuditRetentionDays: 365,
		}
	default:
		return kerneldomain.HardLimits{
			ToolsMax:           10,
			RunRPM:             30,
			AuditRetentionDays: 30,
		}
	}
}

func CanUse(current, limit int) bool {
	if limit <= 0 {
		return false
	}
	return current < limit
}
