# Fuentes de extracción iniciales

Este documento mapea de dónde debería salir el código inicial de cada módulo.

## `saas`

Fuente principal:

- `/home/pablo/Projects/Pablo/saas-core`

Componentes candidatos:

- `org/`
- `users/`
- `identity/`
- `billing/`
- `entitlements/`
- `admin/`
- `usagemetering/`
- `shared/`

Estado actual en `core`:

- dominio base de tenant/principal/API key
- parsing neutral de bearer/api key
- entitlements por plan
- normalización de slug/role
- `kernel/usecases/domain` para tipos compartidos
- `middleware` HTTP reusable para principal resolution
- contexts `org`, `users`, `billing`, `admin`, `usagemetering`
- `handler/dto` y `repository/models` en los contexts con adapters
- `clerkwebhook`
- `identity/executor/jwks` -> `authn/go/jwks`
- `identity/executor/oidc` -> `authn/go/oidc`
- `billing/runtime`, `billing/webhook_handler`, `billing/dunning_worker`, `billing/stripe_client`
- `migrations`
- shims `shared/*` para compatibilidad del layout viejo
- contrato SaaS de notificaciones apoyado en `notifications/`

Conclusión actual:

- `saas-core` ya fue recreado de forma completa dentro de `core/saas/go`
- la obsolescencia real del repo viejo depende solo de migrar sus consumidores

## `authz`

Fuentes principales:

- `/home/pablo/Projects/Pablo/saas-core/shared/authz`

Componentes candidatos:

- constantes de scopes
- checks de scopes
- checks de roles
- adapter liviano para exponer autorización reusable

Estado actual en `core`:

- `authz.go`
- `usecases.go`
- `handler.go`
- `handler/dto`

## `notifications`

Fuentes principales:

- `/home/pablo/Projects/Pablo/core/saas/go/notifications`
- `/home/pablo/Projects/Pablo/pymes/pymes-core/backend/internal/notifications/sender.go`

Componentes candidatos:

- senders `noop`
- senders `smtp`
- senders `ses`
- config reusable por env
- bootstrap reusable de email

Estado actual en `core`:

- `EmailMessage`
- `EmailSender`
- `NewNoopEmailSender`
- `NewSMTPSender`
- `NewSESSender`
- `EmailConfigFromEnv`
- `NewEmailSender`

## `backend`

Fuentes principales:

- `/home/pablo/Projects/Pablo/nexus/v2/pkgs/go-pkg`
- `/home/pablo/Projects/Pablo/nexus/v3/pkgs/go-pkg`
- `/home/pablo/Projects/Pablo/pymes/pkgs/go-pkg`
- `/home/pablo/Projects/Pablo/toollab/toollab-core/internal/shared`

Componentes candidatos:

- http server
- api keys
- observability
- handlers/http helpers
- pagination
- retry
- validation
- context keys

Estado actual en `core`:

- `httpjson`
- `apikey`
- `httpserver`
- `observability`
- `pagination`
- `resilience`
- `validation`

## `authn`

Fuentes principales:

- `/home/pablo/Projects/Pablo/ponti/ponti-frontend/ui/src/pages/login`
- `/home/pablo/Projects/Pablo/ponti/ponti-frontend/ui/src/api/client.ts`
- `/home/pablo/Projects/Pablo/pymes/pkgs/ts-pkg`

Componentes candidatos:

- storage namespaced de browser
- eventos de logout forzado
- fetch auth con token provider
- cliente axios con refresh serializado
- adapters frontend para Clerk

Estado actual en `core`:

- `browser/storage`
- `browser/events`
- `http/fetch`
- `http/axios`
- `providers/clerk`

## `databases/postgres`

Fuentes principales:

- `/home/pablo/Projects/Pablo/nexus/v2/pkgs/go-pkg/postgres`
- `/home/pablo/Projects/Pablo/nexus/v3/pkgs/go-pkg/postgres`

Componentes candidatos:

- pool `pgx`
- healthcheck PostgreSQL
- migrate runner PostgreSQL
- config/env de PostgreSQL

Estado actual en `core`:

- `ConfigFromEnv`
- pool `pgx`
- `Ping`
- migrate runner con `fs.FS`

## `databases/dynamodb`

Fuente principal:

- `/home/pablo/Projects/AlphaCoding/kma-backend/lambdas/shared`

Componentes candidatos:

- DynamoDB
- bootstraps
- marshaling/unmarshaling reusable

