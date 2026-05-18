package models

import (
	"time"

	userdomain "github.com/devpablocristo/core/saas/go/users/usecases/domain"
)

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

func UserFromDomain(item userdomain.User) User {
	return User{
		ID:         item.ID,
		ExternalID: item.ExternalID,
		Email:      item.Email,
		Name:       item.Name,
		AvatarURL:  item.AvatarURL,
		CreatedAt:  item.CreatedAt,
		UpdatedAt:  item.UpdatedAt,
		DeletedAt:  item.DeletedAt,
	}
}

func (m User) ToDomain() userdomain.User {
	return userdomain.User{
		ID:         m.ID,
		ExternalID: m.ExternalID,
		Email:      m.Email,
		Name:       m.Name,
		AvatarURL:  m.AvatarURL,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
		DeletedAt:  m.DeletedAt,
	}
}

func MemberFromDomain(item userdomain.OrgMember) OrgMember {
	return OrgMember{
		ID:       item.ID,
		OrgID:    item.OrgID,
		UserID:   item.UserID,
		Role:     item.Role,
		JoinedAt: item.JoinedAt,
		User:     UserFromDomain(item.User),
	}
}

func (m OrgMember) ToDomain() userdomain.OrgMember {
	return userdomain.OrgMember{
		ID:       m.ID,
		OrgID:    m.OrgID,
		UserID:   m.UserID,
		Role:     m.Role,
		JoinedAt: m.JoinedAt,
		User:     m.User.ToDomain(),
	}
}
