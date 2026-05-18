// Package contracts contiene los DTOs canónicos del contrato AI/chat que
// hablan Companion y sus consumidores (pymes/frontend, nexus/console, etc).
// Cualquier app que consuma o produzca estos shapes debe importar este
// paquete en vez de redeclarar los tipos.
//
// El contrato es deliberadamente abierto/extensible: ChatBlock es una unión
// etiquetada con un campo Type para que se puedan sumar variantes (insight
// cards, KPIs, tablas, acciones) sin breaking change en el wire. Hoy solo
// "text" está activo.
package contracts

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ChatRequest es el body de POST /v1/chat.
type ChatRequest struct {
	Message          string       `json:"message"`
	ChatID           *uuid.UUID   `json:"chat_id,omitempty"`           // continúa conversación existente
	RouteHint        string       `json:"route_hint,omitempty"`        // hint opcional al router
	ConfirmedActions []string     `json:"confirmed_actions,omitempty"` // ids de acciones aprobadas por el usuario
	Handoff          *ChatHandoff `json:"handoff,omitempty"`           // metadata si vino de otro agente
}

// ChatHandoff describe un traspaso desde otro agente.
type ChatHandoff struct {
	Source      string `json:"source"`                 // identificador del agente origen
	TargetAgent string `json:"target_agent,omitempty"` // agente destino sugerido
}

// ChatResponse es el body de respuesta de POST /v1/chat.
type ChatResponse struct {
	ChatID               uuid.UUID                 `json:"chat_id"`
	Reply                string                    `json:"reply"`
	Blocks               []ChatBlock               `json:"blocks,omitempty"`
	ToolCalls            []ChatToolCall            `json:"tool_calls,omitempty"`
	PendingConfirmations []ChatPendingConfirmation `json:"pending_confirmations,omitempty"`
	TokensUsed           int                       `json:"tokens_used,omitempty"`
	RoutedAgent          string                    `json:"routed_agent,omitempty"`
	RoutingSource        string                    `json:"routing_source,omitempty"`
	OutputKind           string                    `json:"output_kind,omitempty"`
}

// ChatBlock es una unión etiquetada por Type. Solo "text" está activo; las
// otras variantes están reservadas y se irán implementando incrementalmente.
//
// Variants reservadas (no emitirlas todavía):
//   - "actions"      → bloques de acciones aprobables/clickables
//   - "insight_card" → tarjeta con insight + métricas
//   - "kpi_group"    → grupo de KPIs
//   - "table"        → tabla tabular
type ChatBlock struct {
	Type string `json:"type"` // "text" hoy; otras variantes futuras
	// Campos por variante (todos omitempty para que el wire solo emita los
	// relevantes a Type).
	Text string `json:"text,omitempty"` // variant "text"
}

// ChatToolCall registra una llamada a tool realizada por el agente durante
// el turno. Útil para auditar/explicar la respuesta.
type ChatToolCall struct {
	Name       string          `json:"name"`
	Args       json.RawMessage `json:"args,omitempty"`
	Result     json.RawMessage `json:"result,omitempty"`
	DurationMS int64           `json:"duration_ms,omitempty"`
	Error      string          `json:"error,omitempty"`
}

// ChatPendingConfirmation describe una acción pendiente de aprobación
// humana (típicamente origen Nexus governance).
type ChatPendingConfirmation struct {
	ID          string `json:"id"`
	Description string `json:"description,omitempty"`
	BindingHash string `json:"binding_hash,omitempty"`
}

// ConversationSummary resumen para listado.
type ConversationSummary struct {
	ID             uuid.UUID `json:"id"`
	Title          string    `json:"title,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
	CreatedAt      time.Time `json:"created_at"`
	MessageCount   int       `json:"message_count"`
	ProductSurface string    `json:"product_surface,omitempty"`
}

// ConversationDetail conversación completa con sus mensajes.
type ConversationDetail struct {
	ID        uuid.UUID             `json:"id"`
	Title     string                `json:"title,omitempty"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
	Messages  []ConversationMessage `json:"messages"`
}

// ConversationMessage un mensaje de la conversación.
type ConversationMessage struct {
	Role      string      `json:"role"` // "user" | "assistant" | "system" | "tool"
	Content   string      `json:"content"`
	Timestamp time.Time   `json:"timestamp,omitempty"`
	Blocks    []ChatBlock `json:"blocks,omitempty"`
}

// ConversationListResponse para GET /v1/chat/conversations.
type ConversationListResponse struct {
	Items      []ConversationSummary `json:"items"`
	NextCursor string                `json:"next_cursor,omitempty"`
}
