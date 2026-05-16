package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gforce/gforce/internal/api/middleware"
	"github.com/gforce/gforce/internal/api/response"
	"github.com/gforce/gforce/internal/models"
	"github.com/gforce/gforce/internal/store"
	"go.uber.org/zap"
)

// PATHandler handles personal access token endpoints.
type PATHandler struct {
	store  store.Store
	logger *zap.Logger
}

// NewPATHandler creates a PATHandler.
func NewPATHandler(s store.Store, logger *zap.Logger) *PATHandler {
	return &PATHandler{store: s, logger: logger}
}

type createPATRequest struct {
	Name      string     `json:"name"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at"`
}

// Create handles POST /api/v1/user/tokens.
func (h *PATHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w)
		return
	}

	var req createPATRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Name == "" {
		response.BadRequest(w, "name is required")
		return
	}
	if len(req.Scopes) == 0 {
		req.Scopes = []string{"repo:read", "repo:write"}
	}

	pat, rawToken, err := h.store.CreatePAT(r.Context(), store.CreatePATParams{
		UserID:    user.ID,
		Name:      req.Name,
		Scopes:    req.Scopes,
		ExpiresAt: req.ExpiresAt,
	})
	if err != nil {
		response.InternalError(w, h.logger, err)
		return
	}

	// Embed the raw token into the response — this is the ONLY time it is returned.
	pat.TokenHash = "" // clear hash before embedding in response
	resp := struct {
		ID         uuid.UUID  `json:"id"`
		UserID     uuid.UUID  `json:"user_id"`
		Name       string     `json:"name"`
		Prefix     string     `json:"prefix"`
		Scopes     []string   `json:"scopes"`
		LastUsedAt *time.Time `json:"last_used_at"`
		ExpiresAt  *time.Time `json:"expires_at"`
		CreatedAt  time.Time  `json:"created_at"`
		Token      string     `json:"token"`
		Message    string     `json:"message"`
	}{
		ID:         pat.ID,
		UserID:     pat.UserID,
		Name:       pat.Name,
		Prefix:     pat.Prefix,
		Scopes:     pat.Scopes,
		LastUsedAt: pat.LastUsedAt,
		ExpiresAt:  pat.ExpiresAt,
		CreatedAt:  pat.CreatedAt,
		Token:      rawToken,
		Message:    "Store this token securely. It will not be shown again.",
	}

	h.logger.Info("PAT created", zap.String("user_id", user.ID.String()), zap.String("name", req.Name))
	response.JSON(w, http.StatusCreated, resp)
}

// List handles GET /api/v1/user/tokens.
func (h *PATHandler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w)
		return
	}

	pats, err := h.store.ListPATs(r.Context(), user.ID)
	if err != nil {
		response.InternalError(w, h.logger, err)
		return
	}

	if pats == nil {
		pats = []*models.PersonalAccessToken{}
	}
	response.JSON(w, http.StatusOK, pats)
}

// Revoke handles DELETE /api/v1/user/tokens/:id.
func (h *PATHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid token ID")
		return
	}

	if err := h.store.RevokePAT(r.Context(), id, user.ID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.InternalError(w, h.logger, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
