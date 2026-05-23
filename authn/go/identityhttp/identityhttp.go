// Package identityhttp provides the canonical HTTP identity contract for
// services that use authn.Principal.
package identityhttp

import (
	"context"
	"net/http"
	"strings"

	authn "github.com/devpablocristo/platform/authn/go"
	"github.com/devpablocristo/platform/errors/go/domainerr"
)

const (
	HeaderOrgID            = "X-Org-ID"
	HeaderUserID           = "X-User-ID"
	HeaderAuthRole         = "X-Auth-Role"
	HeaderAuthScopes       = "X-Auth-Scopes"
	HeaderAuthMethod       = "X-Auth-Method"
	HeaderServicePrincipal = "X-Service-Principal"
)

type contextKey string

const contextPrincipalKey contextKey = "platform.identityhttp.principal"

type Context struct {
	OrgID            string
	Actor            string
	Role             string
	Scopes           []string
	AuthMethod       string
	ServicePrincipal bool
}

func WithPrincipal(r *http.Request, principal *authn.Principal, method string) *http.Request {
	if r == nil || principal == nil {
		return r
	}
	authMethod := strings.TrimSpace(principal.AuthMethod)
	if authMethod == "" {
		authMethod = strings.TrimSpace(method)
	}
	ctx := Context{
		OrgID:            strings.TrimSpace(principal.OrgID),
		Actor:            strings.TrimSpace(principal.Actor),
		Role:             strings.TrimSpace(principal.Role),
		Scopes:           cleanScopes(principal.Scopes),
		AuthMethod:       authMethod,
		ServicePrincipal: principalServicePrincipal(principal),
	}
	req := r.Clone(context.WithValue(r.Context(), contextPrincipalKey, ctx))
	req.Header = r.Header.Clone()
	clearIdentityHeaders(req.Header)
	if ctx.Actor != "" {
		req.Header.Set(HeaderUserID, ctx.Actor)
	}
	if ctx.OrgID != "" {
		req.Header.Set(HeaderOrgID, ctx.OrgID)
	}
	if ctx.Role != "" {
		req.Header.Set(HeaderAuthRole, ctx.Role)
	}
	if len(ctx.Scopes) > 0 {
		req.Header.Set(HeaderAuthScopes, strings.Join(ctx.Scopes, " "))
	}
	if ctx.AuthMethod != "" {
		req.Header.Set(HeaderAuthMethod, ctx.AuthMethod)
	}
	if ctx.ServicePrincipal {
		req.Header.Set(HeaderServicePrincipal, "true")
	}
	return req
}

func FromRequest(r *http.Request) Context {
	if r == nil {
		return Context{}
	}
	if ctx, ok := r.Context().Value(contextPrincipalKey).(Context); ok {
		return ctx
	}
	return Context{
		OrgID:            strings.TrimSpace(r.Header.Get(HeaderOrgID)),
		Actor:            strings.TrimSpace(r.Header.Get(HeaderUserID)),
		Role:             strings.TrimSpace(r.Header.Get(HeaderAuthRole)),
		Scopes:           ParseScopes(r.Header.Get(HeaderAuthScopes)),
		AuthMethod:       strings.TrimSpace(r.Header.Get(HeaderAuthMethod)),
		ServicePrincipal: parseBool(r.Header.Get(HeaderServicePrincipal)),
	}
}

func HasNoAuthContext(r *http.Request) bool {
	ctx := FromRequest(r)
	return ctx.AuthMethod == "" && len(ctx.Scopes) == 0
}

func HasScope(r *http.Request, scope string) bool {
	_, ok := scopeSet(FromRequest(r).Scopes)[strings.TrimSpace(scope)]
	return ok
}

func HasAnyScope(r *http.Request, scopes ...string) bool {
	have := scopeSet(FromRequest(r).Scopes)
	for _, scope := range scopes {
		if _, ok := have[strings.TrimSpace(scope)]; ok {
			return true
		}
	}
	return false
}

