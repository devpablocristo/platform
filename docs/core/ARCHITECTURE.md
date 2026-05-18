# Arquitectura del monorepo `core`

## Objetivo

`core` es el repo de capacidades reutilizables para varios productos.

La unidad principal de organización es la **capacidad**. Dentro de cada capacidad, la implementación se separa explícitamente por lenguaje.

## Módulos raíz

- `saas`
- `browser`
- `http`
- `observability`
- `config`
- `security`
- `validate`
- `errors`
- `utils`
- `concurrency`
- `authz`
- `authn`
- `notifications`
- `databases`
- `providers`
- `eventing`
- `governance`
- `artifact`
- `webhook`
- `activity`
- `ai`

## Estado actual

- `browser/ts`: activo, módulo TypeScript con namespace storage reusable para browser
- `http/ts`: activo, módulo TypeScript con `fetch` JSON, parseo de errores y `event-stream`
- `authz/go`: activo, módulo Go con scopes, roles, checks reusable y adapter liviano de autorización
- `notifications/go`: activo, módulo Go con senders `noop`, `smtp`, `ses` y bootstrap reusable de email
- `http/go`: activo, módulo Go con utilidades backend HTTP reutilizables
- `http/gin/go`: activo, adaptadores reutilizables para Gin
- `observability/go`: activo, módulo Go de observabilidad reusable
- `observability/rust`: activo, runtime Rust de observabilidad reusable
- `config/go`: activo, módulo Go de configuración reusable
- `security/go`: activo, módulo Go de helpers de seguridad reusable
- `validate/go`: activo, módulo Go de validación reusable
- `validate/rust`: activo, runtime Rust de validación reusable
- `errors/go`: activo, módulo Go de errores reusable
- `errors/rust`: activo, runtime Rust de errores reusable
- `utils/go`: activo, utilidades técnicas compartidas
- `utils/pagination/rust`: activo, runtime Rust de paginación reusable
- `utils/resilience/rust`: activo, runtime Rust de resilience reusable
- `concurrency/go`: activo, primitivas de concurrencia reusable
- `concurrency/fsm/rust`: activo, runtime Rust para máquinas de estado
- `concurrency/worker/rust`: activo, runtime Rust para workers
- `databases/postgres/go`: activo, módulo Go con pool/config/migrations
- `databases/dynamodb/go`: activo, módulo Go con adapter reusable para DynamoDB
- `providers/aws/lambda/go`: activo, módulo Go con Lambda HTTP / API Gateway
- `providers/aws/s3/go`: activo, módulo Go con adapter reusable para S3
- `providers/aws/sqs/go`: activo, módulo Go con adapter reusable para SQS
- `eventing/go`: activo, módulo Go con envelopes de eventos asíncronos
- `governance/go`: activo, módulo Go con kernel de dominio, contexts con `usecases/domain` y adapters `handler/dto` + `repository/models`
- `governance/rust`: activo, runtime Rust para el kernel determinista (`risk`, `decision`, `approval`, `evidence`) con puertos y adapters explícitos
- `artifact/go`: activo, módulo Go con `Asset`, naming, tabular, PDF, QR y `attachments`
- `artifact/rust`: activo, runtime Rust para `asset`, `attachments` y `tabular` con codecs XLSX desacoplados
- `webhook/go`: activo, módulo Go con endpoints, signing, retry policy y planning de deliveries
- `activity/go`: activo, módulo Go con audit trail append-only y timeline
- `activity/rust`: activo, runtime Rust para `audit` y `timeline` con puertos, adapters y hashing determinista
- `saas/go`: activo, módulo Go con dominio SaaS, contexts con `usecases/domain`, `handler/dto`, `repository/models`, middleware HTTP reusable y contratos SaaS
- `authn/go`: activo, módulo Go con parsing de credenciales, `jwks` y `oidc`
- `authn/ts`: activo, módulo TypeScript para sesión/browser, fetch auth, axios auth y adapters frontend reutilizables
- `ai/python`: activo, paquete Python con `domain/providers/services/registry/config/api`, middleware FastAPI, app factory y `ai_core` como compatibilidad histórica

## Reglas de frontera

### Lo que sí va en `core`

- capacidades compartidas por dos o más productos;
- código reusable sin dominio específico de una app;
- kernels o runtimes con frontera estable.

### Lo que no va en `core`

- features específicas de Nexus, Pymes, Ponti, Fixguard o KMA/AlphaCoding;
- copy o UX de un producto;
- apps, UIs y wiring de despliegue de una app específica;
- lógica que todavía solo sirve a un repo y no tiene frontera clara.

## Regla multi-lenguaje

Lenguaje distinto no obliga repo distinto.

Cada capacidad usa subdirectorios por lenguaje desde el inicio:

```text
governance/
  go/
```

Si aparece otra implementación:

```text
governance/
  spec/
  go/
  rust/
```

## Regla de versionado

No existe una única versión del repo `core`.

Cada implementación concreta se versiona de forma independiente:

