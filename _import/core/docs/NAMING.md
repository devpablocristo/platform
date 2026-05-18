# Core — Convención de naming

## Regla

```
responsabilidad[/subresponsabilidad]*/lenguaje/
```

- **Responsabilidad**: dominio funcional (`ai`, `authn`, `databases`, `http`, etc.)
- **Subresponsabilidad**: subdominio cuando aplica (`databases/postgres`, `providers/aws/lambda`)
- **Lenguaje**: siempre el **último directorio** del path (`go`, `python`, `ts`, `rust`)
- Después del lenguaje solo hay implementación (`src/`, `internal/`, `go.mod`, `pyproject.toml`, etc.)

## Ejemplos válidos

```
ai/python/                        # responsabilidad/lenguaje
ai/go/                            # responsabilidad/lenguaje
authn/ts/                         # responsabilidad/lenguaje
authn/go/                         # responsabilidad/lenguaje
databases/postgres/go/            # responsabilidad/sub/lenguaje
databases/postgres/rust/          # responsabilidad/sub/lenguaje
providers/aws/lambda/go/          # responsabilidad/sub/sub/lenguaje
http/go/                          # responsabilidad/lenguaje
governance/go/                    # responsabilidad/lenguaje
```

## Prohibido

```
databases/gorm/go/                ❌ gorm es implementación, no responsabilidad
ai/python/src/core_ai/            ❌ nombre de paquete repite el path (core_ redundante)
responsabilidad/go/sub/python/    ❌ lenguaje en el medio, no al final
```

## Nombres de paquetes internos

Dentro del directorio de lenguaje, el paquete se nombra por su función sin repetir el path:

| Path | Paquete | Import |
|------|---------|--------|
| `ai/python/src/runtime/` | `runtime` | `from runtime import ...` |
| `http/python/src/httpserver/` | `devpablocristo-httpserver` | `from httpserver import ...` |
| `ai/go/` | `package ai` | `import "github.com/.../ai/go"` |
| `authn/ts/src/` | `@devpablocristo/core-authn` | `import { } from '@devpablocristo/core-authn'` |

## Subpaquetes dentro de un lenguaje

Cuando una responsabilidad tiene múltiples primitivas en el mismo lenguaje, se organizan como subpaquetes internos:

```
ai/python/src/
  runtime/          # providers, orchestrator, types, auth
  memory/           # conversations, context window
```

No como directorios hermanos de lenguaje:
```
ai/runtime/python/  ❌ no crear un nivel extra por cada primitiva
ai/memory/python/   ❌
```
