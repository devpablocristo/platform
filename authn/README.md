# authn

Capacidad reusable para autenticación, tanto backend como sesión/browser.

Implementaciones actuales:

- `authn/go/`
- `authn/ts/`

## Pertenece

- parsing de credenciales backend
- verificación `jwks`
- helpers `oidc`
- token providers de browser
- fetch helpers con bearer/api-key
- storage namespaced de access/refresh tokens
- eventos de logout forzado
- clientes HTTP con refresh serializado
- adapters opcionales a providers frontend

## No pertenece

- pantallas concretas de login
- copy/UI de auth
- rutas de una app específica
- prompts o lógica de negocio de producto

## Fuente inicial esperada

- `/home/pablo/Projects/Pablo/core/saas/go/identity/executor`
- `/home/pablo/Projects/Pablo/ponti/ponti-frontend/ui/src/pages/login`
- `/home/pablo/Projects/Pablo/ponti/ponti-frontend/ui/src/api/client.ts`
- `/home/pablo/Projects/Pablo/pymes/pkgs/ts-pkg`

## Estructura actual

- `authn/go/` — inbound: `Principal`, `Credential`, `Authenticator`, `Extractor`, `TryInbound`, `BearerJWTAuthenticator`, `APIKeyFuncAuthenticator` (sin “provider universal”: refresh/OAuth/sesiones van fuera de esta capa). Ejemplos: [go/README.md](go/README.md).
- `authn/go/jwks/`
- `authn/go/oidc/` — discovery + verificación + intercambio de código (flujo federado; no mezclar con API key)
- `authn/ts/src/browser/`
- `authn/ts/src/http/`
- `authn/ts/src/providers/`
- `authn/ts/tests/`

## Principio de diseño (Go)

Una sola abstracción inflada (`VerifyToken` + `RefreshToken` + `RevokeSession` en una interfaz) rompe con API keys y JWT stateless. Acá solo: **extraer credencial → autenticar → `Principal`**. El middleware `saas/go/middleware.AuthMiddleware` sigue el mismo orden que `authn.TryInbound` (JWT primero, luego API key si el JWT falla o no hay Bearer).
