# governance

Kernel reusable para toma de decisión, políticas y control.

## Pertenece

- policies
- risk
- approvals
- evidence
- audit contracts
- state machines de decisión
- delegations cuando su semántica sea estable

## No pertenece

- handlers o wiring de una app concreta
- repositorios específicos de un producto
- dashboards o UIs

## Fuentes iniciales esperadas

- `/home/pablo/Projects/Pablo/nexus/v3/review/internal`
- `/home/pablo/Projects/Pablo/nexus/v2/data-plane/internal/action`
- `/home/pablo/Projects/Pablo/fixguard/fixguard-core`
- `/home/pablo/Projects/AlphaCoding/kma-backend/lambdas/shared/statemachine`

## Nota

Este es el módulo más valioso, pero no debería ser el primero en migrarse. Primero conviene estabilizar `backend`, `databases`, `providers/aws` y `artifact`.

## Estructura actual

Implementaciones actuales:

- `governance/go/`
- `governance/rust/`

Esta implementación ya aplica la convención Go de contexto + `usecases/domain` sin capas de compatibilidad internas.

Paquetes activos en `governance/go/`:

- `kernel/usecases/domain/` como fuente de verdad de tipos compartidos
- `policy/` con `usecases.go` y `usecases/domain/`
- `risk/` con `usecases.go` y `usecases/domain/`
- `decision/` con `usecases.go` y `usecases/domain/`
- `approval/` con `usecases.go` y `usecases/domain/`
- `delegations/`
- `audit/`
- `evidence/` con `usecases.go` y `usecases/domain/`
- `handler/dto/` en los contexts de aplicación
- `repository/models/` en los contexts con puertos de persistencia o lookup

Runtime adicional activo:

- `governance/rust/` contiene el kernel determinista con hexagonal architecture
- `domain/`: modelos, risk, approval, evidence
- `application/`: decision engine y evidence service
- `infrastructure/`: adapters de policy matching y signing HMAC
