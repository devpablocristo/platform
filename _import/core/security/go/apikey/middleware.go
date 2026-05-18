package apikey

import (
	"context"
	"net/http"

	"github.com/devpablocristo/core/http/go/httpjson"
)

// Middleware aplica autenticación por API key y excluye health probes.
// Adapter HTTP — usa httpjson para respuestas de error.
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			next.ServeHTTP(w, r)
			return
		}
		rawKey := ExtractKey(r)
		principal, ok := a.Authenticate(rawKey)
		if !ok {
			httpjson.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "valid api key required")
			return
		}
		ctx := context.WithValue(r.Context(), principalContextKey, principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
