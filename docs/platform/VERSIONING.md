# Versionado independiente en `platform`

`platform` es un monorepo, pero no tiene una version global publicable.

La unidad de versionado es cada implementacion concreta:

- `http/go`
- `http/ts`
- `http/python`
- `authn/go`
- `authn/ts`
- `errors/rust`
- `features/scheduling/go`
- `features/scheduling/ts`
- `ui/modal/ts`

## Fuente de verdad

Cada implementacion versionable debe tener `VERSION` en la raiz del runtime.

Ademas:

- Go: `VERSION` y el tag del modulo deben coincidir.
- TypeScript: `VERSION` y `package.json.version` deben coincidir.
- Python: `VERSION` y `pyproject.toml project.version` deben coincidir.
- Rust: `VERSION` y `Cargo.toml package.version` deben coincidir cuando el
  crate sea publicable.

## Tags

Los tags se cortan por subdirectorio:

```text
http/go/v0.2.0
authn/ts/v0.1.1
kernels/ai/runtime/python/v0.1.0
features/scheduling/go/v0.2.0
ui/modal/ts/v0.1.0
```

El prefijo del tag debe ser exactamente el path del modulo.

## Semver

- Patch: fixes compatibles.
- Minor: APIs nuevas compatibles.
- Major: breaking changes.

Mientras una API siga inestable, usar `0.x` y documentar el grado de estabilidad
en el README del modulo.

## Scripts

```bash
npm run validate:versions
bash tooling/scripts/list-module-versions.sh
bash tooling/scripts/bump-module-version.sh <module-path> <version>
bash tooling/scripts/check-remote-tags.sh
```

`validate:versions` es el guardrail obligatorio antes de publicar.
`check-remote-tags.sh` valida tags remotos de modulos publicables hoy
(Go/TypeScript/Python). Rust queda excluido por defecto hasta definir registry
y naming de crates; usar `--include-rust` solo cuando esa publicacion este
habilitada explicitamente.

## Consumers

Los repos consumidores deben depender de versiones publicadas:

- Go: `github.com/devpablocristo/platform/<path> vX.Y.Z`
- npm: `@devpablocristo/platform-*` con semver publicable
- Python: package publicado en PyPI

No commitear `replace` ni `file:` en consumers salvo que el repo lo documente
como excepcion temporal y no productiva.
