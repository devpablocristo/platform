package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	authn "github.com/devpablocristo/platform/authn/go"
	authnjwks "github.com/devpablocristo/platform/authn/go/jwks"
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

	staleWindow    time.Duration
	refreshBackoff time.Duration
	observeStale   authnjwks.StaleObserver
	now            func() time.Time

	mu          sync.RWMutex
	doc         *DiscoveryDocument
	cachedAt    time.Time
	nextRetryAt time.Time
	lastErr     error
	verifier    *authnjwks.Verifier
}

// Option configura el cliente. Lo que aplica al verificador se le reenvía, para que un
// consumidor configure la resiliencia en un solo lugar.
type Option func(*DiscoveryClient)

// WithStaleWindow acota cuánto se sigue usando un documento —y una clave de firma— que ya no
// se pudo reconfirmar. Ver jwks.WithStaleKeyWindow para el razonamiento completo.
func WithStaleWindow(window time.Duration) Option {
	return func(d *DiscoveryClient) { d.staleWindow = window }
}

// WithRefreshBackoff evita martillar un proveedor que está caído.
func WithRefreshBackoff(backoff time.Duration) Option {
	return func(d *DiscoveryClient) { d.refreshBackoff = backoff }
}

// WithStaleObserver registra el hook que se llama al servir algo vencido. El `subject` es el
// `kid` cuando lo emite el verificador, y "discovery document" cuando lo emite este cliente.
func WithStaleObserver(observe authnjwks.StaleObserver) Option {
	return func(d *DiscoveryClient) { d.observeStale = observe }
}

func NewDiscoveryClient(issuerURL string, opts ...Option) *DiscoveryClient {
	client := &DiscoveryClient{
		issuerURL:      strings.TrimRight(strings.TrimSpace(issuerURL), "/"),
		httpClient:     &http.Client{Timeout: 10 * time.Second},
		cacheTTL:       10 * time.Minute,
		staleWindow:    authnjwks.DefaultStaleKeyWindow,
		refreshBackoff: authnjwks.DefaultRefreshBackoff,
		now:            time.Now,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(client)
		}
	}
	return client
}

func (d *DiscoveryClient) Discover(ctx context.Context) (*DiscoveryDocument, error) {
	d.mu.RLock()
	if d.doc != nil && d.now().Sub(d.cachedAt) < d.cacheTTL {
		doc := d.doc
		d.mu.RUnlock()
		return doc, nil
	}
	retryAt := d.nextRetryAt
	lastErr := d.lastErr
	d.mu.RUnlock()

	if lastErr != nil && !retryAt.IsZero() && d.now().Before(retryAt) {
		return d.staleDocument(lastErr)
	}
	doc, err := d.refresh(ctx)
	if err != nil {
		return d.staleDocument(err)
	}
	return doc, nil
}

// staleDocument sirve el documento vencido mientras la ventana lo permita.
//
// El endpoint de discovery es infraestructura del proveedor, no de la credencial: que se
// caiga no dice nada sobre los tokens que ya están en circulación.
func (d *DiscoveryClient) staleDocument(cause error) (*DiscoveryDocument, error) {
	d.mu.RLock()
	doc := d.doc
	cachedAt := d.cachedAt
	window := d.staleWindow
	observe := d.observeStale
	age := d.now().Sub(cachedAt)
	d.mu.RUnlock()

	if doc == nil || window <= 0 || cachedAt.IsZero() || age > window {
		return nil, cause
	}
	if observe != nil {
		observe("discovery document", age, cause)
	}
	return doc, nil
}

func (d *DiscoveryClient) Verifier(ctx context.Context) (*authnjwks.Verifier, error) {
	doc, err := d.Discover(ctx)
	if err != nil {
		// Un verificador que ya existe SOBREVIVE a la caída del discovery.
		//
		// Antes este método pedía el documento primero, siempre, así que el TTL del
		// documento —10 minutos— era lo que en realidad cortaba la autenticación, aunque
		// las claves estuvieran perfectas en memoria. El documento sólo dice DÓNDE está el
		// JWKS, y si ya hay verificador eso ya se sabe.
		d.mu.RLock()
		verifier := d.verifier
		d.mu.RUnlock()
		if verifier != nil {
			return verifier, nil
		}
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
	// La resiliencia se configura una vez y vale para las dos capas.
	d.verifier = authnjwks.NewVerifier(doc.JWKSURI,
		authnjwks.WithStaleKeyWindow(d.staleWindow),
		authnjwks.WithRefreshBackoff(d.refreshBackoff),
		authnjwks.WithStaleKeyObserver(d.observeStale),
	)
	return d.verifier, nil
}

// UsableKeys y LastRefreshError delegan en el verificador, y son lo que necesita una sonda de
// readiness que no puede permitirse una llamada externa. Sin verificador todavía no hubo un
// solo intento de autenticación: cero claves y ningún error, que NO es un estado de falla.
func (d *DiscoveryClient) UsableKeys() int {
	d.mu.RLock()
	verifier := d.verifier
	d.mu.RUnlock()
	if verifier == nil {
		return 0
	}
	return verifier.UsableKeys()
}

func (d *DiscoveryClient) LastRefreshError() error {
	d.mu.RLock()
	verifier := d.verifier
	lastErr := d.lastErr
	d.mu.RUnlock()
	if verifier == nil {
		return lastErr
	}
	if err := verifier.LastRefreshError(); err != nil {
		return err
	}
	return lastErr
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
		return nil, d.refreshFailed(fmt.Errorf("%w: oidc discovery fetch: %w", authn.ErrProviderUnavailable, err))
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, d.refreshFailed(fmt.Errorf("%w: oidc discovery status %d from %s",
			authn.ErrProviderUnavailable, resp.StatusCode, wellKnown))
	}

	var doc DiscoveryDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, d.refreshFailed(fmt.Errorf("%w: oidc discovery decode: %w", authn.ErrProviderUnavailable, err))
	}

	if doc.Issuer == "" || doc.JWKSURI == "" || doc.AuthorizationEndpoint == "" || doc.TokenEndpoint == "" {
		// Un documento incompleto es el proveedor sirviendo algo que no sirve, no una
		// credencial mala: envuelve el sentinel igual que un 5xx.
		return nil, d.refreshFailed(fmt.Errorf("%w: oidc discovery document is incomplete",
			authn.ErrProviderUnavailable))
	}

	now := d.now()
	d.mu.Lock()
	// Una rotación del jwks_uri invalida el verificador: apuntaba a otro lado. Es la razón
	// por la que el documento se sigue refrescando aunque el verificador ya exista.
	if d.doc != nil && d.doc.JWKSURI != doc.JWKSURI {
		d.verifier = nil
	}
	d.doc = &doc
	d.cachedAt = now
	d.nextRetryAt = time.Time{}
	d.lastErr = nil
	d.mu.Unlock()
	return &doc, nil
}

func (d *DiscoveryClient) refreshFailed(err error) error {
	d.mu.Lock()
	d.lastErr = err
	d.nextRetryAt = d.now().Add(d.refreshBackoff)
	d.mu.Unlock()
	return err
}
