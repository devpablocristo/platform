package dto

import (
	"time"

	usage "github.com/devpablocristo/core/saas/go/usagemetering/usecases/domain"
)

type IncrementRequest struct {
	TenantID string        `json:"tenant_id"`
	Counter  usage.Counter `json:"counter"`
}

type CurrentPeriodRequest struct {
	Now time.Time `json:"now,omitempty"`
}

type CurrentPeriodResponse struct {
	Period string `json:"period"`
}
