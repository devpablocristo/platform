# Release flow — `modules`

Cómo publicar una versión de un módulo `modules/*`.

Mismo flujo que `core` (ver [core/docs/RELEASE_FLOW.md](../../core/docs/RELEASE_FLOW.md))
con detalles específicos de UI.

## Convención

- **Tag**: `<path>/v<X.Y.Z>` (ej. `scheduling/ts/v0.6.2`, `ui/conversation-inbox/ts/v0.2.0`).
- **Source of truth**: campo `version` de `package.json` + archivo `VERSION` (cuando exista). Ambos deben coincidir con el tag.

## Flujo estándar (TS)

1. Editar `<path>/package.json` y `<path>/VERSION` con la nueva versión.
2. Actualizar `src/` con cambios funcionales.
3. Tests verdes (`npm test` en el path).
4. Si hay breaking change en el API: actualizar README + bump major.
5. PR → merge a main.
6. Workflow [`publish-ts-package.yml`](../.github/workflows/publish-ts-package.yml):
   - Detecta el cambio de versión
   - Corre tests + typecheck
   - `npm publish` (idempotente)
   - Crea + push del tag

## Cadencia

- **Mensual mínimo** por package con commits (aunque solo sean dev-deps bumps).
- **Inmediato** para features/fixes funcionales.
- Si vas a hacer churn de dev-deps en muchos packages, hacelo en un PR único y publicá todos juntos.

## Post-release housekeeping

Idéntico a core (ver §Post-release housekeeping en core/docs/RELEASE_FLOW.md):

- Preferir acumular para el próximo bump real.
- No más de 5 commits post-tag sin tagear.

## Coordinación con `core`

Si un módulo de `modules` depende de un módulo de `core`:

1. **Publicar primero `core`**.
2. Bumpear la dep en `modules/<path>/package.json`.
3. Publicar `modules/<path>` con la nueva versión de core.

Esto evita versiones de modules referenciando versiones inexistentes de core.

## Coordinación con consumers (apps)

Cuando publicás un breaking change:

1. Comunicar antes (issue + ping a Pablo / dueños de apps).
2. Publicar el major.
3. Apps actualizan en su próximo ciclo.

Para changes no-breaking, las apps reciben PRs automáticos vía Renovate (LP-01 cuando esté implementado).

## Rollback

Idéntico a core:

1. No unpublish.
2. Patch fix + nuevo bump.
3. `npm deprecate` la versión rota.
4. Comunicar.

## Verificación post-release

```bash
npm view @devpablocristo/modules-<name> version
git tag --list '<path>/v*' | tail -1
```

Deben coincidir.
