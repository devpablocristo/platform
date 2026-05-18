# Contribuir a `modules`

Antes de abrir PR, leer [GOVERNANCE.md en core](../core/GOVERNANCE.md).

## Estructura

```
modules/
├── ui/<name>/ts/              # componentes React reusables
├── crud/{ui,paths,archive}/   # CRUD genérico (UI + backend)
├── scheduling/{go,ts}/
├── kanban/board/ts/
├── search/ts/
├── ...
└── <area>/[<sub>/]<lang>/     # paquete individual con package.json o go.mod
```

Convención: `<area>/[<sub>/]<lang>/` igual que `core`.

## Proponer un nuevo módulo en `modules`

Checklist:

1. **Regla de los 3** (GOVERNANCE §2): ≥3 consumidores reales o planificados.
2. **Dominio acotado**: el módulo encapsula UN concepto (un componente UI, un comportamiento de scheduling, etc.). Si abarca varios, splitealo.
3. **No es primitiva técnica**: si es HTTP/auth/observability puro, va a `core` no a `modules`.
4. **No es lógica de producto**: si es específico de pymes/companion/etc., NO va a modules.

Estructura mínima de un nuevo TS package:

```
modules/<area>/ts/
├── package.json          # name: @devpablocristo/modules-<area>
├── tsconfig.json
├── VERSION               # 0.1.0
├── README.md             # qué es, cómo se usa, consumers
├── src/
│   ├── index.ts
│   ├── <Component>.tsx
│   └── styles.css        # si aplica
└── tests/
    └── <Component>.test.tsx
```

## Peer dependencies

Componentes React deben declarar:

```json
"peerDependencies": {
  "react": "^18.0.0 || ^19.0.0",
  "react-dom": "^18.0.0 || ^19.0.0"
}
```

(Política `^18 || ^19` hasta que React 20 salga.)

Si el módulo depende de otro `@devpablocristo/modules-*`, declararlo en `dependencies` (no peerDeps) con `^X.Y.Z`.

## CSS / Estilos

- Exportar como `./styles.css` en `exports` del package.json:
  ```json
  "exports": {
    ".": "./src/index.ts",
    "./styles.css": "./src/styles.css"
  }
  ```
- Consumers importan en su entrypoint: `import '@devpablocristo/modules-<area>/styles.css'`.
- No usar CSS-in-JS pesado; tailwind classes en JSX es OK.

## Tests

- vitest para unitarios.
- @testing-library/react para componentes.
- e2e queda en las apps consumidoras.

## Release

Ver [`docs/RELEASE_FLOW.md`](./docs/RELEASE_FLOW.md).

## Breaking changes

Major bump. Coordinar con consumers (pymes/frontend es el principal hoy).

## Deprecación

Si un módulo pierde consumidores y queda 90 días sin uso:

1. Marcar como deprecated en npm: `npm deprecate '@devpablocristo/modules-<name>@*' '<razón>'`.
2. Mover README a "DEPRECATED" header.
3. Después de 180d sin reactivación: archivar el directorio.

Ejemplo: `modules-{ai-console,ui-filters,ui-forms,work-orders}` deprecados 2026-05-15.

## Preguntas / discusión

Issues en este repo, taggear con `discussion` o `proposal`.
