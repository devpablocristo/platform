package jwks

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	authn "github.com/devpablocristo/platform/authn/go"
)

// Estos tests cubren una sola pregunta: cuando el proveedor de identidad se cae, ¿el sistema
// rechaza a todo el mundo o sigue atendiendo a quienes tienen un token válido?
//
// El comportamiento anterior era el primero, y de la peor forma: pasados 5 minutos —el TTL de
// la caché— cada request fallaba con el MISMO error que un token falsificado, así que el
// consumidor devolvía 401, el cliente se deslogueaba, y ninguna alarma miraba nada porque un
// 401 no es un error de nadie.

// fakeJWKS es un JWKS que se puede tirar abajo y que cuenta cuántas veces lo golpearon.
//
// El contador es la mitad del punto: un fallback sin backoff sigue pagando el timeout HTTP en
// cada request, así que la caída de disponibilidad se vuelve una caída de latencia.
type fakeJWKS struct {
	server *httptest.Server

	mu     sync.Mutex
	body   []byte
	status int
	hits   int
}

func newFakeJWKS(t *testing.T, kid string, pub *rsa.PublicKey) *fakeJWKS {
	t.Helper()
	fake := &fakeJWKS{body: mustJWKSJSON(t, kid, pub), status: http.StatusOK}
	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fake.mu.Lock()
		fake.hits++
		status, body := fake.status, fake.body
		fake.mu.Unlock()

		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(fake.server.Close)
	return fake
}

func (f *fakeJWKS) breakWith(status int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status = status
}

func (f *fakeJWKS) serve(body []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status = http.StatusOK
	f.body = body
}

func (f *fakeJWKS) hitCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.hits
}

// testClock controla el tiempo del verificador. El `exp` del token lo sigue validando la
// librería contra el reloj de verdad, que es exactamente la propiedad que hace que una clave
// stale no extienda la vida de ninguna sesión.
type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func newTestClock() *testClock {
	return &testClock{now: time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)}
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// signedToken arma un token firmado con `exp` bien en el futuro: acá nunca se prueba la
// expiración, se prueba de dónde sale la clave.
func signedToken(t *testing.T, priv *rsa.PrivateKey, kid string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": "user-1",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	})
	token.Header["kid"] = kid
	signed, err := token.SignedString(priv)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

// resilientVerifier arma un verificador con reloj controlado apuntando al fake.
func resilientVerifier(fake *fakeJWKS, clock *testClock, opts ...Option) *Verifier {
	verifier := NewVerifier(fake.server.URL, opts...)
	verifier.httpClient = fake.server.Client()
	verifier.now = clock.Now
	return verifier
}

func rsaKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return priv
}

func TestAKeyAlreadyFetchedKeepsWorkingWhileTheJWKSIsDown(t *testing.T) {
	t.Parallel()

	priv := rsaKey(t)
	fake := newFakeJWKS(t, "kid-1", &priv.PublicKey)
	clock := newTestClock()
	verifier := resilientVerifier(fake, clock)
	token := signedToken(t, priv, "kid-1")

	if _, err := verifier.VerifyToken(context.Background(), token); err != nil {
		t.Fatalf("el primer viaje tenía que andar: %v", err)
	}

	// El JWKS se cae y la caché vence. Antes, acá empezaba el 401 masivo.
	fake.breakWith(http.StatusInternalServerError)
	clock.advance(10 * time.Minute)

	claims, err := verifier.VerifyToken(context.Background(), token)
	if err != nil {
		t.Fatalf("un token válido tiene que seguir andando con el JWKS caído: %v", err)
	}
	if claims["sub"] != "user-1" {
		t.Errorf("claims: %#v", claims)
	}
}

