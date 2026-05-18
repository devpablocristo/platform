package ginmw

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/devpablocristo/core/errors/go/domainerr"
	coreapikey "github.com/devpablocristo/core/security/go/apikey"
)

const apiKeyPrincipalContextKey = "core.api_key_principal"

// RequireAPIKey protege un handler Gin con un authenticator reutilizable.
func RequireAPIKey(auth *coreapikey.Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/healthz" || c.Request.URL.Path == "/readyz" {
			c.Next()
			return
		}

		rawKey := coreapikey.ExtractKey(c.Request)
		principal, ok := auth.Authenticate(rawKey)
		if !ok {
			Respond(c, domainerr.Unauthorized("valid api key required"))
			c.Abort()
			return
		}

		c.Set(apiKeyPrincipalContextKey, principal)
		c.Next()
	}
}

// RequireAPIKeyFromEnv construye un authenticator simple a partir de un env var con una única key.
func RequireAPIKeyFromEnv(envVar string) gin.HandlerFunc {
	secret := strings.TrimSpace(os.Getenv(envVar))
	if secret == "" {
		return RequireAPIKey(nil)
	}

	auth, err := coreapikey.NewAuthenticator("default=" + secret)
	if err != nil {
		return func(c *gin.Context) {
			WriteError(c, 500, "internal_error", "api key authenticator misconfigured")
			c.Abort()
		}
	}

	return RequireAPIKey(auth)
}
