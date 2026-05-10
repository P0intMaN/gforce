// Package handlers contains the HTTP handler functions for the gforce API.
package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gforce/gforce/internal/auth"
	"github.com/gforce/gforce/internal/models"
	"github.com/gforce/gforce/internal/store"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// UserHandler handles HTTP requests for user and authentication resources.
type UserHandler struct {
	users  store.UserStore
	auth   *auth.Service
	logger *zap.Logger
}

// NewUserHandler creates a UserHandler with the supplied dependencies.
func NewUserHandler(users store.UserStore, authSvc *auth.Service, logger *zap.Logger) *UserHandler {
	return &UserHandler{users: users, auth: authSvc, logger: logger}
}

// Register handles POST /api/v1/users — creates a new user account.
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "username, email, and password are required")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.logger.Error("hashing password", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	now := time.Now().UTC()
	user := &models.User{
		ID:           uuid.NewString(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.users.Create(r.Context(), user); err != nil {
		h.logger.Error("creating user", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "could not create user")
		return
	}

	h.logger.Info("user registered", zap.String("user_id", user.ID), zap.String("username", user.Username))
	respondJSON(w, http.StatusCreated, user)
}

// Login handles POST /api/v1/auth/login — issues a JWT on valid credentials.
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.users.GetByUsername(r.Context(), req.Username)
	if err != nil {
		if store.IsNotFound(err) {
			respondError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		h.logger.Error("fetching user for login", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := h.auth.Issue(user.ID, user.Username)
	if err != nil {
		h.logger.Error("issuing token", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "could not issue token")
		return
	}

	h.logger.Info("user logged in", zap.String("user_id", user.ID))
	respondJSON(w, http.StatusOK, models.LoginResponse{Token: token})
}

// GetCurrentUser handles GET /api/v1/users/me — returns the authenticated user's profile.
func (h *UserHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromRequest(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	user, err := h.users.GetByID(r.Context(), claims.UserID)
	if err != nil {
		if store.IsNotFound(err) {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}
		h.logger.Error("fetching current user", zap.String("user_id", claims.UserID), zap.Error(err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	respondJSON(w, http.StatusOK, user)
}
