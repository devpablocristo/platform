package session

import (
	"net/http"

	"github.com/devpablocristo/core/http/go/httperr"
	saasmiddleware "github.com/devpablocristo/core/saas/go/middleware"
)

// RegisterProtected registra GET /session en el mux, detrás de authMW.
// Responde 200 con el Principal del kernel (tenant_id, actor, role, scopes, auth_method).
// Sin campos de producto: los productos pueden envolver esta respuesta o registrar otro handler.
func RegisterProtected(mux *http.ServeMux, authMW func(http.Handler) http.Handler) {
	if mux == nil || authMW == nil {
		return
	}
	mux.Handle("GET /session", authMW(http.HandlerFunc(HandleSession)))
}

// HandleSession escribe el Principal autenticado como JSON o 401 si no hay principal.
func HandleSession(w http.ResponseWriter, r *http.Request) {
	p, ok := saasmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		httperr.Unauthorized(w, "principal not found")
		return
	}
	httperr.WriteJSON(w, http.StatusOK, p)
}
