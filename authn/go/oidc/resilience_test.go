package oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	authn "github.com/devpablocristo/platform/authn/go"
)

// fakeProvider sirve el .well-known y el JWKS, y cada mitad se puede tirar abajo por separado.
// Que se puedan romper por separado es el punto: el defecto que estos tests cubren es que la
// caída de UNA mitad rompía la verificación aunque la otra estuviera perfecta.
type fakeProvider struct {
	server *httptest.Server

	mu             sync.Mutex
	discoveryDown  bool
	jwksDown       bool
	discoveryHits  int
	kid            string
	jwksBody       []byte
	issuerOverride string
}

func newFakeProvider(t *testing.T, pub *rsa.PublicKey, kid string) *fakeProvider {
	t.Helper()
	fake := &fakeProvider{kid: kid, jwksBody: jwksJSON(t, kid, pub)}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		fake.mu.Lock()
		fake.discoveryHits++
		down := fake.discoveryDown
		issuer := fake.issuerOverride
		fake.mu.Unlock()
		if down {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if issuer == "" {
			issuer = fake.server.URL
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 issuer,
			"authorization_endpoint": issuer + "/authorize",
			"token_endpoint":         issuer + "/token",
			"jwks_uri":               fake.server.URL + "/jwks",
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		fake.mu.Lock()
		down := fake.jwksDown
		body := fake.jwksBody
		fake.mu.Unlock()
		if down {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	})
	fake.server = httptest.NewServer(mux)
	t.Cleanup(fake.server.Close)
	return fake
}

func (f *fakeProvider) breakDiscovery() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.discoveryDown = true
}

func (f *fakeProvider) discoveryCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.discoveryHits
}

type clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// TestAUsableVerifierSurvivesADiscoveryOutage cubre el defecto más caro de los dos, porque es
// el que en la práctica decidía cuándo empezaba el 401 masivo.
//
// Verifier() pedía el documento de discovery SIEMPRE, antes de devolver el verificador que ya
// tenía construido. Así que el TTL que en realidad cortaba la autenticación era el del
// documento —10 minutos— y no el de las claves, y cortaba aunque las claves estuvieran
// perfectas en memoria. El documento sólo dice DÓNDE está el JWKS: si ya hay verificador, eso
// ya se sabe y no hace falta volver a preguntarlo para validar una firma.
func TestAUsableVerifierSurvivesADiscoveryOutage(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	fake := newFakeProvider(t, &priv.PublicKey, "kid-1")
	moment := &clock{now: time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)}

	client := NewDiscoveryClient(fake.server.URL, WithStaleWindow(time.Hour))
	client.httpClient = fake.server.Client()
	client.now = moment.Now

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": "user-1",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	})
	token.Header["kid"] = "kid-1"
	signed, err := token.SignedString(priv)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := client.VerifyToken(context.Background(), signed); err != nil {
		t.Fatalf("el primer viaje tenía que andar: %v", err)
	}

	// El .well-known se cae y su caché vence. El JWKS sigue perfecto.
	fake.breakDiscovery()
	moment.advance(20 * time.Minute)

	claims, err := client.VerifyToken(context.Background(), signed)
	if err != nil {
		t.Fatalf("con el JWKS sano, una caída del discovery no puede romper la verificación: %v", err)
	}
	if claims["sub"] != "user-1" {
		t.Errorf("claims: %#v", claims)
	}

	// Y pasada la ventana de stale del documento tampoco, porque el verificador ya existe:
	// el documento no vuelve a hacer falta para validar una firma.
	moment.advance(2 * time.Hour)
	if _, err := client.VerifyToken(context.Background(), signed); err != nil {
		t.Errorf("el verificador ya construido no depende del documento: %v", err)
	}
}

