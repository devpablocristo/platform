# core

Monorepo de capacidades reutilizables compartidas entre varios productos.

Este repo no contiene apps. Contiene módulos por capacidad.

## Módulos

- `saas/`: tenancy, identity, users, billing, entitlements
- `browser/`: storage y utilidades reutilizables del runtime browser
- `http/`: transporte HTTP frontend reusable, `fetch` JSON y `event-stream`
- `observability/`: tracing, metrics e integración reusable de observabilidad
- `config/`: configuración reusable para servicios backend
- `security/`: helpers reutilizables de seguridad backend
- `validate/`: validación reusable
- `errors/`: errores compartidos y contratos de error
- `utils/`: utilidades técnicas reutilizables
- `concurrency/`: primitivas reutilizables de concurrencia y coordinación
- `authz/`: autorización reusable por roles y scopes
- `authn/`: autenticación reusable, tanto backend (`go`) como sesión/browser (`ts`)
- `notifications/`: senders y transporte reusable para notificaciones
- `scheduling/`: primitivas reutilizables de ventanas, slots y bloqueos temporales
- `calendar/`: exportación ICS y sincronización reusable con calendarios externos
- `databases/`: adapters concretos de bases de datos
- `providers/`: adapters concretos de proveedores externos
- `eventing/`: envelopes y contratos de eventos asíncronos
- `governance/`: policies, risk, approvals, evidence y state machines
- `artifact/`: PDF, Excel, CSV, QR, exports y report generation
- `ingestion/`: contrato HTTP agnóstico al dominio para normalización/extracción multimodal (DTOs, cliente `POST /v1/extract`, helpers de ensamblado)
- `webhook/`: outbound delivery, firma HMAC, retries y replay
- `activity/`: audit trail append-only, timeline y exports
- `ai/`: runtime y helpers AI reutilizables

## Reglas de organización

- El repo se llama `core`, así que adentro no usamos sufijos `-core`
- Los módulos se nombran por capacidad, no por lenguaje
- Lenguaje distinto no obliga repo distinto
- Cada capacidad se organiza con subdirectorios por lenguaje desde el día 1
- Si una capacidad suma más implementaciones, conviven bajo la misma raíz de capacidad
- No se admite dominio específico de producto dentro de este repo
- No existe una sola versión global del repo; cada implementación (`http/go`, `ai/python`, etc.) se versiona por separado

## Estructura

```text
core/
  saas/
    go/
  browser/
    ts/
  http/
    ts/
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
    pagination/
      rust/
    resilience/
      rust/
  concurrency/
    go/
    fsm/
      rust/
    worker/
      rust/
  authz/
    go/
  authn/
    go/
    ts/
  notifications/
    go/
  scheduling/
    go/
  calendar/
    ics/
      go/
    sync/
      google/
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
    rust/
  artifact/
    go/
    rust/
  ingestion/
    go/
  webhook/
    go/
  activity/
    go/
    rust/
  ai/
    python/
  docs/
  scripts/
  examples/
```

## Documentación

- [Arquitectura](docs/ARCHITECTURE.md)
- [Versionado](docs/VERSIONING.md)
- [Fuentes de extracción](docs/EXTRACTION-SOURCES.md)
- [Reglas para agentes](AGENTS.md)
- [Scripts](scripts/README.md)

## Tooling y CI

- El tooling TypeScript del repo queda fijado en `20.19.0` vía `.nvmrc`.
- Los módulos TS versionados usan `package-lock.json` y los workflows ejecutan `npm ci` para pruebas/publicación reproducibles.
- Hay `Dependabot` semanal para `github-actions`, `npm` y `pip` en las superficies publicadas del repo.

## Estado actual

Este repo ya tiene:

- reglas para Claude, GPT/Codex y Cursor;
- estructura raíz del monorepo;
- documentación de fronteras y migración;
- bootstrap real en `http/`, `observability/`, `config/`, `security/`, `validate/`, `errors/`, `utils/`, `concurrency/`, `databases/`, `providers/`, `eventing/`, `governance/`, `artifact/`, `webhook/`, `activity/`, `saas/`, `authz/`, `authn/`, `notifications/`, `scheduling/` y `ai/`;
- separación explícita por lenguaje en cada capacidad;
- scripts de validación por módulo y workflow CI del monorepo.

