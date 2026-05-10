// Package store defines the persistence interface for gforce domain objects.
package store

import (
	"context"

	"github.com/gforce/gforce/internal/models"
)

// Store is the top-level dependency that groups all repository interfaces.
type Store interface {
	Users() UserStore
	Repos() RepoStore
	SSHKeys() SSHKeyStore
	Close() error
}

// UserStore defines persistence operations for User entities.
type UserStore interface {
	// Create persists a new user and returns it with the generated ID populated.
	Create(ctx context.Context, u *models.User) error
	// GetByID returns the user with the given UUID, or ErrNotFound.
	GetByID(ctx context.Context, id string) (*models.User, error)
	// GetByUsername returns the user with the given username, or ErrNotFound.
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	// GetByEmail returns the user with the given email address, or ErrNotFound.
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	// Update persists changes to an existing user record.
	Update(ctx context.Context, u *models.User) error
	// Delete removes the user with the given ID.
	Delete(ctx context.Context, id string) error
}

// RepoStore defines persistence operations for Repository entities.
type RepoStore interface {
	// Create persists a new repository and returns it with the generated ID populated.
	Create(ctx context.Context, r *models.Repository) error
	// GetByID returns the repository with the given UUID, or ErrNotFound.
	GetByID(ctx context.Context, id string) (*models.Repository, error)
	// GetByOwnerAndName returns the repo owned by ownerID with the given name.
	GetByOwnerAndName(ctx context.Context, ownerID, name string) (*models.Repository, error)
	// ListByOwner returns all repositories belonging to ownerID.
	ListByOwner(ctx context.Context, ownerID string) ([]*models.Repository, error)
	// Update persists changes to an existing repository record.
	Update(ctx context.Context, r *models.Repository) error
	// Delete removes the repository with the given ID.
	Delete(ctx context.Context, id string) error
}

// SSHKeyStore defines persistence operations for SSH public keys.
type SSHKeyStore interface {
	// Create persists a new SSH key and returns it with the generated ID populated.
	Create(ctx context.Context, k *models.SSHKey) error
	// ListByUser returns all SSH keys registered to userID.
	ListByUser(ctx context.Context, userID string) ([]*models.SSHKey, error)
	// Delete removes the SSH key with the given ID.
	Delete(ctx context.Context, id string) error
}

// ErrNotFound is returned when a requested entity does not exist in the store.
var ErrNotFound = &notFoundError{}

type notFoundError struct{}

func (e *notFoundError) Error() string { return "record not found" }

// IsNotFound reports whether err is or wraps ErrNotFound.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*notFoundError)
	return ok
}
