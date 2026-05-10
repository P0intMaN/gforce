// Package models defines the core domain types for gforce.
package models

import "time"

// User represents a gforce platform user.
type User struct {
	ID           string    `db:"id"            json:"id"`
	Username     string    `db:"username"      json:"username"`
	Email        string    `db:"email"         json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	CreatedAt    time.Time `db:"created_at"    json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"    json:"updated_at"`
}

// Repository represents a git repository hosted on gforce.
type Repository struct {
	ID            string    `db:"id"             json:"id"`
	OwnerID       string    `db:"owner_id"       json:"owner_id"`
	Name          string    `db:"name"           json:"name"`
	Description   string    `db:"description"    json:"description"`
	IsPrivate     bool      `db:"is_private"     json:"is_private"`
	DefaultBranch string    `db:"default_branch" json:"default_branch"`
	DiskPath      string    `db:"disk_path"      json:"disk_path"`
	CreatedAt     time.Time `db:"created_at"     json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"     json:"updated_at"`
}

// Commit is a lightweight representation of a git commit for API responses.
type Commit struct {
	SHA       string    `json:"sha"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	Email     string    `json:"email"`
	Timestamp time.Time `json:"timestamp"`
}

// SSHKey represents a user's registered SSH public key.
type SSHKey struct {
	ID          string    `db:"id"           json:"id"`
	UserID      string    `db:"user_id"      json:"user_id"`
	Title       string    `db:"title"        json:"title"`
	PublicKey   string    `db:"public_key"   json:"public_key"`
	Fingerprint string    `db:"fingerprint"  json:"fingerprint"`
	CreatedAt   time.Time `db:"created_at"   json:"created_at"`
}

// CreateUserRequest is the payload for creating a new user.
type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// CreateRepoRequest is the payload for creating a new repository.
type CreateRepoRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	IsPrivate     bool   `json:"is_private"`
	DefaultBranch string `json:"default_branch"`
}

// LoginRequest is the payload for user authentication.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse carries the JWT issued on successful authentication.
type LoginResponse struct {
	Token string `json:"token"`
}
