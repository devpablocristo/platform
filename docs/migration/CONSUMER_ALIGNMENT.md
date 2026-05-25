# Consumer Alignment

Estado operativo de consumers despues de la fusion `core` + `modules` -> `platform`.

| Consumer | Estado | Debe consumir | Notas |
|---|---|---|---|
| `pymes` | Migrado | `github.com/devpablocristo/platform/*`, `@devpablocristo/platform-*`, Axis Companion y Nexus por HTTP | Es el consumer de referencia para `platform/lifecycle/go`. El runtime IA local `pymes/ai` queda retirado. |
| `ponti-backend` | Migrado parcialmente | `github.com/devpablocristo/platform/*`, Axis Companion por `COMPANION_BASE_URL` | El codigo ya usa Companion; deploy debe setear `COMPANION_BASE_URL` y `COMPANION_INTERNAL_JWT_SECRET`. |
| `ponti-frontend` | Migrado parcialmente | `@devpablocristo/platform-*` | Conserva nombres de archivos `ponti-ai.openapi.*` como contrato legacy del BFF mientras el backend adapta a Companion. |
| `axis/nexus` | En curso | `github.com/devpablocristo/platform/*` | Usa `platform/lifecycle/go` con `replace` local durante desarrollo. Publicar version antes de quitar `replace`. |
| `axis/companion` | En curso | `github.com/devpablocristo/platform/*` | Igual que Nexus; no debe depender de repos legacy `core`/`modules`. |

## Guardrails

- Los repos de producto no deben agregar imports nuevos `github.com/devpablocristo/core/*` ni `@devpablocristo/core-*`.
- Los repos de producto no deben agregar imports nuevos `github.com/devpablocristo/modules/*` ni `@devpablocristo/modules-*`.
- Las integraciones entre productos (`pymes`, `ponti`, `nexus`, `companion`) son por API estable, no imports de dominio cruzado.
- Los nombres legacy de contrato pueden sobrevivir solo en adapters/BFFs y deben documentar el owner actual.
