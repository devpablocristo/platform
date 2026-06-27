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

Estado: **corregido en working tree**.

Evidencia: `.github/workflows/ci.yml` solo imprime que la matriz esta pendiente.

Riesgo: PRs de `platform` pueden mergearse sin validar boundaries, versions ni tests por lenguaje; Axis/Medmory despues consumen paquetes rotos.

Fix: el workflow principal ahora corre guardrails (`validate-runtime-layout`, boundaries, versions, TS deps) y tests separados para Go, TypeScript, Python y Rust.

### HIGH-02 — Guardrails de versionado/boundaries/test Go no estaban auditando el repo real

Estado: **corregido en working tree**.

Evidencia: `validate:versions` reportaba `no modules discovered`; `test:go` salia OK sin ejecutar paquetes; `validate:boundaries` leia bajo `tooling/` y mantenia mensajes/reglas de `core`.

Riesgo: falso verde en release de paquetes base.

Fix: raiz del repo corregida, modulo Go `testing/go/tenancy` incluido y paths esperados migrados a `github.com/devpablocristo/platform/...`.

### HIGH-03 — Tests AI capabilities apuntaban a contracts movidos

Estado: **corregido en working tree**.

Evidencia: `npm run test:python` y luego `npm run test:go` fallaban buscando `contracts/capabilities/v1` bajo `kernels/ai/runtime` o `kernels/ai`.

Riesgo: el contrato neutral de capabilities podia cambiar sin que kernels Go/Python lo validaran.

Fix: tests apuntan a `contracts/ai/capabilities/v1`.

### MED-01 — Workflows de publish permiten continuar si fallan tests

Estado: abierto.

Evidencia: `.github/workflows/publish-ts-package.yml` y `.github/workflows/publish-python-package.yml` tienen `continue-on-error: true` en el paso de test.

Riesgo: un tag puede publicar paquetes que no compilan o no pasan tests.

Fix recomendado: quitar `continue-on-error` despues de cerrar dependencias iniciales y adaptar tests; mientras tanto, documentar explicitamente que publicar manualmente requiere correr la matriz local.

### MED-02 — Docs operativas siguen en estado legacy `core/modules`

Estado: abierto.

Evidencia: `AGENTS.md`, `CONTRIBUTING.md`, `GOVERNANCE.md` y `docs/core`/`docs/modules` todavia describen reglas como si el repo fuera `core` + `modules`; `CONTRIBUTING.md` referencia `docs/RELEASE_FLOW.md`, que no existe en esa ubicacion.

Riesgo: agentes y humanos aplican reglas viejas, paths viejos o release flow equivocado.

Fix recomendado: crear docs canonicas `docs/platform/*` o `docs/RELEASE_FLOW.md`, y dejar `docs/core`/`docs/modules` como historico/migracion.

### MED-03 — Validacion TS interna estaba atada a `modules-*`

Estado: **corregido en working tree**.

Evidencia: `tooling/scripts/validate-internal-ts-deps.py` usaba `INTERNAL_SCOPE = "@devpablocristo/modules-"` y raiz `tooling/`.

Riesgo: no validaba los packages reales `@devpablocristo/platform-*`.

Fix: scope y raiz corregidos; se validan `file:` locales y rangos semver publicables.

### MED-04 — Crates Rust no podian correr por lib targets/deps invalidos

Estado: **corregido en working tree**.

Evidencia: `npm run test:rust` fallaba con `library target names cannot contain hyphens` y luego con path dependency `resilience` no encontrada.

Riesgo: los paquetes Rust existian en el monorepo pero no eran testeables/publicables.

Fix: `[lib] name` usa underscores o alias corto valido; path deps declaran el `package` real manteniendo el alias usado por codigo.

### LOW-01 — Test TS de scheduling dependia de timing ajustado

Estado: **corregido en working tree**.

Evidencia: `features/scheduling/ts/src/PublicSchedulingFlow.test.tsx` podia no encontrar el boton de slot mientras React Query seguia mostrando loading.

Riesgo: flake en `npm run test:ts`.

Fix: timeout explicito de 5s para waits de disponibilidad publica.

## Relacion con Axis y Medmory

Axis y Medmory ya consumen varios paquetes Go de `platform`. En esta pasada no hizo falta publicar un nuevo tag para consumidores porque los fixes fueron de tooling/tests/docs internos de `platform`, no de API publica. Si se mergean estos cambios, no requieren bump en Axis/Medmory.
