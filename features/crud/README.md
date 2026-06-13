# `crud`

Responsabilidad CRUD agnóstica de dominio, dividida por subresponsabilidad:

| Artefacto | Relación con `core` | Contenido |
|-----------|---------------------|-----------|
| `ui/ts` | Depende de `platform/browser/ts` para layout | `@devpablocristo/platform-crud-ui` — `CrudPage`, tipos, strings, rutas REST, `mergeCanonicalCrudDefaults` (`/surface`), preferencias de UI (`createCrudUiPreferencesApi`, `CrudUiPreferencesPanel`). |
| `paths/go` | Módulo Go chico con `go.mod` propio | `github.com/devpablocristo/platform/features/crud/paths/go` — constantes de segmentos URL. |

## Publicar `@devpablocristo/platform-crud-ui` (npm)

Desde `platform/features/crud/ui/ts` (tras `npm ci` y tests verdes): `npm publish --access public`.

## Imports

**TypeScript**

```ts
import { CrudPage, crudStringsEs } from "@devpablocristo/platform-crud-ui";
```

**Go**

```go
import "github.com/devpablocristo/platform/features/crud/paths/go/paths"
// paths.SegmentArchived, etc.
```

`replace` local en el `go.mod` del backend:

```text
replace github.com/devpablocristo/platform/features/crud/paths/go => ../platform/features/crud/paths/go
```

## Verificación con Docker

Desde `platform/`:

```bash
docker compose build crud-ui-ts-check crud-paths-go-check
```

El build falla si `npm run typecheck`, `npm test` o `go test ./...` fallan.

## Por qué `paths/go` no es un CRUD completo

Los handlers, RBAC, GORM y DTOs viven en cada servicio. Acá solo se comparten nombres de rutas para no desalinear FE y BE.
