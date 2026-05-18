package authz

// UseCases agrupa reglas reutilizables de autorización.
type UseCases struct{}

func NewUseCases() *UseCases {
	return &UseCases{}
}

func (u *UseCases) HasScope(scopes []string, required string) bool {
	return HasScope(scopes, required)
}

func (u *UseCases) HasAnyScope(scopes []string, required ...string) bool {
	return HasAnyScope(scopes, required...)
}

func (u *UseCases) IsRole(role *string, accepted ...string) bool {
	return IsRole(role, accepted...)
}