func TestTheDiscoveryDocumentIsServedStaleAndItsFailureIsTyped(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	fake := newFakeProvider(t, &priv.PublicKey, "kid-1")
	moment := &clock{now: time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)}

	var (
		mu      sync.Mutex
		avisos  []string
		lastAge time.Duration
	)
	client := NewDiscoveryClient(fake.server.URL,
		WithStaleWindow(time.Hour),
		WithRefreshBackoff(30*time.Second),
		WithStaleObserver(func(subject string, age time.Duration, _ error) {
			mu.Lock()
			defer mu.Unlock()
			avisos = append(avisos, subject)
			lastAge = age
		}))
	client.httpClient = fake.server.Client()
	client.now = moment.Now

	if _, err := client.Discover(context.Background()); err != nil {
		t.Fatal(err)
	}
	fake.breakDiscovery()
	moment.advance(20 * time.Minute)

	doc, err := client.Discover(context.Background())
	if err != nil {
		t.Fatalf("dentro de la ventana el documento vencido se sirve: %v", err)
	}
	if doc.JWKSURI == "" {
		t.Error("el documento servido tiene que ser el real, no uno vacío")
	}

	mu.Lock()
	if len(avisos) != 1 || avisos[0] != "discovery document" {
		t.Errorf("servir un documento vencido tiene que avisar: %v", avisos)
	}
	if lastAge != 20*time.Minute {
		t.Errorf("la edad informada tiene que ser real: %v", lastAge)
	}
	mu.Unlock()

	// El backoff también aplica al discovery: no se martilla un .well-known caído.
	before := fake.discoveryCount()
	for range 10 {
		if _, err := client.Discover(context.Background()); err != nil {
			t.Fatalf("el fallback tenía que sostener estas lecturas: %v", err)
		}
	}
	if hits := fake.discoveryCount() - before; hits != 0 {
		t.Errorf("dentro del backoff no se vuelve a pedir el documento y se pidió %d veces", hits)
	}

	// Fuera de la ventana se deja de servir, y el error dice que no se pudo verificar.
	moment.advance(2 * time.Hour)
	_, err = client.Discover(context.Background())
	if err == nil {
		t.Fatal("pasada la ventana el documento vencido no se sirve más")
	}
	if !errors.Is(err, authn.ErrProviderUnavailable) {
		t.Errorf("el fallo del proveedor tiene que ser distinguible de una credencial mala: %v", err)
	}
}

func TestReadinessSurvivesACompleteOutageWithoutLyingAboutIt(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	fake := newFakeProvider(t, &priv.PublicKey, "kid-1")
	client := NewDiscoveryClient(fake.server.URL)
	client.httpClient = fake.server.Client()

	// Recién construido: sin verificador todavía no hubo un solo intento de autenticación.
	// Cero claves y ningún error NO es un estado de falla, y una sonda que fallara acá
	// bloquearía cualquier deploy.
	if client.UsableKeys() != 0 {
		t.Errorf("claves antes del primer uso: %d", client.UsableKeys())
	}
	if client.LastRefreshError() != nil {
		t.Errorf("no puede haber error antes del primer intento: %v", client.LastRefreshError())
	}

	fake.breakDiscovery()
	if _, err := client.Discover(context.Background()); err == nil {
		t.Fatal("sin documento y sin caché no hay nada que servir")
	}
	// Ahora sí hay un fallo real, y la sonda tiene que poder verlo.
	if !errors.Is(client.LastRefreshError(), authn.ErrProviderUnavailable) {
		t.Errorf("la sonda tiene que poder leer el último fallo: %v", client.LastRefreshError())
	}
}

func jwksJSON(t *testing.T, kid string, pub *rsa.PublicKey) []byte {
	t.Helper()
	exponent := big.NewInt(int64(pub.E))
	body, err := json.Marshal(map[string]any{
		"keys": []map[string]any{{
			"kty": "RSA",
			"kid": kid,
			"alg": "RS256",
			"use": "sig",
			"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(exponent.Bytes()),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return body
}
