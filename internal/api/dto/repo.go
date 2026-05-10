package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreateRepoRequest is the body for POST /api/v1/user/repos.
type CreateRepoRequest struct {
	Name          string `json:"name"           validate:"required,min=1,max=100,slug"`
	Description   string `json:"description"    validate:"max=500"`
	IsPrivate     bool   `json:"is_private"`
	InitRepo      bool   `json:"init"`
	DefaultBranch string `json:"default_branch" validate:"omitempty,max=255"`
}

// UpdateRepoRequest is the body for PATCH /api/v1/repos/:owner/:repo.
type UpdateRepoRequest struct {
	Description   string `json:"description"    validate:"max=500"`
	IsPrivate     bool   `json:"is_private"`
	DefaultBranch string `json:"default_branch" validate:"omitempty,max=255"`
}

// RepoResponse is the public view of a repository.
// disk_path and fork_of are internal fields and must never appear here.
type RepoResponse struct {
	ID            uuid.UUID    `json:"id"`
	Name          string       `json:"name"`
	FullName      string       `json:"full_name"`
	Description   string       `json:"description"`
	IsPrivate     bool         `json:"is_private"`
	DefaultBranch string       `json:"default_branch"`
	CloneURL      string       `json:"clone_url"`
	StarCount     int          `json:"star_count"`
	ForkCount     int          `json:"fork_count"`
	Owner         UserResponse `json:"owner"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}
