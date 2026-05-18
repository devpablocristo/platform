# Versionado independiente en `modules`

`modules` es un solo repo, pero no tiene una sola versión global.

La unidad de versionado es cada implementación concreta:

- `ai/console/ts`
- `admin/insights/ts`
- `calendar/board/ts`
- `crud/ui/ts`
- `crud/paths/go`
- `kanban/board/ts`
- `scheduling/go`
- `ui/data-display/ts`
- `ui/filters/ts`
- `ui/forms/ts`
- `ui/modal/ts`
- `ui/page-shell/ts`
- `ui/section-hub/ts`
- `ui/notification-feed/ts`

## Regla

Cada implementación tiene su propio archivo `VERSION` en la raíz del runtime:

```text
ai/
  console/
    ts/
      VERSION
      package.json

admin/
  insights/
    ts/
      VERSION
      package.json

calendar/
  board/
    ts/
      VERSION
      package.json

crud/
  ui/
    ts/
      VERSION
      package.json

crud/
  paths/
    go/
      VERSION
      go.mod

scheduling/
  go/
    VERSION
    go.mod

kanban/
  board/
    ts/
      VERSION
      package.json

ui/
  data-display/
    ts/
      VERSION
      package.json
  filters/
    ts/
      VERSION
      package.json
  forms/
    ts/
      VERSION
      package.json
  modal/
    ts/
      VERSION
      package.json
  notification-feed/
    ts/
      VERSION
      package.json
  page-shell/
    ts/
      VERSION
      package.json
  section-hub/
    ts/
      VERSION
      package.json
```

## Semántica

- usar `semver`
- en `VERSION` guardar `0.1.0`, no `v0.1.0`
- empezar en `0.x` mientras la API siga moviéndose
- no inventar una versión global del repo
- solo se sube la versión del módulo que cambió

## Tags

Los tags se cortan por subdirectorio:

- `ai/console/ts/v0.1.0`
- `admin/insights/ts/v0.1.0`
- `calendar/board/ts/v0.1.0`
- `crud/ui/ts/v0.1.0`
- `crud/paths/go/v0.1.0`
- `kanban/board/ts/v0.1.0`
- `scheduling/go/v0.1.0`
- `ui/data-display/ts/v0.1.0`
- `ui/filters/ts/v0.1.0`
- `ui/forms/ts/v0.1.0`
- `ui/modal/ts/v0.1.0`
- `ui/notification-feed/ts/v0.1.0`
- `ui/page-shell/ts/v0.1.0`
- `ui/section-hub/ts/v0.1.0`

Para Go esto sigue la convención correcta de módulos versionados en subdirectorios del monorepo.

## Fuente de verdad

- Go: `VERSION` es la fuente de verdad del release
- TypeScript: `VERSION` y `package.json` deben coincidir

## Scripts

- `scripts/list-module-versions.sh`: lista versiones y tags esperados
- `scripts/validate-module-versions.sh`: valida semver y consistencia
- `scripts/bump-module-version.sh <modulo/runtime> <version>`: sube una versión localmente

## Flujo recomendado

1. hacer cambios en un solo módulo
2. correr validación local
3. subir la versión de ese módulo
4. correr tests del repo
5. crear el tag del subdirectorio correspondiente

## Ejemplos

Listar versiones:

```bash
./scripts/list-module-versions.sh
```

Validar consistencia:

```bash
./scripts/validate-module-versions.sh
```

Subir `crud/ui/ts` a `0.2.0`:

```bash
./scripts/bump-module-version.sh crud/ui/ts 0.2.0
```
