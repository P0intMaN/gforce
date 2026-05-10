package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	gforcev1alpha1 "github.com/gforce/gforce/operator/api/v1alpha1"
	"github.com/gforce/gforce/internal/api/dto"
	"github.com/gforce/gforce/internal/api/middleware"
	"github.com/gforce/gforce/internal/api/response"
	"github.com/gforce/gforce/internal/api/validate"
	"github.com/gforce/gforce/internal/gitserver"
	"github.com/gforce/gforce/internal/models"
	"github.com/gforce/gforce/internal/store"
	"github.com/gforce/gforce/pkg/gitutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"go.uber.org/zap"
)

// RepoHandler handles HTTP requests for repository resources.
type RepoHandler struct {
	store        store.Store
	gitRootPath  string
	baseURL      string
	logger       *zap.Logger
	k8sClient    k8sclient.Client // nil when running outside Kubernetes
	k8sNamespace string
}

// NewRepoHandler creates a RepoHandler. k8sClient may be nil — CR creation is skipped in that case.
func NewRepoHandler(s store.Store, gitRootPath, baseURL string, logger *zap.Logger, k8sClient k8sclient.Client, k8sNamespace string) *RepoHandler {
	return &RepoHandler{
		store:        s,
		gitRootPath:  gitRootPath,
		baseURL:      baseURL,
		logger:       logger,
		k8sClient:    k8sClient,
		k8sNamespace: k8sNamespace,
	}
}

// Create handles POST /api/v1/user/repos — requires auth.
func (h *RepoHandler) Create(w http.ResponseWriter, r *http.Request) {
	owner, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w)
		return
	}

	req, err := validate.DecodeAndValidate[dto.CreateRepoRequest](r)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	if req.DefaultBranch == "" {
		req.DefaultBranch = "main"
	}

	diskPath := gitserver.GetRepoPath(h.gitRootPath, owner.Username, req.Name)

	if err := gitserver.EnsureOwnerDir(h.gitRootPath, owner.Username); err != nil {
		response.InternalError(w, h.logger, err)
		return
	}

	var desc *string
	if req.Description != "" {
		desc = &req.Description
	}

	repo, err := h.store.CreateRepo(r.Context(), models.CreateRepoParams{
		OwnerID:       owner.ID,
		Name:          req.Name,
		Description:   desc,
		IsPrivate:     req.IsPrivate,
		DefaultBranch: req.DefaultBranch,
		DiskPath:      diskPath,
	})
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			response.Error(w, http.StatusConflict, "repository with that name already exists")
			return
		}
		response.InternalError(w, h.logger, err)
		return
	}

	if err := gitserver.InitBareRepo(diskPath); err != nil {
		h.logger.Error("initialising bare repo", zap.String("path", diskPath), zap.Error(err))
		if delErr := h.store.DeleteRepo(r.Context(), repo.ID); delErr != nil {
			h.logger.Error("cleaning up repo record", zap.Error(delErr))
		}
		response.InternalError(w, h.logger, err)
		return
	}

	if req.InitRepo {
		if err := gitutil.CreateInitialCommit(diskPath, req.Name, req.DefaultBranch, h.logger); err != nil {
			h.logger.Warn("creating initial commit failed", zap.String("path", diskPath), zap.Error(err))
			// Non-fatal: repo is usable, just empty.
		}
	}

	// Best-effort: create the Repository CR so the operator can reconcile it.
	// Non-fatal — the repo already exists in the DB and on disk.
	h.createRepositoryCR(r.Context(), owner.Username, owner.ID.String(), repo, req)

	h.logger.Info("repository created", zap.String("repo_id", repo.ID.String()), zap.String("full_name", owner.Username+"/"+req.Name))
	response.JSON(w, http.StatusCreated, repoToDTO(repo, owner, h.baseURL))
}

// createRepositoryCR creates the Kubernetes Repository CR for the operator to manage.
// Failures are logged and not propagated to the caller.
func (h *RepoHandler) createRepositoryCR(ctx context.Context, username, userID string, repo *models.Repository, req dto.CreateRepoRequest) {
	if h.k8sClient == nil {
		return
	}

	crName := fmt.Sprintf("%s-%s", username, repo.Name)
	cr := &gforcev1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      crName,
			Namespace: h.k8sNamespace,
		},
		Spec: gforcev1alpha1.RepositorySpec{
			OwnerRef: gforcev1alpha1.OwnerReference{
				Username: username,
				UserID:   userID,
			},
			Name:          repo.Name,
			Description:   req.Description,
			IsPrivate:     repo.IsPrivate,
			DefaultBranch: repo.DefaultBranch,
		},
	}

	if err := h.k8sClient.Create(ctx, cr); err != nil {
		h.logger.Warn("creating Repository CR",
			zap.String("cr_name", crName),
			zap.Error(err),
		)
	} else {
		h.logger.Info("Repository CR created",
			zap.String("cr_name", crName),
		)
	}
}

