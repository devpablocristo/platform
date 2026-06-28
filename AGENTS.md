# platform — Reglas para agentes

## Contexto

`platform` es el monorepo de paquetes reutilizables del ecosistema `pablo`.
Unifica los repos legacy `core/` y `modules/` bajo:

- Go: `github.com/devpablocristo/platform/<path>`
- TypeScript: `@devpablocristo/platform-*`
- Python: `devpablocristo-platform-*`
- Rust: crates `platform-*`

No es un repo de apps. No debe contener reglas de negocio de Axis, Medmory,
Pymes, Ponti ni otros productos.

## Lectura inicial

1. Leer este archivo.
2. Leer `README.md`.
3. Leer `GOVERNANCE.md`.
4. Para releases/versionado, leer `docs/platform/RELEASE_FLOW.md` y
   `docs/platform/VERSIONING.md`.
5. Si el trabajo toca migracion legacy, leer `docs/migration/`.

## Boundaries

- Cada paquete tiene ownership por path y runtime: `{capability}/{runtime}` o
  `{feature}/.../{runtime}`.
- No hay version global del repo para publicar paquetes; cada modulo mantiene
  su propio `VERSION`.
- No agregar dominio de producto en paquetes transversales.
- `features/` puede contener componentes o verticales reutilizables, pero no
  reglas privadas de una app.
- `sdks/` es para clientes/adapters a servicios externos.
- `contracts/` es para contratos compartidos cross-runtime.
- Los consumers deben depender de versiones publicadas; evitar `replace` o
  `file:` en repos de apps salvo durante iteracion local no commiteada.

## Estructura vigente

```text
platform/
  authn/ authz/ browser/ calendar/ concurrency/ config/
  contracts/ databases/ errors/ eventing/ http/ ingestion/
  jobs/ lifecycle/ notifications/ observability/ persistence/
  sdks/ security/ testing/ utils/ validate/ webhook/
  kernels/
  features/
  ui/
  docs/
  tooling/
```

El codigo real vive en subdirectorios por runtime (`go`, `ts`, `python`,
`rust`) y no directamente en la raiz de una capability, salvo documentos y
metadatos.

## Checks

Antes de cerrar cambios, correr lo mas acotado que cubra el riesgo:

```bash
npm run validate:boundaries
npm run validate:versions
npm run validate:ts-deps
npm run test:go
npm run test:ts
npm run test:python
npm run test:rust
```

Para una pasada completa:

```bash
npm run test:all
```

## Releases

- Bump de version por modulo con `tooling/scripts/bump-module-version.sh`.
- Tags por subdirectorio: `<module-path>/vX.Y.Z`.
- Workflows de publish deben fallar si fallan los tests del modulo taggeado.
- No publicar paquetes sin tag git correspondiente.

Ver `docs/platform/RELEASE_FLOW.md`.

## Docs legacy

`docs/core/` y `docs/modules/` son historicos de la migracion. No usarlos como
fuente operativa actual salvo que una tarea pida investigar el origen legacy.

## Estilo

- Codigo y APIs en ingles.
- Documentacion operativa en espanol cuando el repo ya lo usa.
- Mantener cambios chicos y verificables.
- No borrar codigo o docs legacy sin evidencia de no uso y sin dejar rastro de
  migracion cuando aplique.
