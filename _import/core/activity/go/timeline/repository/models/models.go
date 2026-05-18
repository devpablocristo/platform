package models

import "time"

type EntryModel struct {
	ID           string
	TenantID     string
	EntityType   string
	EntityID     string
	EventType    string
	Title        string
	Description  string
	Actor        string
	MetadataJSON []byte
	CreatedAt    time.Time
}
