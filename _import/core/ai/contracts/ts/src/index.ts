// DTOs canónicos del contrato AI/chat para Companion y sus consumidores.
// Espejo TypeScript de github.com/devpablocristo/core/ai/contracts/go.
// Cualquier app que produzca o consuma estos shapes debe importar este
// paquete en vez de redeclarar los tipos.

// ────────────────────────────────────────────────────────────────
// POST /v1/chat
// ────────────────────────────────────────────────────────────────

/** ChatBlock es una unión etiquetada por type. Hoy solo "text" está activo.
 *  Variantes reservadas para iteraciones siguientes: "actions",
 *  "insight_card", "kpi_group", "table". */
export type ChatBlock = ChatTextBlock; // unión extensible (sumar variantes acá)

export interface ChatTextBlock {
  type: 'text';
  text: string;
}

export interface ChatHandoff {
  source: string;
  target_agent?: string;
}

export interface ChatToolCall {
  name: string;
  args?: unknown;
  result?: unknown;
  duration_ms?: number;
  error?: string;
}

export interface ChatPendingConfirmation {
  id: string;
  description?: string;
  binding_hash?: string;
}

export interface ChatRequest {
  message: string;
  chat_id?: string;
  route_hint?: string;
  confirmed_actions?: string[];
  handoff?: ChatHandoff;
}

export interface ChatResponse {
  chat_id: string;
  reply: string;
  blocks?: ChatBlock[];
  tool_calls?: ChatToolCall[];
  pending_confirmations?: ChatPendingConfirmation[];
  tokens_used?: number;
  routed_agent?: string;
  routing_source?: string;
  output_kind?: string;
}

// ────────────────────────────────────────────────────────────────
// GET /v1/chat/conversations(/{id})
// ────────────────────────────────────────────────────────────────

export interface ConversationSummary {
  id: string;
  title?: string;
  created_at: string;
  updated_at: string;
  message_count: number;
  product_surface?: string;
}

export interface ConversationMessage {
  role: 'user' | 'assistant' | 'system' | 'tool';
  content: string;
  timestamp?: string;
  blocks?: ChatBlock[];
}

export interface ConversationDetail {
  id: string;
  title?: string;
  created_at: string;
  updated_at: string;
  messages: ConversationMessage[];
}

export interface ConversationListResponse {
  items: ConversationSummary[];
  next_cursor?: string;
}

// ────────────────────────────────────────────────────────────────
// POST /v1/notifications
// ────────────────────────────────────────────────────────────────

export interface NotificationsRequest {
  kind?: 'insight' | string;
  period?: 'today' | 'week' | 'month';
  compare?: boolean;
  top_limit?: number;
  preferred_language?: 'es' | 'en' | string;
}

export interface NotificationItem {
  id: string;
  title: string;
  body?: string;
  severity?: 'info' | 'warning' | 'critical' | string;
  period?: string;
  scope?: string;
  context?: unknown;
}

export interface NotificationsResponse {
  items: NotificationItem[];
  service_kind?: string;
  output_kind?: string;
}
