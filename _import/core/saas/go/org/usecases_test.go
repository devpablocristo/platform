package org

import (
	"context"
	"testing"

	orgdomain "github.com/devpablocristo/core/saas/go/org/usecases/domain"
)

func TestResolvePrincipal(t *testing.T) {
	t.Parallel()

	usecases := NewUseCases(fakeKeyResolver{
		principal: orgdomain.Principal{TenantID: "acme", Scopes: []string{"tools:read"}},
	})
	principal, err := usecases.ResolvePrincipal(context.Background(), "hash-1")
	if err != nil {
		t.Fatalf("ResolvePrincipal returned error: %v", err)
	}
	if principal.TenantID != "acme" {
		t.Fatalf("unexpected principal: %#v", principal)
	}
}

type fakeKeyResolver struct {
	principal orgdomain.Principal
}

func (f fakeKeyResolver) FindPrincipalByAPIKeyHash(context.Context, string) (orgdomain.Principal, string, error) {
	return f.principal, "stored", nil
}
