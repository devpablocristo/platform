# AUDIT.md — Auditoria de calidad Platform

Fecha: 2026-06-27. Alcance: segunda pasada sobre salud del monorepo, packages reutilizables, workflows, versionado y consumidores Axis/Medmory.

## Estado

Checks iniciales relevantes:
- `npm run validate:boundaries` fallaba por script con raiz en `tooling/` y reglas legacy `core/modules`.
- `npm run validate:versions` fallaba con `no modules discovered`.
- `npm run test:go` devolvia OK sin ejecutar modulos reales.
- `npm run test:python` fallaba en runtime AI porque los tests buscaban contracts en una ruta legacy.
- `npm run test:rust` fallaba por `[lib] name` con guiones y luego por path deps sin `package = ...`.
- `npm run test:ts` tuvo un fallo flaky en `PublicSchedulingFlow` por timeout demasiado corto esperando disponibilidad async.

Fixes aplicados en esta pasada:
- `.github/workflows/ci.yml` dejo de ser placeholder y ahora corre guardrails + tests Go/TS/Python/Rust en jobs separados.
- `authn/ts` ya no depende de tipos ambientales `vite/client` no declarados; el typecheck limpio de CI valida el modulo sin dependencias hoisted.
- Los workflows de publish TS/Python ahora fallan si falla el test del modulo taggeado; TS instala el workspace con `pnpm` para resolver dependencias internas sin depender de paquetes ya publicados.
- Docs operativas raiz reemplazadas por guias vigentes de `platform`; `docs/platform/` contiene release/versioning actuales y `docs/core`/`docs/modules` quedan como historico.
- Scripts de tooling ahora resuelven la raiz real del repo.
- `validate-module-versions`, `test-go`, `list-module-versions` y `check-remote-tags` descubren tambien `testing/go/tenancy`.
- `validate-boundaries` valida imports/manifests de codigo contra dependencias legacy `core/modules` sin fallar por docs de migracion.
- `validate-internal-ts-deps.py` usa scope `@devpablocristo/platform-*`, raiz real y permite semver publicable ademas de `file:`.
- Tests Go/Python de capabilities apuntan al contrato canonico `contracts/ai/capabilities/v1`.
- Rust mantiene package names publicables con guiones, pero usa lib target names validos y aliases cortos donde los tests externos importan `activity`, `artifact` y `governance`.
- `PublicSchedulingFlow.test.tsx` espera disponibilidad async con timeout explicito.

Verificacion despues de fixes:
- `npm run validate:boundaries` OK.
- `npm run validate:versions` OK, 70 modulos versionados.
- `bash tooling/scripts/validate-runtime-layout.sh` OK.
- `npm run validate:ts-deps` OK, 16 referencias internas validadas.
- `npm run test:python` OK, 1 + 71 tests.
- `npm run test:go` OK.
- `npm run test:rust` OK.
- `npm run test:ts` OK.

## Hallazgos

### HIGH-01 — CI principal es un placeholder

Estado: **corregido en PR #8**.

Evidencia: `.github/workflows/ci.yml` solo imprime que la matriz esta pendiente.

Riesgo: PRs de `platform` pueden mergearse sin validar boundaries, versions ni tests por lenguaje; Axis/Medmory despues consumen paquetes rotos.

Fix: el workflow principal ahora corre guardrails (`validate-runtime-layout`, boundaries, versions, TS deps) y tests separados para Go, TypeScript, Python y Rust.

### HIGH-02 — Guardrails de versionado/boundaries/test Go no estaban auditando el repo real

Estado: **corregido en PR #8**.

Evidencia: `validate:versions` reportaba `no modules discovered`; `test:go` salia OK sin ejecutar paquetes; `validate:boundaries` leia bajo `tooling/` y mantenia mensajes/reglas de `core`.

Riesgo: falso verde en release de paquetes base.