### Bootstrap por módulo

- `http/go/`: helpers backend HTTP reutilizables
- `http/gin/go/`: middleware y utilidades específicas de Gin
- `observability/go/`: observabilidad reusable para servicios backend
- `config/go/`: configuración reusable para servicios
- `security/go/`: helpers de seguridad backend
- `validate/go/`: validación reusable para backends
- `errors/go/`: contratos y helpers de error compartidos
- `utils/go/`: utilidades técnicas compartidas
- `concurrency/go/`: primitivas de concurrencia reutilizables
- `browser/ts/`: namespace storage para browser, lectura/escritura string/JSON y cleanup por prefijo
- `http/ts/`: `fetch` reusable, parseo uniforme de errores HTTP y JSON `event-stream`
- `authz/go/`: scopes, roles, checks reusable y adapter liviano de autorización
- `notifications/go/`: `noop`, `smtp`, `ses`, config reusable y bootstrap de email senders
- `calendar/ics/go/`: generación reusable de calendarios y eventos `.ics`
- `calendar/sync/google/go/`: OAuth reusable para sincronización con Google Calendar
- `databases/postgres/go/`: config/pool `pgx` y migrate runner
- `databases/dynamodb/go/`: acceso reusable a DynamoDB
- `providers/aws/lambda/go/`: `lambdahttp`
- `providers/aws/s3/go/`: storage y presigned URLs para S3
- `providers/aws/sqs/go/`: envío reusable a SQS
- `eventing/go/`: envelope tipado reusable para eventos asíncronos
- `governance/go/`: `kernel/usecases/domain`, `policy`, `risk`, `decision`, `approval`, `delegations`, `audit`, `evidence`, `handler/dto`, `repository/models`, más compatibilidad en `domain/`
- `governance/rust/`: kernel determinista con hexagonal architecture para `risk`, `decision`, `approval` y `evidence`, listo para adapters de integración
- `artifact/go/`: root `Asset` + naming/content types, `storage`, `tabular`, `pdf`, `qr`, `attachments`
- `artifact/rust/`: runtime Rust para `asset`, `attachments` y `tabular` CSV/XLSX con puertos explícitos para codecs e infraestructura
- `webhook/go/`: gestión de endpoints, firma HMAC, headers, backoff y planning de deliveries
- `activity/go/`: `kernel/usecases/domain`, `audit`, `timeline` y export helpers
- `activity/rust/`: runtime hexagonal para `audit` y `timeline`, con hash chaining, exports deterministas y adapters explícitos
- `saas/go/`: `kernel/usecases/domain`, `identity`, `org`, `users`, `billing`, `admin`, `entitlement`, `tenant`, `usagemetering`, `middleware`, `handler/dto`, `repository/models` y contrato SaaS de notificaciones
- `authn/go/`: parsing de credenciales, `jwks` y `oidc` reusable para autenticación backend
- `authn/ts/`: storage namespaced de auth, eventos de logout, fetch auth, axios auth con refresh serializado y adapters frontend
- `ai/python/`: paquete `runtime` como base compartida de AI con clients, contracts, completions, config, auth, rate-limit, observabilidad y wiring FastAPI reusable

Estado clave:

- `saas/go/` ya absorbió completamente el contenido funcional de `/home/pablo/Projects/Pablo/saas-core`
- `ai/python/` ya absorbió completamente el contenido funcional de `/home/pablo/Projects/Pablo/ai-core`
- cada implementación ya tiene versionado independiente con `VERSION` y tags por subdirectorio
- el siguiente trabajo para `saas` ya no es reconstrucción interna sino migración de consumidores y retiro del repo viejo
- el siguiente trabajo para `ai` ya no es reconstrucción interna sino migración de consumidores y retiro del repo viejo

El trabajo que sigue ya no es “crear el repo”, sino profundizar y migrar código desde los repos origen módulo por módulo.
