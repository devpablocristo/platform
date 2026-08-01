package jwks

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	authn "github.com/devpablocristo/platform/authn/go"
)

// Los valores por defecto del fallback. Ver WithStaleKeyWindow y WithRefreshBackoff.
const (
	DefaultStaleKeyWindow = 24 * time.Hour
	DefaultRefreshBackoff = 30 * time.Second
)

// StaleObserver se llama CADA VEZ que se sirve algo que ya no se pudo reconfirmar.
//
// Existe porque un fallback silencioso es peor que la caída que viene a evitar: cambiaría un
// error ruidoso por una degradación que nadie ve. Este paquete no loguea ni emite métricas
// —no conoce el destino— pero tampoco puede callarse.
type StaleObserver func(subject string, age time.Duration, cause error)

type Verifier struct {
	jwksURL    string
	httpClient *http.Client
	cacheTTL   time.Duration

	staleWindow    time.Duration
	refreshBackoff time.Duration
	observeStale   StaleObserver
	now            func() time.Time

	mu         sync.RWMutex
	keysByKID  map[string]*rsa.PublicKey
	cacheUntil time.Time
	// fetchedAt es cuándo se trajeron las claves que hay en memoria, y es lo que mide la
	// edad de una clave stale. Distinto de cacheUntil, que sólo dice si están frescas.
	fetchedAt   time.Time
	nextRetryAt time.Time
	lastErr     error
}

// Option configura el verificador. Es variádico en NewVerifier, así que los llamadores que
// existían siguen compilando y reciben los defaults.
type Option func(*Verifier)

// WithStaleKeyWindow acota cuánto tiempo se sigue sirviendo una clave que ya no se pudo
// reconfirmar contra el JWKS. Cero desactiva el fallback.
//
// El default son 24 horas, y la decisión tiene tres patas:
//
//   - El fallback sólo alcanza kids que YA se trajeron con éxito. Una clave nueva es un
//     cache miss: se intenta el refresh y si el proveedor está caído falla. No se inventa
//     confianza, así que rotar una clave HACIA ADENTRO no se ve afectado.
//   - No extiende la vida de ningún token: el `exp` se sigue validando local contra el
//     reloj. Por eso el riesgo NO se compone con la duración de la sesión, que es lo que
//     hace a una clave stale mucho menos peligrosa que una sesión stale.
//   - El único riesgo real es una clave REVOCADA: si el proveedor rota su clave de firma
//     porque se comprometió, se seguiría aceptando durante la ventana. Es una exposición
//     del mismo orden que el TTL máximo de un token, y por eso se acota.
//
// Más de un día no es un transitorio: es un incidente del proveedor que amerita una
// decisión humana, no auto-sanación indefinida.
func WithStaleKeyWindow(window time.Duration) Option {
	return func(v *Verifier) { v.staleWindow = window }
}

// WithRefreshBackoff evita martillar un JWKS que está caído.
//
// Va junto con el fallback y no aparte: sin backoff, cada request con la caché vencida paga
// el timeout completo del cliente HTTP, así que una caída de disponibilidad se convierte
// además en una caída de latencia. Los dos cambios juntos o ninguno.
func WithRefreshBackoff(backoff time.Duration) Option {
	return func(v *Verifier) { v.refreshBackoff = backoff }
}

// WithStaleKeyObserver registra el hook que se llama al servir una clave vencida.
func WithStaleKeyObserver(observe StaleObserver) Option {
	return func(v *Verifier) { v.observeStale = observe }
}

type jwksDocument struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func NewVerifier(jwksURL string, opts ...Option) *Verifier {
	verifier := &Verifier{
		jwksURL:        strings.TrimSpace(jwksURL),
		httpClient:     &http.Client{Timeout: 5 * time.Second},
		cacheTTL:       5 * time.Minute,
		staleWindow:    DefaultStaleKeyWindow,
		refreshBackoff: DefaultRefreshBackoff,
		now:            time.Now,
		keysByKID:      make(map[string]*rsa.PublicKey),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(verifier)
		}
	}
	return verifier
}

// UsableKeys informa cuántas claves hay en memoria SIN salir a la red.
//
// Junto con LastRefreshError son los dos datos que necesita una sonda de readiness, que no
// puede permitirse una llamada externa: un readyz que hace un GET al JWKS flapea con la
// latencia del proveedor y termina bloqueando deploys.
func (v *Verifier) UsableKeys() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.keysByKID)
}

// LastRefreshError devuelve el último fallo de refresh, o nil si el último intento anduvo.
func (v *Verifier) LastRefreshError() error {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.lastErr
}

func (v *Verifier) VerifyToken(ctx context.Context, token string) (map[string]any, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("empty token")
	}
	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(item *jwt.Token) (any, error) {
		kid, _ := item.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("missing kid")
		}
		key, keyErr := v.getKey(ctx, kid)
		if keyErr != nil {
			return nil, keyErr
		}
		return key, nil
	}, jwt.WithValidMethods([]string{"RS256", "RS384", "RS512"}))
	if err != nil {
		return nil, err
	}
	if !parsed.Valid {
		return nil, errors.New("token not valid")
	}

	out := make(map[string]any, len(claims))
	for key, value := range claims {
		out[key] = value
	}
	return out, nil
}

