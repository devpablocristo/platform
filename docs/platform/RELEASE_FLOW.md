# Release Flow de `platform`

Cada release publica un modulo concreto, no todo el monorepo.

## Preflight

Desde `main` actualizado:

```bash
git checkout main
git pull --ff-only origin main
git status --short
npm run validate:boundaries
npm run validate:versions
npm run validate:ts-deps
```

Ejecutar tambien los tests del runtime tocado.

## Bump

```bash
bash tooling/scripts/bump-module-version.sh <module-path> <version>
```

Ejemplos:

```bash
bash tooling/scripts/bump-module-version.sh http/go 0.2.0
bash tooling/scripts/bump-module-version.sh authn/ts 0.1.1
```

Abrir PR con el cambio, esperar CI verde y mergear a `main`.

## Go

1. Confirmar `VERSION`.
2. Crear tag `<module-path>/vX.Y.Z`.
3. Pushear el tag.
4. Verificar el proxy.

```bash
git tag http/go/v0.2.0 -m "http/go: v0.2.0"
git push origin http/go/v0.2.0
GOPROXY=https://proxy.golang.org go list -m github.com/devpablocristo/platform/http/go@v0.2.0
```

Para la publicacion inicial masiva existe:

```bash
bash tooling/scripts/publish-go-tags.sh --dry-run
bash tooling/scripts/publish-go-tags.sh
```

## TypeScript

1. Confirmar `VERSION` y `package.json.version`.
2. Crear tag `<module-path>/vX.Y.Z`.
3. Pushear el tag.
4. El workflow `publish-ts-package` corre typecheck/test del modulo y publica.
5. Verificar npm.

```bash
git tag authn/ts/v0.1.1 -m "authn/ts: v0.1.1"
git push origin authn/ts/v0.1.1
npm view @devpablocristo/platform-authn version
```

El workflow instala el workspace con `pnpm` y ejecuta:

```bash
pnpm --filter <package-name> run typecheck
pnpm --filter <package-name> run test
```

## Python

1. Confirmar `VERSION` y `pyproject.toml`.
2. Crear tag `<module-path>/vX.Y.Z`.
3. Pushear el tag.
4. El workflow `publish-python-package` corre tests, build, `twine check` y
   publish.
5. Verificar PyPI.

```bash
git tag http/python/v0.1.1 -m "http/python: v0.1.1"
git push origin http/python/v0.1.1
pip index versions devpablocristo-platform-http
```

## Rust

Rust se valida en CI con `npm run test:rust`. Publicar crates requiere decision
explicita de registry/naming antes de automatizar.

## Post-release

- Verificar registry/proxy.
- Bump consumers en PRs separados.
- No usar pseudo-versions Go en consumers.
- No hacer unpublish. Ante error, publicar patch correctivo.
- Si se publica un fix reusable para Axis/Medmory, el orden es:
  `platform` -> tag -> consumer PR -> CI/smoke del consumer.
