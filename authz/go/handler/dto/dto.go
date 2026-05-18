package dto

type HasScopeRequest struct {
	Scopes   []string `json:"scopes,omitempty"`
	Required string   `json:"required"`
}

type HasAnyScopeRequest struct {
	Scopes   []string `json:"scopes,omitempty"`
	Required []string `json:"required,omitempty"`
}

type IsRoleRequest struct {
	Role     *string  `json:"role,omitempty"`
	Accepted []string `json:"accepted,omitempty"`
}

type BoolResponse struct {
	Allowed bool `json:"allowed"`
}
