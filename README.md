# platform

Monorepo unificado del ecosistema `pablo`. Fusiona los repos legacy `core/`
(primitivas técnicas) y `modules/` (features verticales reutilizables) en una
única fuente de verdad con renombre coherente bajo
`github.com/devpablocristo/platform/*` (Go) y `@devpablocristo/platform-*` (npm).

## Estructura

```
platform/
├── errors/{go,rust}/                 # capabilities transversales (L0)
├── validate/, config/, observability/, http/, concurrency/, security/, authn/, authz/
├── databases/, eventing/, webhook/, notifications/, ingestion/
├── calendar/ics/, jobs/, utils/, browser/
├── lifecycle/{go,ts}/                # CRUDAR normalizado (L0.5)
├── kernels/                          # mini-aplicaciones embebibles (L1)
│   ├── saas/, governance/, activity/, artifact/, ai/
├── features/                         # verticales reutilizables (L2)
│   ├── scheduling/, crud/, kanban-board/, calendar-board/, search/,
│   ├── admin-insights/, conversation-inbox/, notification-feed/
├── ui/                               # primitivas del design system
│   ├── modal/, data-display/, section-hub/, page-shell/, shell-sidebar/
├── sdks/                             # clientes a servicios externos
│   ├── aws/, google-calendar/, clerk/ (segunda ola)
├── contracts/ai/                     # contratos cross-lenguaje
├── docs/, tooling/, examples/
└── .github/workflows/
```

## Origen

Este repo nace de la fusión de:

- `github.com/devpablocristo/core` (47 paquetes Go/Rust/TS/Python — capabilities
  transversales y kernels de dominio).
- `github.com/devpablocristo/modules` (20 paquetes — features verticales y
  primitivas UI).

Ambos repos siguen accesibles **read-only** (archivados al final de la Ola A).
Sus versiones publicadas permanecen disponibles durante la migración de los
consumers no migrados (ponti, nexus, companion, medmory, toollab).

Ver [`docs/migration/MIGRATION_FROM_CORE_MODULES.md`](docs/migration/MIGRATION_FROM_CORE_MODULES.md)
para detalles del proceso y registro de equivalencias de paquetes.

## Consumers

- **pymes** — consumer de referencia: ya consume `platform-*`,
  `platform/lifecycle/go`, Axis Companion y Nexus por HTTP.
- **ponti** — migración parcial: backend y frontend usan `platform-*`; el
  backend conversa con Axis Companion y conserva shapes legacy en el BFF.
- **nexus** y **companion** — consumen piezas de `platform` durante desarrollo;
  quitar `replace` locales requiere publicar la version correspondiente.
- **medmory, toollab** — validar antes de eliminar definitivamente repos legacy.

Ver [`docs/migration/CONSUMER_ALIGNMENT.md`](docs/migration/CONSUMER_ALIGNMENT.md).

## Documentación

- [`AGENTS.md`](AGENTS.md) / [`CLAUDE.md`](CLAUDE.md) — reglas para agentes y Claude Code.
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — flujo de trabajo, releases, validación.
- [`GOVERNANCE.md`](GOVERNANCE.md) — gobernanza del ecosistema (en transición).
- [`docs/`](docs/) — ADRs, naming, release flow, versioning.

## Workspace

El monorepo usa:

- **Go workspace** (`go.work`) para los ~31 `go.mod` internos.
- **pnpm workspace** (`pnpm-workspace.yaml`) para los ~19 paquetes npm.
- **Cargo workspace** (`Cargo.toml` root) para los 14 crates Rust.
- Python packages independientes (`pyproject.toml`) en sus paths respectivos.

## Estado de la migración

- ✅ Ola A — Fusión core + modules → platform.
- 🔄 Ola B — Extracción de transversales de pymes a platform + diseño `lifecycle/`.
- 🔄 Ola C — Refactor de módulos CRUDAR de pymes para uniformizar via `lifecycle/`.

Ver [`docs/migration/MIGRATION_FROM_CORE_MODULES.md`](docs/migration/MIGRATION_FROM_CORE_MODULES.md)
para el plan detallado.