func TestAStaleKeyIsNotServedBeyondItsBound(t *testing.T) {
	t.Parallel()

	priv := rsaKey(t)
	fake := newFakeJWKS(t, "kid-1", &priv.PublicKey)
	clock := newTestClock()
	verifier := resilientVerifier(fake, clock, WithStaleKeyWindow(time.Hour), WithRefreshBackoff(0))
	token := signedToken(t, priv, "kid-1")

	if _, err := verifier.VerifyToken(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	fake.breakWith(http.StatusInternalServerError)

	clock.advance(59 * time.Minute)
	if _, err := verifier.VerifyToken(context.Background(), token); err != nil {
		t.Fatalf("dentro de la ventana tenía que servir la clave vencida: %v", err)
	}

	// Fuera de la ventana se deja de adivinar. Una caída más larga que el tope no es un
	// transitorio: es un incidente que amerita que alguien lo mire.
	clock.advance(2 * time.Minute)
	_, err := verifier.VerifyToken(context.Background(), token)
	if err == nil {
		t.Fatal("pasada la ventana, servir la clave sería aceptar una firma que nadie confirmó")
	}
	if !errors.Is(err, authn.ErrProviderUnavailable) {
		t.Errorf("el error tiene que decir que no se pudo verificar, no que el token es malo: %v", err)
	}
}

func TestAKidThatWasNeverFetchedIsNotInvented(t *testing.T) {
	t.Parallel()

	priv := rsaKey(t)
	fake := newFakeJWKS(t, "kid-1", &priv.PublicKey)
	clock := newTestClock()
	verifier := resilientVerifier(fake, clock)

	if _, err := verifier.VerifyToken(context.Background(), signedToken(t, priv, "kid-1")); err != nil {
		t.Fatal(err)
	}
	fake.breakWith(http.StatusInternalServerError)
	clock.advance(10 * time.Minute)

	// Un kid que nunca se trajo con éxito no está en la caché, así que el fallback no lo
	// alcanza. Rotar una clave HACIA ADENTRO no se ve afectado por el fallback.
	_, err := verifier.VerifyToken(context.Background(), signedToken(t, priv, "kid-nuevo"))
	if err == nil {
		t.Fatal("una clave que nunca se confirmó no se puede aceptar")
	}
	if !errors.Is(err, authn.ErrProviderUnavailable) {
		t.Errorf("con el proveedor caído no se sabe si el kid existe: no se puede afirmar que el token es malo: %v", err)
	}
}

// TestAnInfrastructureFailureIsDistinguishableFromABadToken es la tabla que codifica toda la
// decisión, y de paso prueba que el sentinel sobrevive al envoltorio de jwt.ParseWithClaims
// —el keyfunc devuelve el error adentro del parser, y si no se pudiera alcanzar con errors.Is
// desde afuera todo lo demás no serviría de nada—.
func TestAnInfrastructureFailureIsDistinguishableFromABadToken(t *testing.T) {
	t.Parallel()

	priv := rsaKey(t)
	other := rsaKey(t)

	cases := []struct {
		name        string
		unavailable bool
		// arrange devuelve el token a verificar y deja el fake en el estado del caso.
		arrange func(t *testing.T, fake *fakeJWKS) string
	}{
		{
			name:        "el JWKS devuelve 500",
			unavailable: true,
			arrange: func(_ *testing.T, fake *fakeJWKS) string {
				fake.breakWith(http.StatusInternalServerError)
				return signedToken(t, priv, "kid-1")
			},
		},
		{
			name:        "el JWKS devuelve 404",
			unavailable: true,
			arrange: func(_ *testing.T, fake *fakeJWKS) string {
				fake.breakWith(http.StatusNotFound)
				return signedToken(t, priv, "kid-1")
			},
		},
		{
			name:        "el cuerpo del JWKS no decodifica",
			unavailable: true,
			arrange: func(_ *testing.T, fake *fakeJWKS) string {
				fake.serve([]byte("esto no es json"))
				return signedToken(t, priv, "kid-1")
			},
		},
		{
			name:        "el JWKS no tiene una sola clave usable",
			unavailable: true,
			arrange: func(_ *testing.T, fake *fakeJWKS) string {
				fake.serve([]byte(`{"keys":[]}`))
				return signedToken(t, priv, "kid-1")
			},
		},
		{
			name:        "el kid no está DESPUÉS de un refresh exitoso",
			unavailable: false,
			arrange: func(_ *testing.T, _ *fakeJWKS) string {
				return signedToken(t, priv, "kid-que-no-publica")
			},
		},
		{
			name:        "la firma es de otra clave",
			unavailable: false,
			arrange: func(_ *testing.T, _ *fakeJWKS) string {
				return signedToken(t, other, "kid-1")
			},
		},
		{
			name:        "el token no declara kid",
			unavailable: false,
			arrange: func(t *testing.T, _ *fakeJWKS) string {
				token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"sub": "x"})
				signed, err := token.SignedString(priv)
				if err != nil {
					t.Fatal(err)
				}
				return signed
			},
		},
		{
			name:        "el token está expirado",
			unavailable: false,
			arrange: func(t *testing.T, _ *fakeJWKS) string {
				token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
					"sub": "x",
					"exp": float64(time.Now().Add(-time.Hour).Unix()),
				})
				token.Header["kid"] = "kid-1"
				signed, err := token.SignedString(priv)
				if err != nil {
					t.Fatal(err)
				}
				return signed
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			fake := newFakeJWKS(t, "kid-1", &priv.PublicKey)
			// Sin ventana de stale: cada caso arranca sin nada cacheado, así que lo que se
			// mide es la clasificación del error y no el fallback.
			verifier := resilientVerifier(fake, newTestClock(), WithStaleKeyWindow(0))
			token := testCase.arrange(t, fake)

			_, err := verifier.VerifyToken(context.Background(), token)
			if err == nil {
				t.Fatal("todos estos casos tienen que fallar")
			}
			got := errors.Is(err, authn.ErrProviderUnavailable)
			if got != testCase.unavailable {
				t.Errorf("ErrProviderUnavailable=%v, esperado %v\nerror: %v", got, testCase.unavailable, err)
			}
		})
	}
}

