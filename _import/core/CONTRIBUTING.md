# Contribuir a `core`

Antes de abrir PR, leer [GOVERNANCE.md](./GOVERNANCE.md).

## Estructura

```
core/
├── ai/{go,ts,python}/         # contratos + clientes LLM
├── authn/{go,ts}/             # auth primitivas
├── http/{go,ts}/              # HTTP server/client
├── databases/{postgres,dynamodb}/go/
├── providers/aws/{lambda,s3,sqs}/go/
├── ...
└── <area>/<lang>/             # módulo individual con su go.mod o package.json
```

Cada módulo es un Go module (`go.mod`) o package npm (`package.json`)
independiente. La convención de path es `<area>/[<sub>/]<lang>/`.

## Proponer un nuevo módulo en `core`

Checklist:

1. **Verificar la regla de los 3** (ver GOVERNANCE §2): ¿hay ≥3 consumidores reales o planificados?
2. **No es lógica de producto**: si vive en una app de dominio, no pertenece a core. Considerá `modules/` si es UI/dominio acotado.
3. **No duplica algo existente**: revisar inventario actual.
4. Abrir PR con:
   - Carpeta `<path>/` con go.mod o package.json
   - `VERSION` file (empezar en `0.1.0`)
   - `README.md` mínimo (qué es, cómo se usa, consumers actuales)
   - Tests (ver §Tests)
   - Si querés que arranque como experimental sin tag estable: README explícito + sin tag (ejemplo: `activity/go`).

## Tests

- **Go**: tests al lado del archivo (`*_test.go`); cobertura no obligatoria pero recomendada en lógica crítica.
- **TS**: vitest; `tests/` o `*.test.ts` colocados.
- CI corre tests en cada PR (`.github/workflows/`).

## Release

Ver [`docs/RELEASE_FLOW.md`](./docs/RELEASE_FLOW.md).

TL;DR:
1. Bump `VERSION` (o `package.json` version).
2. PR + merge.
3. Workflow auto-publica a npm/Go-proxy + auto-tag.
4. Verificar tag en GitHub + presencia en registry.

## Naming

- Go: `github.com/devpablocristo/core/<area>/<sub>/go` — paths kebab-case
- TS: `@devpablocristo/core-<area>` — packages kebab-case, sin sub-paths
  (ej. `@devpablocristo/core-fsm` no `@devpablocristo/core-concurrency-fsm`)
- Funciones exported: camelCase (TS) / PascalCase (Go)

## Breaking changes

- Major bump obligatorio.
- ADR escrito.
- Coordinar con consumers (ver lista de imports en cada README).

## Preguntas / discusión

Issues en este repo, taggear con `discussion` o `proposal`.
