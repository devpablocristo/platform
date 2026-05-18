# scripts

Acá viven scripts transversales del monorepo `modules`.

## Uso esperado

- validación de versionado por módulo
- chequeos de publicación por tag
- testing transversal del repo

## Scripts actuales

- `validate-module-versions.sh`: valida `VERSION`, semver y consistencia con manifests por módulo
- `validate-internal-ts-deps.py`: valida que dependencias TS internas `modules -> modules` usen `file:` local y no registry
- `list-module-versions.sh`: lista módulos versionados y el tag esperado
- `check-remote-tags.sh`: compara los tags remotos publicados contra los tags esperados por `VERSION`
- `bump-module-version.sh`: sube la versión de un módulo concreto
- `test-go-modules.sh`: corre `go test ./...` en cada módulo Go
- `test-ts-modules.sh`: instala el workspace TS desde la raíz y corre `typecheck` y `test` en cada módulo TypeScript
- `test-all.sh`: ejecuta validación de versionado y toda la suite del repo
- `prepare-ts-package-release.py`: arma un paquete TS publicable sin romper los `file:` locales del workspace
