# @devpablocristo/modules-crud-ui

Componentes React + utilidades para construir páginas CRUD configurables
(lista paginada + búsqueda + filtros + bulk actions + edición). Pensado para
montar verticales de negocio (billing, scheduling, etc.) sin reescribir
shell de tabla.

## Instalación

```bash
npm install @devpablocristo/modules-crud-ui
```

## Uso

```tsx
import { CrudPage } from '@devpablocristo/modules-crud-ui'

<CrudPage
  resource="invoices"
  columns={invoiceColumns}
  fetchPage={fetchInvoicesPage}
  features={{ sort: true, filters: true, export: true }}
/>
```

## Qué incluye

- `CrudPage.tsx` — shell de página CRUD con paginación cursor/offset
- `crudFeatureDefaults.ts` — opt-in de features (sort, filters, export, etc.)
- `crudUiPreferences.ts` + `CrudUiPreferencesPanel.tsx` — persistencia local de columnas/sort por usuario
- `columnSort.ts`, `csvToolbarMerge.ts` — helpers internos

## Peer deps

- `react`: `^18.0.0 || ^19.0.0`

## Consumidores

pymes/frontend
