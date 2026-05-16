# Migración a pnpm workspace + turbo

**Estado**: PROPUESTA — no implementada. Sigue [GOVERNANCE §11 (excepciones)](../../core/GOVERNANCE.md#11-excepciones) para aplicación.

Parte del audit core+modules 2026-05-15 (MP-03).

## Por qué

Hoy cada TS package en `modules/` (16 packages) tiene:

- `package.json` independiente
- `package-lock.json` propio
- `node_modules/` por package
- Dev-deps duplicadas entre packages (TypeScript, vitest, React types, etc.)
- Bumps de devDeps requieren PR en cada package o "mega-PR" como `e37d39b "Align modules with published core packages"` que tocó 16+ packages

Migrar a **pnpm workspace + turbo**:

- Un único `pnpm-lock.yaml` en root
- `node_modules` hoisted; menos espacio en disco y CI más rápido
- Bumps coordinados: editás un `package.json` o el root y todos se alinean
- `turbo run typecheck --filter=...` para CI incremental
- Cache de turbo entre builds

## Costo / riesgo

- **Migración inicial**: ~1 día. Hay que regenerar lockfile, ajustar scripts, posiblemente CI workflow.
- **Riesgo: romper publish workflow**. El actual `publish-ts-package.yml` corre `npm ci` dentro de cada `<path>/`. Con pnpm hoisting, eso cambia.
- **Aprendizaje**: si alguien no sabe pnpm vs npm, comandos pueden confundir.

## Plan de migración (cuando se decida ejecutar)

### Paso 1 — Branch + setup

```bash
git checkout -b chore/modules-workspace
echo "" > pnpm-workspace.yaml
cat > pnpm-workspace.yaml <<EOF
packages:
  - "ai/console/ts"
  - "admin/insights/ts"
  - "calendar/board/ts"
  - "crud/archive/go"  # NO — Go module, no incluir
  - "crud/paths/go"     # NO
  - "crud/ui/ts"
  - "kanban/board/ts"
  - "scheduling/ts"
  - "search/ts"
  - "sidebar/ts"
  - "ui/conversation-inbox/ts"
  - "ui/data-display/ts"
  - "ui/filters/ts"
  - "ui/forms/ts"
  - "ui/modal/ts"
  - "ui/notification-feed/ts"
  - "ui/page-shell/ts"
  - "ui/section-hub/ts"
  - "work-orders/ts"
EOF
```

### Paso 2 — Limpiar lockfiles individuales + instalar

```bash
find . -name "package-lock.json" -not -path "*/node_modules/*" -delete
find . -name "node_modules" -type d -not -path "*/node_modules/*" -exec rm -rf {} +
pnpm install
```

### Paso 3 — Agregar turbo

```bash
pnpm add -Dw turbo
cat > turbo.json <<EOF
{
  "$schema": "https://turbo.build/schema.json",
  "pipeline": {
    "typecheck": { "outputs": [] },
    "test": { "outputs": ["coverage/**"] },
    "build": { "outputs": ["dist/**"] }
  }
}
EOF
```

### Paso 4 — Ajustar workflows

`publish-ts-package.yml` hoy hace `cd <path> && npm ci && npm run typecheck && npm test`.
Cambiar a:

```yaml
- run: pnpm install --frozen-lockfile
- run: pnpm -F "@devpablocristo/modules-${module##*/}" run typecheck
- run: pnpm -F "@devpablocristo/modules-${module##*/}" run test
```

(El `${module##*/}` extrae el último segmento del path.)

### Paso 5 — Verificar

```bash
pnpm -r run typecheck   # corre typecheck en todos
pnpm -r run test
turbo run build --filter='@devpablocristo/modules-scheduling'
```

### Paso 6 — Smoke test publish

Antes de mergear: bumpear un package de prueba (p.ej. `modules-search` patch),
verificar que el workflow corre correctamente con pnpm en CI.

### Paso 7 — Merge + dejar marcadores

- Mergear a main.
- Actualizar este doc a "implementado en <PR#>".
- Actualizar `core/docs/RELEASE_FLOW.md` con notas pnpm.

## Rollback

Branch `chore/modules-workspace` queda como rollback. Si publish rompe:

1. Revertir el merge.
2. `pnpm install` ya no aplica; restaurar `package-lock.json` desde branch base.
3. Investigar causa raíz en branch.

## Cuándo

**No prioritario.** Costo/beneficio favorable solo cuando:

- Empezamos a hacer bumps coordinados >1/mes
- CI tiempo total >5min por package
- Aparecen breaking changes que requieren coordinación cross-package

Hoy el ecosistema tolera el modelo actual. Re-evaluar en 6 meses (Q4 2026).
