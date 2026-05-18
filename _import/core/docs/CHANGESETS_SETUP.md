# Setup de changesets — propuesta

**Estado**: PROPUESTA — no implementada. Sigue [GOVERNANCE §11 (excepciones)](../GOVERNANCE.md#11-excepciones) para aplicación.

Parte del audit core+modules 2026-05-15 (MP-04).

## Por qué

Hoy cada package en `core/*ts` tiene:

- Bumps manuales editando `VERSION` + `package.json`
- Sin CHANGELOG.md por package
- Auditoría de "¿qué cambió entre v0.2.0 y v0.3.0?" → `git log` manual

Con `@changesets/cli`:

- Cada PR que cambia un package incluye un `changeset` markdown describiendo el cambio
- En release, los changesets se agrupan automáticamente en CHANGELOG.md por package
- Bump de versión automático según el tipo declarado (patch/minor/major)
- Permite publicar varios packages con sus cambios consolidados de una

## Costo / riesgo

- **Migración inicial**: medio día. Setup + adaptar publish workflow.
- **Carga por PR**: cada contribuyente debe agregar un changeset (`pnpm changeset`). Es 1 archivo md de 5 líneas pero hay que recordar.
- **Riesgo**: si se mezcla con flujo actual (editar VERSION manual), conviven dos sources of truth.

## Plan de migración (cuando se decida ejecutar)

### Prerrequisito

Idealmente correr **después** de MP-03 (workspace pnpm) — changesets se integra mejor en un workspace.

### Paso 1 — Instalar

```bash
cd /home/pablocristo/Proyectos/pablo/core
pnpm add -Dw @changesets/cli  # si MP-03 ya está aplicado
# Si no, instalar a nivel root con npm:
# npm install -D @changesets/cli
npx changeset init
```

### Paso 2 — Config

Editar `.changeset/config.json`:

```json
{
  "$schema": "https://unpkg.com/@changesets/config@2.3.1/schema.json",
  "changelog": "@changesets/changelog-github",
  "commit": false,
  "fixed": [],
  "linked": [],
  "access": "public",
  "baseBranch": "main",
  "updateInternalDependencies": "patch",
  "ignore": []
}
```

### Paso 3 — Workflow

Reemplazar `publish-ts-package.yml` por publish via changesets:

```yaml
name: core-publish-ts

on:
  push:
    branches: [main]

jobs:
  publish:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
        with:
          fetch-depth: 0
      - uses: actions/setup-node@v6
        with:
          node-version: "20.19.0"
      - run: npm ci

      - name: Create release PR or publish
        uses: changesets/action@v1
        with:
          publish: pnpm run release
          version: pnpm run version-packages
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          NPM_TOKEN: ${{ secrets.NPM_TOKEN }}
```

Define en root `package.json`:

```json
"scripts": {
  "release": "changeset publish",
  "version-packages": "changeset version"
}
```

### Paso 4 — Convención de PR

Agregar a `CONTRIBUTING.md`:

> Cada PR que cambia código de un package debe incluir un changeset.
> Correr `npx changeset` y describir el cambio. Tipos: patch (fix), minor (feature), major (breaking).

### Paso 5 — CHANGELOG.md por package

Changesets genera automáticamente `<path>/CHANGELOG.md` en cada release.

### Paso 6 — Coexistencia con VERSION

Durante transición:

- VERSION file queda como source-of-truth secundaria (CI valida que `VERSION == package.json.version == git tag`)
- Eventualmente borrar VERSION una vez que el equipo se acostumbre

## Rollback

Revertir el commit que agregó `.changeset/` + el workflow. Volver a edición manual.

## Cuándo

**No prioritario.** Adoptar cuando:

- Releases frecuentes (>5/mes en modules o core)
- Necesidad de CHANGELOG por package para audit/cliente
- MP-03 ya esté aplicado

Hoy el flujo manual funciona — adoptar prematuro agrega carga sin retorno.
Re-evaluar en 6 meses (Q4 2026).
