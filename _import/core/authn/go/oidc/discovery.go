package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	authnjwks "github.com/devpablocristo/core/authn/go/jwks"
)

type DiscoveryDocument struct {
	Issuer                        string   `json:"issuer"`
	AuthorizationEndpoint         string   `json:"authorization_endpoint"`
	TokenEndpoint                 string   `json:"token_endpoint"`
	UserinfoEndpoint              string   `json:"userinfo_endpoint,omitempty"`
	JWKSURI                       string   `json:"jwks_uri"`
	ScopesSupported               []string `json:"scopes_supported,omitempty"`
	ResponseTypesSupported        []string `json:"response_types_supported,omitempty"`
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported,omitempty"`
}

type DiscoveryClient struct {
	issuerURL  string
	httpClient *http.Client
	cacheTTL   time.Duration

	mu       sync.RWMutex
	doc      *DiscoveryDocument
	cachedAt time.Time
	verifier *authnjwks.Verifier
}

func NewDiscoveryClient(issuerURL string) *DiscoveryClient {
	return &DiscoveryClient{
		issuerURL:  strings.TrimRight(strings.TrimSpace(issuerURL), "/"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		cacheTTL:   10 * time.Minute,
	}
}

func (d *DiscoveryClient) Discover(ctx context.Context) (*DiscoveryDocument, error) {
	d.mu.RLock()
	if d.doc != nil && time.Since(d.cachedAt) < d.cacheTTL {
		doc := d.doc
		d.mu.RUnlock()
		return doc, nil
	}
	d.mu.RUnlock()
	return d.refresh(ctx)
}

func (d *DiscoveryClient) Verifier(ctx context.Context) (*authnjwks.Verifier, error) {
	doc, err := d.Discover(ctx)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery: %w", err)
	}

	d.mu.RLock()
	verifier := d.verifier
	d.mu.RUnlock()
	if verifier != nil {
		return verifier, nil
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	if d.verifier != nil {
		return d.verifier, nil
	}
	d.verifier = authnjwks.NewVerifier(doc.JWKSURI)
	return d.verifier, nil
}

func (d *DiscoveryClient) VerifyToken(ctx context.Context, token string) (map[string]any, error) {
	verifier, err := d.Verifier(ctx)
	if err != nil {
		return nil, err
	}
	return verifier.VerifyToken(ctx, token)
}

func (d *DiscoveryClient) refresh(ctx context.Context) (*DiscoveryDocument, error) {
	wellKnown := d.issuerURL + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wellKnown, nil)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("oidc discovery status %d from %s", resp.StatusCode, wellKnown)
	}

	var doc DiscoveryDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("oidc discovery decode: %w", err)
	}

	if doc.Issuer == "" || doc.JWKSURI == "" || doc.AuthorizationEndpoint == "" || doc.TokenEndpoint == "" {
		return nil, fmt.Errorf("oidc discovery document is incomplete")
	}

	d.mu.Lock()
	if d.doc != nil && d.doc.JWKSURI != doc.JWKSURI {
		d.verifier = nil
	}
	d.doc = &doc
	d.cachedAt = time.Now()
	d.mu.Unlock()
	return &doc, nil
}