func PrincipalOrgID(r *http.Request) string {
	return FromRequest(r).OrgID
}

func CanAccessOrg(r *http.Request, orgID string, crossOrgScope string) bool {
	if HasNoAuthContext(r) {
		return true
	}
	if crossOrgScope != "" && HasScope(r, crossOrgScope) {
		return true
	}
	principalOrg := strings.TrimSpace(PrincipalOrgID(r))
	return principalOrg != "" && principalOrg == strings.TrimSpace(orgID)
}

// RequireOrgMatch resuelve el org id efectivo o retorna error tipado.
// Variante error-aware de `EffectiveOrgID`:
//   - sin auth context (dev mode / pruebas) → devuelve `requestedOrgID, nil` tal cual.
//   - principal con `crossOrgScope` → bypassea el match: devuelve `requestedOrgID`
//     si fue provisto, sino el `principalOrgID`.
//   - principal sin org → `domainerr.TenantMissing()`.
//   - `requestedOrgID` distinto al `principalOrgID` → `domainerr.TenantMismatch()`.
//   - match → `principalOrgID, nil`.
func RequireOrgMatch(r *http.Request, requestedOrgID, crossOrgScope string) (string, error) {
	requestedOrgID = strings.TrimSpace(requestedOrgID)
	if HasNoAuthContext(r) {
		return requestedOrgID, nil
	}
	principalOrg := strings.TrimSpace(PrincipalOrgID(r))
	if crossOrgScope != "" && HasScope(r, crossOrgScope) {
		if requestedOrgID != "" {
			return requestedOrgID, nil
		}
		return principalOrg, nil
	}
	if principalOrg == "" {
		return "", domainerr.TenantMissing()
	}
	if requestedOrgID != "" && requestedOrgID != principalOrg {
		return "", domainerr.TenantMismatch()
	}
	return principalOrg, nil
}

func EffectiveOrgID(r *http.Request, requestedOrgID, crossOrgScope string) (string, bool) {
	requestedOrgID = strings.TrimSpace(requestedOrgID)
	if HasNoAuthContext(r) {
		return requestedOrgID, true
	}
	principalOrg := strings.TrimSpace(PrincipalOrgID(r))
	if crossOrgScope != "" && HasScope(r, crossOrgScope) {
		if requestedOrgID != "" {
			return requestedOrgID, true
		}
		return principalOrg, true
	}
	if principalOrg == "" {
		return "", false
	}
	if requestedOrgID != "" && requestedOrgID != principalOrg {
		return "", false
	}
	return principalOrg, true
}

func ParseScopes(raw string) []string {
	raw = strings.NewReplacer(",", " ", ";", " ", "+", " ").Replace(raw)
	return cleanScopes(strings.Fields(raw))
}

func clearIdentityHeaders(h http.Header) {
	for _, header := range []string{
		HeaderOrgID,
		HeaderUserID,
		HeaderAuthRole,
		HeaderAuthScopes,
		HeaderAuthMethod,
		HeaderServicePrincipal,
	} {
		h.Del(header)
	}
}

func cleanScopes(scopes []string) []string {
	out := make([]string, 0, len(scopes))
	seen := make(map[string]struct{}, len(scopes))
	for _, item := range scopes {
		scope := strings.TrimSpace(item)
		if scope == "" {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}
	return out
}

func scopeSet(scopes []string) map[string]struct{} {
	out := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope != "" {
			out[scope] = struct{}{}
		}
	}
	return out
}

func principalServicePrincipal(principal *authn.Principal) bool {
	if principal == nil || principal.Claims == nil {
		return false
	}
	for _, key := range []string{"service_principal", "service"} {
		switch value := principal.Claims[key].(type) {
		case bool:
			if value {
				return true
			}
		case string:
			if parseBool(value) {
				return true
			}
		}
	}
	return false
}

func parseBool(raw string) bool {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "1", "true", "yes", "y", "service":
		return true
	default:
		return false
	}
}