Fix: raiz del repo corregida, modulo Go `testing/go/tenancy` incluido y paths esperados migrados a `github.com/devpablocristo/platform/...`.

### HIGH-03 — Tests AI capabilities apuntaban a contracts movidos

Estado: **corregido en PR #8**.

Evidencia: `npm run test:python` y luego `npm run test:go` fallaban buscando `contracts/capabilities/v1` bajo `kernels/ai/runtime` o `kernels/ai`.

Riesgo: el contrato neutral de capabilities podia cambiar sin que kernels Go/Python lo validaran.

Fix: tests apuntan a `contracts/ai/capabilities/v1`.

### MED-01 — Workflows de publish permiten continuar si fallan tests

Estado: **corregido en PR #8**.

Evidencia: `.github/workflows/publish-ts-package.yml` y `.github/workflows/publish-python-package.yml` tienen `continue-on-error: true` en el paso de test.

Riesgo: un tag puede publicar paquetes que no compilan o no pasan tests.

Fix: ambos workflows quedaron fail-fast. En TS, el test del modulo taggeado corre via `pnpm --filter` despues de instalar el workspace, asi las dependencias internas se resuelven localmente antes de preparar/publicar el paquete.

### MED-02 — Docs operativas siguen en estado legacy `core/modules`

Estado: **corregido en PR #8**.

Evidencia: `AGENTS.md`, `CONTRIBUTING.md`, `GOVERNANCE.md` y `docs/core`/`docs/modules` todavia describen reglas como si el repo fuera `core` + `modules`; `CONTRIBUTING.md` referencia `docs/RELEASE_FLOW.md`, que no existe en esa ubicacion.

Riesgo: agentes y humanos aplican reglas viejas, paths viejos o release flow equivocado.

Fix: `AGENTS.md`, `CONTRIBUTING.md` y `GOVERNANCE.md` ahora describen `platform`; `docs/platform/RELEASE_FLOW.md` y `docs/platform/VERSIONING.md` son la fuente operativa; docs legacy tienen README/punteros historicos.

### MED-03 — Validacion TS interna estaba atada a `modules-*`

Estado: **corregido en PR #8**.

Evidencia: `tooling/scripts/validate-internal-ts-deps.py` usaba `INTERNAL_SCOPE = "@devpablocristo/modules-"` y raiz `tooling/`.

Riesgo: no validaba los packages reales `@devpablocristo/platform-*`.

Fix: scope y raiz corregidos; se validan `file:` locales y rangos semver publicables.

### MED-04 — Crates Rust no podian correr por lib targets/deps invalidos

Estado: **corregido en PR #8**.

Evidencia: `npm run test:rust` fallaba con `library target names cannot contain hyphens` y luego con path dependency `resilience` no encontrada.

Riesgo: los paquetes Rust existian en el monorepo pero no eran testeables/publicables.

Fix: `[lib] name` usa underscores o alias corto valido; path deps declaran el `package` real manteniendo el alias usado por codigo.

### LOW-01 — Test TS de scheduling dependia de timing ajustado

Estado: **corregido en PR #8**.

Evidencia: `features/scheduling/ts/src/PublicSchedulingFlow.test.tsx` podia no encontrar el boton de slot mientras React Query seguia mostrando loading.

Riesgo: flake en `npm run test:ts`.

Fix: timeout explicito de 5s para waits de disponibilidad publica.

## Relacion con Axis y Medmory

Axis y Medmory ya consumen varios paquetes Go de `platform`. En esta pasada no hizo falta publicar un nuevo tag para consumidores porque los fixes fueron de tooling/tests/docs internos de `platform`, no de API publica. Si se mergean estos cambios, no requieren bump en Axis/Medmory.

## Follow-up — 2026-06-28

### HIGH-2P-01 — `platform-chat-ui` publicado y consumido no entraba al workspace ni al publish workflow

Estado: **corregido en PR #9**.

