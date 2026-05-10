package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/gforce/gforce/internal/models"
	"github.com/gforce/gforce/internal/store"
	"go.uber.org/zap"
)

// RepoHandler handles HTTP requests for repository resources.
type RepoHandler struct {
	store       store.RepoStore
	gitRootPath string
	logger      *zap.Logger
}

// NewRepoHandler creates a RepoHandler with the supplied dependencies.
func NewRepoHandler(s store.RepoStore, gitRootPath string, logger *zap.Logger) *RepoHandler {
	return &RepoHandler{store: s, gitRootPath: gitRootPath, logger: logger}
}

// Create handles POST /api/v1/repos — creates a new repository record.
func (h *RepoHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromRequest(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	ownerID, err := parseUUID(claims.UserID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id in token")
		return
	}

	var req struct {
		Name          string  `json:"name"`
		Description   *string `json:"description"`
		IsPrivate     bool    `json:"is_private"`
		DefaultBranch string  `json:"default_branch"`
	}
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

	repo, err := h.store.CreateRepo(r.Context(), models.CreateRepoParams{
		OwnerID:       ownerID,
		Name:          req.Name,
		Description:   req.Description,
		IsPrivate:     req.IsPrivate,
		DefaultBranch: req.DefaultBranch,
		DiskPath:      filepath.Join(h.gitRootPath, claims.Username, req.Name+".git"),
	})
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			respondError(w, http.StatusConflict, "repository with that name already exists")
			return
		}
		h.logger.Error("creating repository", zap.String("name", req.Name), zap.Error(err))
		respondError(w, http.StatusInternalServerError, "could not create repository")
		return
	}

	h.logger.Info("repository created", zap.String("repo_id", repo.ID.String()), zap.String("name", repo.Name))
	respondJSON(w, http.StatusCreated, repo)
}

// Get handles GET /api/v1/repos/{repoID} — returns repository metadata.
func (h *RepoHandler) Get(w http.ResponseWriter, r *http.Request) {
	repoID, err := parseUUID(chi.URLParam(r, "repoID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid repository id")
		return
	}

	repo, err := h.store.GetRepoByID(r.Context(), repoID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, "repository not found")
			return
		}
		h.logger.Error("fetching repository", zap.String("repo_id", repoID.String()), zap.Error(err))
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
	ownerID, err := parseUUID(claims.UserID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id in token")
		return
	}

	limit, offset := parsePagination(r)
	repos, err := h.store.ListReposByOwner(r.Context(), ownerID, limit, offset)
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

// ListPublic handles GET /api/v1/explore/repos — lists all public repositories.
func (h *RepoHandler) ListPublic(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	repos, err := h.store.ListPublicRepos(r.Context(), limit, offset)
	if err != nil {
		h.logger.Error("listing public repositories", zap.Error(err))
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
	repoID, err := parseUUID(chi.URLParam(r, "repoID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid repository id")
		return
	}
	claims, ok := claimsFromRequest(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	ownerID, _ := parseUUID(claims.UserID)

	repo, err := h.store.GetRepoByID(r.Context(), repoID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, "repository not found")
			return
		}
		h.logger.Error("fetching repository for update", zap.String("repo_id", repoID.String()), zap.Error(err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if repo.OwnerID != ownerID {
		respondError(w, http.StatusForbidden, "not your repository")
		return
	}

	var params models.UpdateRepoParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updated, err := h.store.UpdateRepo(r.Context(), repoID, params)
	if err != nil {
		h.logger.Error("updating repository", zap.String("repo_id", repoID.String()), zap.Error(err))
		respondError(w, http.StatusInternalServerError, "could not update repository")
		return
	}

	respondJSON(w, http.StatusOK, updated)
}

// Delete handles DELETE /api/v1/repos/{repoID} — removes a repository record.
func (h *RepoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	repoID, err := parseUUID(chi.URLParam(r, "repoID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid repository id")
		return
	}
	claims, ok := claimsFromRequest(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	ownerID, _ := parseUUID(claims.UserID)

	repo, err := h.store.GetRepoByID(r.Context(), repoID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, "repository not found")
			return
		}
		h.logger.Error("fetching repository for delete", zap.String("repo_id", repoID.String()), zap.Error(err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if repo.OwnerID != ownerID {
		respondError(w, http.StatusForbidden, "not your repository")
		return
	}

	if err := h.store.DeleteRepo(r.Context(), repoID); err != nil {
		h.logger.Error("deleting repository", zap.String("repo_id", repoID.String()), zap.Error(err))
		respondError(w, http.StatusInternalServerError, "could not delete repository")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parsePagination(r *http.Request) (limit, offset int) {
	limit = 30
	offset = 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return limit, offset
}