Estado actual en `core`:

- `dynamodbtable`

## `providers/aws/lambda`

Fuente principal:

- `/home/pablo/Projects/AlphaCoding/kma-backend/lambdas/shared`

Componentes candidatos:

- runtime Lambda
- API Gateway responses
- Lambda HTTP
- routing reusable

Estado actual en `core`:

- `lambdahttp`

## `providers/aws/s3`

Fuente principal:

- `/home/pablo/Projects/AlphaCoding/kma-backend/lambdas/shared`

Componentes candidatos:

- S3
- presigned URLs
- uploads/downloads reutilizables

Estado actual en `core`:

- `s3store`

## `providers/aws/sqs`

Fuente principal:

- `/home/pablo/Projects/AlphaCoding/kma-backend/lambdas/shared`

Componentes candidatos:

- SQS
- envío JSON
- bootstrap/config AWS

Estado actual en `core`:

- `sqsqueue`

## `eventing`

Fuente principal:

- `/home/pablo/Projects/AlphaCoding/kma-backend/lambdas/shared`

Componentes candidatos:

- envelopes
- contratos de eventos
- metadata reusable

Estado actual en `core`:

- `envelope`

## `governance`

Fuentes principales:

- `/home/pablo/Projects/Pablo/nexus/v3/review/internal`
- `/home/pablo/Projects/Pablo/nexus/v2/data-plane/internal/action`
- `/home/pablo/Projects/Pablo/fixguard/fixguard-core`
- `/home/pablo/Projects/AlphaCoding/kma-backend/lambdas/shared/statemachine`

Componentes candidatos:

- policy evaluation
- risk
- approvals
- evidence
- audit contracts
- state machines

Estado actual en `core`:

- `kernel/usecases/domain`
- evaluator CEL
- risk engine
- decision engine
- approvals
- delegations
- audit replay
- evidence pack builder
- `handler/dto` y `repository/models` en los contexts con adapters
- compatibilidad en `domain/`

## `artifact`

Fuentes principales:

- `/home/pablo/Projects/Pablo/pymes/pymes-core/backend/internal/pdfgen`
- `/home/pablo/Projects/Pablo/pymes/pymes-core/backend/internal/dataio`
- `/home/pablo/Projects/Pablo/pymes/pymes-core/backend/internal/paymentgateway`
- `/home/pablo/Projects/Pablo/ponti/ponti-backend/internal/labor/excel`
- `/home/pablo/Projects/Pablo/ponti/ponti-backend/internal/labor/excel-service.go`
- `/home/pablo/Projects/AlphaCoding/kma-backend/lambdas/shared/excel`
- `/home/pablo/Projects/AlphaCoding/kma-backend/lambdas/reports-worker/internal/generate-report`

Componentes candidatos:

- PDF generation
- Excel parsing/export
- CSV export
- QR generation
- report generation
- artifact storage contracts

Estado actual en `core`:

- root `Asset`
- naming/content type helpers
- storage contracts
- export tabular CSV/XLSX
- PDF simple reusable
- QR PNG reusable

## `ai`

Fuente principal:

- `/home/pablo/Projects/Pablo/ai-core`

Componentes candidatos:

- orchestrator
- provider factory
- auth
- rate limit
- logging
- contexts
- resilience
- FastAPI helpers

Estado actual en `core`:

- `runtime.domain`
- `runtime.providers`
- `runtime.services`
- `runtime.registry`
- `runtime.config`
- `runtime.api`
- middleware FastAPI de auth, rate-limit y request context
- app factory reusable
- paquete `runtime` como namespace único compartido
- tests estándar con `unittest`

Conclusión actual:

- `ai-core` ya fue recreado de forma completa dentro de `core/ai/python`
- la obsolescencia real del repo viejo depende solo de migrar sus consumidores

## Pendientes detectados en la última pasada

### `webhook` como capacidad nueva

Fuentes principales:

- `/home/pablo/Projects/Pablo/pymes/pymes-core/backend/internal/outwebhooks/usecases.go`
- `/home/pablo/Projects/Pablo/pymes/pymes-core/backend/internal/outwebhooks/repository.go`

Señal:

- gestión de endpoints
- firma HMAC
- outbox
- retries/backoff
- delivery log
- replay

Recomendación:

- crear un módulo nuevo `webhook/go`
- no meterlo dentro de `backend`, porque ya es una capacidad con semántica propia

Estado actual:

