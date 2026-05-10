package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gforce/gforce/internal/api/handlers"
	"github.com/gforce/gforce/internal/api/middleware"
	"github.com/gforce/gforce/internal/models"
	"github.com/gforce/gforce/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testBaseURL = "http://localhost:8080"

func newRepoHandler(t *testing.T, st store.Store) *handlers.RepoHandler {
	t.Helper()
	dir := t.TempDir()
	return handlers.NewRepoHandler(st, dir, testBaseURL, zap.NewNop())
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestCreateRepo_Unauthenticated(t *testing.T) {
	t.Parallel()

	h := newRepoHandler(t, &mockStore{})

	body := `{"name":"myrepo"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	// No user in context → RequireAuth would block, but we test handler directly
	w := httptest.NewRecorder()
	h.Create(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateRepo_Success(t *testing.T) {
	t.Parallel()

	owner := fixedUser()
	repoID := uuid.New()

	st := &mockStore{
		onCreateRepo: func(_ context.Context, p models.CreateRepoParams) (*models.Repository, error) {
			return &models.Repository{
				ID:            repoID,
				OwnerID:       owner.ID,
				Name:          p.Name,
				DefaultBranch: p.DefaultBranch,
				DiskPath:      p.DiskPath,
			}, nil
		},
		onDeleteRepo: func(_ context.Context, _ uuid.UUID) error { return nil },
	}

	h := newRepoHandler(t, st)

	body := `{"name":"myrepo","default_branch":"main"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req = withUser(req, owner)
	w := httptest.NewRecorder()
	h.Create(w, req)

	// InitBareRepo will run on the temp dir, which should succeed.
	// Check 201 and clone_url in body.
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "clone_url")
	assert.NotContains(t, w.Body.String(), "disk_path",
		"disk_path must never appear in API responses")
}

func TestCreateRepo_InvalidName_NotSlug(t *testing.T) {
	t.Parallel()

	h := newRepoHandler(t, &mockStore{})
	owner := fixedUser()

	// Uppercase name violates slug validation
	body := `{"name":"MyRepo"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req = withUser(req, owner)
	w := httptest.NewRecorder()
	h.Create(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateRepo_Duplicate(t *testing.T) {
	t.Parallel()

	owner := fixedUser()
	st := &mockStore{
		onCreateRepo: func(_ context.Context, _ models.CreateRepoParams) (*models.Repository, error) {
			return nil, store.ErrConflict
		},
	}
	h := newRepoHandler(t, st)

	body := `{"name":"existing-repo"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req = withUser(req, owner)
	w := httptest.NewRecorder()
	h.Create(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

// ── Get ───────────────────────────────────────────────────────────────────────

func TestGetRepo_Public_NoAuth(t *testing.T) {
	t.Parallel()

	owner := fixedUser()
	repo := fixedRepo(owner.ID)

	st := &mockStore{
		onGetUserByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return owner, nil
		},
		onGetRepoByOwnerAndName: func(_ context.Context, _ uuid.UUID, _ string) (*models.Repository, error) {
			return repo, nil
		},
	}
	h := newRepoHandler(t, st)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withURLParam(req, "owner", "alice")
	req = withURLParam(req, "repo", "myrepo")
	// No user in context — public repo should be accessible
	w := httptest.NewRecorder()
	h.Get(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "disk_path")
	assert.Contains(t, w.Body.String(), "clone_url")
}

func TestGetRepo_Private_NoAuth(t *testing.T) {
	t.Parallel()

	owner := fixedUser()
	repo := fixedRepo(owner.ID)
	repo.IsPrivate = true

	st := &mockStore{
		onGetUserByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return owner, nil
		},
		onGetRepoByOwnerAndName: func(_ context.Context, _ uuid.UUID, _ string) (*models.Repository, error) {
			return repo, nil
		},
	}
	h := newRepoHandler(t, st)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withURLParam(req, "owner", "alice")
	req = withURLParam(req, "repo", "private-repo")
	// No user in context — private repo should return 403
	w := httptest.NewRecorder()
	h.Get(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetRepo_Private_WithOwner(t *testing.T) {
	t.Parallel()

	owner := fixedUser()
	repo := fixedRepo(owner.ID)
	repo.IsPrivate = true

	st := &mockStore{
		onGetUserByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return owner, nil
		},
		onGetRepoByOwnerAndName: func(_ context.Context, _ uuid.UUID, _ string) (*models.Repository, error) {
			return repo, nil
		},
	}
	h := newRepoHandler(t, st)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withURLParam(req, "owner", "alice")
	req = withURLParam(req, "repo", "private-repo")
	req = withUser(req, owner) // owner is authenticated
	w := httptest.NewRecorder()
	h.Get(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetRepo_NotFound(t *testing.T) {
	t.Parallel()

	st := &mockStore{
		onGetUserByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return nil, store.ErrNotFound
		},
	}
	h := newRepoHandler(t, st)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withURLParam(req, "owner", "nobody")
	req = withURLParam(req, "repo", "ghost")
	w := httptest.NewRecorder()
	h.Get(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestDeleteRepo_NotOwner(t *testing.T) {
	t.Parallel()

	owner := fixedUser()
	other := &models.User{ID: uuid.New(), Username: "bob", IsActive: true}
	repo := fixedRepo(owner.ID)

	st := &mockStore{
		onGetUserByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return owner, nil
		},
		onGetRepoByOwnerAndName: func(_ context.Context, _ uuid.UUID, _ string) (*models.Repository, error) {
			return repo, nil
		},
	}
	h := newRepoHandler(t, st)

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = withURLParam(req, "owner", "alice")
	req = withURLParam(req, "repo", "myrepo")
	req = withUser(req, other) // bob is NOT the owner
	w := httptest.NewRecorder()
	h.Delete(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestDeleteRepo_Unauthenticated(t *testing.T) {
	t.Parallel()

	h := newRepoHandler(t, &mockStore{})
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = withURLParam(req, "owner", "alice")
	req = withURLParam(req, "repo", "myrepo")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ── middleware.ContextWithUser is used in tests ── keep the import live ───────

var _ = middleware.ContextWithUser

// ── assert store is used in tests ────────────────────────────────────────────

var _ store.Store = (*mockStore)(nil)

// ── assert helpers compile ────────────────────────────────────────────────────

func TestWithURLParam_Compiles(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withURLParam(req, "key", "val")
	require.NotNil(t, req)
}
