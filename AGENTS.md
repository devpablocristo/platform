# core — Reglas para agentes

## 1. Contexto

Este repo es el monorepo de capacidades reutilizables para productos como Nexus, Pymes, Ponti, Fixguard y AlphaCoding.

Acá solo vive código reusable, estable y agnóstico al producto.

No es un repo de apps.
No es un repo de UIs.
No es un repo de features de negocio específicas de un producto.

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
- No introducir abstracción prematura
- No duplicar lógica entre módulos ni entre productos si ya pertenece a este repo

---

## 4. Flujo obligatorio

1. Antes de proponer un cambio, rastrear todos los consumidores y productores afectados.
2. Si el cambio es cross-module o no trivial, explicar primero:
   - forma ideal
   - forma recomendada
3. Si no verificaste todos los paths afectados, decirlo explícitamente.
4. No decir "listo", "funciona" o equivalente sin evidencia de verificación en este turno.

---

## 5. Estructura objetivo del repo

El repo se llama `core`, por lo tanto adentro NO usamos sufijos `-core`.

La estructura objetivo es:

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
  docs/
  scripts/
  examples/
```

### Regla de naming

- Los módulos se nombran por capacidad o adapter concreto estable: `saas`, `browser`, `http`, `observability`, `config`, `security`, `validate`, `errors`, `utils`, `concurrency`, `authz`, `authn`, `notifications`, `databases/postgres`, `databases/dynamodb`, `providers/aws/lambda`, `providers/aws/s3`, `providers/aws/sqs`, `eventing`, `governance`, `artifact`, `webhook`, `activity`, `ai`
- NUNCA usar `saas-core/`, `http-core/`, etc. dentro de este repo
- NUNCA crear roots genéricos como `common/`, `shared/`, `utils/`, `libs/` en la raíz del repo
- `shared/` solo puede existir dentro de un módulo concreto y con ownership claro
- La raíz de una capacidad es solo contenedor organizativo; el código real vive siempre dentro del subdirectorio por lenguaje

---

## 6. Multi-lenguaje sin caos

El repo puede contener Go, Rust y Python. Lenguaje distinto NO obliga repo distinto.

### Regla

- Toda capacidad usa subcarpetas por lenguaje desde el inicio
- Para Go, la ruta válida es siempre `{modulo}/go/...`
- Para Python, la ruta válida es siempre `{modulo}/python/...`
- Para Rust, la ruta válida es siempre `{modulo}/rust/...`
- Para TypeScript, la ruta válida es siempre `{modulo}/ts/...`
- NUNCA crear `go.mod`, `pyproject.toml`, `Cargo.toml`, `src/` o paquetes de código directamente en la raíz de la capacidad
- NUNCA crear archivos Go en `saas/`, `http/`, `observability/`, `governance/`, etc. fuera de `go/`
- NUNCA crear archivos Python en `ai/` fuera de `python/`
- NUNCA crear archivos TypeScript fuera de `ts/`
- NUNCA usar una sola versión global para todo el repo
- Toda implementación concreta (`saas/go`, `ai/python`, etc.) debe tener su propio archivo `VERSION`
- Los tags de release siempre se cortan por subdirectorio: `{modulo}/{runtime}/vX.Y.Z`
- Si una capacidad tiene más de una implementación, se agregan sin romper la raíz de capacidad:

```text
{modulo}/
  go/
```

### Ejemplos

Correcto con una sola implementación:

```text
governance/
  go/

ai/
  python/
```

Correcto si hay múltiples implementaciones:

```text
governance/
  spec/
  go/
  rust/
```

Incorrecto:

```text
governance/
  Cargo.toml
  src/

saas/
  go.mod
  identity/

ai/
  pyproject.toml
  src/
