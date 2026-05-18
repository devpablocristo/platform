# governance/rust

Runtime Rust para el kernel determinista de `governance`.

## Alcance

Esta implementación cubre solo las piezas que sí ganan con Rust:

- `risk`: scoring determinista y CPU-bound
- `decision`: composición de evaluación y fallback por riesgo
- `approval`: reglas de quorum y break-glass
- `evidence`: armado y firmado de evidence packs

La evaluación de policies sigue modelada como puerto. Hoy el runtime incluye dos adapters:

- `CelPolicyMatcher`: adapter más sólido, con CEL real para expresiones
- `DeterministicPolicyMatcher`: adapter mínimo útil para tests y bootstrap controlado

## Arquitectura

Se usa hexagonal architecture porque este runtime es un kernel reusable y no una aplicación:

- `src/domain/`: modelos, reglas puras y value objects
- `src/application/`: casos de uso y puertos
- `src/infrastructure/`: adapters concretos de policy matching y signing

## Principios

- lógica determinista en `domain/`
- dependencias hacia adentro, nunca desde dominio hacia adapters
- serialización estable con `BTreeMap` para evidence reproducible
- adapters reemplazables para policy evaluation y signing

## Validación

Checks esperados para este módulo:

```bash
cargo fmt --all
cargo clippy --all-targets -- -D warnings
cargo test
```
