# webhook

Capacidad reusable para outbound webhooks.

Implementación actual: `webhook/go/`

## Pertenece

- endpoints outbound
- firma HMAC
- headers estándar de delivery
- backoff y retry policy
- replay y planificación de deliveries

## No pertenece

- handlers o rutas de una app concreta
- persistencia específica de un producto
- workers atados a una plataforma concreta

## Fuente inicial esperada

- `/home/pablo/Projects/Pablo/pymes/pymes-core/backend/internal/outwebhooks`
