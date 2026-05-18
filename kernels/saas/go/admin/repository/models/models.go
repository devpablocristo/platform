package models

import (
	"time"

	admindomain "github.com/devpablocristo/core/saas/go/admin/usecases/domain"
)

type TenantSettings = admindomain.TenantSettings
type AdminActivityEvent = admindomain.AdminActivityEvent
type ProtectedResource = admindomain.ProtectedResource
type RestoreEvidence = admindomain.RestoreEvidence

type TenantStatus = admindomain.TenantStatus

const (
	TenantStatusActive    = admindomain.TenantStatusActive
	TenantStatusSuspended = admindomain.TenantStatusSuspended
	TenantStatusDeleted   = admindomain.TenantStatusDeleted
)

type LifecycleRecord struct {
	TenantID  string                   `json:"tenant_id"`
	Status    admindomain.TenantStatus `json:"status"`
	DeletedAt *time.Time               `json:"deleted_at,omitempty"`
	UpdatedBy *string                  `json:"updated_by,omitempty"`
}
