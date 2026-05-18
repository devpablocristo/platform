package models

import "time"

type EndpointModel struct {
	ID        string
	TenantID  string
	URL       string
	Secret    string
	Events    []string
	IsActive  bool
	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type DeliveryModel struct {
	ID           string
	EndpointID   string
	EventType    string
	PayloadJSON  []byte
	StatusCode   *int
	ResponseBody string
	Attempts     int
	NextRetry    *time.Time
	DeliveredAt  *time.Time
	CreatedAt    time.Time
}
