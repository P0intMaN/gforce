package handlers_test

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/gforce/gforce/internal/models"
	"github.com/gforce/gforce/internal/store"
)

// mockStore implements store.Store with optional function fields.
// Unset methods return zero values / ErrNotFound.
type mockStore struct {
	onCreateUser            func(context.Context, models.CreateUserParams) (*models.User, error)
	onGetUserByID           func(context.Context, uuid.UUID) (*models.User, error)
	onGetUserByUsername     func(context.Context, string) (*models.User, error)
	onGetUserByEmail        func(context.Context, string) (*models.User, error)
	onUpdateUser            func(context.Context, uuid.UUID, models.UpdateUserParams) (*models.User, error)
	onListUsers             func(context.Context, int, int) ([]*models.User, error)
	onCreateRepo            func(context.Context, models.CreateRepoParams) (*models.Repository, error)
	onGetRepoByID           func(context.Context, uuid.UUID) (*models.Repository, error)
	onGetRepoByOwnerAndName func(context.Context, uuid.UUID, string) (*models.Repository, error)
	onListReposByOwner      func(context.Context, uuid.UUID, int, int) ([]*models.Repository, error)
	onListPublicRepos       func(context.Context, int, int) ([]*models.Repository, error)
	onListPublicReposByOwner func(context.Context, uuid.UUID, int, int) ([]*models.Repository, error)
	onUpdateRepo            func(context.Context, uuid.UUID, models.UpdateRepoParams) (*models.Repository, error)
	onDeleteRepo            func(context.Context, uuid.UUID) error
	onIncrementStarCount    func(context.Context, uuid.UUID, int) error
	onCreateSSHKey          func(context.Context, models.CreateSSHKeyParams) (*models.SSHKey, error)
	onGetSSHKeyByFingerprint func(context.Context, string) (*models.SSHKey, error)
	onListSSHKeysByUser     func(context.Context, uuid.UUID) ([]*models.SSHKey, error)
	onDeleteSSHKey          func(context.Context, uuid.UUID, uuid.UUID) error
}