Evidencia: Axis y Medmory consumen `@devpablocristo/platform-chat-ui@^0.1.2`; el paquete vive en `features/chat/ui/ts`, pero `pnpm-workspace.yaml` no lo incluia y `.github/workflows/publish-ts-package.yml` no escuchaba tags `features/chat/ui/ts/v*`.

Riesgo: `npm run test:ts` no typecheckeaba/testeaba un paquete publicado y usado por apps; futuros tags del paquete no dispararian el workflow de publish.

Fix: agregar `features/chat/ui/ts` al workspace pnpm y al workflow de publish TS.

### MED-2P-02 — `check-remote-tags` exigia tags Rust aunque Rust no esta publicado

Estado: **corregido en PR #10**.

Evidencia: despues de taggear los paquetes TS publicados, `bash tooling/scripts/check-remote-tags.sh` seguia fallando por 14 tags Rust `v0.1.0` ausentes. La documentacion vigente indica que Rust todavia requiere decision explicita de registry/naming antes de publicarse.

Riesgo: el guardrail de tags mezclaba modulos publicables hoy con crates no publicables todavia, generando falso rojo e incentivando tags prematuros de Rust.

Fix: `check-remote-tags.sh` valida por defecto Go/TypeScript/Python y expone `--include-rust` para el modo estricto cuando Rust publishing quede habilitado. `docs/platform/VERSIONING.md` documenta la distincion.

### MED-2P-03 — Codigo/docs publicables conservaban referencias legacy `core/modules`

Estado: **corregido en PR #11**.

Evidencia: `ui/page-shell/ts/src/styles.css` importaba `@devpablocristo/modules-shell-sidebar/styles.css`; `features/scheduling/ts/src/styles.css` importaba `@devpablocristo/modules-calendar-board/styles.css`; varios READMEs de paquetes Go/TS mostraban imports `github.com/devpablocristo/core/...` o `@devpablocristo/core-*`. `docker-compose.yml` y Dockerfiles CI locales apuntaban a paths `modules/...` y paquetes ya inexistentes.

Riesgo: consumers que instalaran solo paquetes `platform-*` podian resolver mal CSS compartido, y la documentacion de paquetes publicables seguia guiando hacia repos legacy.

Fix: actualizar imports CSS y READMEs a nombres `platform`, corregir compose/Dockerfiles locales a paths reales, remover servicios legacy inexistentes y ampliar `validate-boundaries` para cubrir CSS, READMEs de paquete y Dockerfiles.

## Follow-up — 2026-06-29

### 2P-VERIFY-04 — Tags, tests y consumidores revalidados tras la segunda pasada

Estado: **verificado sin cambios de codigo requeridos**.

Evidencia:
- `bash tooling/scripts/check-remote-tags.sh` OK; todos los tags esperados para modulos publicables estan presentes en `origin`.
- `npm run validate:boundaries` OK.
- `npm run validate:versions` OK, 70 modulos versionados.
- `npm run validate:ts-deps` OK, 16 referencias internas validadas.
- `npm run test:go` OK.
- `npm run test:ts` OK.
- `npm run test:python` OK, 1 + 71 tests; queda solo warning upstream de Starlette/httpx.
- `npm run test:rust` OK.
- Busqueda de consumidores en Axis/Medmory: ambos consumen paquetes `github.com/devpablocristo/platform/...` y `@devpablocristo/platform-*` por version publicada; no se detectaron `replace`/`file:` locales accidentalmente commiteados hacia `platform`.

Conclusion: la segunda pasada de `platform` no deja HIGH/MED abiertos confirmados. Las referencias legacy restantes fuera de `AUDIT.md` estan acotadas a tooling de migracion/deprecacion (`rename-imports.py`, `deprecate-legacy-npm.sh`) y a docs historicos excluidos por guardrails. No hace falta publicar nuevos tags ni hacer bumps en Axis/Medmory porque no hubo cambio de API publica.
