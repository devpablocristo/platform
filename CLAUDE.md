# CLAUDE.md - Reglas para Claude Code

## 1. Contexto

Este repo es el monorepo `core` para capacidades reutilizables compartidas entre productos.

Acá viven módulos por capacidad:

- `saas/`
- `browser/`
- `http/`
- `observability/`
- `config/`
- `security/`
- `validate/`
- `errors/`
- `utils/`
- `concurrency/`
- `authz/`
- `authn/`
- `notifications/`
- `databases/postgres/`
- `databases/dynamodb/`
- `providers/aws/lambda/`
- `providers/aws/s3/`
- `providers/aws/sqs/`
- `eventing/`
- `governance/`
- `artifact/`
- `webhook/`
- `activity/`
- `ai/`

No es un repo de apps ni de features específicas de un producto.

---

## 2. Idioma

- Código: inglés
- Comentarios: español
- TODOs: inglés
- Respuestas: español siempre

---

## 3. Principios

- DRY, YAGNI, SOLID, KISS, fail fast
- Cambios quirúrgicos
- Ideal primero, luego recomendación práctica si difieren
- Verificar antes de afirmar

---

## 4. Antes de proponer cualquier cambio

- OBLIGATORIO: rastrear todos los consumidores y productores afectados
- OBLIGATORIO: revisar impacto cross-module y, si aplica, impacto en repos consumidores
- PROHIBIDO: afirmar "es seguro", "se puede simplificar" o equivalente sin verificar los paths afectados
- Si no verificaste todo, decilo explícitamente

---

## 5. Estructura del repo

El repo se llama `core`, así que adentro NO usamos sufijos `-core`.

Correcto:

```text
core/
  saas/
    go/
  browser/
    ts/
  http/
    go/
    gin/go/
    ts/
    python/
  observability/
    go/
    rust/
  config/
    go/
  security/
    go/
  validate/
    go/
    rust/
  errors/
    go/
    rust/
  utils/
    go/
    pagination/rust/
    resilience/rust/
  concurrency/
    go/
    fsm/rust/
    worker/rust/
  authz/
    go/
  authn/
    ts/
  notifications/
    go/
  databases/
    postgres/
      go/
    dynamodb/
      go/
  providers/
    aws/
      lambda/
        go/
      s3/
        go/
      sqs/
        go/
  eventing/
    go/
  governance/
    go/
  artifact/
    go/
  webhook/
    go/
  activity/
    go/
  ai/
    python/
```

Incorrecto:

```text
saas-core/
backend-core/
```

### Reglas de organización

- Nombrar por capacidad, no por lenguaje
- No crear `common/`, `shared/`, `utils/`, `libs/` en la raíz
- `shared/` solo puede existir dentro de un módulo concreto
- Toda capacidad se separa por lenguaje desde el inicio
- Para Go, usar siempre `{modulo}/go/...`
- Para Python, usar siempre `{modulo}/python/...`
- Para Rust, usar siempre `{modulo}/rust/...`
- Para TypeScript, usar siempre `{modulo}/ts/...`
- No crear `go.mod`, `pyproject.toml`, `Cargo.toml`, `src/` ni paquetes reales en la raíz de la capacidad
- No crear código Go fuera de `go/`, código Python fuera de `python/` ni TypeScript fuera de `ts/`
- No usar una versión global del repo
- Toda implementación concreta debe tener su propio archivo `VERSION`
- Los tags de release se cortan por subdirectorio: `{modulo}/{runtime}/vX.Y.Z`
- Si tiene múltiples implementaciones, usar:

```text
{modulo}/
  go/
```

si suma más runtimes:

```text
{modulo}/
  spec/
  go/
  rust/
  python/
```

Incorrecto aunque hoy solo exista un runtime:

```text
saas/
  go.mod
  identity/

ai/
  pyproject.toml
  src/
```

---

## 6. Qué pertenece a `core`

Pertenece si:

- es reusable entre productos;
- no contiene negocio específico de un producto;
- tiene frontera estable;
- puede exportarse como capacidad reusable.

No pertenece si:

- contiene copy o dominio específico;
- solo sirve para una app;
- todavía no maduró como capacidad compartida.

---

## 7. Fronteras por módulo

### `saas`

- orgs, users, identity, billing, entitlements, usage metering

### `authz`

- autorización reusable por roles y scopes

### `authn`

- autenticación reusable: parsing de credenciales, `jwks`, `oidc`, sesión/browser auth para frontends

### `notifications`

- senders reutilizables `noop`, `smtp`, `ses`

### Módulos técnicos backend

- `http/`: server, transporte y adapters HTTP reutilizables
- `observability/`: métricas, tracing y sinks reutilizables
- `config/`: configuración reusable de servicios
- `security/`: helpers reutilizables de seguridad backend
- `validate/`: validación reusable
- `errors/`: errores técnicos compartidos
- `utils/`: utilidades backend genéricas
- `concurrency/`: primitivas reutilizables de concurrencia

