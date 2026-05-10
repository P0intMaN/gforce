package gitserver_test

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/gforce/gforce/internal/auth"
	"github.com/gforce/gforce/internal/models"
	"github.com/gforce/gforce/internal/store"
	. "github.com/gforce/gforce/internal/gitserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

// testValidator implements auth.TokenValidator with a pluggable function.
type testValidator struct {
	validate func(token string) (*auth.Claims, error)
}

func (v *testValidator) Validate(token string) (*auth.Claims, error) {
	return v.validate(token)
}

// noopValidator accepts every token as belonging to a fixed user.
type noopValidator struct {
	claims *auth.Claims
}

func (v *noopValidator) Validate(_ string) (*auth.Claims, error) {
	return v.claims, nil
}

// testStore is a hand-written mock of store.Store.
// Each method field is optional; methods fall back to sensible defaults.
type testStore struct {
	onGetUserByUsername    func(ctx context.Context, username string) (*models.User, error)
	onGetUserByID          func(ctx context.Context, id uuid.UUID) (*models.User, error)
	onGetRepoByOwnerAndName func(ctx context.Context, ownerID uuid.UUID, name string) (*models.Repository, error)
}

// GetUserByUsername delegates to onGetUserByUsername or returns ErrNotFound.
func (s *testStore) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	if s.onGetUserByUsername != nil {
		return s.onGetUserByUsername(ctx, username)
	}
	return nil, store.ErrNotFound
}

// GetUserByID delegates to onGetUserByID or returns ErrNotFound.
func (s *testStore) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	if s.onGetUserByID != nil {
		return s.onGetUserByID(ctx, id)
	}
	return nil, store.ErrNotFound
}

// GetRepoByOwnerAndName delegates to onGetRepoByOwnerAndName or returns ErrNotFound.
func (s *testStore) GetRepoByOwnerAndName(ctx context.Context, ownerID uuid.UUID, name string) (*models.Repository, error) {
	if s.onGetRepoByOwnerAndName != nil {
		return s.onGetRepoByOwnerAndName(ctx, ownerID, name)
	}
	return nil, store.ErrNotFound
}

// --- Unused store.Store methods: all return zero-values so the mock compiles. ---

func (s *testStore) CreateUser(_ context.Context, _ models.CreateUserParams) (*models.User, error) {
	return nil, nil
}
func (s *testStore) GetUserByEmail(_ context.Context, _ string) (*models.User, error) {
	return nil, store.ErrNotFound
}
func (s *testStore) UpdateUser(_ context.Context, _ uuid.UUID, _ models.UpdateUserParams) (*models.User, error) {
	return nil, nil
}
func (s *testStore) ListUsers(_ context.Context, _, _ int) ([]*models.User, error) {
	return nil, nil
}
func (s *testStore) CreateRepo(_ context.Context, _ models.CreateRepoParams) (*models.Repository, error) {
	return nil, nil
}
func (s *testStore) GetRepoByID(_ context.Context, _ uuid.UUID) (*models.Repository, error) {
	return nil, store.ErrNotFound
}
func (s *testStore) ListReposByOwner(_ context.Context, _ uuid.UUID, _, _ int) ([]*models.Repository, error) {
	return nil, nil
}
func (s *testStore) ListPublicRepos(_ context.Context, _, _ int) ([]*models.Repository, error) {
	return nil, nil
}
func (s *testStore) ListPublicReposByOwner(_ context.Context, _ uuid.UUID, _, _ int) ([]*models.Repository, error) {
	return nil, nil
}
func (s *testStore) UpdateRepo(_ context.Context, _ uuid.UUID, _ models.UpdateRepoParams) (*models.Repository, error) {
	return nil, nil
}
func (s *testStore) DeleteRepo(_ context.Context, _ uuid.UUID) error { return nil }
func (s *testStore) IncrementStarCount(_ context.Context, _ uuid.UUID, _ int) error { return nil }
func (s *testStore) CreateSSHKey(_ context.Context, _ models.CreateSSHKeyParams) (*models.SSHKey, error) {
	return nil, nil
}
func (s *testStore) GetSSHKeyByFingerprint(_ context.Context, _ string) (*models.SSHKey, error) {
	return nil, store.ErrNotFound
}
func (s *testStore) ListSSHKeysByUser(_ context.Context, _ uuid.UUID) ([]*models.SSHKey, error) {
	return nil, nil
}
func (s *testStore) DeleteSSHKey(_ context.Context, _, _ uuid.UUID) error { return nil }
func (s *testStore) BeginTx(_ context.Context) (store.Store, error)       { return nil, nil }
func (s *testStore) Commit() error                                         { return nil }
func (s *testStore) Rollback() error                                       { return nil }
func (s *testStore) Ping(_ context.Context) error                          { return nil }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTestRepo creates a temporary bare git repository and returns its path.
func newTestRepo(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "repo.git")
	require.NoError(t, InitBareRepo(path))
	return path
}