```

---

## 7. Qué pertenece a este repo

### Regla de entrada

Una capacidad pertenece a `core` si:

- es reusable entre productos;
- no contiene copy ni dominio específico de un producto;
- tiene frontera estable;
- puede consumirse con cambios mínimos o sin cambios fuera de este repo.

### Si NO cumple eso

No entra a `core`. Se queda en el repo origen hasta madurar.

### Test rápido

Pregunta:

`¿esto lo pueden consumir al menos dos productos sin reescribirlo ni meterle copy/negocio específico?`

- Si sí: probablemente pertenece a `core`
- Si no: no pertenece a `core`

---

## 8. Fronteras por módulo

### `saas/`

Pertenece:
- orgs
- users
- identity
- billing
- entitlements
- usage metering

No pertenece:
- onboarding específico de un producto
- UI/admin específico
- copy comercial de una app

### `authz/`

Pertenece:
- scopes
- roles
- checks de autorización reusable

No pertenece:
- identidad
- storage de sesión browser
- tenancy o billing

### `authn/`

Pertenece:
- parsing de credenciales backend
- `jwks`
- `oidc`
- storage de tokens browser
- refresh serializado
- logout forzado
- helpers HTTP frontend

No pertenece:
- authz por scopes/roles
- UI de login específica de una app

### `notifications/`

Pertenece:
- senders reutilizables `noop`, `smtp`, `ses`
- config reusable por env
- bootstrap reusable de email

No pertenece:
- preferencias de notificación de una app
- templates/copy de producto
- colas o workers específicos de una app

### Módulos técnicos backend

Pertenece:
- `http/` para server, request/response helpers, auth transport y api keys reutilizables
- `observability/` para métricas, tracing y sinks compartidos
- `validate/` para validación reusable
- `errors/` para errores técnicos compartidos
- `utils/` para paginación, resiliencia y helpers genéricos
- `config/`, `security/` y `concurrency/` para infraestructura reusable transversal

No pertenece:
- dominio de negocio
- reglas SaaS
- wrappers acoplados a una app específica
- adapters concretos de base de datos

### `databases/postgres/`

Pertenece:
- pool PostgreSQL
- healthcheck PostgreSQL
- config PostgreSQL
- migrations runner PostgreSQL
- helpers `pgx` o `database/sql` específicos de PostgreSQL

No pertenece:
- runtime HTTP genérico fuera de `http/`
- dominio de negocio
- adapters de otras bases de datos

### `databases/dynamodb/`

Pertenece:
- client bootstrap DynamoDB
- helpers de serialización DynamoDB
- operaciones reutilizables sobre tablas DynamoDB

No pertenece:
- runtime Lambda
- API Gateway
- dominio KMA
- flows, audits, facilities, projects, etc.

### `providers/aws/lambda/`

Pertenece:
- runtime Lambda
- API Gateway adapters
- Lambda HTTP reusable
- helpers de routing para Lambda

No pertenece:
- dominio KMA
- flows, audits, facilities, projects, etc.

### `providers/aws/s3/`

Pertenece:
- storage reusable sobre S3
- presigned URLs
- bootstrap AWS específico de S3

No pertenece:
- storage abstractions de negocio
- dominio KMA

### `providers/aws/sqs/`

Pertenece:
- envío reusable a SQS
- serialización JSON común
- bootstrap AWS específico de SQS

No pertenece:
- contratos de eventos de negocio
- dominio KMA

### `eventing/`

Pertenece:
- envelopes HTTP/eventos
- contratos de eventos asíncronos
- metadata reusable para publicación/consumo

No pertenece:
- routing Lambda
- adapters concretos de proveedor
- dominio KMA

### `governance/`

Pertenece:
- policies
- risk
- approvals
- evidence
- audit contracts
- state machines de decisión
- delegations cuando su semántica sea estable

No pertenece:
- handlers de una app concreta
- repositorios específicos de un producto
- wiring de servicios

### `artifact/`

Pertenece:
- PDF
- Excel
- CSV
- QR
- report generation
- file naming
- metadata de artefactos
- storage contracts de artefactos

No pertenece:
- templates o copy cerrados a un producto
- reportes con dominio acoplado a una sola app

### `webhook/`

Pertenece:
- endpoints outbound
- firma HMAC
- retries/backoff
- replay y delivery planning

No pertenece:
- workers o rutas de una app específica
- persistencia cerrada a un producto

### `activity/`

Pertenece:
- audit append-only
- hash chaining
- export CSV/JSONL
- timeline por entidad

No pertenece:
- dashboards o auditoría específica de una sola app

### `ai/`

Pertenece:
- orchestration
- provider factory
- auth/rate limit AI
- runtime helpers
- context handling
- resilience/circuit breaker
- adapters AI reusable

No pertenece:
- prompts o tools de negocio específicos
- contratos cerrados a una app

---

## 9. Dependencias entre módulos

Regla general:

- Sin ciclos entre módulos
- Ningún módulo puede importar internals de otro módulo
- Por defecto, cada módulo debe ser autocontenido

### Excepciones permitidas

- `saas` puede depender de `backend` para infraestructura técnica reusable
- `saas` puede depender de `authz` y `notifications` para capacidades transversales

### Reglas fuertes

- `backend` no depende de otros módulos
- `authz` debe intentar mantenerse independiente
- `notifications` debe intentar mantenerse independiente
- `databases/postgres` debe intentar mantenerse independiente
- `databases/dynamodb` debe intentar mantenerse independiente
- `providers/aws/lambda` debe intentar mantenerse independiente
- `providers/aws/s3` debe intentar mantenerse independiente
- `providers/aws/sqs` debe intentar mantenerse independiente
- `eventing` debe intentar mantenerse independiente
- `governance` debe intentar mantenerse independiente
- `artifact` debe intentar mantenerse independiente
- `webhook` debe intentar mantenerse independiente
- `activity` debe intentar mantenerse independiente
- `ai` debe intentar mantenerse independiente
- Si una dependencia entre módulos no es obvia, documentarla primero

---

## 10. No dominio de producto en `core`

NUNCA meter en este repo:

- `nexus` domain específico
- `pymes` domain específico
- `ponti` domain específico
- `fixguard` domain específico
- `kma` domain específico
- nombres de clientes, verticales o features de una sola app

Este repo contiene capacidades, no productos.

---

## 11. Arquitectura Go para módulos librería

Usar arquitectura hexagonal solo cuando hay dominio real o comportamiento complejo.

La arquitectura NO se define por la estructura de directorios.

Hexagonal significa:

- dependencias hacia adentro;
- dominio aislado de HTTP, DB, AWS y frameworks;
- puertos y adapters con fronteras claras.

La estructura de carpetas de esta sección es una convención obligatoria del repo para legibilidad y consistencia, no la definición de la arquitectura.

### A. Paquetes de dominio o comportamiento rico

Patrón preferido:

```text
{contexto}/
  usecases.go
  usecases/domain/entities.go
  handler.go                 # solo si el módulo expone adapter HTTP
  handler/dto/dto.go         # solo si hay HTTP
  repository.go              # solo si el módulo posee adapter de persistencia
  repository/models/models.go
  {adapter}.go
  {adapter}/
  *_test.go