### `databases/postgres`

- pool PostgreSQL, healthcheck, config y migraciones PostgreSQL

### `databases/dynamodb`

- client bootstrap DynamoDB, marshaling reusable y operaciones comunes sobre tablas

### `providers/aws/lambda`

- Lambda, API Gateway y Lambda HTTP reusable

### `providers/aws/s3`

- storage S3 reusable y presigned URLs

### `providers/aws/sqs`

- envío reusable a SQS

### `eventing`

- envelopes y contratos de eventos asíncronos

### `governance`

- policies, risk, approvals, evidence, audit contracts, state machines de decisión

### `artifact`

- PDF, Excel, CSV, QR, report generation, file naming y metadata

### `webhook`

- endpoints outbound, firma HMAC, retries/backoff, replay y delivery planning

### `activity`

- audit append-only, hash chaining, export CSV/JSONL y timeline por entidad

### `ai`

- orchestration, provider factory, auth/rate limit AI, runtime helpers y resilience

### No pertenece a ningún módulo de `core`

- dominio específico de Nexus, Pymes, Ponti, Fixguard o KMA/AlphaCoding
- apps, UIs, dashboards, copy producto

---

## 8. Dependencias entre módulos

- Sin ciclos
- Ningún módulo importa internals de otro
- Por default, cada módulo es autocontenido
- `backend` no depende de otros módulos
- `authz` debe intentar mantenerse independiente
- `notifications` debe intentar mantenerse independiente
- `databases/postgres` debe intentar mantenerse independiente
- `databases/dynamodb` debe intentar mantenerse independiente
- `saas` puede depender de `backend`, `authz` y `notifications`
- `providers/aws/lambda`, `providers/aws/s3` y `providers/aws/sqs` deben intentar mantenerse independientes
- `eventing` debe intentar mantenerse independiente
- `governance`, `artifact`, `webhook`, `activity` y `ai` deben intentar mantenerse independientes

---

## 9. Go

Usar hexagonal solo donde haya dominio real. No forzarla en paquetes técnicos pequeños.

La arquitectura NO se define por la estructura de directorios.

Hexagonal se evalúa por:

- dirección de dependencias;
- dominio aislado de infraestructura;
- puertos/adapters claros.

La estructura `usecases/domain`, `handler/dto` y `repository/models` es convención de este repo para módulos Go con dominio real.

### Patrones válidos

Para dominio/comportamiento:

```text
{contexto}/
  usecases.go
  usecases/domain/entities.go
  handler.go
  handler/dto/dto.go
  repository.go
  repository/models/models.go
```

Para runtime/adapters técnicos:

```text
{package}/
  client.go
  config.go
  errors.go
  types.go
```

### Reglas Go

- `context.Context` primer parámetro
- no `panic()`
- no ignorar errores con `_`
- `slog` por default
- `errors.Is`
- `fmt.Errorf("...: %w", err)`
- interfaces en el consumidor
- accept interfaces, return structs
- DTOs explícitos si hay HTTP
- no modificar migraciones publicadas

---

## 10. Rust

Usar Rust para kernels deterministas/performance-sensitive.

### Reglas Rust

- `unsafe` prohibido salvo pedido explícito
- no `panic!` en library code
- API pública pequeña
- dominio puro separado de adapters
- implementar contra `spec/` si existe

---

## 11. Python

- type hints siempre
- Pydantic o dataclasses para tipos públicos
- `Protocol` para interfaces
- `async/await` para I/O
- no `print()`
- no `except: pass`
- config no hardcodeada
- no retornar `dict` sueltos como API pública

Si hay FastAPI, separar domain/service/schemas/repository/router/dependencies.

---

## 12. Testing

Antes de cerrar un cambio:

- Go: `go build ./...`, `go vet ./...`, `go test ./...`
- Rust: `cargo fmt --check`, `cargo clippy --all-targets --all-features -- -D warnings`, `cargo test`
- Python: `ruff check .`, `mypy .`, `pytest -q`

Si no se puede probar, decirlo explícitamente.

---

## 13. Git

- NUNCA `git push` sin autorización explícita
- NUNCA commit, merge, rebase, reset, checkout o cambio de rama sin autorización explícita
- Permitido: `git status`, `git diff`, `git log`, `git show`

---

## 14. Reglas críticas

- NUNCA valores hardcodeados salvo autorización explícita
- NUNCA meter dominio específico de producto en `core`
- NUNCA crear `common/`, `shared/`, `utils/` en la raíz
- NUNCA duplicar una capacidad en dos módulos
- NUNCA mezclar implementación directa en la raíz de una capacidad
- NUNCA afirmar que algo está listo sin evidencia de verificación

Fuente de verdad equivalente para GPT/Codex: `AGENTS.md`.
