# authn/go

Autenticación **inbound** para servicios HTTP: credencial → verificación → `Principal`.

No incluye: refresh obligatorio, revoke de sesión, ni login browser completo como una sola interfaz “universal”. OIDC (redirect + exchange) vive en `oidc/` como flujo aparte.

## Paquetes

| Paquete | Rol |
|---------|-----|
| Raíz (`authn`) | Tipos `Principal`, `Credential`, `Authenticator`, `TryInbound`, extractores, `BearerJWTAuthenticator`, `APIKeyFuncAuthenticator` |
| `jwks` | Verificación JWT RS256/384/512 contra JWKS remoto (caché) |
| `oidc` | Discovery `.well-known`, verificación vía JWKS del issuer, PKCE + intercambio de código |

## Ejemplo: JWT + JWKS + mapa de claims

```go
import (
  "context"
  authn "github.com/devpablocristo/platform/authn/go"
  "github.com/devpablocristo/platform/authn/go/jwks"
)

v := jwks.NewVerifier("https://issuer.example/.well-known/jwks.json")
a := &authn.BearerJWTAuthenticator{
  Verify: v,
  Map: func(ctx context.Context, claims map[string]any) (authn.Principal, error) {
    // Mapear org_id, roles, scopes según tu producto
    return authn.Principal{OrgID: "...", Actor: "..."}, nil
  },
}
p, err := a.Authenticate(ctx, authn.BearerCredential{Token: rawBearer})
```

## Ejemplo: mismo orden que SaaS (Bearer luego API key)

```go
p, method, err := authn.TryInbound(ctx, jwtAuth, apiKeyAuth, authorizationHeader, xAPIKeyHeader)
// method: "jwt" | "api_key"
```

`platform/kernels/saas/go/middleware.AuthMiddleware` delega en `TryInbound` con adaptadores al `PrincipalVerifier` del kernel.

## Cuando el proveedor de identidad se cae

Desde `0.3.0`, una caída del JWKS o del `.well-known` **no rechaza a quien tiene un token
válido**, y el motivo del rechazo es distinguible cuando sí hay que rechazar.

**`ErrProviderUnavailable`** significa "no pude verificar", que no es "la credencial es
inválida". Un consumidor que colapse las dos cosas en 401 le miente al cliente —lo manda a
loguearse de nuevo con un token perfecto— y además se queda sin señal, porque un 401 no es un
error de nadie y ninguna alarma lo mira. El mapeo natural es **503**:

```go
if errors.Is(err, authn.ErrProviderUnavailable) {
  w.Header().Set("Retry-After", "30")
  http.Error(w, "identity provider unavailable", http.StatusServiceUnavailable)
  return
}
```

**Claves y documento stale.** Una clave que ya se trajo con éxito se sigue sirviendo hasta 24 h
después de que el proveedor deja de responder (`WithStaleKeyWindow`), y el documento de
discovery igual (`WithStaleWindow`). Tres propiedades hacen que eso sea aceptable:

- El fallback **sólo** alcanza `kid`s ya confirmados. Una clave nueva es un cache miss, así que
  rotar una clave *hacia adentro* no se ve afectado.
- **No extiende la vida de ningún token**: el `exp` se sigue validando local contra el reloj.
  El riesgo no se compone con la duración de la sesión.
- El único riesgo real es una clave **revocada**, que se seguiría aceptando durante la ventana.
  Es una exposición del mismo orden que el TTL máximo de un token, y por eso se acota.

**Nunca es silencioso.** `WithStaleKeyObserver` / `WithStaleObserver` se llaman cada vez que se
sirve algo vencido. Sin ese hook, esto cambiaría una caída ruidosa por una degradación que
nadie ve, que es peor:

```go
client := oidc.NewDiscoveryClient(issuer,
  oidc.WithStaleObserver(func(subject string, age time.Duration, cause error) {
    // métrica + log: es lo que permite enterarse ANTES de que haya impacto
  }))
```

**Readiness sin red.** `UsableKeys()` y `LastRefreshError()` responden desde memoria. Una sonda
que hiciera un `GET` al JWKS flapearía con la latencia del proveedor y terminaría bloqueando
deploys. Servir stale es *degradado*, no *no-listo*: la sonda sólo debería fallar con cero
claves usables **y** un último refresh fallido.

`WithRefreshBackoff` (30 s por default) va junto con el fallback y no aparte: sin él cada
request con la caché vencida paga el timeout HTTP completo, así que la caída de disponibilidad
se convierte además en una caída de latencia.

## Versión

Ver `VERSION` en este directorio.
