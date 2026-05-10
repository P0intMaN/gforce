package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gforce/gforce/internal/api/handlers"
	"github.com/gforce/gforce/internal/api/middleware"
	"github.com/gforce/gforce/internal/auth"
	"github.com/gforce/gforce/internal/models"
	"github.com/gforce/gforce/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func newAuthSvc(t *testing.T) *auth.Service {
	t.Helper()
	svc, err := auth.NewService("test-secret-for-handlers", 24*time.Hour)
	require.NoError(t, err)
	return svc
}

// withURLParam injects a chi URL parameter into the request context.
func withURLParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// withUser injects an authenticated user into the request context (simulates RequireAuth).
func withUser(r *http.Request, u *models.User) *http.Request {
	return r.WithContext(middleware.ContextWithUser(r.Context(), u))
}

// jsonBody creates a JSON body for a request.
func jsonBody(t *testing.T, v any) *strings.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return strings.NewReader(string(b))
}

// ── Register ─────────────────────────────────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	t.Parallel()

	user := fixedUser()
	st := &mockStore{
		onCreateUser: func(_ context.Context, _ models.CreateUserParams) (*models.User, error) {
			return user, nil
		},
	}
	h := handlers.NewUserHandler(st, newAuthSvc(t), zap.NewNop())

	body := `{"username":"alice","email":"alice@example.com","password":"securepass"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Register(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.NotContains(t, w.Body.String(), "password_hash",
		"password_hash must never appear in API responses")

	var env map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Contains(t, env, "data")

	var data map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(env["data"], &data))
	assert.Equal(t, `"alice"`, string(data["username"]))
}

func TestRegister_DuplicateUsername(t *testing.T) {
	t.Parallel()

	st := &mockStore{
		onCreateUser: func(_ context.Context, _ models.CreateUserParams) (*models.User, error) {
			return nil, store.ErrConflict
		},
	}
	h := handlers.NewUserHandler(st, newAuthSvc(t), zap.NewNop())

	body := `{"username":"alice","email":"alice@example.com","password":"securepass"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.Register(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestRegister_InvalidUsername_Slug(t *testing.T) {
	t.Parallel()

	h := handlers.NewUserHandler(&mockStore{}, newAuthSvc(t), zap.NewNop())

	// Username with uppercase — violates slug rule
	body := `{"username":"Alice","email":"alice@example.com","password":"securepass"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.Register(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── Login ─────────────────────────────────────────────────────────────────────

func TestLogin_Success(t *testing.T) {
	t.Parallel()

	hash, err := bcrypt.GenerateFromPassword([]byte("correctpass"), bcrypt.MinCost)
	require.NoError(t, err)

	user := fixedUser()
	user.PasswordHash = string(hash)

	st := &mockStore{
		onGetUserByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return user, nil
		},
	}
	h := handlers.NewUserHandler(st, newAuthSvc(t), zap.NewNop())

	body := `{"login":"alice","password":"correctpass"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.Login(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var env map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	var data map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(env["data"], &data))
	assert.Contains(t, data, "token")
	token := strings.Trim(string(data["token"]), `"`)
	assert.NotEmpty(t, token)
}

func TestLogin_WrongPassword(t *testing.T) {
	t.Parallel()

	hash, _ := bcrypt.GenerateFromPassword([]byte("rightpass"), bcrypt.MinCost)
	user := fixedUser()
	user.PasswordHash = string(hash)

	st := &mockStore{
		onGetUserByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return user, nil
		},
	}
	h := handlers.NewUserHandler(st, newAuthSvc(t), zap.NewNop())

	body := `{"login":"alice","password":"wrongpass"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.Login(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogin_FallbackToEmail(t *testing.T) {
	t.Parallel()

	hash, _ := bcrypt.GenerateFromPassword([]byte("mypass123"), bcrypt.MinCost)
	user := fixedUser()
	user.PasswordHash = string(hash)

	st := &mockStore{
		onGetUserByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return nil, store.ErrNotFound
		},
		onGetUserByEmail: func(_ context.Context, _ string) (*models.User, error) {
			return user, nil
		},
	}
	h := handlers.NewUserHandler(st, newAuthSvc(t), zap.NewNop())

	body := `{"login":"alice@example.com","password":"mypass123"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.Login(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ── GetUser ───────────────────────────────────────────────────────────────────

func TestGetUser_Public(t *testing.T) {
	t.Parallel()

	user := fixedUser()
	st := &mockStore{
		onGetUserByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return user, nil
		},
	}
	h := handlers.NewUserHandler(st, newAuthSvc(t), zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withURLParam(req, "username", "alice")
	w := httptest.NewRecorder()
	h.GetUser(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "password_hash")
}

func TestGetUser_NotFound(t *testing.T) {
	t.Parallel()

	h := handlers.NewUserHandler(&mockStore{}, newAuthSvc(t), zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withURLParam(req, "username", "ghost")
	w := httptest.NewRecorder()
	h.GetUser(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── GetCurrentUser ────────────────────────────────────────────────────────────

func TestGetCurrentUser_WithAuth(t *testing.T) {
	t.Parallel()

	user := fixedUser()
	h := handlers.NewUserHandler(&mockStore{}, newAuthSvc(t), zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withUser(req, user)
	w := httptest.NewRecorder()
	h.GetCurrentUser(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "password_hash")
}

func TestGetCurrentUser_Unauthenticated(t *testing.T) {
	t.Parallel()

	h := handlers.NewUserHandler(&mockStore{}, newAuthSvc(t), zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.GetCurrentUser(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ── DeleteSSHKey ──────────────────────────────────────────────────────────────

func TestDeleteSSHKey_NotOwner(t *testing.T) {
	t.Parallel()

	keyID := uuid.New()
	user := fixedUser()

	st := &mockStore{
		onDeleteSSHKey: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
			return store.ErrNotFound
		},
	}
	h := handlers.NewUserHandler(st, newAuthSvc(t), zap.NewNop())

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = withUser(req, user)
	req = withURLParam(req, "id", keyID.String())
	w := httptest.NewRecorder()
	h.DeleteSSHKey(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
