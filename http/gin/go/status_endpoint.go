package ginmw

import (
	"context"
	"net/http"
	"strings"

	"github.com/devpablocristo/platform/errors/go/domainerr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// StatusUpdater is the signature of a usecase that changes a resource's
// status. The usecase is responsible for:
//
//   - reading the current state,
//   - validating the transition through an FSM (and mapping sentinels via
//     platform/concurrency/go/fsm.MapDomainError),
//   - persisting,
//   - emitting audit / timeline / webhook side effects.
//
// It returns the pure domain value (the caller-provided mapper turns it
// into a DTO).
//
// tenantID is opaque (uuid.UUID) — the consumer typically resolves it from
// AuthContext at the wire site (see ParseAuthTenantAndParamID for pymes).
type StatusUpdater[T any] func(ctx context.Context, tenantID, id uuid.UUID, nextStatus, actor string) (T, error)

// StatusResponseMapper converts the domain value into the JSON response shape.
type StatusResponseMapper[T any] func(T) any

// TenantIDExtractor resolves the tenant identifier from the Gin request.
// Default: GetAuthContext(c).OrgID parsed as UUID (assumes AuthMiddleware
// already populated the context). Consumers with non-UUID tenant identifiers
// or different identity wiring can supply their own.
type TenantIDExtractor func(c *gin.Context) (uuid.UUID, bool)

// IDParamExtractor resolves the resource ID from the request URL param.
// Default: ParseUUIDParam(c, "id").
type IDParamExtractor func(c *gin.Context) (uuid.UUID, bool)

// StatusEndpointConfig customizes RegisterStatusEndpoint. Zero value uses
// reasonable defaults (UUID tenant from AuthContext.OrgID, UUID :id param,
// PATCH method, "/status" suffix).
type StatusEndpointConfig struct {
	ExtractTenantID TenantIDExtractor
	ExtractID       IDParamExtractor
	// Suffix appended to BasePath to form the route (e.g. "/status" or "/state").
	// Defaults to "/:id/status" when empty (the legacy pymes convention).
	RouteSuffix string
}

// RegisterStatusEndpoint registers a PATCH endpoint for status transitions
// on a resource, wired to RBAC. It handles:
//
//   - tenant + id parsing,
//   - request body shape (`{"status": "..."}`),
//   - normalization (lowercase + trim),
//   - RBAC check via the supplied middleware,
//   - calling the user-provided updater,
//   - mapping domain errors to HTTP via Respond.
//
// The handler is FSM-agnostic and audit-agnostic: the updater is the one
// that knows the state machine and the side effects.
//
// Example wiring (pymes-style):
//
//	ginmw.RegisterStatusEndpoint[sales.Sale](
//	    auth, rbac,
//	    "sales", "update", "/sales",
//	    salesUsecases.UpdateStatus,
//	    func(s sales.Sale) any { return saleDTO.From(s) },
//	    nil, // defaults
//	)
func RegisterStatusEndpoint[T any](
	auth *gin.RouterGroup,
	rbac *RBACMiddleware,
	resource, permission, basePath string,
	update StatusUpdater[T],
	mapper StatusResponseMapper[T],
	cfg *StatusEndpointConfig,
) {
	extractTenant := defaultTenantIDExtractor
	extractID := defaultIDParamExtractor
	suffix := "/:id/status"
	if cfg != nil {
		if cfg.ExtractTenantID != nil {
			extractTenant = cfg.ExtractTenantID
		}
		if cfg.ExtractID != nil {
			extractID = cfg.ExtractID
		}
		if strings.TrimSpace(cfg.RouteSuffix) != "" {
			suffix = cfg.RouteSuffix
		}
	}

	auth.PATCH(basePath+suffix, rbac.RequirePermission(resource, permission), func(c *gin.Context) {
		tenantID, ok := extractTenant(c)
		if !ok {
			return
		}
		id, ok := extractID(c)
		if !ok {
			return
		}
		var req statusRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			WriteValidation(c, "invalid request body")
			return
		}
		next := strings.TrimSpace(strings.ToLower(req.Status))
		if next == "" {
			Respond(c, domainerr.Validation("status is required"))
			return
		}
		actor := GetAuthContext(c).Actor
		out, err := update(c.Request.Context(), tenantID, id, next, actor)
		if err != nil {
			Respond(c, err)
			return
		}
		c.JSON(http.StatusOK, mapper(out))
	})
}

// statusRequest is the uniform body for status endpoints.
type statusRequest struct {
	Status string `json:"status" binding:"required"`
}

func defaultTenantIDExtractor(c *gin.Context) (uuid.UUID, bool) {
	raw := strings.TrimSpace(GetAuthContext(c).OrgID)
	if raw == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"code":    "UNAUTHORIZED",
			"message": "unauthorized",
		})
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		WriteValidation(c, "invalid tenant")
		return uuid.Nil, false
	}
	return id, true
}

func defaultIDParamExtractor(c *gin.Context) (uuid.UUID, bool) {
	value := strings.TrimSpace(c.Param("id"))
	id, err := uuid.Parse(value)
	if err != nil {
		// Use VALIDATION envelope (consistent with WriteValidation) instead of
		// the legacy SimpleErrorResponse shape returned by ParseUUIDParam, so
		// the body matches what the status endpoint produces elsewhere.
		WriteValidation(c, "invalid id")
		return uuid.Nil, false
	}
	return id, true
}
