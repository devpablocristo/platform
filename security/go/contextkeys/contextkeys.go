package ctxkeys

type Key string

const (
	RequestID  Key = "request_id"
	OrgID      Key = "org_id"
	TenantID   Key = "tenant_id"
	Actor      Key = "actor"
	Role       Key = "role"
	Scopes     Key = "scopes"
	AuthMethod Key = "auth_method"
)

const (
	CtxKeyRequestID  = string(RequestID)
	CtxKeyOrgID      = string(OrgID)
	CtxKeyTenantID   = string(TenantID)
	CtxKeyActor      = string(Actor)
	CtxKeyRole       = string(Role)
	CtxKeyScopes     = string(Scopes)
	CtxKeyAuthMethod = string(AuthMethod)
)
