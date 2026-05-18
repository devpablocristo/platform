# ai/python

Implementación Python de la capacidad `ai`.

## Paquetes

- `src/runtime/`: implementación actual organizada para FastAPI/Python

## Cobertura funcional

- orchestrator
- provider factory
- JSON completion clients
- conversation memory
- operational memory consolidation
- LLM settings helpers
- auth
- rate limit
- logging
- contexts
- resilience
- FastAPI helpers

## Estado

`ai-core` ya fue recreado dentro de esta implementación.

El trabajo pendiente fuera de este directorio ya no es reconstrucción interna sino migración de consumidores y retiro del repo viejo.

## Release 0.5.0

Incluye capa reusable de memoria operativa:

- facts del negocio
- preferencias por usuario
- open loops
- decisiones recientes
- vista curada para prompts
