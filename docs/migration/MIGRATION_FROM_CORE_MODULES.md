# Migración `core/` + `modules/` → `platform/`

## Resumen

Este documento registra la fusión de los repos legacy
`github.com/devpablocristo/core` y `github.com/devpablocristo/modules` en el
nuevo monorepo unificado `github.com/devpablocristo/platform`. La migración
se ejecuta en tres olas (A → B → C) con `pymes` como rata de laboratorio.

El plan completo vive en
`~/.claude/plans/actu-como-staff-principal-engineer-immutable-yao.md`.

## Decisiones rectoras

| # | Decisión | Implicancia |
|---|---|---|
| D1 | Preservar historia git con `git subtree add` | Reversible, sin reescribir SHAs |
| D2 | Reorganización profunda por intención | Estructura por capability/feature, no por origen |
| D3 | `core/scheduling/go` (primitiva) → `platform/jobs/go` | Libera el nombre `scheduling` para el feature vertical |
| D4 | Cutover hard sin shims | Pymes es el único consumer migrado en esta ola |
| D5 | Reset todas las versiones a `v0.1.0` bajo el nuevo nombre | Limpieza completa, changelogs nuevos |
| D6 | NO migrar deprecados | Skip: ai-console, ui-filters, ui-forms, work-orders, .release-scheduling-ts |
| D7 | Incluir Rust y Python | `platform/<capacidad>/{go,rust,ts,python}/` cuando aplique |
| D8 | `sdks/` solo para servicios externos | AWS + Google Calendar día 1; Clerk en segunda ola |
| D9 | `contracts/` top-level reservada para crecimiento | Hoy solo `contracts/ai/` |
| D10 | Renombre npm: `@devpablocristo/platform-<paquete>` | Sin prefijo `core-`/`modules-` |
| D11 | Renombre Go: `github.com/devpablocristo/platform/<path>` | Conservando `/go` final del path |
| D12 | Scope reducido a pymes | Otros consumers (ponti, nexus, companion, medmory, toollab) quedan fuera de esta ola |
| D13 | Crear `platform/lifecycle/{go,ts}` como capacidad nueva | Normaliza CRUDAR end-to-end |
| D14 | Refactor completo de los 12 módulos CRUDAR de pymes | Todos pasan a usar lifecycle uniformemente |
| D15 | Extraer transversales de pymes a platform | Backend Gin glue, audit hash-chain, FSM glue, Frontend CRUD shell |
| D16 | Tres olas secuenciales A → B → C | A: fusión, B: extracción, C: refactor |
| D17 | `platform/` es agnóstico al negocio de las apps | Sin nombres de entidades pymes ni de otros consumers |
| D18 | Paquetes publicados de core/modules: mantener vivos + deprecar formalmente | Cero rotura para consumers no migrados |
| D19 | Repos GitHub core/modules: archivar al final de Ola A, eliminar tras 90 días | Backup local + branch `backup/pre-platform-migration` |

## Invariantes de agnosticidad

`platform/` no conoce las entidades de negocio de los consumers. Doce
invariantes (I1-I12) detallados en el plan rector. En cada PR de Olas B y C,
correr el checklist anti-filtración:

```bash
# Ninguna entidad pymes en platform
grep -riE '\b(customer|quote|invoice|pricelist|cashflow|sale|purchase|payment|supplier|employee|product|service|recurring|return|inventory|procurement|wooko|beauty|medical|restaurant|workshop|professional)s?\b' \
  --include="*.go" --include="*.ts" --include="*.tsx" --include="*.rs" --include="*.py" \
  /home/pablocristo/Proyectos/pablo/platform/ \
  | grep -v "//.*example\|//.*e\.g\.\|// example\|_test\.go\|\.test\."
# Debe estar vacío

# Sin referencias a Clerk fuera de sdks/clerk
grep -rE '"@clerk/|"@auth0/|clerk-sdk' --include="*.go" --include="*.ts" \
  /home/pablocristo/Proyectos/pablo/platform/{errors,http,authn,authz,...}/ \
  | grep -v sdks/clerk
# Solo permitido en platform/sdks/clerk/

# Sin paths REST concretos
grep -rE '"/v1/(customers|quotes|invoices|...)"' \
  --include="*.go" --include="*.ts" \
  /home/pablocristo/Proyectos/pablo/platform/
# Debe estar vacío
```

## Estructura del proceso (resumen)

### Ola A — Fusión `core` + `modules` → `platform`

10 fases (A0-A9 + A10 deferred):

- **A0** Pre-flight: snapshots, branches `backup/pre-platform-migration` en GitHub.
- **A1** Esqueleto platform/ (este commit).
- **A2** Importar `core/` con `git subtree add --prefix=_import/core`.
- **A3** Importar `modules/` con `git subtree add --prefix=_import/modules`.
- **A4** Reorganización física a estructura final con `git mv` (tandas por capacidad).
- **A5** Renombre Go modules y npm packages.
- **A6** Workspace tooling: `go.work`, `pnpm-workspace.yaml`, `Cargo.toml`.
- **A7** Publicación inicial de paquetes `platform-*` v0.1.0 (orden topológico).
- **A8** Cutover de pymes a imports nuevos.
- **A9** CI/CD + deprecation formal (`npm deprecate`) + archivado de repos + `DEPRECATED_PACKAGES.md`.
- **A10** (futuro, +90d) Eliminación de repos archivados.

### Ola B — Extracción de transversales de pymes + diseño `lifecycle/`

- **B0** Diseño completo de `platform/lifecycle/{go,ts}`.
- **B1** Extracción Backend Gin glue (RBAC, HTTP errors, pagination/params).
- **B2** Extracción Audit hash-chain → `platform/kernels/activity/go`.
- **B3** Extracción FSM glue backend (MapFSMError, RegisterStatusEndpoint).
- **B4** Extracción FSM glue frontend (reemplazo del builder casero por `platform-fsm`).
- **B5** Extracción Frontend CRUD shell.
- **B6** Implementación y publicación de `platform/lifecycle/{go,ts}`.

### Ola C — Refactor de los 12 módulos CRUDAR de pymes

Aplicar `platform/lifecycle/*` mecánicamente a los 12+ módulos con
CRUDAR, unificar naming `deleted_at` → `archived_at`, activar audit
centralizada, agregar policies por entidad. Pymes queda como referencia.

## Estado actual

| Fase | Estado |
|------|--------|
| A0 | ✅ completed |
| A1 | 🔄 in progress (este commit) |
| A2-A9 | ⏳ pending |
| A10 | ⏳ futuro (+90 días tras A9) |
| Olas B, C | ⏳ pending |

## Snapshots de estado pre-migración

Guardados en `/tmp/platform-migration/`:

- `tags-core.txt` — 66 tags
- `tags-modules.txt` — 39 tags
- `head-{core,modules,pymes}.txt` — commits de referencia
- `versions-{core,modules}-pre.txt` — versiones publicadas al momento de iniciar la migración

Y branches `backup/pre-platform-migration` en GitHub para los 3 repos.

## Documentos relacionados

- [`AGENTS.md`](../../AGENTS.md) — reglas para agentes.
- [`CLAUDE.md`](../../CLAUDE.md) — reglas para Claude Code.
- [`GOVERNANCE.md`](../../GOVERNANCE.md) — gobernanza (en transición).
- `docs/migration/DEPRECATED_PACKAGES.md` — registro de paquetes deprecados (se crea en Fase A9).
- `docs/adr/` — Architecture Decision Records.
