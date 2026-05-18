package domain

type ActorRef struct {
	Legacy string `json:"legacy,omitempty"`
	Type   string `json:"type,omitempty"`
	ID     string `json:"id,omitempty"`
	Label  string `json:"label,omitempty"`
}
