# @devpablocristo/platform-crud-ui

React components and utilities for configurable CRUD pages with explicit
lifecycle views.

The lifecycle surface is:

- `active`
- `archived`
- `trash`

Archive is not deletion. Trash is reversible deletion. Purge is irreversible.

## Install

```bash
npm install @devpablocristo/platform-crud-ui
```

## Usage

```tsx
import { CrudPage } from "@devpablocristo/platform-crud-ui";

<CrudPage
  basePath="/v1/documents"
  supportsArchived
  supportsTrash
  label="document"
  labelPlural="documents"
  labelPluralCap="Documents"
  columns={columns}
  formFields={fields}
  searchText={(row) => row.title}
  toFormValues={(row) => ({ title: row.title })}
  isValid={(values) => String(values.title ?? "").trim().length > 0}
/>;
```

## Included

- `CrudPage.tsx`: CRUD page shell with lifecycle views.
- `restPaths.ts`: route helpers aligned with `platform/lifecycle/go/paths`.
- `crudUiPreferences.ts` and `CrudUiPreferencesPanel.tsx`: local UI preferences.
- `columnSort.ts`, `csvToolbarMerge.ts`: table helpers.
