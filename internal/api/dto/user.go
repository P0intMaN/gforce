// Package dto defines the Data Transfer Objects for the GForce API.
// Handlers must use DTOs — domain models are never written directly to responses.
package dto

import (
	"time"

	"github.com/google/uuid"
)

// RegisterRequest is the body for POST /api/v1/auth/register.
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=39,slug"`
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// LoginRequest accepts a username or email in the Login field.
type LoginRequest struct {
	Login    string `json:"login"    validate:"required"`
	Password string `json:"password" validate:"required"`
}

// TokenResponse carries the issued JWT and its expiry.
type TokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// UserResponse is the public view of a user. Never includes password_hash.
type UserResponse struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url"`
	Bio         string    `json:"bio"`
	IsAdmin     bool      `json:"is_admin"`
	CreatedAt   time.Time `json:"created_at"`
}

// UpdateProfileRequest is the body for PATCH /api/v1/user.
type UpdateProfileRequest struct {
	DisplayName string `json:"display_name" validate:"max=100"`
	Bio         string `json:"bio"          validate:"max=500"`
	AvatarURL   string `json:"avatar_url"   validate:"omitempty,url"`
}

// AddSSHKeyRequest is the body for POST /api/v1/user/keys.
type AddSSHKeyRequest struct {
	Title     string `json:"title"      validate:"required,max=255"`
	PublicKey string `json:"public_key" validate:"required"`
}

// SSHKeyResponse is the public view of an SSH key.
type SSHKeyResponse struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Fingerprint string     `json:"fingerprint"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedAt   time.Time  `json:"created_at"`
}
