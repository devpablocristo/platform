# Arquitectura — `platform`

> Generado de hechos de Codebase Memory (grafo del código, no adivinado) y anclado con drift a los
> símbolos que describe. Escala: **7.338 nodos / 27.890 edges** · Go 307 · TS 127 · Python 76 · Rust 68.

## Qué es

Un **monorepo políglota de kernels reutilizables** para construir SaaS multi-tenant. No es un producto:
es la base (identidad, tenancy, HTTP, errores, persistencia, billing, observabilidad, eventing) sobre la
que se montan productos. `features/scheduling` es el **consumidor de referencia** (un SaaS real de
reservas/colas). Cada capacidad se publica por lenguaje (`go/`, `ts/`, `rust/`, `python/`) según dónde
se consuma.

## Capas (del grafo: fan-in/out)

- **core** (alto fan-in, lo reusado por todos): `security`(tenant), `errors`, `persistence/gorm`,
  `databases/postgres`, `observability`, `http`, `sdks`(aws/clerk/stripe).
- **entry** (consumidores): `features/scheduling`, `kernels/saas`, `kernels/ai`, `http/gin`.
- **internal**: paquetes TS de front.

## Kernels compartidos (los hotspots reales — fan_in del grafo)

Estos son los símbolos más reusados = el contrato de facto del kernel:

| símbolo | fan_in | módulo | qué es |
|---|--:|---|---|
| `security/go/tenant.String` | **77** | security | el tipo TenantID (identidad de tenant en todo el kernel) |
| `sdks/aws/lambda/go/lambdahttp.JSON` | 76 | sdks | responder JSON para Lambda |
| `persistence/gorm/go/tenancy.Where` | 54 | persistence | **scoping por tenant en gorm** (fila-nivel) |
| `errors/go/domainerr.New` / `.Validation` | 54 / 44 | errors | **kernel de errores de dominio** (Kind, mapeo HTTP) |
| `databases/postgres/go.Close` | 53 | databases | pool Postgres |
| `observability/go.WriteHeader` | 52 | observability | middleware de tracing/log |
| `http/gin/go.Respond` | 47 | http | **responder HTTP unificado** |
| `http/ts/src/fetch.request` | 45 | http | cliente fetch TS |

> ⚠️ **Duplicación con ponti**: `http.Respond`, `errors/domainerr`, `persistence/tenancy.Where` y el
> tipo `tenant` son primitivas que `ponti/apps/core` reimplementa (RespondError, sus error types, su
> tenant scoping). Son los candidatos directos a que ponti consuma platform (ver
> `ponti/docs/architecture/core.md`).

## Módulos (por directorio · políglota)

### `kernels/` — los kernels de producto
`saas` (Go: **identidad multi-tenant** — `ClaimsResolver`, `MembershipResolver`, adapters
clerk/cognito/firebase; ver más abajo), `ai`, `activity`, `artifact`, `governance`.

### `authn/` — autenticación (Go + Rust + TS)
Cluster de 86 nodos (cohesión 0.84): `NewAuthenticator`/`Authenticate`/`sign`/`ValidateCapabilityManifest`
+ `authn/go/jwks` (verify RS256, JWK n/e — Firebase-compatible). TS: `authn/ts` (browser token storage,
providers clerk, axios/fetch autenticados). Cluster 141 (identity errors: `TenantMissing`, `FromString`).

### `http/` — transporte (Go + gin + Python + TS + client/server)
Cluster kernel de 240 nodos (cohesión 0.87): `HandlerFunc`/`WriteHeader`/`Respond`. El responder y el
cliente fetch viven acá.

### `errors/` (Go + Rust) — `domainerr` (Kind/New/Validation), mapeo a HTTP.
### `persistence/` + `databases/` — gorm tenancy + pools Postgres/DynamoDB.
### `security/` — el tipo `tenant` (TenantID), scoping.
### `sdks/` — aws (lambda/dynamodb), clerk, stripe.
### `kernels/saas` billing — cluster 65: `CreateCheckoutSession`/`handleStripeWebhook`/`ApplyOrgPlanChange` (Stripe).
### Otros: `authz`, `eventing`, `observability`, `notifications`, `jobs`, `ingestion`, `lifecycle`, `concurrency`, `config`, `contracts`, `validate`, `webhook`, `browser`, `ui`, `utils`, `testing`, `tooling`.

## `kernels/saas/go/identity` — el kernel de identidad (donde aterrizó A1/A2)

- `ClaimsResolver` — deriva el `Principal` de custom claims del token (org/role/scopes), fail-closed
  en issuer/audience.
- `MembershipResolver` (A2) — para IdPs que **no** ponen org en el token (Firebase/Identity Platform):
  verifica el token y deriva tenant/rol/scopes desde las **memberships** del actor, con política de
  selección 0/1/>1 (`WithRequestedTenant` para el caso >1). Satisface `PrincipalVerifier` →
  intercambiable con `ClaimsResolver` en el middleware.
- Adapters (A1): `adapters/firebase` (`NewVerifier`/`IssuerFor`/`ClaimsConfig`), clerk, cognito.

## Features (consumidores de referencia)

`features/scheduling` (Go backend + TS/React front): el SaaS de reservas. Rutas
`/v1/scheduling/{bookings,calendar-events,blocked-ranges,queues,services,branches}` con máquina de estados
de booking (confirm/cancel/check-in/start-service/complete/no-show/reschedule) y colas
(`queues/:id/tickets`, pause). Otras: `calendar-board`, `kanban-board`, `chat`, `conversation-inbox`,
`notification-feed`, `search`, `admin-insights`, `crud`.

## Gaps de documentación (ver reporte)

- No había `ARCHITECTURE.md` vigente (este doc lo cubre); `docs/core/` es legacy.
- `CLAUDE.md`/`GPT.md` desalineados; sin `docs/adr/`; Rust sin tags de release.
