# notifications

Capacidad reusable para notificaciones por canal y para la bandeja in-app de avisos no-chat.

Implementación actual: `notifications/go/`

## Pertenece

- contratos de email sender
- adaptadores `noop`, `smtp` y `ses`
- config reusable por env
- bootstrap reusable de senders
- inbox/bandeja in-app por tenant y destinatario (`go/inbox`)
- lifecycle reusable de insight/event candidates persistidos (`go/candidates`)
- metadata reusable para handoffs o contexto de UI

## No pertenece

- preferencias de notificación de una app
- plantillas/copy de producto
- colas, workers o handlers específicos de un producto
- approvals de un producto o de governance
- UIs de producto

## Fuente inicial esperada

- `/home/pablo/Projects/Pablo/core/saas/go/notifications` como puerto de dominio, no como implementación
- `/home/pablo/Projects/Pablo/pymes/pymes-core/backend/internal/notifications/sender.go`
