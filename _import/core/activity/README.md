# activity

Capacidad reusable para audit trails y timelines.

Implementaciones actuales:

- `activity/go/`
- `activity/rust/`

## Pertenece

- audit append-only
- hash chaining
- export CSV/JSONL
- timeline por entidad

## No pertenece

- auditoría específica de una app
- dashboards o UIs
- repos concretos de producto

## Fuentes iniciales esperadas

- `/home/pablo/Projects/Pablo/pymes/pymes-core/backend/internal/audit`
- `/home/pablo/Projects/Pablo/pymes/pymes-core/backend/internal/timeline`
- `/home/pablo/Projects/Pablo/nexus/v3/review/internal/audit`

## Estructura actual

- `activity/go/`: `kernel/usecases/domain`, `audit`, `timeline`
- `activity/rust/`: `domain`, `application`, `infrastructure`
