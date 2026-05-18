package ginmw

import (
	"net/http"

	"github.com/gin-gonic/gin"

	authn "github.com/devpablocristo/platform/authn/go"
	ctxkeys "github.com/devpablocristo/platform/security/go/contextkeys"
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
//
// Looks up tenant identity under BOTH ctxkeys.CtxKeyTenantID (agnostic, new)
// and ctxkeys.CtxKeyOrgID (legacy). TenantID takes precedence when both are
// set, since it's the canonical key for new code. The result is exposed as
// AuthContext.OrgID for backward compat — consumers can rename their field
// access in a separate ola.
func GetAuthContext(c *gin.Context) AuthContext {
	tenantID, hasTenant := c.Get(ctxkeys.CtxKeyTenantID)
	orgID, _ := c.Get(ctxkeys.CtxKeyOrgID)
	actor, _ := c.Get(ctxkeys.CtxKeyActor)
	role, _ := c.Get(ctxkeys.CtxKeyRole)
	scopes, _ := c.Get(ctxkeys.CtxKeyScopes)
	authMethod, _ := c.Get(ctxkeys.CtxKeyAuthMethod)

	resolvedTenant := asString(orgID)
	if hasTenant {
		if s := asString(tenantID); s != "" {
			resolvedTenant = s
		}
	}

	ctxScopes, _ := scopes.([]string)
	return AuthContext{
		OrgID:      resolvedTenant,
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
