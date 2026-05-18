# Modules

Paquetes listos para integrar **encima de [`core`](../core)**, sin lógica de negocio y ordenados por `responsabilidad/subresponsabilidad/lenguaje/paquete`.

## Layout

```
modules/
  admin/
    insights/ts     → npm @devpablocristo/modules-admin-insights
  ai/
    console/ts      → npm @devpablocristo/modules-ai-console
  calendar/
    board/ts        → npm @devpablocristo/modules-calendar-board
  crud/
    ui/ts           → npm @devpablocristo/modules-crud-ui
    paths/go        → module github.com/devpablocristo/modules/crud/paths/go
  kanban/
    board/ts        → npm @devpablocristo/modules-kanban-board
  scheduling/
    go             → module github.com/devpablocristo/modules/scheduling/go
  ui/
    data-display/ts → npm @devpablocristo/modules-ui-data-display
    filters/ts      → npm @devpablocristo/modules-ui-filters
    forms/ts        → npm @devpablocristo/modules-ui-forms
    modal/ts        → npm @devpablocristo/modules-ui-modal
    page-shell/ts   → npm @devpablocristo/modules-ui-page-shell
    section-hub/ts  → npm @devpablocristo/modules-ui-section-hub
    notification-feed/ts → npm @devpablocristo/modules-ui-notification-feed
```

## Responsabilidades

- `ai`: UI reusable para consolas con copilot e insights.
- `admin`: superficies reutilizables de administración y observabilidad para consolas.
- `calendar`: calendario reusable basado en FullCalendar.
- `crud`: shell CRUD y segmentos HTTP compartidos.
- `kanban`: tablero por estados y drag and drop.
- `scheduling`: bounded context reusable de agenda y cola virtual, montado encima de primitivas de scheduling en `core`.
- `ui`: primitivas de interfaz genéricas que no pertenecen a CRUD ni a Kanban.
  Incluye `notification-feed` para bandejas de avisos no-chat, `page-shell` para shells de consola sobre `core-browser` + `modules-shell-sidebar`, `modal` para superficies/dialog shells reutilizables, `section-hub` para hubs de navegación por secciones, y entradas accionables.

## Referencias

- Ver [`crud/README.md`](./crud/README.md) para el detalle de la familia CRUD.
- El chequeo con Docker vive en [`docker-compose.yml`](./docker-compose.yml).
- La política de versionado vive en [`docs/VERSIONING.md`](./docs/VERSIONING.md).

## Tooling y CI

- El tooling TS del repo queda fijado en `20.19.0` vía `.nvmrc`.
- Cuando un paquete tiene `package-lock.json`, la CI y los scripts usan `npm ci`; si todavía no tiene lock, cae a `npm install`.
- `Dependabot` semanal cubre `github-actions` y todos los paquetes npm versionados del repo.

## Versionado

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

Cada una tiene su propio archivo `VERSION`.

- La validación de consistencia se corre con:

```bash
npm run version:check
```

- La suite transversal del repo se corre con:

```bash
npm run test:all
```

- El listado de versiones y tags esperados se obtiene con:

```bash
npm run version:list
```

- La auditoría de tags remotos publicados se corre con:

```bash
npm run release:audit
```
