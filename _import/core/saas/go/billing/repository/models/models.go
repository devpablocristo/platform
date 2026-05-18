package models

import (
	"time"

	billingdomain "github.com/devpablocristo/core/saas/go/billing/usecases/domain"
)

type HardLimits struct {
	ToolsMax           int `json:"tools_max"`
	RunRPM             int `json:"run_rpm"`
	AuditRetentionDays int `json:"audit_retention_days"`
}

type UsageCounters struct {
	APICalls        int64 `json:"api_calls"`
	EventsIngested  int64 `json:"events_ingested"`
	IncidentsOpened int64 `json:"incidents_opened"`
	ActionsExecuted int64 `json:"actions_executed"`
}

type UsageSummary struct {
	Period   string        `json:"period"`
	Counters UsageCounters `json:"counters"`
}

type TenantBilling struct {
	TenantID           string                      `json:"tenant_id"`
	PlanCode           billingdomain.PlanCode      `json:"plan_code"`
	HardLimits         HardLimits                  `json:"hard_limits"`
	BillingStatus      billingdomain.BillingStatus `json:"billing_status"`
	PastDueSince       *time.Time                  `json:"past_due_since,omitempty"`
	ProviderCustomerID *string                     `json:"provider_customer_id,omitempty"`
	ProviderContractID *string                     `json:"provider_contract_id,omitempty"`
	UpdatedAt          time.Time                   `json:"updated_at"`
	CreatedAt          time.Time                   `json:"created_at"`
}

func FromDomain(item billingdomain.TenantBilling) TenantBilling {
	return TenantBilling{
		TenantID:           item.TenantID,
		PlanCode:           item.PlanCode,
		HardLimits:         HardLimits(item.HardLimits),
		BillingStatus:      item.BillingStatus,
		PastDueSince:       item.PastDueSince,
		ProviderCustomerID: item.ProviderCustomerID,
		ProviderContractID: item.ProviderContractID,
		UpdatedAt:          item.UpdatedAt,
		CreatedAt:          item.CreatedAt,
	}
}

func (m TenantBilling) ToDomain() billingdomain.TenantBilling {
	return billingdomain.TenantBilling{
		TenantID:           m.TenantID,
		PlanCode:           m.PlanCode,
		HardLimits:         billingdomain.HardLimits(m.HardLimits),
		BillingStatus:      m.BillingStatus,
		PastDueSince:       m.PastDueSince,
		ProviderCustomerID: m.ProviderCustomerID,
		ProviderContractID: m.ProviderContractID,
		UpdatedAt:          m.UpdatedAt,
		CreatedAt:          m.CreatedAt,
	}
}