func (m *mockStore) CreateUser(ctx context.Context, p models.CreateUserParams) (*models.User, error) {
	if m.onCreateUser != nil {
		return m.onCreateUser(ctx, p)
	}
	return nil, store.ErrNotFound
}
func (m *mockStore) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	if m.onGetUserByID != nil {
		return m.onGetUserByID(ctx, id)
	}
	return nil, store.ErrNotFound
}
func (m *mockStore) GetUserByUsername(ctx context.Context, u string) (*models.User, error) {
	if m.onGetUserByUsername != nil {
		return m.onGetUserByUsername(ctx, u)
	}
	return nil, store.ErrNotFound
}
func (m *mockStore) GetUserByEmail(ctx context.Context, e string) (*models.User, error) {
	if m.onGetUserByEmail != nil {
		return m.onGetUserByEmail(ctx, e)
	}
	return nil, store.ErrNotFound
}
func (m *mockStore) UpdateUser(ctx context.Context, id uuid.UUID, p models.UpdateUserParams) (*models.User, error) {
	if m.onUpdateUser != nil {
		return m.onUpdateUser(ctx, id, p)
	}
	return nil, store.ErrNotFound
}
func (m *mockStore) ListUsers(ctx context.Context, l, o int) ([]*models.User, error) {
	if m.onListUsers != nil {
		return m.onListUsers(ctx, l, o)
	}
	return nil, nil
}
func (m *mockStore) CreateRepo(ctx context.Context, p models.CreateRepoParams) (*models.Repository, error) {
	if m.onCreateRepo != nil {
		return m.onCreateRepo(ctx, p)
	}
	return nil, store.ErrNotFound
}
func (m *mockStore) GetRepoByID(ctx context.Context, id uuid.UUID) (*models.Repository, error) {
	if m.onGetRepoByID != nil {
		return m.onGetRepoByID(ctx, id)
	}
	return nil, store.ErrNotFound
}
func (m *mockStore) GetRepoByOwnerAndName(ctx context.Context, ownerID uuid.UUID, name string) (*models.Repository, error) {
	if m.onGetRepoByOwnerAndName != nil {
		return m.onGetRepoByOwnerAndName(ctx, ownerID, name)
	}
	return nil, store.ErrNotFound
}
func (m *mockStore) ListReposByOwner(ctx context.Context, id uuid.UUID, l, o int) ([]*models.Repository, error) {
	if m.onListReposByOwner != nil {
		return m.onListReposByOwner(ctx, id, l, o)
	}
	return nil, nil
}
func (m *mockStore) ListPublicRepos(ctx context.Context, l, o int) ([]*models.Repository, error) {
	if m.onListPublicRepos != nil {
		return m.onListPublicRepos(ctx, l, o)
	}
	return nil, nil
}
func (m *mockStore) ListPublicReposByOwner(ctx context.Context, id uuid.UUID, l, o int) ([]*models.Repository, error) {
	if m.onListPublicReposByOwner != nil {
		return m.onListPublicReposByOwner(ctx, id, l, o)
	}
	return nil, nil
}
func (m *mockStore) UpdateRepo(ctx context.Context, id uuid.UUID, p models.UpdateRepoParams) (*models.Repository, error) {
	if m.onUpdateRepo != nil {
		return m.onUpdateRepo(ctx, id, p)
	}
	return nil, store.ErrNotFound
}
func (m *mockStore) DeleteRepo(ctx context.Context, id uuid.UUID) error {
	if m.onDeleteRepo != nil {
		return m.onDeleteRepo(ctx, id)
	}
	return nil
}
func (m *mockStore) IncrementStarCount(ctx context.Context, id uuid.UUID, d int) error {
	if m.onIncrementStarCount != nil {
		return m.onIncrementStarCount(ctx, id, d)
	}
	return nil
}
func (m *mockStore) CreateSSHKey(ctx context.Context, p models.CreateSSHKeyParams) (*models.SSHKey, error) {
	if m.onCreateSSHKey != nil {
		return m.onCreateSSHKey(ctx, p)
	}
	return nil, nil
}
func (m *mockStore) GetSSHKeyByFingerprint(ctx context.Context, f string) (*models.SSHKey, error) {
	if m.onGetSSHKeyByFingerprint != nil {
		return m.onGetSSHKeyByFingerprint(ctx, f)
	}
	return nil, store.ErrNotFound
}
func (m *mockStore) ListSSHKeysByUser(ctx context.Context, id uuid.UUID) ([]*models.SSHKey, error) {
	if m.onListSSHKeysByUser != nil {
		return m.onListSSHKeysByUser(ctx, id)
	}
	return nil, nil
}
func (m *mockStore) DeleteSSHKey(ctx context.Context, id, uid uuid.UUID) error {
	if m.onDeleteSSHKey != nil {
		return m.onDeleteSSHKey(ctx, id, uid)
	}
	return nil
}
func (m *mockStore) BeginTx(_ context.Context) (store.Store, error) { return m, nil }
func (m *mockStore) Commit() error                                    { return nil }
func (m *mockStore) Rollback() error                                  { return nil }
func (m *mockStore) Ping(_ context.Context) error                     { return nil }
func (m *mockStore) RecordEvent(_ context.Context, _ store.RecordEventParams) error {
	return nil
}
func (m *mockStore) ListUserActivity(_ context.Context, _ uuid.UUID, _ int) ([]*models.ActivityEvent, error) {
	return nil, nil
}

// --- test fixtures ---

func fixedUser() *models.User {
	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	return &models.User{
		ID:        id,
		Username:  "alice",
		Email:     "alice@example.com",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func fixedRepo(ownerID uuid.UUID) *models.Repository {
	return &models.Repository{
		ID:            uuid.New(),
		OwnerID:       ownerID,
		Name:          "myrepo",
		IsPrivate:     false,
		DefaultBranch: "main",
		DiskPath:      "/tmp/repos/alice/myrepo.git",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}
