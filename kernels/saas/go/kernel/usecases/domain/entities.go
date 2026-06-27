package domain

import "time"

type Org struct {
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Tenant is a source-compatible alias for legacy consumers; new contracts use Org.
type Tenant = Org

type Membership struct {
	OrgID    string    `json:"org_id"`
	TenantID string    `json:"-"`
	UserID   string    `json:"user_id"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

type APIKey struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	TenantID  string    `json:"-"`
	Name      string    `json:"name"`
	Scopes    []string  `json:"scopes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Principal struct {
	OrgID      string   `json:"org_id"`
	TenantID   string   `json:"-"`
	Actor      string   `json:"actor,omitempty"`
	Role       string   `json:"role,omitempty"`
	Scopes     []string `json:"scopes,omitempty"`
	AuthMethod string   `json:"auth_method,omitempty"`
}

func (p Principal) EffectiveOrgID() string {
	if p.OrgID != "" {
		return p.OrgID
	}
	return p.TenantID
}

type PlanCode string

const (
	PlanStarter    PlanCode = "starter"
	PlanGrowth     PlanCode = "growth"
	PlanEnterprise PlanCode = "enterprise"
)

type BillingStatus string

const (
	BillingTrialing BillingStatus = "trialing"
	BillingActive   BillingStatus = "active"
	BillingPastDue  BillingStatus = "past_due"
	BillingCanceled BillingStatus = "canceled"
	BillingUnpaid   BillingStatus = "unpaid"
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

type OrgBilling struct {
	OrgID              string        `json:"org_id"`
	TenantID           string        `json:"-"`
	PlanCode           PlanCode      `json:"plan_code"`
	HardLimits         HardLimits    `json:"hard_limits"`
	BillingStatus      BillingStatus `json:"billing_status"`
	PastDueSince       *time.Time    `json:"past_due_since,omitempty"`
	ProviderCustomerID *string       `json:"provider_customer_id,omitempty"`
	ProviderContractID *string       `json:"provider_contract_id,omitempty"`
	UpdatedAt          time.Time     `json:"updated_at"`
	CreatedAt          time.Time     `json:"created_at"`
}

func (b OrgBilling) EffectiveOrgID() string {
	if b.OrgID != "" {
		return b.OrgID
	}
	return b.TenantID
}

// TenantBilling is kept as a source-compatible alias while consumers migrate to OrgBilling.
type TenantBilling = OrgBilling
