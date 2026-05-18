package domain

import "time"

const (
	StatusNew      = "new"
	StatusNotified = "notified"
	StatusResolved = "resolved"
)

type Candidate struct {
	ID              string
	TenantID        string
	Kind            string
	EventType       string
	EntityType      string
	EntityID        string
	Fingerprint     string
	Severity        string
	Status          string
	Title           string
	Body            string
	Evidence        map[string]any
	OccurrenceCount int
	FirstSeenAt     time.Time
	LastSeenAt      time.Time
	FirstNotifiedAt *time.Time
	LastNotifiedAt  *time.Time
	ResolvedAt      *time.Time
	LastActor       string
}

type UpsertInput struct {
	TenantID    string
	Kind        string
	EventType   string
	EntityType  string
	EntityID    string
	Fingerprint string
	Severity    string
	Title       string
	Body        string
	Evidence    map[string]any
	Actor       string
	Now         time.Time
}
