# Paquetes y módulos deprecados (post-migración core+modules → platform)

> Generado al final de la **Ola A** de la migración. Fecha de deprecation: **2026-05-18**.
> Eliminación de repos legacy programada para: **2026-08-16** (+90 días).

Tras la fusión de `github.com/devpablocristo/core` y `github.com/devpablocristo/modules`
en el monorepo `github.com/devpablocristo/platform`, todos los paquetes publicados de
los repos legacy quedan **deprecados** y reemplazados por sus equivalentes platform-*.

Las versiones publicadas permanecen accesibles indefinidamente en npm, PyPI y el
Go proxy. Los consumers no migrados (ponti, nexus, companion, medmory) siguen
funcionando con las versiones existentes hasta su propia migración futura.

---

## npm — paquetes deprecados

Comando estándar de migración para consumers:
```bash
# Reemplazá el nombre viejo por el nuevo y bumpeá la versión.
npm uninstall @devpablocristo/<old> && npm install @devpablocristo/<new>@^0.1.0
```

| Paquete viejo | Última versión | Reemplazo | Versión nueva | Notas |
|---|---|---|---|---|
| `@devpablocristo/core-authn` | — | `@devpablocristo/platform-authn` | 0.1.0 | Drop-in (rename de import). |
| `@devpablocristo/core-browser` | 0.4.0 | `@devpablocristo/platform-browser` | 0.1.0 | Drop-in. |
| `@devpablocristo/core-http` | — | `@devpablocristo/platform-http` | 0.1.0 | Drop-in. |
| `@devpablocristo/core-fsm` | — | `@devpablocristo/platform-fsm` | 0.1.0 | Drop-in (paquete `concurrency/fsm/ts`). |
| `@devpablocristo/core-ai-contracts` | — | `@devpablocristo/platform-ai-contracts` | 0.1.0 | Drop-in. |
| `@devpablocristo/modules-scheduling` | 0.6.2 | `@devpablocristo/platform-scheduling` | 0.1.0 | Drop-in. |
| `@devpablocristo/modules-search` | 0.2.0 | `@devpablocristo/platform-search` | 0.1.0 | Drop-in. |
| `@devpablocristo/modules-shell-sidebar` | 0.3.0 | `@devpablocristo/platform-shell-sidebar` | 0.1.0 | Drop-in. |
| `@devpablocristo/modules-admin-insights` | 0.1.1 | `@devpablocristo/platform-admin-insights` | 0.1.0 | Drop-in. |
| `@devpablocristo/modules-calendar-board` | 0.2.0 | `@devpablocristo/platform-calendar-board` | 0.1.1 | v0.1.1 incluye LegacyRef cast para FullCalendar 6.x. |
| `@devpablocristo/modules-kanban-board` | 0.3.1 | `@devpablocristo/platform-kanban-board` | 0.1.0 | Drop-in. |
| `@devpablocristo/modules-crud-ui` | 0.9.1 | `@devpablocristo/platform-crud-ui` | 0.1.0 | En Ola B integra con `@devpablocristo/platform-lifecycle`. |
| `@devpablocristo/modules-ui-data-display` | 0.3.3 | `@devpablocristo/platform-ui-data-display` | 0.1.0 | Drop-in. |
| `@devpablocristo/modules-ui-modal` | 0.2.0 | `@devpablocristo/platform-ui-modal` | 0.1.0 | Drop-in. |
| `@devpablocristo/modules-ui-section-hub` | 0.2.0 | `@devpablocristo/platform-ui-section-hub` | 0.1.0 | Drop-in. |
| `@devpablocristo/modules-ui-page-shell` | 0.2.0 | `@devpablocristo/platform-ui-page-shell` | 0.1.0 | Drop-in. |
| `@devpablocristo/modules-ui-conversation-inbox` | 0.2.0 | `@devpablocristo/platform-conversation-inbox` | 0.1.0 | Drop del prefijo `ui-` (es feature vertical, no primitiva). |
| `@devpablocristo/modules-ui-notification-feed` | 0.2.0 | `@devpablocristo/platform-notification-feed` | 0.1.0 | Drop del prefijo `ui-`. |

### npm — NO reemplazados (sin migración, sin consumers vivos)

| Paquete viejo | Razón | Acción si lo necesitás |
|---|---|---|
| `@devpablocristo/modules-ai-console` | Deprecado pre-migración | Quedó publicado, sin soporte. Reimplementar en consumer. |
| `@devpablocristo/modules-ui-filters` | Deprecado pre-migración | Idem. |
| `@devpablocristo/modules-ui-forms` | Deprecado pre-migración | Idem. |
| `@devpablocristo/modules-work-orders` | Sin consumers (toollab = 0) | Idem. |

---

## Go — módulos deprecados

