package governanceclient

import (
	"github.com/devpablocristo/core/http/go/httpclient"
)

// RequestOption modifica una llamada individual sin tocar la config global
// del Client. Es la forma de pasar scope de tenant e idempotencia per-call
// sin acoplar al consumer al package httpclient ni al header interno de Nexus.
type RequestOption func(*requestOptions)

type requestOptions struct {
	tenantID       string
	idempotencyKey string
}

// WithTenantID scopea la llamada a un tenant. Sirve para callers
// multi-tenant que operan para varios tenants con una unica API key con scope
// nexus:cross_org. Sin esta opción, Nexus usa el tenant del principal de la
// API key.
func WithTenantID(tenantID string) RequestOption {
	return func(o *requestOptions) { o.tenantID = tenantID }
}

// WithOrgID mantiene compatibilidad con callers externos viejos. Código nuevo
// debe usar WithTenantID.
func WithOrgID(orgID string) RequestOption {
	return WithTenantID(orgID)
}

// WithIdempotencyKey setea el header Idempotency-Key en la llamada.
func WithIdempotencyKey(key string) RequestOption {
	return func(o *requestOptions) { o.idempotencyKey = key }
}

func collectOptions(opts ...RequestOption) []httpclient.RequestOption {
	var ro requestOptions
	for _, o := range opts {
		if o != nil {
			o(&ro)
		}
	}
	var out []httpclient.RequestOption
	if ro.tenantID != "" {
		out = append(out, httpclient.WithHeader("X-Org-ID", ro.tenantID))
	}
	if ro.idempotencyKey != "" {
		out = append(out, httpclient.WithIdempotencyKey(ro.idempotencyKey))
	}
	return out
}