- `webhook/go` ya fue creado con `usecases/domain`, `repository/models`, signing HMAC, headers estándar y backoff de retries

### `activity` o `auditlog` como capacidad nueva

Fuentes principales:

- `/home/pablo/Projects/Pablo/pymes/pymes-core/backend/internal/audit/usecases.go`
- `/home/pablo/Projects/Pablo/pymes/pymes-core/backend/internal/audit/repository.go`
- `/home/pablo/Projects/Pablo/pymes/pymes-core/backend/internal/timeline/usecases.go`
- `/home/pablo/Projects/Pablo/pymes/pymes-core/backend/internal/timeline/repository.go`
- `/home/pablo/Projects/Pablo/nexus/v3/review/internal/audit/usecases.go`

Señal:

- audit trail append-only
- export CSV/JSONL
- hash chaining / tamper-evident log
- timeline por entidad
- replay de eventos

Recomendación:

- no forzarlo dentro de `governance` todavía
- evaluar un módulo nuevo `activity/go` o `auditlog/go`
- si se mantiene en `governance`, solo para audit ligado a decisiones/workflows

Estado actual:

- `activity/go` ya fue creado con contexts `audit/` y `timeline/`

### ampliar `databases/dynamodb/go` y `providers/aws/s3/go`

Fuentes principales:

- `/home/pablo/Projects/AlphaCoding/kma-backend/lambdas/shared/databases/nosql/dynamodb/bootstrap.go`
- `/home/pablo/Projects/AlphaCoding/kma-backend/lambdas/shared/databases/nosql/dynamodb/client.go`
- `/home/pablo/Projects/AlphaCoding/kma-backend/lambdas/public-upload/cmd/main.go`
- `/home/pablo/Projects/AlphaCoding/kma-backend/lambdas/uploads/cmd/main.go`
- `/home/pablo/Projects/AlphaCoding/kma-backend/lambdas/audits-review/internal/shared/services/s3_presigner.go`

Señal:

- cliente/bootstrapping DynamoDB más completo
- presigned PUT para S3
- upload flows reutilizables
- URLs públicas/privadas y contratos de upload

Recomendación:

- ampliar `databases/dynamodb/go`
- ampliar `providers/aws/s3/go`

Estado actual:

- `s3store` ya incluye `PresignPut` y `ExtractKey`
- `dynamodbtable` ya incluye `DeleteJSON`, `QueryJSON` y `Health`

### ampliar `artifact/go`

Fuentes principales:

- `/home/pablo/Projects/Pablo/pymes/pymes-core/backend/internal/attachments/usecases.go`
- `/home/pablo/Projects/Pablo/pymes/pymes-core/backend/internal/attachments/usecases/domain/entities.go`

Señal:

- metadata de adjuntos
- upload/download request contracts
- storage key conventions
- asociación de archivos a entidades

Recomendación:

- ampliar `artifact/go`
- probablemente con `attachments` o `blobs` como subpaquete

Estado actual:

- `artifact/go/attachments` ya fue creado con metadata, storage keys y contratos de upload/download

### ampliar `notifications/go` o crear `notify` más adelante

Fuentes principales:

- `/home/pablo/Projects/Pablo/pymes/pymes-core/backend/internal/notifications/sender.go`

Señal:

- SMTP sender
- SES sender
- backend noop

Recomendación:

- por ahora ampliaría `notifications/go`
- solo crearía `notify/go` si aparece otro consumidor no-SaaS

Estado actual:

- `saas/go/notifications` quedó solo como puerto de dominio SaaS

### todavía no extraería

Fuentes evaluadas:

- `/home/pablo/Projects/Pablo/pymes/pymes-core/backend/internal/whatsapp`
- `/home/pablo/Projects/Pablo/pymes/pymes-core/backend/internal/paymentgateway/gateway/mercadopago.go`
- `/home/pablo/Projects/Pablo/pymes/pkgs/ts-pkg/src/auth.ts`
- `/home/pablo/Projects/AlphaCoding/kma_web/src/features/auth/api/cognito.repo.impl.ts`
- `/home/pablo/Projects/Pablo/ponti/ponti-backend/internal/admin/idp/firebase_admin.go`
- `/home/pablo/Projects/AlphaCoding/KMA_app/src/core/repos/sqliteOutboxRepo.ts`

Motivo:

- siguen demasiado pegados a producto, canal o runtime específico
- o todavía solo tienen un consumidor claro
