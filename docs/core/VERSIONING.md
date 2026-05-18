# Versionado independiente en `core`

`core` es un solo repo, pero no tiene una sola versión global.

La unidad de versionado es cada implementación concreta:

- `config/go`
- `concurrency/go`
- `errors/go`
- `http/go`
- `http/gin/go`
- `observability/go`
- `security/go`
- `utils/go`
- `validate/go`
- `authz/go`
- `browser/ts`
- `http/ts`
- `authn/go`
- `authn/rust`
- `notifications/go`
- `calendar/ics/go`
- `calendar/sync/google/go`
- `databases/postgres/go`
- `databases/postgres/rust`
- `databases/dynamodb/go`
- `providers/aws/lambda/go`
- `providers/aws/s3/go`
- `providers/aws/sqs/go`
- `eventing/go`
- `governance/go`
- `governance/rust`
- `artifact/go`
- `artifact/rust`
- `webhook/go`
- `activity/go`
- `activity/rust`
- `saas/go`
- `authn/ts`
- `http/python`
- `http/client/rust`
- `http/server/rust`
- `utils/pagination/rust`
- `utils/resilience/rust`
- `concurrency/fsm/rust`
- `concurrency/worker/rust`
- `errors/rust`
- `validate/rust`
- `observability/rust`
- `ai/go`
- `ai/python`

## Regla

Cada implementación tiene su propio archivo `VERSION` en la raíz del runtime:

```text
http/
  go/
    VERSION
    go.mod

ai/
  python/
    VERSION
    pyproject.toml

authn/
  ts/
    VERSION
    package.json
```

## Semántica

- usar `semver`
- empezar en `0.x` mientras la API siga moviéndose
- no inventar una versión global del repo
- solo se sube la versión del módulo que cambió

## Tags

Los tags se cortan por subdirectorio:

- `config/go/v0.1.0`
- `concurrency/go/v0.1.0`
- `errors/go/v0.1.0`
- `http/go/v0.1.0`
- `http/gin/go/v0.1.0`
- `observability/go/v0.1.0`
- `security/go/v0.1.0`
- `utils/go/v0.1.0`
- `validate/go/v0.1.0`
- `authz/go/v0.1.0`
- `browser/ts/v0.1.0`
- `http/ts/v0.1.0`
- `authn/go/v0.1.0`
- `authn/rust/v0.1.0`
- `notifications/go/v0.1.0`
- `calendar/ics/go/v0.1.0`
- `calendar/sync/google/go/v0.1.0`
- `databases/postgres/go/v0.1.0`
- `databases/postgres/rust/v0.1.0`
- `databases/dynamodb/go/v0.1.0`
- `providers/aws/lambda/go/v0.1.0`
- `providers/aws/s3/go/v0.1.0`
- `providers/aws/sqs/go/v0.1.0`
- `eventing/go/v0.1.0`
- `webhook/go/v0.1.0`
- `activity/go/v0.1.0`
- `activity/rust/v0.1.0`
- `governance/go/v0.1.0`
- `governance/rust/v0.1.0`
- `artifact/go/v0.1.0`
- `artifact/rust/v0.1.0`
- `saas/go/v0.1.0`
- `authn/ts/v0.1.0`
- `http/python/v0.1.0`
- `http/client/rust/v0.1.0`
- `http/server/rust/v0.1.0`
- `utils/pagination/rust/v0.1.0`
- `utils/resilience/rust/v0.1.0`
- `concurrency/fsm/rust/v0.1.0`
- `concurrency/worker/rust/v0.1.0`
- `errors/rust/v0.1.0`
- `validate/rust/v0.1.0`
- `observability/rust/v0.1.0`
- `ai/go/v0.1.0`
- `ai/python/v0.1.0`

Para Go esto sigue la convención correcta de módulos versionados en subdirectorios del monorepo.

## Fuente de verdad

- Go: `VERSION` es la fuente de verdad del release
- TypeScript: `VERSION` y `package.json` deben coincidir
- Python: `VERSION` y `pyproject.toml` deben coincidir
- Rust: `VERSION` y `Cargo.toml` deben coincidir cuando exista `rust/`

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

Subir `authz/go` a `0.2.0`:

```bash
./scripts/bump-module-version.sh authz/go 0.2.0
```
