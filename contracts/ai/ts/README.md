# @devpablocristo/core-ai-contracts

DTOs canónicos del contrato AI/chat para Companion y sus consumidores
(`pymes/frontend`, `nexus/console`, etc).

Cualquier app que produzca o consuma estos shapes (response de
`POST /v1/chat`, listado de conversations, notifications) debe importar
este paquete en vez de redeclarar los tipos. Mantiene una sola fuente de
verdad y permite evolucionar el contrato con bumps semver coordinados.

## Tipos principales

- `ChatRequest` / `ChatResponse` — `POST /v1/chat`.
- `ChatBlock` — unión etiquetada por `type`. Hoy solo `'text'` está activo.
  Variantes reservadas para iteraciones siguientes: `'actions'`,
  `'insight_card'`, `'kpi_group'`, `'table'`.
- `ChatToolCall`, `ChatPendingConfirmation`, `ChatHandoff` — auxiliares del
  response.
- `ConversationSummary`, `ConversationDetail`, `ConversationMessage`,
  `ConversationListResponse` — endpoints `GET /v1/chat/conversations(/{id})`.
- `NotificationsRequest`, `NotificationsResponse`, `NotificationItem` —
  `POST /v1/notifications`.

## Versionado

`v0.1.0` — primera publicación con `ChatBlock` text-only. Sumar variantes
nuevas a la unión `ChatBlock` no es breaking porque amplia el tipo;
remover variantes existentes sí lo es. Tag git: `ai/contracts/ts/v0.1.0`.

Versión Go paralela: `github.com/devpablocristo/core/ai/contracts/go`.
