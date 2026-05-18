package dto

import identitydomain "github.com/devpablocristo/core/saas/go/identity/usecases/domain"

type VerifyRequest struct {
	Token string `json:"token"`
}

type VerifyResponse struct {
	Principal identitydomain.Principal `json:"principal"`
}

type TokenHeaderRequest struct {
	Authorization string `json:"authorization"`
}

type APIKeyHeaderRequest struct {
	Authorization string `json:"authorization"`
	XAPIKey       string `json:"x_api_key"`
}

type TokenHeaderResponse struct {
	Token string `json:"token,omitempty"`
	Found bool   `json:"found"`
}
