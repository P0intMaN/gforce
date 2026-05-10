// Package handlers contains the HTTP handler functions for the GForce API.
package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gforce/gforce/internal/api/dto"
	"github.com/gforce/gforce/internal/api/middleware"
	"github.com/gforce/gforce/internal/api/response"
	"github.com/gforce/gforce/internal/api/validate"
	"github.com/gforce/gforce/internal/auth"
	"github.com/gforce/gforce/internal/models"
	"github.com/gforce/gforce/internal/store"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	gossh "golang.org/x/crypto/ssh"
)

// UserHandler handles HTTP requests for user and authentication resources.
type UserHandler struct {
	store  store.Store
	auth   *auth.Service
	logger *zap.Logger
}

// NewUserHandler creates a UserHandler.
func NewUserHandler(s store.Store, authSvc *auth.Service, logger *zap.Logger) *UserHandler {
	return &UserHandler{store: s, auth: authSvc, logger: logger}
}

// Register handles POST /api/v1/auth/register.
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	req, err := validate.DecodeAndValidate[dto.RegisterRequest](r)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		response.InternalError(w, h.logger, err)
		return
	}

	user, err := h.store.CreateUser(r.Context(), models.CreateUserParams{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
	})
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			response.Error(w, http.StatusConflict, "username or email already taken")
			return
		}
		response.InternalError(w, h.logger, err)
		return
	}

	h.logger.Info("user registered", zap.String("user_id", user.ID.String()), zap.String("username", user.Username))
	response.JSON(w, http.StatusCreated, userToDTO(user))
}

// Login handles POST /api/v1/auth/login.
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	req, err := validate.DecodeAndValidate[dto.LoginRequest](r)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	user, err := h.store.GetUserByUsername(r.Context(), req.Login)
	if errors.Is(err, store.ErrNotFound) {
		user, err = h.store.GetUserByEmail(r.Context(), req.Login)
	}
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.Error(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		response.InternalError(w, h.logger, err)
		return
	}

	if !user.IsActive {
		response.Error(w, http.StatusForbidden, "account is disabled")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		response.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := h.auth.Issue(user.ID.String(), user.Username)
	if err != nil {
		response.InternalError(w, h.logger, err)
		return
	}

	h.logger.Info("user logged in", zap.String("user_id", user.ID.String()))
	response.JSON(w, http.StatusOK, dto.TokenResponse{
		Token:     token,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	})
}

// GetUser handles GET /api/v1/users/:username — public endpoint.
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	user, err := h.store.GetUserByUsername(r.Context(), username)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.InternalError(w, h.logger, err)
		return
	}
	response.JSON(w, http.StatusOK, userToDTO(user))
}

// GetCurrentUser handles GET /api/v1/user — requires auth.
func (h *UserHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w)
		return
	}
	response.JSON(w, http.StatusOK, userToDTO(user))
}

// UpdateProfile handles PATCH /api/v1/user — requires auth.
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	current, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w)
		return
	}

	req, err := validate.DecodeAndValidate[dto.UpdateProfileRequest](r)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	params := models.UpdateUserParams{}
	if req.DisplayName != "" {
		params.DisplayName = &req.DisplayName
	}
	if req.Bio != "" {
		params.Bio = &req.Bio
	}
	if req.AvatarURL != "" {
		params.AvatarURL = &req.AvatarURL
	}

	updated, err := h.store.UpdateUser(r.Context(), current.ID, params)
	if err != nil {
		response.InternalError(w, h.logger, err)
		return
	}
	response.JSON(w, http.StatusOK, userToDTO(updated))
}

// AddSSHKey handles POST /api/v1/user/keys — requires auth.
func (h *UserHandler) AddSSHKey(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w)
		return
	}

	req, err := validate.DecodeAndValidate[dto.AddSSHKeyRequest](r)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	parsed, _, _, _, err := gossh.ParseAuthorizedKey([]byte(req.PublicKey))
	if err != nil {
		response.BadRequest(w, "invalid SSH public key format")
		return
	}
	fingerprint := gossh.FingerprintSHA256(parsed)

	key, err := h.store.CreateSSHKey(r.Context(), models.CreateSSHKeyParams{
		UserID:      user.ID,
		Title:       req.Title,
		PublicKey:   req.PublicKey,
		Fingerprint: fingerprint,
	})
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			response.Error(w, http.StatusConflict, "SSH key already registered")
			return
		}
		response.InternalError(w, h.logger, err)
		return
	}

	response.JSON(w, http.StatusCreated, sshKeyToDTO(key))
}

// ListSSHKeys handles GET /api/v1/user/keys — requires auth.
func (h *UserHandler) ListSSHKeys(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w)
		return
	}

	keys, err := h.store.ListSSHKeysByUser(r.Context(), user.ID)
	if err != nil {
		response.InternalError(w, h.logger, err)
		return
	}

	out := make([]dto.SSHKeyResponse, 0, len(keys))
	for _, k := range keys {
		out = append(out, sshKeyToDTO(k))
	}
	response.JSON(w, http.StatusOK, out)
}

// DeleteSSHKey handles DELETE /api/v1/user/keys/:id — requires auth.
func (h *UserHandler) DeleteSSHKey(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w)
		return
	}

	keyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid key ID")
		return
	}

	if err := h.store.DeleteSSHKey(r.Context(), keyID, user.ID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.InternalError(w, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- DTO conversion helpers -------------------------------------------------

func userToDTO(u *models.User) dto.UserResponse {
	return dto.UserResponse{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		DisplayName: derefStr(u.DisplayName),
		AvatarURL:   derefStr(u.AvatarURL),
		Bio:         derefStr(u.Bio),
		IsAdmin:     u.IsAdmin,
		CreatedAt:   u.CreatedAt,
	}
}

func sshKeyToDTO(k *models.SSHKey) dto.SSHKeyResponse {
	return dto.SSHKeyResponse{
		ID:          k.ID,
		Title:       k.Title,
		Fingerprint: k.Fingerprint,
		LastUsedAt:  k.LastUsedAt,
		CreatedAt:   k.CreatedAt,
	}
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
