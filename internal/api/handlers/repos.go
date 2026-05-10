package handlers

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gforce/gforce/internal/models"
	"github.com/gforce/gforce/internal/store"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RepoHandler handles HTTP requests for repository resources.
type RepoHandler struct {
	repos       store.RepoStore
	gitRootPath string
	logger      *zap.Logger
}

// NewRepoHandler creates a RepoHandler with the supplied dependencies.
func NewRepoHandler(repos store.RepoStore, gitRootPath string, logger *zap.Logger) *RepoHandler {
	return &RepoHandler{repos: repos, gitRootPath: gitRootPath, logger: logger}
}

// Create handles POST /api/v1/repos — creates a new repository record.
func (h *RepoHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromRequest(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req models.CreateRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.DefaultBranch == "" {
		req.DefaultBranch = "main"
	}

	now := time.Now().UTC()
	repo := &models.Repository{
		ID:            uuid.NewString(),
		OwnerID:       claims.UserID,
		Name:          req.Name,
		Description:   req.Description,
		IsPrivate:     req.IsPrivate,
		DefaultBranch: req.DefaultBranch,
		DiskPath:      filepath.Join(h.gitRootPath, claims.UserID, req.Name+".git"),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := h.repos.Create(r.Context(), repo); err != nil {
		h.logger.Error("creating repository", zap.String("name", req.Name), zap.Error(err))
		respondError(w, http.StatusInternalServerError, "could not create repository")
		return
	}

	h.logger.Info("repository created", zap.String("repo_id", repo.ID), zap.String("name", repo.Name))
	respondJSON(w, http.StatusCreated, repo)
}

// Get handles GET /api/v1/repos/{repoID} — returns repository metadata.
func (h *RepoHandler) Get(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "repoID")

	repo, err := h.repos.GetByID(r.Context(), repoID)
	if err != nil {
		if store.IsNotFound(err) {
			respondError(w, http.StatusNotFound, "repository not found")
			return
		}
		h.logger.Error("fetching repository", zap.String("repo_id", repoID), zap.Error(err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	respondJSON(w, http.StatusOK, repo)
}

// List handles GET /api/v1/repos — lists repositories owned by the authenticated user.
func (h *RepoHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromRequest(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	repos, err := h.repos.ListByOwner(r.Context(), claims.UserID)
	if err != nil {
		h.logger.Error("listing repositories", zap.String("user_id", claims.UserID), zap.Error(err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if repos == nil {
		repos = []*models.Repository{}
	}
	respondJSON(w, http.StatusOK, repos)
}

// Update handles PATCH /api/v1/repos/{repoID} — updates mutable repository fields.
func (h *RepoHandler) Update(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "repoID")
	claims, ok := claimsFromRequest(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	repo, err := h.repos.GetByID(r.Context(), repoID)
	if err != nil {
		if store.IsNotFound(err) {
			respondError(w, http.StatusNotFound, "repository not found")
			return
		}
		h.logger.Error("fetching repository for update", zap.String("repo_id", repoID), zap.Error(err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if repo.OwnerID != claims.UserID {
		respondError(w, http.StatusForbidden, "not your repository")
		return
	}

	var req models.CreateRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != "" {
		repo.Name = req.Name
	}
	if req.Description != "" {
		repo.Description = req.Description
	}
	repo.IsPrivate = req.IsPrivate
	repo.UpdatedAt = time.Now().UTC()

	if err := h.repos.Update(r.Context(), repo); err != nil {
		h.logger.Error("updating repository", zap.String("repo_id", repoID), zap.Error(err))
		respondError(w, http.StatusInternalServerError, "could not update repository")
		return
	}

	respondJSON(w, http.StatusOK, repo)
}

// Delete handles DELETE /api/v1/repos/{repoID} — removes a repository record.
func (h *RepoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "repoID")
	claims, ok := claimsFromRequest(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	repo, err := h.repos.GetByID(r.Context(), repoID)
	if err != nil {
		if store.IsNotFound(err) {
			respondError(w, http.StatusNotFound, "repository not found")
			return
		}
		h.logger.Error("fetching repository for delete", zap.String("repo_id", repoID), zap.Error(err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if repo.OwnerID != claims.UserID {
		respondError(w, http.StatusForbidden, "not your repository")
		return
	}

	if err := h.repos.Delete(r.Context(), repoID); err != nil {
		h.logger.Error("deleting repository", zap.String("repo_id", repoID), zap.Error(err))
		respondError(w, http.StatusInternalServerError, "could not delete repository")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
