// Package store defines the persistence interface for gforce domain objects.
package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/gforce/gforce/internal/models"
)

// Sentinel errors returned by store implementations.
var (
	// ErrNotFound is returned when the requested record does not exist.
	ErrNotFound = errors.New("record not found")
	// ErrConflict is returned when an insert violates a uniqueness constraint.
	ErrConflict = errors.New("record already exists")
)

// UserStore defines persistence operations for User entities.
type UserStore interface {
	// CreateUser persists a new user and returns the created record.
	CreateUser(ctx context.Context, params models.CreateUserParams) (*models.User, error)
	// GetUserByID returns the user with the given UUID, or ErrNotFound.
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	// GetUserByUsername returns the user with the given username, or ErrNotFound.
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	// GetUserByEmail returns the user with the given email address, or ErrNotFound.
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	// UpdateUser applies params to the user identified by id and returns the updated record.
	UpdateUser(ctx context.Context, id uuid.UUID, params models.UpdateUserParams) (*models.User, error)
	// ListUsers returns a paginated list of all users ordered by creation time descending.
	ListUsers(ctx context.Context, limit, offset int) ([]*models.User, error)
}

// RepoStore defines persistence operations for Repository entities.
type RepoStore interface {
	// CreateRepo persists a new repository and returns the created record.
	CreateRepo(ctx context.Context, params models.CreateRepoParams) (*models.Repository, error)
	// GetRepoByID returns the repository with the given UUID, or ErrNotFound.
	GetRepoByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	// GetRepoByOwnerAndName returns the repository owned by ownerID with the given name.
	GetRepoByOwnerAndName(ctx context.Context, ownerID uuid.UUID, name string) (*models.Repository, error)
	// ListReposByOwner returns a paginated list of repositories belonging to ownerID.
	ListReposByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*models.Repository, error)
	// ListPublicRepos returns a paginated list of all public repositories.
	ListPublicRepos(ctx context.Context, limit, offset int) ([]*models.Repository, error)
	// UpdateRepo applies params to the repository identified by id and returns the updated record.
	UpdateRepo(ctx context.Context, id uuid.UUID, params models.UpdateRepoParams) (*models.Repository, error)
	// DeleteRepo removes the repository with the given id.
	DeleteRepo(ctx context.Context, id uuid.UUID) error
	// IncrementStarCount atomically adds delta (positive or negative) to star_count.
	IncrementStarCount(ctx context.Context, id uuid.UUID, delta int) error
}

// SSHKeyStore defines persistence operations for SSH public keys.
type SSHKeyStore interface {
	// CreateSSHKey persists a new SSH key and returns the created record.
	CreateSSHKey(ctx context.Context, params models.CreateSSHKeyParams) (*models.SSHKey, error)
	// GetSSHKeyByFingerprint returns the SSH key with the given fingerprint, or ErrNotFound.
	GetSSHKeyByFingerprint(ctx context.Context, fingerprint string) (*models.SSHKey, error)
	// ListSSHKeysByUser returns all SSH keys registered to userID.
	ListSSHKeysByUser(ctx context.Context, userID uuid.UUID) ([]*models.SSHKey, error)
	// DeleteSSHKey removes the SSH key identified by id, scoped to userID for ownership enforcement.
	DeleteSSHKey(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}

// Store is the top-level dependency that composes all sub-stores and adds
// transaction support.
type Store interface {
	UserStore
	RepoStore
	SSHKeyStore

	// BeginTx starts a database transaction and returns a Store whose methods
	// run within that transaction. Commit or Rollback must be called on the
	// returned Store.
	BeginTx(ctx context.Context) (Store, error)
	// Commit commits the current transaction. Returns an error if called on a
	// non-transactional Store.
	Commit() error
	// Rollback aborts the current transaction. Returns an error if called on a
	// non-transactional Store.
	Rollback() error
	// Ping verifies the underlying connection is still alive.
	Ping(ctx context.Context) error
}
