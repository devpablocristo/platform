package domain

import "time"

type Endpoint struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	URL       string    `json:"url"`
	Secret    string    `json:"secret,omitempty"`
	Events    []string  `json:"events"`
	IsActive  bool      `json:"is_active"`
	CreatedBy string    `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Delivery struct {
	ID           string         `json:"id"`
	EndpointID   string         `json:"endpoint_id"`
	EventType    string         `json:"event_type"`
	Payload      map[string]any `json:"payload"`
	StatusCode   *int           `json:"status_code,omitempty"`
	ResponseBody string         `json:"response_body,omitempty"`
	Attempts     int            `json:"attempts"`
	NextRetry    *time.Time     `json:"next_retry,omitempty"`
	DeliveredAt  *time.Time     `json:"delivered_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

type DispatchRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Body    []byte            `json:"body"`
	Headers map[string]string `json:"headers"`
}