- `authz/go`
- `browser/ts`
- `http/ts`
- `notifications/go`
- `config/go`
- `concurrency/go`
- `errors/go`
- `http/go`
- `http/gin/go`
- `observability/go`
- `security/go`
- `utils/go`
- `validate/go`
- `authn/go`
- `authn/ts`
- `authn/rust`
- `databases/postgres/go`
- `databases/postgres/rust`
- `databases/dynamodb/go`
- `providers/aws/lambda/go`
- `providers/aws/s3/go`
- `providers/aws/sqs/go`
- `eventing/go`
- `governance/go`
- `governance/rust`
- `artifact/go`
- `artifact/rust`
- `webhook/go`
- `activity/go`
- `activity/rust`
- `saas/go`
- `http/python`
- `http/client/rust`
- `http/server/rust`
- `utils/pagination/rust`
- `utils/resilience/rust`
- `concurrency/fsm/rust`
- `concurrency/worker/rust`
- `errors/rust`
- `validate/rust`
- `observability/rust`
- `ai/go`
- `ai/python`

Cada una debe tener un archivo `VERSION` y sus tags se cortan por subdirectorio:

- `config/go/v0.1.0`
- `concurrency/go/v0.1.0`
- `errors/go/v0.1.0`
- `http/go/v0.1.0`
- `http/gin/go/v0.1.0`
- `observability/go/v0.1.0`
- `security/go/v0.1.0`
- `utils/go/v0.1.0`
- `validate/go/v0.1.0`
- `authz/go/v0.1.0`
- `browser/ts/v0.1.0`
- `http/ts/v0.1.0`
- `authn/go/v0.1.0`
- `authn/rust/v0.1.0`
- `notifications/go/v0.1.0`
- `authn/ts/v0.1.0`
- `databases/postgres/go/v0.1.0`
- `databases/postgres/rust/v0.1.0`
- `databases/dynamodb/go/v0.1.0`
- `providers/aws/lambda/go/v0.1.0`
- `providers/aws/s3/go/v0.1.0`
- `providers/aws/sqs/go/v0.1.0`
- `eventing/go/v0.1.0`
- `saas/go/v0.1.0`
- `http/python/v0.1.0`
- `http/client/rust/v0.1.0`
- `http/server/rust/v0.1.0`
- `utils/pagination/rust/v0.1.0`
- `utils/resilience/rust/v0.1.0`
- `concurrency/fsm/rust/v0.1.0`
- `concurrency/worker/rust/v0.1.0`
- `errors/rust/v0.1.0`
- `validate/rust/v0.1.0`
- `observability/rust/v0.1.0`
- `ai/go/v0.1.0`
- `ai/python/v0.1.0`

Para más detalle, ver [VERSIONING.md](VERSIONING.md).

## Dependencias permitidas

Regla general:

- sin ciclos entre módulos;
- no importar internals de otro módulo;
- cada módulo debe ser autocontenido por defecto.

Reglas específicas:

- `authz` debe intentar mantenerse independiente;
- `notifications` debe intentar mantenerse independiente;
- `databases/*` debe intentar mantenerse independiente;
- `providers/aws/*` debe intentar mantenerse independiente;
- `eventing` debe intentar mantenerse independiente;
- `saas` puede depender de `http`, `observability`, `config`, `security`, `authn`, `authz` y `notifications`;
- `governance`, `artifact`, `webhook`, `activity` y `ai` deben intentar mantenerse independientes.

## Patrones por tipo de módulo

### Módulos Go con dominio real

La arquitectura hexagonal se evalúa por la dirección de dependencias y el aislamiento del dominio, no por el árbol de carpetas.

La convención de estructura para módulos Go con dominio real es:

```text
{contexto}/
  usecases.go
  usecases/domain/entities.go
  handler.go                 # cuando expone adapter de aplicación
  handler/dto/dto.go
  repository.go              # cuando define puerto de persistencia o lookup
  repository/models/models.go
```

### Módulos técnicos o runtime

No forzar hexagonal en paquetes simples:

```text
{package}/
  client.go
  config.go
  errors.go
  types.go
```

### Rust

Preferido para kernels deterministas o sensibles a performance:

```text
{modulo}/rust/
  Cargo.toml
  src/
    lib.rs
    domain/
    application/
    adapters/
```

### Python

Preferido para runtime AI y librerías Python:

```text
{modulo}/python/
  pyproject.toml
  src/
  tests/
```

Si expone FastAPI, separar `domain`, `providers`, `services`, `config` y `api` con `router/dependencies/schemas/app`.

## Validación del repo

El repo tiene validación local y CI por módulos independientes:

- `scripts/test-go-modules.sh`
- `scripts/test-ai.sh`
- `scripts/test-all.sh`
- `.github/workflows/ci.yml`

## Orden recomendado de profundización y migración

1. `http`
2. `authz`
3. `browser`
4. `observability`
5. `notifications`
6. `databases/postgres`
7. `databases/dynamodb`
8. `providers/aws/*`
9. `eventing`
10. `artifact`
11. `webhook`
12. `activity`
13. `governance`
14. `saas`
15. `ai`

Ese orden sigue minimizando riesgo de extracción y reduce acoplamiento temprano, incluso después del bootstrap inicial.