// newFixedUser returns a deterministic user and UUID for tests.
func newFixedUser() (uuid.UUID, *models.User) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	return id, &models.User{ID: id, Username: "alice", Email: "alice@example.com", IsActive: true}
}

// ---------------------------------------------------------------------------
// TestInfoRefs_PublicRepo_NoAuth
// ---------------------------------------------------------------------------

func TestInfoRefs_PublicRepo_NoAuth(t *testing.T) {
	t.Parallel()

	repoPath := newTestRepo(t)
	userID, owner := newFixedUser()

	st := &testStore{
		onGetUserByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return owner, nil
		},
		onGetRepoByOwnerAndName: func(_ context.Context, _ uuid.UUID, _ string) (*models.Repository, error) {
			return &models.Repository{
				ID:            uuid.New(),
				OwnerID:       userID,
				IsPrivate:     false,
				DiskPath:      repoPath,
				DefaultBranch: "main",
			}, nil
		},
	}

	h := NewGitHandler(st, &noopValidator{}, t.TempDir(), zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/alice/myrepo.git/info/refs?service=git-upload-pack", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/x-git-upload-pack-advertisement", w.Header().Get("Content-Type"))
}

// ---------------------------------------------------------------------------
// TestInfoRefs_PrivateRepo_NoAuth
// ---------------------------------------------------------------------------

func TestInfoRefs_PrivateRepo_NoAuth(t *testing.T) {
	t.Parallel()

	userID, owner := newFixedUser()

	st := &testStore{
		onGetUserByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return owner, nil
		},
		onGetRepoByOwnerAndName: func(_ context.Context, _ uuid.UUID, _ string) (*models.Repository, error) {
			return &models.Repository{
				ID:        uuid.New(),
				OwnerID:   userID,
				IsPrivate: true,
				DiskPath:  "/nonexistent",
			}, nil
		},
	}

	h := NewGitHandler(st, &noopValidator{}, t.TempDir(), zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/alice/private.git/info/refs?service=git-upload-pack", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Header().Get("WWW-Authenticate"), "Basic realm=")
}

// ---------------------------------------------------------------------------
// TestInfoRefs_PrivateRepo_WithAuth
// ---------------------------------------------------------------------------

