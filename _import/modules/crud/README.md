# `crud`

Responsabilidad CRUD agnóstica de dominio, dividida por subresponsabilidad:

| Artefacto | Relación con `core` | Contenido |
|-----------|---------------------|-----------|
| `ui/ts` | Depende de `core/browser/ts` (`@devpablocristo/core-browser/crud` para layout) | `@devpablocristo/modules-crud-ui` — `CrudPage`, tipos, strings, rutas REST, `mergeCanonicalCrudDefaults` (`/surface`), preferencias de UI (`createCrudUiPreferencesApi`, `CrudUiPreferencesPanel`). |
| `paths/go` | Módulo Go chico con `go.mod` propio | `github.com/devpablocristo/modules/crud/paths/go` — constantes de segmentos URL. |

## Publicar `@devpablocristo/modules-crud-ui` (npm)

Desde `modules/crud/ui/ts` (tras `npm ci` y tests verdes): `npm publish --access public`. Tag Git del repo `modules`: `crud/ui/ts/v<versión>` (ver `docs/VERSIONING.md`). En el producto, fijar dependencia `^0.7.0` o superior que incluya `surface` y preferencias de UI.

## Imports

**TypeScript**

```ts
import { CrudPage, crudStringsEs } from "@devpablocristo/modules-crud-ui";
```

**Go**

```go
import "github.com/devpablocristo/modules/crud/paths/go/paths"
// paths.SegmentArchived, etc.
```

`replace` local en el `go.mod` del backend:

```text
replace github.com/devpablocristo/modules/crud/paths/go => ../modules/crud/paths/go
```

## Verificación con Docker

Desde `modules/`:

```bash
docker compose build crud-ui-ts-check crud-paths-go-check
```

El build falla si `npm run typecheck`, `npm test` o `go test ./...` fallan.

## Por qué `paths/go` no es un CRUD completo

Los handlers, RBAC, GORM y DTOs viven en cada servicio. Acá solo se comparten nombres de rutas para no desalinear FE y BE.
