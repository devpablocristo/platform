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
  authn "github.com/devpablocristo/core/authn/go"
  "github.com/devpablocristo/core/authn/go/jwks"
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

`core/saas/go/middleware.AuthMiddleware` delega en `TryInbound` con adaptadores al `PrincipalVerifier` del kernel.

## Versión

Ver `VERSION` en este directorio.
