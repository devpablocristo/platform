package domain

import "time"

type User struct {
	ID         string     `json:"id"`
	ExternalID string     `json:"external_id"`
	Email      string     `json:"email"`
	Name       string     `json:"name"`
	AvatarURL  *string    `json:"avatar_url,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}

type OrgMember struct {
	ID       string    `json:"id"`
	OrgID    string    `json:"org_id"`
	UserID   string    `json:"user_id"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
	User     User      `json:"user"`
}

type APIKey struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	Name      string    `json:"name"`
	Scopes    []string  `json:"scopes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type CreatedAPIKey struct {
	APIKey APIKey `json:"api_key"`
	Secret string `json:"secret"`
}

type RotatedAPIKey struct {
	APIKey APIKey `json:"api_key"`
	Secret string `json:"secret"`
}

type MeProfile struct {
	OrgID      string   `json:"org_id"`
	ExternalID string   `json:"external_id"`
	Role       string   `json:"role"`
	Scopes     []string `json:"scopes,omitempty"`
	User       *User    `json:"user,omitempty"`
}
