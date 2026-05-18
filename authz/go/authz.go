package authz

import "strings"

const (
	ScopeToolsRead  = "tools:read"
	ScopeToolsWrite = "tools:write"

	ScopePolicyRead  = "policy:read"
	ScopePolicyWrite = "policy:write"

	ScopeEgressRead  = "egress:read"
	ScopeEgressWrite = "egress:write"

	ScopeAuditRead = "audit:read"

	ScopeGatewayRun      = "gateway:run"
	ScopeGatewaySimulate = "gateway:simulate"

	ScopeMCPRead = "mcp:read"
	ScopeMCPCall = "mcp:call"

	ScopeA2ACall = "a2a:call"

	ScopeAdminConsoleRead  = "admin:console:read"
	ScopeAdminConsoleWrite = "admin:console:write"

	ScopeSecretsAdmin = "admin:secrets"
)

func HasScope(scopes []string, required string) bool {
	required = strings.TrimSpace(required)
	if required == "" {
		return false
	}
	for _, item := range scopes {
		if strings.TrimSpace(item) == required {
			return true
		}
	}
	return false
}

func HasAnyScope(scopes []string, required ...string) bool {
	for _, item := range required {
		if HasScope(scopes, item) {
			return true
		}
	}
	return false
}

func IsRole(role *string, accepted ...string) bool {
	if role == nil {
		return false
	}
	current := strings.TrimSpace(*role)
	for _, item := range accepted {
		if current == strings.TrimSpace(item) {
			return true
		}
	}
	return false
}