Comando estándar de migración para consumers:
```bash
# Reemplazá el module path en imports (.go) y require directives (go.mod).
# Lista completa de mappings en tooling/scripts/rename-imports.py.
sed -i 's|github.com/devpablocristo/core/X|github.com/devpablocristo/platform/X|g' file.go
go mod tidy
```

| Módulo viejo | Última versión publicada | Reemplazo | Versión nueva | Notas |
|---|---|---|---|---|
| `github.com/devpablocristo/core/errors/go` | 0.1.0 | `github.com/devpablocristo/platform/errors/go` | 0.1.0 | |
| `github.com/devpablocristo/core/validate/go` | 0.1.1 | `github.com/devpablocristo/platform/validate/go` | 0.1.0 | |
| `github.com/devpablocristo/core/config/go` | 0.1.0 | `github.com/devpablocristo/platform/config/go` | 0.1.0 | |
| `github.com/devpablocristo/core/observability/go` | 0.1.0 | `github.com/devpablocristo/platform/observability/go` | 0.1.0 | |
| `github.com/devpablocristo/core/http/go` | 0.1.1 | `github.com/devpablocristo/platform/http/go` | 0.1.0 | |
| `github.com/devpablocristo/core/http/gin/go` | 0.1.1 | `github.com/devpablocristo/platform/http/gin/go` | 0.1.0 | |
| `github.com/devpablocristo/core/security/go` | 0.1.0 | `github.com/devpablocristo/platform/security/go` | 0.1.0 | |
| `github.com/devpablocristo/core/concurrency/go` | 0.1.1 | `github.com/devpablocristo/platform/concurrency/go` | 0.1.0 | |
| `github.com/devpablocristo/core/authn/go` | 0.2.1 | `github.com/devpablocristo/platform/authn/go` | 0.1.0 | |
| `github.com/devpablocristo/core/authz/go` | 0.1.0 | `github.com/devpablocristo/platform/authz/go` | 0.1.0 | |
| `github.com/devpablocristo/core/databases/postgres/go` | 0.1.1 | `github.com/devpablocristo/platform/databases/postgres/go` | 0.1.0 | |
| `github.com/devpablocristo/core/databases/dynamodb/go` | 0.1.0 | `github.com/devpablocristo/platform/databases/dynamodb/go` | 0.1.0 | |
| `github.com/devpablocristo/core/eventing/go` | 0.1.0 | `github.com/devpablocristo/platform/eventing/go` | 0.1.0 | |
| `github.com/devpablocristo/core/webhook/go` | 0.1.0 | `github.com/devpablocristo/platform/webhook/go` | 0.1.0 | |
| `github.com/devpablocristo/core/notifications/go` | 0.3.0 | `github.com/devpablocristo/platform/notifications/go` | 0.1.1 | v0.1.1 restaura `TenantID` (HEAD del core había revertido a `OrgID`). |
| `github.com/devpablocristo/core/ingestion/go` | 0.1.0 | `github.com/devpablocristo/platform/ingestion/go` | 0.1.0 | |
| `github.com/devpablocristo/core/calendar/ics/go` | 0.1.0 | `github.com/devpablocristo/platform/calendar/ics/go` | 0.1.0 | |
| `github.com/devpablocristo/core/calendar/sync/google/go` | 0.1.0 | `github.com/devpablocristo/platform/sdks/google-calendar/go` | 0.1.0 | Promovido a `sdks/` (cliente a servicio externo). |
| `github.com/devpablocristo/core/scheduling/go` | 0.1.0 | `github.com/devpablocristo/platform/jobs/go` | 0.1.0 | **Renombre de capability**: era una primitiva cron/jobs; el nombre `scheduling` quedó para el feature vertical. |
| `github.com/devpablocristo/core/providers/aws/lambda/go` | 0.1.0 | `github.com/devpablocristo/platform/sdks/aws/lambda/go` | 0.1.0 | Promovido a `sdks/`. |
| `github.com/devpablocristo/core/providers/aws/s3/go` | 0.1.0 | `github.com/devpablocristo/platform/sdks/aws/s3/go` | 0.1.0 | Promovido a `sdks/`. |
| `github.com/devpablocristo/core/providers/aws/sqs/go` | 0.1.0 | `github.com/devpablocristo/platform/sdks/aws/sqs/go` | 0.1.0 | Promovido a `sdks/`. |
| `github.com/devpablocristo/core/saas/go` | 0.1.0 | `github.com/devpablocristo/platform/kernels/saas/go` | 0.1.0 | Promovido a `kernels/` (mini-app embebible). |
| `github.com/devpablocristo/core/governance/go` | 0.4.1 | `github.com/devpablocristo/platform/kernels/governance/go` | 0.1.0 | Promovido a `kernels/`. |
| `github.com/devpablocristo/core/activity/go` | 0.1.0 | `github.com/devpablocristo/platform/kernels/activity/go` | 0.1.0 | Promovido a `kernels/`. |
| `github.com/devpablocristo/core/artifact/go` | 0.1.0 | `github.com/devpablocristo/platform/kernels/artifact/go` | 0.1.0 | Promovido a `kernels/`. |
| `github.com/devpablocristo/core/ai/go` | 0.3.0 | `github.com/devpablocristo/platform/kernels/ai/go` | 0.1.0 | Promovido a `kernels/`. |
| `github.com/devpablocristo/core/ai/contracts/go` | 0.1.0 | `github.com/devpablocristo/platform/contracts/ai/go` | 0.1.0 | Promovido a `contracts/` top-level. |
| `github.com/devpablocristo/modules/scheduling/go` | 0.5.0 | `github.com/devpablocristo/platform/features/scheduling/go` | 0.1.0 | Promovido a `features/` (vertical de producto). |
| `github.com/devpablocristo/modules/crud/paths/go` | 0.1.0 | `github.com/devpablocristo/platform/features/crud/paths/go` | 0.1.0 | Promovido a `features/`. |
| `github.com/devpablocristo/modules/crud/archive/go` | 0.1.0 | `github.com/devpablocristo/platform/features/crud/archive/go` | 0.1.0 | En Ola B será re-exportado por `platform/lifecycle/go/archive`. |