// Get handles GET /api/v1/repos/:owner/:repo — public for public repos, auth for private.
func (h *RepoHandler) Get(w http.ResponseWriter, r *http.Request) {
	repo, ownerUser, err := h.loadRepo(r)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.InternalError(w, h.logger, err)
		return
	}

	if repo.IsPrivate {
		caller, ok := middleware.UserFromContext(r.Context())
		if !ok || caller.ID != repo.OwnerID {
			response.Forbidden(w)
			return
		}
	}

	response.JSON(w, http.StatusOK, repoToDTO(repo, ownerUser, h.baseURL))
}

// ListByUser handles GET /api/v1/users/:username/repos — public repos always visible,
// private repos only visible to their owner.
func (h *RepoHandler) ListByUser(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	limit, offset := parsePagination(r)

	ownerUser, err := h.store.GetUserByUsername(r.Context(), username)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.InternalError(w, h.logger, err)
		return
	}

	caller, isAuth := middleware.UserFromContext(r.Context())
	isOwner := isAuth && caller.ID == ownerUser.ID

	var repos []*models.Repository
	if isOwner {
		repos, err = h.store.ListReposByOwner(r.Context(), ownerUser.ID, limit, offset)
	} else {
		repos, err = h.store.ListPublicReposByOwner(r.Context(), ownerUser.ID, limit, offset)
	}
	if err != nil {
		response.InternalError(w, h.logger, err)
		return
	}

	out := make([]dto.RepoResponse, 0, len(repos))
	for _, rp := range repos {
		out = append(out, repoToDTO(rp, ownerUser, h.baseURL))
	}
	response.JSON(w, http.StatusOK, out)
}

// Update handles PATCH /api/v1/repos/:owner/:repo — requires auth, must be owner.
func (h *RepoHandler) Update(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w)
		return
	}

	repo, ownerUser, err := h.loadRepo(r)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.InternalError(w, h.logger, err)
		return
	}
	if caller.ID != repo.OwnerID {
		response.Forbidden(w)
		return
	}

	req, err := validate.DecodeAndValidate[dto.UpdateRepoRequest](r)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	params := models.UpdateRepoParams{IsPrivate: &req.IsPrivate}
	if req.Description != "" {
		params.Description = &req.Description
	}
	if req.DefaultBranch != "" {
		params.DefaultBranch = &req.DefaultBranch
	}

	updated, err := h.store.UpdateRepo(r.Context(), repo.ID, params)
	if err != nil {
		response.InternalError(w, h.logger, err)
		return
	}
	response.JSON(w, http.StatusOK, repoToDTO(updated, ownerUser, h.baseURL))
}

// Delete handles DELETE /api/v1/repos/:owner/:repo — requires auth, must be owner.
func (h *RepoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w)
		return
	}

	repo, _, err := h.loadRepo(r)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.InternalError(w, h.logger, err)
		return
	}
	if caller.ID != repo.OwnerID {
		response.Forbidden(w)
		return
	}

	if err := h.store.DeleteRepo(r.Context(), repo.ID); err != nil {
		response.InternalError(w, h.logger, err)
		return
	}

	// Best-effort disk cleanup; log but don't fail the request.
	if err := os.RemoveAll(repo.DiskPath); err != nil {
		h.logger.Warn("removing repo from disk", zap.String("path", repo.DiskPath), zap.Error(err))
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- helpers ----------------------------------------------------------------

// loadRepo resolves the :owner and :repo URL params into DB records.
func (h *RepoHandler) loadRepo(r *http.Request) (*models.Repository, *models.User, error) {
	ownerName := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")

	ownerUser, err := h.store.GetUserByUsername(r.Context(), ownerName)
	if err != nil {
		return nil, nil, fmt.Errorf("owner %q: %w", ownerName, err)
	}

	repo, err := h.store.GetRepoByOwnerAndName(r.Context(), ownerUser.ID, repoName)
	if err != nil {
		return nil, nil, fmt.Errorf("repo %q/%q: %w", ownerName, repoName, err)
	}

	return repo, ownerUser, nil
}

// repoToDTO converts a Repository model to the public RepoResponse DTO.
// disk_path, fork_of, and other internal fields are excluded.
func repoToDTO(r *models.Repository, owner *models.User, baseURL string) dto.RepoResponse {
	fullName := owner.Username + "/" + r.Name
	return dto.RepoResponse{
		ID:            r.ID,
		Name:          r.Name,
		FullName:      fullName,
		Description:   derefStr(r.Description),
		IsPrivate:     r.IsPrivate,
		DefaultBranch: r.DefaultBranch,
		CloneURL:      baseURL + "/" + fullName + ".git",
		StarCount:     r.StarCount,
		ForkCount:     r.ForkCount,
		Owner:         userToDTO(owner),
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}

func parsePagination(r *http.Request) (limit, offset int) {
	limit = 30
	if v := r.URL.Query().Get("limit"); v != "" {
		if n := parseInt(v, 1, 100); n > 0 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n := parseInt(v, 0, 1<<31); n >= 0 {
			offset = n
		}
	}
	return limit, offset
}

func parseInt(s string, min, max int) int {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil || n < min || n > max {
		return -1
	}
	return n
}

// diskPathFor returns the expected disk path for owner/repo.
func diskPathFor(gitRootPath, username, repoName string) string {
	return filepath.Join(gitRootPath, username, repoName+".git")
}

// ensure diskPathFor is referenced (avoids "declared and not used" if not called elsewhere)
var _ = diskPathFor
