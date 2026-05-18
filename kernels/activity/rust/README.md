# activity/rust

Runtime Rust para el kernel reusable de `activity`.

## Alcance

Esta implementación agrega un runtime nuevo sin reemplazar `activity/go`.

Contextos incluidos:

- `audit`: append-only trail, hash chaining y export `csv/jsonl`
- `timeline`: registro y listado de eventos por entidad

## Arquitectura

Se usa hexagonal architecture porque el módulo es un kernel reusable:

- `src/domain/`: modelos y lógica determinista
- `src/application/`: casos de uso y puertos
- `src/infrastructure/`: adapters de reloj, IDs, repos in-memory y repos persistentes por filesystem en JSONL

## Hardening

- `FileSystemAuditRepository`: persistencia durable por tenant con append-only JSONL y detección de conflicto si la cadena hash quedó stale entre `last_hash` y `append`
- `FileSystemTimelineRepository`: persistencia durable para timeline con append seguro y recarga entre procesos

## Validación

```bash
cargo fmt --all
cargo clippy --all-targets -- -D warnings
cargo test
```
