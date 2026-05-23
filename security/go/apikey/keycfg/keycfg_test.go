package keycfg_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/devpablocristo/platform/security/go/apikey/keycfg"
)

func TestParse_SimpleNoMetadata(t *testing.T) {
	t.Parallel()
	sanitized, meta := keycfg.Parse("admin=secret1,data=secret2")
	if sanitized != "admin=secret1,data=secret2" {
		t.Errorf("sanitized=%q", sanitized)
	}
	if len(meta) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(meta))
	}
	if meta["admin"].Actor != "admin" || meta["admin"].Role != "admin" {
		t.Errorf("admin metadata defaults wrong: %+v", meta["admin"])
	}
	if meta["data"].Actor != "data" {
		t.Errorf("data actor default wrong: %+v", meta["data"])
	}
}

func TestParse_FullMetadata(t *testing.T) {
	t.Parallel()
	raw := "admin=secret-abc|org_id=local-dev|role=root|actor=service|scope=read+write+a:b|service_principal=true"
	sanitized, meta := keycfg.Parse(raw)
	if sanitized != "admin=secret-abc" {
		t.Errorf("sanitized=%q want admin=secret-abc", sanitized)
	}
	m := meta["admin"]
	if m.OrgID != "local-dev" {
		t.Errorf("OrgID=%q", m.OrgID)
	}
	if m.Role != "root" {
		t.Errorf("Role=%q", m.Role)
	}
	if m.Actor != "service" {
		t.Errorf("Actor=%q", m.Actor)
	}
	if !m.ServicePrincipal {
		t.Error("expected ServicePrincipal=true")
	}
	wantScopes := map[string]bool{"read": true, "write": true, "a:b": true}
	if len(m.Scopes) != 3 {
		t.Fatalf("expected 3 scopes, got %v", m.Scopes)
	}
	for _, s := range m.Scopes {
		if !wantScopes[s] {
			t.Errorf("unexpected scope %q", s)
		}
	}
}

func TestParse_MultipleEntries(t *testing.T) {
	t.Parallel()
	raw := "admin=sec1|org_id=A|scope=companion:tasks:read+companion:tasks:write,bot=sec2|org_id=B|role=bot"
	sanitized, meta := keycfg.Parse(raw)
	if !strings.Contains(sanitized, "admin=sec1") || !strings.Contains(sanitized, "bot=sec2") {
		t.Errorf("sanitized lost names: %q", sanitized)
	}
	if meta["admin"].OrgID != "A" || meta["bot"].OrgID != "B" {
		t.Errorf("orgs wrong: %+v", meta)
	}
	if meta["bot"].Role != "bot" {
		t.Errorf("bot role=%q", meta["bot"].Role)
	}
}

func TestParse_AcceptsLegacyOrgAlias(t *testing.T) {
	t.Parallel()
	for _, alias := range []string{"org", "org_id", "tenant", "tenant_id"} {
		raw := "k=s|" + alias + "=AAA"
		_, meta := keycfg.Parse(raw)
		if meta["k"].OrgID != "AAA" {
			t.Errorf("alias %q failed: %+v", alias, meta["k"])
		}
	}
}

func TestParse_EmptyAndMalformed(t *testing.T) {
	t.Parallel()
	if s, m := keycfg.Parse(""); s != "" || len(m) != 0 {
		t.Errorf("empty got %q %+v", s, m)
	}
	// piece without `=` is preserved
	sanitized, _ := keycfg.Parse("just-a-thing,k=v")
	if !strings.Contains(sanitized, "just-a-thing") || !strings.Contains(sanitized, "k=v") {
		t.Errorf("malformed preservation failed: %q", sanitized)
	}
}

func TestParse_NewlineDelimiter(t *testing.T) {
	t.Parallel()
	raw := "admin=sec1|org_id=A\nbot=sec2|org_id=B"
	_, meta := keycfg.Parse(raw)
	if len(meta) != 2 {
		t.Errorf("expected 2 entries from newline-delim, got %d", len(meta))
	}
}

func TestParseScopeList_Deduplicates(t *testing.T) {
	t.Parallel()
	scopes := keycfg.ParseScopeList("a b a c+a;b")
	want := []string{"a", "b", "c"}
	if len(scopes) != len(want) {
		t.Fatalf("got %v want %v", scopes, want)
	}
}

func TestRegister_DefaultScopesFor_RoundTrip(t *testing.T) {
	t.Parallel()
	keycfg.ResetProfiles()
	keycfg.Register(keycfg.Profile{
		Name:   "admin",
		Scopes: []string{"x:read", "x:write"},
	})
	got := keycfg.DefaultScopesFor("ADMIN") // case-insensitive
	if len(got) != 2 || got[0] != "x:read" || got[1] != "x:write" {
		t.Errorf("DefaultScopesFor=%v", got)
	}
	if keycfg.DefaultScopesFor("nope") != nil {
		t.Error("expected nil for unknown profile")
	}
}

func TestRegister_LastWriteWins(t *testing.T) {
	t.Parallel()
	keycfg.ResetProfiles()
	keycfg.Register(keycfg.Profile{Name: "k", Scopes: []string{"a"}})
	keycfg.Register(keycfg.Profile{Name: "k", Scopes: []string{"b", "c"}})
	got := keycfg.DefaultScopesFor("k")
	if len(got) != 2 || got[0] != "b" || got[1] != "c" {
		t.Errorf("got %v want [b c]", got)
	}
}

func TestRegister_ReturnedSliceIsCopy(t *testing.T) {
	t.Parallel()
	keycfg.ResetProfiles()
	keycfg.Register(keycfg.Profile{Name: "x", Scopes: []string{"a"}})
	out1 := keycfg.DefaultScopesFor("x")
	out1[0] = "MUTATED"
	out2 := keycfg.DefaultScopesFor("x")
	if out2[0] != "a" {
		t.Errorf("registry mutated via returned slice: %v", out2)
	}
}

func TestRegister_ConcurrentSafe(t *testing.T) {
	keycfg.ResetProfiles()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			keycfg.Register(keycfg.Profile{Name: "shared", Scopes: []string{"s"}})
		}()
		go func() {
			defer wg.Done()
			_ = keycfg.DefaultScopesFor("shared")
		}()
	}
	wg.Wait()
}