func TestInfoRefs_PrivateRepo_WithAuth(t *testing.T) {
	t.Parallel()

	repoPath := newTestRepo(t)
	userID, owner := newFixedUser()

	tv := &testValidator{
		validate: func(_ string) (*auth.Claims, error) {
			return &auth.Claims{UserID: userID.String(), Username: "alice"}, nil
		},
	}

	st := &testStore{
		onGetUserByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return owner, nil
		},
		onGetRepoByOwnerAndName: func(_ context.Context, _ uuid.UUID, _ string) (*models.Repository, error) {
			return &models.Repository{
				ID:        uuid.New(),
				OwnerID:   userID,
				IsPrivate: true,
				DiskPath:  repoPath,
			}, nil
		},
		onGetUserByID: func(_ context.Context, _ uuid.UUID) (*models.User, error) {
			return owner, nil
		},
	}

	h := NewGitHandler(st, tv, t.TempDir(), zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/alice/private.git/info/refs?service=git-upload-pack", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	// Auth passed — response is 200 (git ran against temp bare repo).
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// TestReceivePack_RequiresAuth
// ---------------------------------------------------------------------------

func TestReceivePack_RequiresAuth(t *testing.T) {
	t.Parallel()

	userID, owner := newFixedUser()

	st := &testStore{
		onGetUserByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return owner, nil
		},
		onGetRepoByOwnerAndName: func(_ context.Context, _ uuid.UUID, _ string) (*models.Repository, error) {
			return &models.Repository{
				ID:       uuid.New(),
				OwnerID:  userID,
				DiskPath: "/nonexistent",
			}, nil
		},
	}

	h := NewGitHandler(st, &noopValidator{}, t.TempDir(), zap.NewNop())

	// POST with no Authorization header.
	req := httptest.NewRequest(http.MethodPost, "/alice/myrepo.git/git-receive-pack", http.NoBody)
	req.Header.Set("Content-Type", "application/x-git-receive-pack-request")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------------------------------------------------------------------------
// TestResolveRepo_NotFound
// ---------------------------------------------------------------------------

func TestResolveRepo_NotFound(t *testing.T) {
	t.Parallel()

	st := &testStore{
		onGetUserByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return nil, store.ErrNotFound
		},
	}

	h := NewGitHandler(st, &noopValidator{}, t.TempDir(), zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/nobody/ghost.git/info/refs?service=git-upload-pack", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// TestAuthenticateBearer
// ---------------------------------------------------------------------------

func TestAuthenticateBearer(t *testing.T) {
	t.Parallel()

	repoPath := newTestRepo(t)
	userID, owner := newFixedUser()
	const bearerToken = "eyJhbGciOiJIUzI1NiJ9.test"

	tv := &testValidator{
		validate: func(tok string) (*auth.Claims, error) {
			if tok == bearerToken {
				return &auth.Claims{UserID: userID.String(), Username: "alice"}, nil
			}
			return nil, assert.AnError
		},
	}

	st := &testStore{
		onGetUserByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return owner, nil
		},
		onGetRepoByOwnerAndName: func(_ context.Context, _ uuid.UUID, _ string) (*models.Repository, error) {
			return &models.Repository{
				ID: uuid.New(), OwnerID: userID,
				IsPrivate: true, DiskPath: repoPath,
			}, nil
		},
		onGetUserByID: func(_ context.Context, _ uuid.UUID) (*models.User, error) {
			return owner, nil
		},
	}

	h := NewGitHandler(st, tv, t.TempDir(), zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/alice/repo.git/info/refs?service=git-upload-pack", nil)
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	// Valid bearer token → authenticated → NOT 401.
	assert.NotEqual(t, http.StatusUnauthorized, w.Code, "bearer auth should succeed")
}

// ---------------------------------------------------------------------------
// TestAuthenticateBasic
// ---------------------------------------------------------------------------

func TestAuthenticateBasic(t *testing.T) {
	t.Parallel()

	repoPath := newTestRepo(t)
	userID, owner := newFixedUser()
	const tokenInPassword = "eyJhbGciOiJIUzI1NiJ9.test"

	tv := &testValidator{
		validate: func(tok string) (*auth.Claims, error) {
			if tok == tokenInPassword {
				return &auth.Claims{UserID: userID.String(), Username: "alice"}, nil
			}
			return nil, assert.AnError
		},
	}

	st := &testStore{
		onGetUserByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return owner, nil
		},
		onGetRepoByOwnerAndName: func(_ context.Context, _ uuid.UUID, _ string) (*models.Repository, error) {
			return &models.Repository{
				ID: uuid.New(), OwnerID: userID,
				IsPrivate: true, DiskPath: repoPath,
			}, nil
		},
		onGetUserByID: func(_ context.Context, _ uuid.UUID) (*models.User, error) {
			return owner, nil
		},
	}

	h := NewGitHandler(st, tv, t.TempDir(), zap.NewNop())

	// Basic auth: username=alice, password=<jwt token>
	creds := base64.StdEncoding.EncodeToString([]byte("alice:" + tokenInPassword))
	req := httptest.NewRequest(http.MethodGet, "/alice/repo.git/info/refs?service=git-upload-pack", nil)
	req.Header.Set("Authorization", "Basic "+creds)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	// Valid basic auth → authenticated → NOT 401.
	assert.NotEqual(t, http.StatusUnauthorized, w.Code, "basic auth should succeed")
}

// ---------------------------------------------------------------------------
// TestUnknownPath
// ---------------------------------------------------------------------------

func TestUnknownPath(t *testing.T) {
	t.Parallel()
	h := NewGitHandler(&testStore{}, &noopValidator{}, t.TempDir(), zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/not-a-git-path", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// TestInvalidService
// ---------------------------------------------------------------------------

func TestInvalidService(t *testing.T) {
	t.Parallel()
	h := NewGitHandler(&testStore{}, &noopValidator{}, t.TempDir(), zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/alice/repo.git/info/refs?service=git-evil", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