func TestTheJWKSIsNotHammeredWhileItIsDown(t *testing.T) {
	t.Parallel()

	priv := rsaKey(t)
	fake := newFakeJWKS(t, "kid-1", &priv.PublicKey)
	clock := newTestClock()
	verifier := resilientVerifier(fake, clock, WithRefreshBackoff(30*time.Second))
	token := signedToken(t, priv, "kid-1")

	if _, err := verifier.VerifyToken(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	fake.breakWith(http.StatusInternalServerError)
	clock.advance(10 * time.Minute)

	before := fake.hitCount()
	for range 20 {
		if _, err := verifier.VerifyToken(context.Background(), token); err != nil {
			t.Fatalf("el fallback tenía que sostener estos requests: %v", err)
		}
	}
	if hits := fake.hitCount() - before; hits != 1 {
		t.Errorf("dentro de la ventana de backoff el JWKS se golpea UNA vez y se golpeó %d: "+
			"sin backoff cada request paga el timeout HTTP completo y la caída de "+
			"disponibilidad se vuelve además una caída de latencia", hits)
	}

	// Vencido el backoff se reintenta, y cuando el proveedor vuelve el sistema se recupera
	// solo sin que nadie lo toque.
	clock.advance(31 * time.Second)
	fake.serve(mustJWKSJSON(t, "kid-1", &priv.PublicKey))
	if _, err := verifier.VerifyToken(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	if verifier.LastRefreshError() != nil {
		t.Errorf("un refresh exitoso tiene que limpiar el último error: %v", verifier.LastRefreshError())
	}
}

func TestServingAStaleKeyIsNeverSilent(t *testing.T) {
	t.Parallel()

	priv := rsaKey(t)
	fake := newFakeJWKS(t, "kid-1", &priv.PublicKey)
	clock := newTestClock()

	type observation struct {
		subject string
		age     time.Duration
		cause   error
	}
	var (
		mu   sync.Mutex
		seen []observation
	)
	verifier := resilientVerifier(fake, clock, WithStaleKeyObserver(
		func(subject string, age time.Duration, cause error) {
			mu.Lock()
			defer mu.Unlock()
			seen = append(seen, observation{subject, age, cause})
		}))
	token := signedToken(t, priv, "kid-1")

	if _, err := verifier.VerifyToken(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	fake.breakWith(http.StatusInternalServerError)
	clock.advance(20 * time.Minute)
	if _, err := verifier.VerifyToken(context.Background(), token); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 1 {
		t.Fatalf("servir una clave vencida tiene que avisar exactamente una vez y avisó %d veces: "+
			"sin este hook el arreglo cambia una caída ruidosa por una degradación que nadie ve, "+
			"que es peor", len(seen))
	}
	if seen[0].subject != "kid-1" {
		t.Errorf("el aviso tiene que nombrar la clave: %q", seen[0].subject)
	}
	if seen[0].age != 20*time.Minute {
		t.Errorf("la edad informada tiene que ser real, no cero: %v", seen[0].age)
	}
	if !errors.Is(seen[0].cause, authn.ErrProviderUnavailable) {
		t.Errorf("el aviso tiene que llevar por qué no se pudo reconfirmar: %v", seen[0].cause)
	}
}

func TestReadinessCanBeAnsweredWithoutTouchingTheNetwork(t *testing.T) {
	t.Parallel()

	priv := rsaKey(t)
	fake := newFakeJWKS(t, "kid-1", &priv.PublicKey)
	clock := newTestClock()
	verifier := resilientVerifier(fake, clock)

	// Recién construido: cero claves y NINGÚN error. No es un estado de falla, es que
	// todavía nadie se autenticó — y un readyz que fallara acá rompería todo deploy.
	if verifier.UsableKeys() != 0 || verifier.LastRefreshError() != nil {
		t.Fatalf("estado inicial: claves=%d error=%v", verifier.UsableKeys(), verifier.LastRefreshError())
	}

	before := fake.hitCount()
	if _, err := verifier.VerifyToken(context.Background(), signedToken(t, priv, "kid-1")); err != nil {
		t.Fatal(err)
	}
	if verifier.UsableKeys() != 1 {
		t.Errorf("claves usables: %d", verifier.UsableKeys())
	}

	fake.breakWith(http.StatusInternalServerError)
	clock.advance(10 * time.Minute)
	if _, err := verifier.VerifyToken(context.Background(), signedToken(t, priv, "kid-1")); err != nil {
		t.Fatal(err)
	}
	// Degradado, no caído: hay claves Y hay error. Distinguirlo es lo que permite que la
	// sonda no falle mientras el servicio sigue atendiendo.
	if verifier.UsableKeys() == 0 {
		t.Error("sirviendo stale todavía hay claves usables")
	}
	if !errors.Is(verifier.LastRefreshError(), authn.ErrProviderUnavailable) {
		t.Errorf("el último error tiene que estar disponible para la sonda: %v", verifier.LastRefreshError())
	}
	if fake.hitCount() <= before {
		t.Error("el test no ejercitó ningún viaje real")
	}
}