---

## Python (PyPI) — paquetes deprecados

Comando estándar:
```bash
pip uninstall devpablocristo-<old>
pip install devpablocristo-platform-<new>
# y actualizá imports en código.
```

| Paquete viejo | Última versión | Reemplazo | Versión nueva | Notas |
|---|---|---|---|---|
| `devpablocristo-httpserver` | 0.2.0 | `devpablocristo-platform-http` | 0.1.0 | Mismo módulo `httpserver` por compatibilidad de imports. |
| `devpablocristo-core-ai` | 0.8.2 | `devpablocristo-platform-ai-runtime` | 0.1.0 | Mismo paquete `runtime`. Cambian referencias a `contracts/capabilities/v1` (ahora en `platform/contracts/ai/`). |

---

## Calendario y procedimientos

| Hito | Fecha | Estado |
|---|---|---|
| Inicio Ola A (cutover pymes Go + npm + Python) | 2026-05-18 | ✅ completado |
| Fin Ola A — paquetes `platform-*` publicados | 2026-05-18 | ✅ completado |
| Deprecation formal ejecutada (`npm deprecate`, marker en Go) | 2026-05-18 | ✅ |
| Repos GitHub `core` y `modules` archivados | 2026-05-18 | ✅ |
| 90 días post-archivado — ventana de hotfix abierta | 2026-08-16 | pendiente |
| Eliminación definitiva de los repos | ≥ 2026-08-16 | pendiente |
| Migración de ponti a platform-* | TBD | pendiente |
| Migración de nexus a platform-* | TBD | pendiente |
| Migración de companion a platform-* | TBD | pendiente |
| Migración de medmory a platform-* | TBD | pendiente |
| Migración de toollab a platform-* | n/a | toollab no consume core/modules |

### Procedimiento para hotfix durante la ventana de 90 días

1. Documentar el bug y por qué no se puede aplicar el fix solo en `platform/`.
2. Desarchivar el repo legacy en GitHub Settings.
3. Aplicar fix, tagear, publicar versión patch (ej. `core-authn@0.3.1`).
4. Volver a archivar.
5. Actualizar este `DEPRECATED_PACKAGES.md` con la nueva versión y el motivo del hotfix.
6. Notificar al consumer no migrado que existe el patch.

### Cuándo eliminar definitivamente los repos legacy

Pre-requisitos para eliminar (todos deben cumplirse):

- [ ] `ponti` consume solo `platform-*`
- [ ] `nexus` consume solo `platform-*`
- [ ] `companion` consume solo `platform-*`
- [ ] `medmory` consume solo `platform-*`
- [x] `toollab` confirmado sin consumers en core/modules (ya verificado en Ola A)
- [ ] 90 días transcurridos desde el archivado (≥ 2026-08-16)
- [ ] Backup local del contenido de los repos guardado fuera del workspace

Cuando los 7 checks pasen: eliminar los repos GitHub.

### Backups

- Branches en GitHub: `backup/pre-platform-migration` en `core`, `modules` y `pymes` (pusheados al inicio de Ola A0).
- Snapshots locales: `/tmp/platform-migration/` con tags, head, dirty state y versiones publicadas pre-migración.
- Plan de migración: `~/.claude/plans/actu-como-staff-principal-engineer-immutable-yao.md`.