```

### B. Paquetes de infraestructura o runtime liviano

Patrón preferido:

```text
{package}/
  client.go
  config.go
  errors.go
  types.go
  *_test.go
```

### Reglas Go

- `context.Context` siempre primer parámetro
- NUNCA `panic()` en library code
- NUNCA ignorar errores con `_`
- `slog` como logger por default
- `time.Duration` para duraciones
- `errors.Is` para comparación
- `fmt.Errorf("...: %w", err)` para wrapping
- Interfaces en el consumidor, no en el proveedor
- Accept interfaces, return structs
- Si hay HTTP adapters: DTOs explícitos, nunca `var body struct{...}`
- Si hay migraciones: nunca modificar una migración ya publicada

No forzar estructura de `usecases/handler/repository` en paquetes que solo son adapters técnicos o helpers de runtime.

---

## 12. Rust para kernels

Usar Rust cuando una capacidad necesite:

- kernel determinista
- safety fuerte
- mejor performance
- API chica y estable

### Reglas Rust

- `unsafe` prohibido salvo pedido explícito
- No `panic!` en library code salvo bugs imposibles/documentados
- Tipos explícitos y enums reales, no stringly typed APIs
- Separar dominio puro de adapters
- Mantener la API pública pequeña
- Si existe `spec/`, implementar contra esa spec

Estructura sugerida:

```text
{modulo}/
  rust/
    Cargo.toml
    src/
      lib.rs
      domain/
      application/
      adapters/
    tests/
```

---

## 13. Python para librerías y runtime AI

### Reglas Python

- Type hints siempre
- Pydantic o dataclasses para tipos públicos
- `Protocol` para interfaces
- `async/await` para I/O
- `|` syntax para opcionales
- No `print()`
- No `except: pass`
- Config siempre por settings/parámetros, nunca hardcoded
- No retornar `dict` sueltos como API pública

Si un módulo Python expone FastAPI adapters, usar arquitectura clean/layered y separar:

- domain
- service
- schemas
- repository
- router
- dependencies

---

## 14. Testing y verificación

Antes de cerrar un cambio:

- Go: `go build ./...`, `go vet ./...`, `go test ./...` en el módulo afectado
- Rust: `cargo fmt --check`, `cargo clippy --all-targets --all-features -- -D warnings`, `cargo test`
- Python: `ruff check .`, `mypy .`, `pytest -q`

Si el cambio afecta múltiples módulos, probar todos los módulos afectados.

Si el entorno no permite probar, decirlo explícitamente.

---

## 15. Git y seguridad operativa

- NUNCA `git push` sin autorización explícita
- NUNCA commits, merges, rebases, resets, checkouts o cambios de rama sin autorización explícita
- Permitido: `git status`, `git diff`, `git log`, `git show`, `git branch`, `git remote -v`

---

## 16. Reglas críticas

- NUNCA valores hardcodeados salvo autorización explícita
- NUNCA meter dominio específico de producto en `core`
- NUNCA crear `common/`, `shared/`, `utils/` en la raíz
- NUNCA duplicar una capacidad en dos módulos
- NUNCA mezclar archivos de implementación directamente en la raíz de una capacidad
- NUNCA afirmar seguridad de un cambio sin revisar consumidores y productores afectados
- NUNCA decir "listo" sin pruebas o sin explicar por qué no pudieron correrse
