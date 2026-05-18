# Release flow — `core`

Cómo publicar una versión de un módulo `core/*`.

## Convención

- **Tag**: `<path>/v<X.Y.Z>` (ej. `authn/ts/v0.3.0`, `ai/go/v0.3.0`).
- **Source of truth**: el archivo `VERSION` (Go) o el campo `version` de `package.json` (TS).
- **El tag debe igualar el VERSION del commit que apunta.**

## TS packages — flujo estándar

1. Editar `<path>/package.json` y `<path>/VERSION` con la nueva versión.
2. Si hay cambios funcionales: actualizar README + tests.
3. PR → merge a main.
4. El workflow [`publish-ts-package.yml`](../.github/workflows/publish-ts-package.yml):
   - Detecta el cambio de VERSION en el path
   - Corre tests y typecheck
   - `npm publish` (idempotente — si la versión ya existe, no rompe)
   - Crea + push del tag `<path>/v<X.Y.Z>`

## Go modules — flujo estándar

1. Editar `<path>/VERSION` con la nueva versión.
2. PR → merge a main.
3. Workflow corta tag `<path>/v<X.Y.Z>` automáticamente.
4. Go proxy indexará la versión en minutos.

Para verificar: `go list -m -versions github.com/devpablocristo/core/<path>`

## Cadencia (ver GOVERNANCE §8)

- **Mínimo mensual** por módulo con commits (aunque sean dev-deps bumps).
- **Inmediato** al mergear feature/fix funcional.

## Post-release housekeeping

Si necesitás mergear docs/chores después de tagear:

- Preferir esperar al próximo bump real (acumular en el mismo PR).
- Si urgen, cortar patch `vX.Y.Z+1` inmediato.
- **Prohibido acumular >5 commits post-tag sin tagear** (rompe trazabilidad).

## Rollback

Si una versión publicada rompe consumers:

1. **No unpublish**: npm bloquea unpublish después de 72h, y aunque se pueda, rompe builds aguas abajo.
2. **Patch fix**: cortar `vX.Y.Z+1` con el rollback.
3. **Deprecate la versión rota** (solo TS): `npm deprecate '<name>@X.Y.Z' 'broken — upgrade to X.Y.Z+1'`.
4. **Comunicar a consumers**: issue en core + ping a dueños de companion/nexus/pymes.

## Pre-release / experimental

Para código sin compromiso de API stability:

- **Sin tag**: README marca "experimental, no consumir desde código productivo". Patrón actual de `core/{activity,artifact,eventing}/go`.
- **Pre-release tag** (alternativo): `<path>/v0.1.0-alpha.1`. Útil si hay un consumer pidiendo probarlo sin compromiso.

## Verificación post-release

Después de cada release:

```bash
# TS
npm view @devpablocristo/core-<name> version
git tag --list '<path>/ts/v*' | tail -1

# Go
go list -m -versions github.com/devpablocristo/core/<path>
git tag --list '<path>/v*' | tail -1
```

VERSION en repo, tag git, y registro publicado deben coincidir.
