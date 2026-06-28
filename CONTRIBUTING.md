# Contribuir a `platform`

Antes de abrir PR, leer:

- [`GOVERNANCE.md`](./GOVERNANCE.md)
- [`docs/platform/VERSIONING.md`](./docs/platform/VERSIONING.md)
- [`docs/platform/RELEASE_FLOW.md`](./docs/platform/RELEASE_FLOW.md)

## Principios

- `platform` contiene paquetes reutilizables, no aplicaciones.
- No introducir dominio privado de consumers.
- Preferir APIs chicas, estables y testeables.
- No duplicar primitivas que ya existan en otro paquete del repo.
- Mantener PRs por tema: bugfix, package release, tooling, docs o refactor.

## Estructura

Cada implementacion versionable vive en un path con runtime propio:

```text
http/go
http/ts
http/python
authn/go
authn/ts
features/scheduling/go
features/scheduling/ts
ui/modal/ts
```

La raiz de cada implementacion contiene su manifest (`go.mod`,
`package.json`, `pyproject.toml` o `Cargo.toml`) y su `VERSION`.

## Proponer un paquete nuevo

Checklist:

1. Confirmar que no existe una primitiva equivalente.
2. Confirmar consumidores reales o planificados.
3. Ubicarlo en la capa correcta: transversal, `contracts`, `sdks`, `kernels`,
   `features` o `ui`.
4. Agregar README minimo cuando el uso no sea obvio.
5. Agregar tests del runtime correspondiente.
6. Agregar `VERSION` inicial.
7. Correr guardrails y tests relevantes.

## Tests

- Go: `go test ./...` dentro del modulo, o `npm run test:go` para todos.
- TS: `pnpm --filter <package> run typecheck` y `pnpm --filter <package> run test`.
- Python: scripts de `tooling/scripts/test-*.sh`.
- Rust: `cargo test` dentro del crate, o `npm run test:rust` para todos.

## Release

Ver [`docs/platform/RELEASE_FLOW.md`](./docs/platform/RELEASE_FLOW.md).

Resumen:

1. Cambiar codigo y tests.
2. Bump del modulo con `tooling/scripts/bump-module-version.sh <path> <version>`.
3. PR + CI verde.
4. Merge a `main`.
5. Crear/pushear tag `<path>/vX.Y.Z`.
6. Verificar registry/proxy y consumers.

## Naming

- Go modules: `github.com/devpablocristo/platform/<path>`.
- npm packages: `@devpablocristo/platform-*`.
- Python packages: `devpablocristo-platform-*`.
- Rust crates: `platform-*`.

## Breaking changes

- Major bump obligatorio.
- Documentar impacto y consumidores afectados.
- Coordinar tag/publicacion con PRs de consumo en Axis, Medmory u otros repos.
