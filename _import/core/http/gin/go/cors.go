// Package ginmw provides reusable Gin middleware and HTTP utilities.
package ginmw

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORSConfig configura el middleware CORS.
type CORSConfig struct {
	// Origins adicionales al default de desarrollo local.
	Origins []string
	// AllowHeaders adicionales a los defaults (Authorization, Content-Type, X-API-KEY).
	AllowHeaders []string
	// MaxAge en segundos (default 86400).
	MaxAge int
}

// DefaultCORSHeaders headers permitidos por default.
var DefaultCORSHeaders = []string{
	"Authorization", "Content-Type", "X-API-KEY", "X-Org-ID", "X-Scopes",
	"X-Request-Id", "X-Internal-Service-Token",
}

// NewCORS crea un middleware CORS con defaults seguros para desarrollo local.
// Los origins de desarrollo local (localhost:5173, 5180) se incluyen siempre.
func NewCORS(cfg CORSConfig) gin.HandlerFunc {
	origins := []string{
		"http://localhost:5173",
		"http://localhost:5180",
		"http://localhost:5190",
		"http://localhost:5191",
		"http://127.0.0.1:5173",
		"http://127.0.0.1:5180",
		"http://127.0.0.1:5190",
		"http://127.0.0.1:5191",
	}

	seen := make(map[string]struct{}, len(origins))
	for _, o := range origins {
		seen[o] = struct{}{}
	}
	for _, o := range cfg.Origins {
		u := strings.TrimSuffix(strings.TrimSpace(o), "/")
		if u == "" {
			continue
		}
		if _, ok := seen[u]; !ok {
			origins = append(origins, u)
			seen[u] = struct{}{}
		}
	}

	headers := DefaultCORSHeaders
	for _, h := range cfg.AllowHeaders {
		h = strings.TrimSpace(h)
		if h != "" {
			headers = append(headers, h)
		}
	}

	maxAge := cfg.MaxAge
	if maxAge <= 0 {
		maxAge = 86400
	}

	return cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     headers,
		AllowCredentials: true,
		MaxAge:           time.Duration(maxAge) * time.Second,
	})
}
