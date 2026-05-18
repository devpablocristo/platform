package ginmw

import (
	"net/http"

	"github.com/gin-gonic/gin"

	authn "github.com/devpablocristo/core/authn/go"
	ctxkeys "github.com/devpablocristo/core/security/go/contextkeys"
)

// AuthMiddleware middleware de autenticación JWT + API Key para Gin.
// Delega a authn.TryInbound (credencial entra → principal sale).
type AuthMiddleware struct {
	jwtAuth authn.Authenticator
	apiKey  authn.Authenticator
}

// NewAuthMiddleware crea un middleware de autenticación.
// jwtAuth y apiKey pueden ser nil (se omite ese mecanismo).
func NewAuthMiddleware(jwtAuth, apiKey authn.Authenticator) *AuthMiddleware {
	return &AuthMiddleware{
		jwtAuth: jwtAuth,
		apiKey:  apiKey,
	}
}

// RequireAuth retorna el handler Gin que requiere autenticación.
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		xAPIKey := c.GetHeader("X-API-KEY")

		principal, method, err := authn.TryInbound(
			c.Request.Context(),
			m.jwtAuth,
			m.apiKey,
			authorization,
			xAPIKey,
		)
		if err != nil || principal == nil {
			c.JSON(http.StatusUnauthorized, SimpleErrorResponse{Error: "unauthorized"})
			c.Abort()
			return
		}

		c.Set(ctxkeys.CtxKeyOrgID, principal.OrgID)
		c.Set(ctxkeys.CtxKeyActor, principal.Actor)
		c.Set(ctxkeys.CtxKeyRole, principal.Role)
		c.Set(ctxkeys.CtxKeyScopes, principal.Scopes)
		c.Set(ctxkeys.CtxKeyAuthMethod, method)
		c.Next()
	}
}

// AuthContext contexto de autenticación extraído de un request.
type AuthContext struct {
	OrgID      string   `json:"org_id"`
	Actor      string   `json:"actor"`
	Role       string   `json:"role"`
	Scopes     []string `json:"scopes"`
	AuthMethod string   `json:"auth_method"`
}

// GetAuthContext extrae el contexto de autenticación del Gin context.
func GetAuthContext(c *gin.Context) AuthContext {
	orgID, _ := c.Get(ctxkeys.CtxKeyOrgID)
	actor, _ := c.Get(ctxkeys.CtxKeyActor)
	role, _ := c.Get(ctxkeys.CtxKeyRole)
	scopes, _ := c.Get(ctxkeys.CtxKeyScopes)
	authMethod, _ := c.Get(ctxkeys.CtxKeyAuthMethod)

	ctxScopes, _ := scopes.([]string)
	return AuthContext{
		OrgID:      asString(orgID),
		Actor:      asString(actor),
		Role:       asString(role),
		Scopes:     ctxScopes,
		AuthMethod: asString(authMethod),
	}
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
