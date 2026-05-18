package ginmw

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// PermissionChecker is the abstract authorization decision: given an actor's
// identity context, decide whether they can perform `action` on `resource`.
//
// Resource and action are opaque strings (§ Invariante I6 — no fixed
// taxonomy in platform). Consumers wire their own checker (Casbin, OpenFGA,
// in-process map, etc.). tenantID is also opaque — typically the
// organization or workspace identifier.
type PermissionChecker interface {
	HasPermission(
		ctx context.Context,
		tenantID, actor, role string,
		scopes []string,
		authMethod, resource, action string,
	) bool
}

// RBACMiddleware enforces a PermissionChecker decision on each request.
// It reads identity from the AuthContext stored on the Gin context (see
// GetAuthContext); consumers that put identity somewhere else can override
// extraction via NewRBACMiddlewareWithResolver.
type RBACMiddleware struct {
	checker  PermissionChecker
	resolver AuthContextResolver
}

// AuthContextResolver extracts the identity context from a Gin request.
// Default: ginmw.GetAuthContext (reads keys set by AuthMiddleware).
type AuthContextResolver func(c *gin.Context) AuthContext

// NewRBACMiddleware wires a PermissionChecker with the default
// AuthContext resolver.
func NewRBACMiddleware(checker PermissionChecker) *RBACMiddleware {
	return &RBACMiddleware{checker: checker, resolver: GetAuthContext}
}

// NewRBACMiddlewareWithResolver lets the consumer supply a custom AuthContext
// resolver, e.g. when identity is stored under non-default context keys.
func NewRBACMiddlewareWithResolver(checker PermissionChecker, resolver AuthContextResolver) *RBACMiddleware {
	if resolver == nil {
		resolver = GetAuthContext
	}
	return &RBACMiddleware{checker: checker, resolver: resolver}
}

// RequirePermission returns a gin.HandlerFunc that aborts the request with
// 401/403 unless the resolved AuthContext allows (resource, action). It
// returns 500 if the middleware is misconfigured.
func (m *RBACMiddleware) RequirePermission(resource, action string) gin.HandlerFunc {
	resource = strings.TrimSpace(resource)
	action = strings.TrimSpace(action)
	return func(c *gin.Context) {
		if m == nil || m.checker == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code":    "INTERNAL",
				"message": "rbac_not_configured",
			})
			return
		}
		authCtx := m.resolver(c)
		// AuthContext.OrgID holds the tenant identifier (kept as OrgID for
		// backward compatibility with consumers that pre-date the TenantID
		// rename). It is passed to PermissionChecker as tenantID — the
		// interface name is the agnostic one.
		tenantID := authCtx.OrgID
		if tenantID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "unauthorized",
			})
			return
		}
		if m.checker.HasPermission(
			c.Request.Context(),
			tenantID, authCtx.Actor, authCtx.Role,
			authCtx.Scopes, authCtx.AuthMethod,
			resource, action,
		) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"code":     "FORBIDDEN",
			"message":  "forbidden",
			"required": resource + ":" + action,
		})
	}
}