func (v *Verifier) getKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.keysByKID[kid]
	validCache := v.now().Before(v.cacheUntil)
	v.mu.RUnlock()
	if ok && validCache {
		return key, nil
	}

	// El refresh se intenta también cuando las claves están frescas pero el kid no está:
	// puede ser una clave que el proveedor acaba de rotar hacia adentro.
	if err := v.refreshKeys(ctx); err != nil {
		return v.staleKey(kid, err)
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	key, ok = v.keysByKID[kid]
	if !ok {
		// Refresh exitoso y el kid no está: el token lo firmó una clave que este proveedor
		// no publica. Es un token malo, NO una caída, y por eso este error no envuelve el
		// sentinel.
		return nil, fmt.Errorf("kid %s not found", kid)
	}
	return key, nil
}

// staleKey decide si se sirve una clave que ya no se pudo reconfirmar.
//
// `cause` ya envuelve ErrProviderUnavailable, así que devolverla tal cual es lo que hace que
// el consumidor distinga "no pude validar" de "token inválido".
func (v *Verifier) staleKey(kid string, cause error) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.keysByKID[kid]
	fetchedAt := v.fetchedAt
	window := v.staleWindow
	observe := v.observeStale
	age := v.now().Sub(fetchedAt)
	v.mu.RUnlock()

	// Un kid que nunca se trajo con éxito no se inventa: preferir el error a adivinar es
	// toda la diferencia entre degradar y aceptar una firma que nadie confirmó.
	if !ok || window <= 0 || fetchedAt.IsZero() || age > window {
		return nil, cause
	}
	if observe != nil {
		observe(kid, age, cause)
	}
	return key, nil
}

// refreshKeys trae el documento, o devuelve el último error sin salir a la red si el backoff
// todavía corre.
func (v *Verifier) refreshKeys(ctx context.Context) error {
	v.mu.RLock()
	retryAt := v.nextRetryAt
	lastErr := v.lastErr
	v.mu.RUnlock()
	if lastErr != nil && !retryAt.IsZero() && v.now().Before(retryAt) {
		return lastErr
	}
	return v.fetchKeys(ctx)
}

func (v *Verifier) fetchKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		// Una URL mal formada es un error de configuración, no una caída del proveedor: no
		// se guarda backoff porque reintentar no lo va a arreglar.
		return err
	}
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return v.refreshFailed(fmt.Errorf("%w: jwks fetch: %w", authn.ErrProviderUnavailable, err))
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return v.refreshFailed(fmt.Errorf("%w: jwks status %d", authn.ErrProviderUnavailable, resp.StatusCode))
	}

	var doc jwksDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return v.refreshFailed(fmt.Errorf("%w: jwks decode: %w", authn.ErrProviderUnavailable, err))
	}

	next := make(map[string]*rsa.PublicKey, len(doc.Keys))
	for _, item := range doc.Keys {
		if !strings.EqualFold(item.Kty, "RSA") || item.Kid == "" {
			continue
		}
		pub, err := rsaFromModExp(item.N, item.E)
		if err != nil {
			continue
		}
		next[item.Kid] = pub
	}
	if len(next) == 0 {
		// Un documento sin una sola clave usable es un proveedor que no está sirviendo lo
		// que tiene que servir, no una credencial mala.
		return v.refreshFailed(fmt.Errorf("%w: no usable jwks keys", authn.ErrProviderUnavailable))
	}

	now := v.now()
	v.mu.Lock()
	v.keysByKID = next
	v.cacheUntil = now.Add(v.cacheTTL)
	v.fetchedAt = now
	v.nextRetryAt = time.Time{}
	v.lastErr = nil
	v.mu.Unlock()
	return nil
}

// refreshFailed registra el fallo y arma el backoff. Devuelve el mismo error para que el
// llamador lo propague sin recordar hacer las dos cosas.
func (v *Verifier) refreshFailed(err error) error {
	v.mu.Lock()
	v.lastErr = err
	v.nextRetryAt = v.now().Add(v.refreshBackoff)
	v.mu.Unlock()
	return err
}

func rsaFromModExp(nEncoded, eEncoded string) (*rsa.PublicKey, error) {
	nb, err := base64.RawURLEncoding.DecodeString(nEncoded)
	if err != nil {
		return nil, err
	}
	eb, err := base64.RawURLEncoding.DecodeString(eEncoded)
	if err != nil {
		return nil, err
	}
	if len(nb) == 0 || len(eb) == 0 {
		return nil, errors.New("invalid rsa params")
	}
	n := new(big.Int).SetBytes(nb)
	e := int(new(big.Int).SetBytes(eb).Int64())
	if e <= 1 {
		return nil, errors.New("invalid exponent")
	}
	return &rsa.PublicKey{N: n, E: e}, nil
}
