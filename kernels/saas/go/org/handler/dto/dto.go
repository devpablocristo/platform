package dto

import orgdomain "github.com/devpablocristo/core/saas/go/org/usecases/domain"

type ResolvePrincipalRequest struct {
	APIKeyHash string `json:"api_key_hash"`
}

type ResolvePrincipalResponse struct {
	Principal orgdomain.Principal `json:"principal"`
}
