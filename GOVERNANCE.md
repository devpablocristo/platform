# Gobernanza de `platform`

`platform` es la fuente de verdad para paquetes reutilizables del ecosistema
`pablo`. Reemplaza a los repos legacy `core` y `modules`.

## Capas

| Capa | Path | Contenido |
|---|---|---|
| Transversal | raices como `http`, `authn`, `errors`, `observability` | Primitivas tecnicas agnosticas |
| Contracts | `contracts/` | Contratos compartidos cross-runtime |
| SDKs | `sdks/` | Clientes/adapters a servicios externos |
| Kernels | `kernels/` | Mini-dominios embebibles o engines reutilizables |
| Features | `features/` | Verticales o componentes reutilizables de dominio acotado |
| UI | `ui/` | Primitivas visuales compartidas |
| Testing | `testing/` | Helpers de test reutilizables |

## Reglas duras

- `platform` no contiene codigo especifico de una app.
- Las apps no deben reinventar primitives ya publicadas en `platform`.
- Los consumers productivos usan versiones publicadas, no paths locales.
- Cada modulo versionable tiene `VERSION` propio.
- El tag, el manifest y `VERSION` deben coincidir.
- No publicar si fallan tests del modulo.

## Promocion a `platform`

Para extraer codigo desde una app:

1. Debe tener frontera reusable y estable.
2. Debe tener consumidores reales o planificados.
3. No debe arrastrar modelos privados, rutas privadas ni copy de producto.
4. Debe incluir tests y documentacion suficiente para consumo externo.
5. Debe publicarse antes de que otros repos lo consuman.

La "regla de 3 consumidores" sigue siendo una buena heuristica, pero no es
absoluta: una primitiva critica puede entrar antes si evita duplicacion riesgosa
o establece un contrato comun.

## Versionado

- Semver estricto.
- Minor: API compatible.
- Patch: bugfix compatible.
- Major: breaking change.
- Sin version global del repo.

Ver [`docs/platform/VERSIONING.md`](docs/platform/VERSIONING.md).

## Release

- Go publica via tags `<path>/vX.Y.Z` y Go proxy.
- TS publica via workflow `publish-ts-package` al pushear tags TS.
- Python publica via workflow `publish-python-package` al pushear tags Python.
- Rust se valida en CI; publicar crates requiere decision explicita.

Ver [`docs/platform/RELEASE_FLOW.md`](docs/platform/RELEASE_FLOW.md).

## Consumers

Axis, Medmory y otros repos deben:

- consumir tags/versiones publicadas;
- evitar `replace` o `file:` commiteados;
- abrir PR de bump despues de publicar platform;
- correr sus checks propios antes de cerrar la adopcion.

## Docs legacy

`docs/core/` y `docs/modules/` quedan como historico de migracion. La fuente
operativa actual es `docs/platform/` y este archivo.
