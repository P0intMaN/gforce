// Package models defines the core domain types for gforce.
package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a gforce platform user.
type User struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // never serialised
	DisplayName  *string   `json:"display_name"`
	AvatarURL    *string   `json:"avatar_url"`
	Bio          *string   `json:"bio"`
	IsAdmin      bool      `json:"is_admin"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Repository represents a git repository hosted on gforce.
type Repository struct {
	ID            uuid.UUID  `json:"id"`
	OwnerID       uuid.UUID  `json:"owner_id"`
	Name          string     `json:"name"`
	Description   *string    `json:"description"`
	IsPrivate     bool       `json:"is_private"`
	DefaultBranch string     `json:"default_branch"`
	DiskPath      string     `json:"disk_path"`
	ForkOf        *uuid.UUID `json:"fork_of"`
	StarCount     int        `json:"star_count"`
	ForkCount     int        `json:"fork_count"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// SSHKey represents a user's registered SSH public key.
type SSHKey struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	Title       string     `json:"title"`
	PublicKey   string     `json:"public_key"`
	Fingerprint string     `json:"fingerprint"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// LoginRequest is the payload for user authentication.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse carries the JWT issued on successful authentication.
type LoginResponse struct {
	Token    string    `json:"token"`
	IssuedAt time.Time `json:"issued_at"`
}

// Commit is a lightweight representation of a git commit for API responses.
type Commit struct {
	SHA       string    `json:"sha"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	Email     string    `json:"email"`
	Timestamp time.Time `json:"timestamp"`
}

// --- params -----------------------------------------------------------------

// CreateUserParams carries the fields required to create a new user.
type CreateUserParams struct {
	Username     string
	Email        string
	PasswordHash string  `json:"-"` // never serialised
	DisplayName  *string
	AvatarURL    *string
	Bio          *string
}

// UpdateUserParams carries the mutable fields for a user update.
// A nil pointer means "leave unchanged".
type UpdateUserParams struct {
	DisplayName *string
	AvatarURL   *string
	Bio         *string
}

// CreateRepoParams carries the fields required to create a new repository.
type CreateRepoParams struct {
	OwnerID       uuid.UUID
	Name          string
	Description   *string
	IsPrivate     bool
	DefaultBranch string
	DiskPath      string
	ForkOf        *uuid.UUID
}

// UpdateRepoParams carries the mutable fields for a repository update.
type UpdateRepoParams struct {
	Name          *string
	Description   *string
	IsPrivate     *bool
	DefaultBranch *string
}

// CreateSSHKeyParams carries the fields required to register an SSH key.
type CreateSSHKeyParams struct {
	UserID      uuid.UUID
	Title       string
	PublicKey   string
	Fingerprint string
}

// ActivityEvent records a user action for the activity feed.
type ActivityEvent struct {
	ID        uuid.UUID              `json:"id"`
	ActorID   uuid.UUID              `json:"actor_id"`
	EventType string                 `json:"event_type"`
	RepoID    *uuid.UUID             `json:"repo_id,omitempty"`
	Payload   map[string]interface{} `json:"payload"`
	CreatedAt time.Time              `json:"created_at"`
}
