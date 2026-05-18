package contracts

import "encoding/json"

// NotificationsRequest es el body de POST /v1/notifications.
type NotificationsRequest struct {
	Kind              string `json:"kind,omitempty"`               // default "insight"
	Period            string `json:"period,omitempty"`             // "today" | "week" | "month"
	Compare           bool   `json:"compare,omitempty"`            // comparar con período previo
	TopLimit          int    `json:"top_limit,omitempty"`          // máximo de items
	PreferredLanguage string `json:"preferred_language,omitempty"` // "es" | "en"
}

// NotificationsResponse es la respuesta de POST /v1/notifications.
type NotificationsResponse struct {
	Items       []NotificationItem `json:"items"`
	ServiceKind string             `json:"service_kind,omitempty"`
	OutputKind  string             `json:"output_kind,omitempty"`
}

// NotificationItem es un insight/notification individual.
type NotificationItem struct {
	ID       string          `json:"id"`
	Title    string          `json:"title"`
	Body     string          `json:"body,omitempty"`
	Severity string          `json:"severity,omitempty"` // "info" | "warning" | "critical"
	Period   string          `json:"period,omitempty"`
	Scope    string          `json:"scope,omitempty"`
	Context  json.RawMessage `json:"context,omitempty"`
}
