package models

import "time"

type EntryModel struct {
	ID           string
	TenantID     string
	Actor        string
	ActorType    string
	ActorID      string
	ActorLabel   string
	Action       string
	ResourceType string
	ResourceID   string
	PayloadJSON  []byte
	PrevHash     string
	Hash         string
	CreatedAt    time.Time
}
