// Package health provides standard /healthz and /readyz endpoints for net/http services.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// ReadinessCheck verifica que un servicio esté listo.
type ReadinessCheck func(context.Context) error

// RegisterEndpoints registra /healthz (liveness) y /readyz (readiness) en un ServeMux.
func RegisterEndpoints(mux *http.ServeMux, ready ReadinessCheck) {
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if ready != nil {
			ctx, cancel := context.WithTimeout(r.Context(), time.Second)
			defer cancel()
			if err := ready(ctx); err != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
}

// ComposeChecks combina varias verificaciones en una sola.
func ComposeChecks(checks ...ReadinessCheck) ReadinessCheck {
	return func(ctx context.Context) error {
		for _, check := range checks {
			if check == nil {
				continue
			}
			if err := check(ctx); err != nil {
				return err
			}
		}
		return nil
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
