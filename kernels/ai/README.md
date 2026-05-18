# ai

Capacidad reusable para runtime AI, providers y orquestación.

## Pertenece

- orchestration
- provider factory
- auth/rate limit AI
- runtime helpers
- contexts
- resilience/circuit breaker
- adapters AI reutilizables

## No pertenece

- prompts específicos de negocio
- tools específicas de una app
- contratos cerrados a un solo producto

## Fuente inicial esperada

- `/home/pablo/Projects/Pablo/ai-core`

## Estructura actual

Implementación actual: `ai/python/`

La implementación ya sigue una estructura Python/FastAPI estándar:

- `src/runtime/domain/`
- `src/runtime/providers/`
- `src/runtime/services/`
- `src/runtime/registry/`
- `src/runtime/config/`
- `src/runtime/api/`
- middleware reusable de auth, rate limit y request context
- `create_app()` para wiring FastAPI estándar
- namespace único `runtime` para imports compartidos
- `tests/`

## Estado

`/home/pablo/Projects/Pablo/ai-core` ya quedó absorbido funcionalmente dentro de `ai/python/`.

Lo que sigue ya no es reconstrucción interna sino migración de consumidores y retiro del repo viejo.
