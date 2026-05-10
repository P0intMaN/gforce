// Package handlers contains the HTTP handler functions for the gforce API.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gforce/gforce/internal/auth"
	"github.com/gforce/gforce/internal/models"
	"github.com/gforce/gforce/internal/store"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// UserHandler handles HTTP requests for user and authentication resources.
type UserHandler struct {
	store  store.UserStore
	auth   *auth.Service
	logger *zap.Logger
}

// NewUserHandler creates a UserHandler with the supplied dependencies.
func NewUserHandler(s store.UserStore, authSvc *auth.Service, logger *zap.Logger) *UserHandler {
	return &UserHandler{store: s, auth: authSvc, logger: logger}
}

// Register handles POST /api/v1/users — creates a new user account.
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username    string  `json:"username"`
		Email       string  `json:"email"`
		Password    string  `json:"password"`
		DisplayName *string `json:"display_name"`
	}
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

	user, err := h.store.CreateUser(r.Context(), models.CreateUserParams{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
		DisplayName:  req.DisplayName,
	})
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			respondError(w, http.StatusConflict, "username or email already taken")
			return
		}
		h.logger.Error("creating user", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "could not create user")
		return
	}

	h.logger.Info("user registered", zap.String("user_id", user.ID.String()), zap.String("username", user.Username))
	respondJSON(w, http.StatusCreated, user)
}

// Login handles POST /api/v1/auth/login — issues a JWT on valid credentials.
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.store.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		h.logger.Error("fetching user for login", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if !user.IsActive {
		respondError(w, http.StatusForbidden, "account is disabled")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := h.auth.Issue(user.ID.String(), user.Username)
	if err != nil {
		h.logger.Error("issuing token", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "could not issue token")
		return
	}

	h.logger.Info("user logged in", zap.String("user_id", user.ID.String()), zap.String("username", user.Username))
	respondJSON(w, http.StatusOK, models.LoginResponse{Token: token, IssuedAt: time.Now().UTC()})
}

// GetCurrentUser handles GET /api/v1/users/me — returns the authenticated user's profile.
func (h *UserHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromRequest(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	userID, err := parseUUID(claims.UserID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id in token")
		return
	}

	user, err := h.store.GetUserByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}
		h.logger.Error("fetching current user", zap.String("user_id", claims.UserID), zap.Error(err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	respondJSON(w, http.StatusOK, user)
}
