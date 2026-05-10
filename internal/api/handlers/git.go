package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gforce/gforce/internal/store"
	"github.com/gforce/gforce/pkg/gitutil"
	"go.uber.org/zap"
)

// GitHandler handles API requests that expose git data (commits, branches, trees).
// Raw git smart-HTTP protocol is handled separately by gitserver.Handler.
type GitHandler struct {
	repos  store.RepoStore
	logger *zap.Logger
}

// NewGitHandler creates a GitHandler with the supplied dependencies.
func NewGitHandler(repos store.RepoStore, logger *zap.Logger) *GitHandler {
	return &GitHandler{repos: repos, logger: logger}
}

// ListCommits handles GET /api/v1/repos/{repoID}/commits — returns recent commits.
func (h *GitHandler) ListCommits(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "repoID")

	repo, err := h.repos.GetByID(r.Context(), repoID)
	if err != nil {
		if store.IsNotFound(err) {
			respondError(w, http.StatusNotFound, "repository not found")
			return
		}
		h.logger.Error("fetching repo for commits", zap.String("repo_id", repoID), zap.Error(err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	commits, err := gitutil.ListCommits(repo.DiskPath, repo.DefaultBranch, 30)
	if err != nil {
		h.logger.Error("listing commits", zap.String("disk_path", repo.DiskPath), zap.Error(err))
		respondError(w, http.StatusInternalServerError, "could not list commits")
		return
	}

	respondJSON(w, http.StatusOK, commits)
}

// ListBranches handles GET /api/v1/repos/{repoID}/branches — returns all branches.
func (h *GitHandler) ListBranches(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "repoID")

	repo, err := h.repos.GetByID(r.Context(), repoID)
	if err != nil {
		if store.IsNotFound(err) {
			respondError(w, http.StatusNotFound, "repository not found")
			return
		}
		h.logger.Error("fetching repo for branches", zap.String("repo_id", repoID), zap.Error(err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	branches, err := gitutil.ListBranches(repo.DiskPath)
	if err != nil {
		h.logger.Error("listing branches", zap.String("disk_path", repo.DiskPath), zap.Error(err))
		respondError(w, http.StatusInternalServerError, "could not list branches")
		return
	}

	respondJSON(w, http.StatusOK, branches)
}
