package authn

import (
	"context"
	"errors"
	"testing"
)

type fnAuthenticator func(context.Context, Credential) (*Principal, error)

func (f fnAuthenticator) Authenticate(ctx context.Context, c Credential) (*Principal, error) {
	return f(ctx, c)
}

func TestTryInboundJWTFirst(t *testing.T) {
	t.Parallel()
	jwt := fnAuthenticator(func(ctx context.Context, c Credential) (*Principal, error) {
		if c.Kind() != KindBearer {
			t.Fatalf("want bearer got %s", c.Kind())
		}
		return &Principal{Actor: "jwt-user"}, nil
	})
	key := fnAuthenticator(func(context.Context, Credential) (*Principal, error) {
		t.Fatal("api key must not run when jwt ok")
		return nil, nil
	})
	p, m, err := TryInbound(context.Background(), jwt, key, "Bearer tok", "psk")
	if err != nil || m != "jwt" || p == nil || p.Actor != "jwt-user" {
		t.Fatalf("got p=%v m=%q err=%v", p, m, err)
	}
}

func TestTryInboundJWTFailsThenAPIKey(t *testing.T) {
	t.Parallel()
	jwt := fnAuthenticator(func(context.Context, Credential) (*Principal, error) {
		return nil, errors.New("bad jwt")
	})
	key := fnAuthenticator(func(ctx context.Context, c Credential) (*Principal, error) {
		if c.Kind() != KindAPIKey {
			t.Fatalf("want api_key got %s", c.Kind())
		}
		return &Principal{Actor: "key-user"}, nil
	})
	p, m, err := TryInbound(context.Background(), jwt, key, "Bearer bad", "psk-local")
	if err != nil || m != "api_key" || p.Actor != "key-user" {
		t.Fatalf("got p=%v m=%q err=%v", p, m, err)
	}
}

func TestTryInboundJWTErrorNoAPIKeyHeader(t *testing.T) {
	t.Parallel()
	jwt := fnAuthenticator(func(context.Context, Credential) (*Principal, error) {
		return nil, errors.New("invalid signature")
	})
	p, m, err := TryInbound(context.Background(), jwt, nil, "Bearer x", "")
	if err == nil || p != nil || m != "" {
		t.Fatalf("want jwt error, got p=%v m=%q err=%v", p, m, err)
	}
}

func TestTryInboundNoCredential(t *testing.T) {
	t.Parallel()
	_, _, err := TryInbound(context.Background(),
		fnAuthenticator(func(context.Context, Credential) (*Principal, error) {
			t.Fatal("jwt should not run")
			return nil, nil
		}),
		fnAuthenticator(func(context.Context, Credential) (*Principal, error) {
			t.Fatal("key should not run")
			return nil, nil
		}),
		"", "",
	)
	if !errors.Is(err, ErrNoValidCredential) {
		t.Fatalf("want ErrNoValidCredential got %v", err)
	}
}

func TestDefaultHeaderExtractor(t *testing.T) {
	t.Parallel()
	var ex DefaultHeaderExtractor
	c, err := ex.Extract("Bearer a", "")
	if err != nil || c == nil || c.Kind() != KindBearer {
		t.Fatalf("bearer: %v %v", c, err)
	}
	c, err = ex.Extract("", "k1")
	if err != nil || c == nil || c.Kind() != KindAPIKey {
		t.Fatalf("key: %v %v", c, err)
	}
	c, err = ex.Extract("", "")
	if err != nil || c != nil {
		t.Fatalf("none: %v %v", c, err)
	}
}

func TestBearerJWTAuthenticatorMinimalMap(t *testing.T) {
	t.Parallel()
	v := stubVerifier{claims: map[string]any{"sub": "u1", "org_id": "o1"}}
	a := &BearerJWTAuthenticator{Verify: v}
	p, err := a.Authenticate(context.Background(), BearerCredential{Token: "any"})
	if err != nil || p.Actor != "u1" || p.OrgID != "o1" {
		t.Fatalf("got %+v err=%v", p, err)
	}
}

func TestBearerJWTAuthenticatorWrongKind(t *testing.T) {
	t.Parallel()
	v := stubVerifier{claims: map[string]any{"sub": "u1"}}
	a := &BearerJWTAuthenticator{Verify: v}
	_, err := a.Authenticate(context.Background(), APIKeyCredential{Key: "x"})
	if !errors.Is(err, ErrWrongCredentialKind) {
		t.Fatalf("got %v", err)
	}
}

func TestAPIKeyFuncAuthenticator(t *testing.T) {
	t.Parallel()
	a := &APIKeyFuncAuthenticator{Resolve: func(ctx context.Context, raw string) (*Principal, error) {
		if raw != "secret" {
			return nil, errors.New("no")
		}
		return &Principal{Actor: "svc"}, nil
	}}
	p, err := a.Authenticate(context.Background(), APIKeyCredential{Key: "secret"})
	if err != nil || p.Actor != "svc" {
		t.Fatalf("got %+v err=%v", p, err)
	}
}

type stubVerifier struct {
	claims map[string]any
}

func (s stubVerifier) VerifyToken(ctx context.Context, rawToken string) (map[string]any, error) {
	return s.claims, nil
}
