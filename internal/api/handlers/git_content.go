package handlers

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/gforce/gforce/internal/api/dto"
	"github.com/gforce/gforce/internal/api/middleware"
	"github.com/gforce/gforce/internal/api/response"
	"github.com/gforce/gforce/internal/models"
	"github.com/gforce/gforce/internal/store"
	"go.uber.org/zap"
)

// GitContentHandler serves repository contents (tree, blob, commits, branches)
// by reading the on-disk bare git repository via go-git.
// It is distinct from the git smart-HTTP server (gitserver package).
type GitContentHandler struct {
	store   store.Store
	baseURL string
	logger  *zap.Logger
}

// NewGitContentHandler creates a GitContentHandler.
func NewGitContentHandler(s store.Store, baseURL string, logger *zap.Logger) *GitContentHandler {
	return &GitContentHandler{store: s, baseURL: baseURL, logger: logger}
}

// GetTree handles GET /api/v1/repos/:owner/:repo/tree/:ref[?path=subdir].
func (h *GitContentHandler) GetTree(w http.ResponseWriter, r *http.Request) {
	dbRepo, _, gitRepo, ok := h.openAndAuthorize(w, r)
	if !ok {
		return
	}

	ref := chi.URLParam(r, "ref")
	commit, err := resolveCommit(gitRepo, ref)
	if err != nil {
		response.NotFound(w)
		return
	}

	tree, err := commit.Tree()
	if err != nil {
		response.InternalError(w, h.logger, err)
		return
	}

	subPath := r.URL.Query().Get("path")
	if subPath != "" {
		entry, err := tree.FindEntry(subPath)
		if err != nil {
			response.NotFound(w)
			return
		}
		if entry.Mode == filemode.Dir || entry.Mode == filemode.Submodule {
			subtree, err := gitRepo.TreeObject(entry.Hash)
			if err != nil {
				response.InternalError(w, h.logger, err)
				return
			}
			tree = subtree
		}
	}

	entries := make([]dto.TreeEntry, 0, len(tree.Entries))
	for _, e := range tree.Entries {
		entryType := "blob"
		if e.Mode == filemode.Dir || e.Mode == filemode.Submodule {
			entryType = "tree"
		}
		var size int64
		if entryType == "blob" {
			if blob, err := gitRepo.BlobObject(e.Hash); err == nil {
				size = blob.Size
			}
		}
		fullPath := e.Name
		if subPath != "" {
			fullPath = subPath + "/" + e.Name
		}
		entries = append(entries, dto.TreeEntry{
			Name: e.Name,
			Path: fullPath,
			Type: entryType,
			Size: size,
			SHA:  e.Hash.String(),
			Mode: e.Mode.String(),
		})
	}

	response.JSON(w, http.StatusOK, dto.TreeResponse{
		SHA:     commit.TreeHash.String(),
		Path:    subPath,
		Entries: entries,
	})
	_ = dbRepo
}

// GetBlob handles GET /api/v1/repos/:owner/:repo/blob/:ref/*path.
func (h *GitContentHandler) GetBlob(w http.ResponseWriter, r *http.Request) {
	_, _, gitRepo, ok := h.openAndAuthorize(w, r)
	if !ok {
		return
	}

	ref := chi.URLParam(r, "ref")
	filePath := chi.URLParam(r, "*")

	commit, err := resolveCommit(gitRepo, ref)
	if err != nil {
		response.NotFound(w)
		return
	}

	file, err := commit.File(filePath)
	if err != nil {
		response.NotFound(w)
		return
	}

	reader, err := file.Reader()
	if err != nil {
		response.InternalError(w, h.logger, err)
		return
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		response.InternalError(w, h.logger, err)
		return
	}

	// ?raw=true serves the decoded file bytes directly — used by the Raw button in the UI.
	if r.URL.Query().Get("raw") == "true" {
		contentType := "text/plain; charset=utf-8"
		// Serve binary files as octet-stream so the browser downloads them.
		if !isTextContent(content) {
			contentType = "application/octet-stream"
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(content)
		return
	}

	response.JSON(w, http.StatusOK, dto.BlobResponse{
		Path:     filePath,
		Content:  base64.StdEncoding.EncodeToString(content),
		Encoding: "base64",
		Size:     file.Size,
		SHA:      file.Hash.String(),
	})
}

// ListCommits handles GET /api/v1/repos/:owner/:repo/commits/:ref.
func (h *GitContentHandler) ListCommits(w http.ResponseWriter, r *http.Request) {
	dbRepo, ownerUser, gitRepo, ok := h.openAndAuthorize(w, r)
	if !ok {
		return
	}

	ref := chi.URLParam(r, "ref")
	limit, offset := parsePagination(r)

	hash, err := gitRepo.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		response.NotFound(w)
		return
	}

	iter, err := gitRepo.Log(&gogit.LogOptions{From: *hash})
	if err != nil {
		response.InternalError(w, h.logger, err)
		return
	}
	defer iter.Close()

	var commits []dto.CommitResponse
	i := 0
	_ = iter.ForEach(func(c *object.Commit) error {
		if i < offset {
			i++
			return nil
		}
		if len(commits) >= limit {
			return fmt.Errorf("stop")
		}
		parents := make([]string, 0, len(c.ParentHashes))
		for _, p := range c.ParentHashes {
			parents = append(parents, p.String())
		}
		commits = append(commits, dto.CommitResponse{
			SHA:       c.Hash.String(),
			Message:   c.Message,
			Author:    dto.CommitAuthor{Name: c.Author.Name, Email: c.Author.Email, Date: c.Author.When},
			Committer: dto.CommitAuthor{Name: c.Committer.Name, Email: c.Committer.Email, Date: c.Committer.When},
			Parents:   parents,
			HTMLURL: fmt.Sprintf("%s/%s/%s/commit/%s",
				h.baseURL, ownerUser.Username, dbRepo.Name, c.Hash.String()),
		})
		i++
		return nil
	})

	if commits == nil {
		commits = []dto.CommitResponse{}
	}
	response.JSON(w, http.StatusOK, commits)
}

// ListBranches handles GET /api/v1/repos/:owner/:repo/branches.
func (h *GitContentHandler) ListBranches(w http.ResponseWriter, r *http.Request) {
	dbRepo, _, gitRepo, ok := h.openAndAuthorize(w, r)
	if !ok {
		return
	}

	head, _ := gitRepo.Head()
	defaultBranch := dbRepo.DefaultBranch

	iter, err := gitRepo.Branches()
	if err != nil {
		response.InternalError(w, h.logger, err)
		return
	}
	defer iter.Close()

	var branches []dto.BranchResponse
	_ = iter.ForEach(func(ref *plumbing.Reference) error {
		isDefault := (head != nil && ref.Name() == head.Name()) ||
			ref.Name().Short() == defaultBranch
		branches = append(branches, dto.BranchResponse{
			Name:      ref.Name().Short(),
			CommitSHA: ref.Hash().String(),
			IsDefault: isDefault,
		})
		return nil
	})

	if branches == nil {
		branches = []dto.BranchResponse{}
	}
	response.JSON(w, http.StatusOK, branches)
}

// RefExists handles GET /api/v1/repos/:owner/:repo/refs/:ref/exists.
func (h *GitContentHandler) RefExists(w http.ResponseWriter, r *http.Request) {
	_, _, gitRepo, ok := h.openAndAuthorize(w, r)
	if !ok {
		return
	}

	ref := chi.URLParam(r, "ref")
	_, err := gitRepo.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		response.NotFound(w)
		return
	}
	response.JSON(w, http.StatusOK, struct {
		Exists bool `json:"exists"`
	}{true})
}

// --- helpers ----------------------------------------------------------------

// openAndAuthorize loads the DB repo record, checks private-repo access,
// and opens the on-disk git repository. Writes an error response and returns
// ok=false when any step fails.
func (h *GitContentHandler) openAndAuthorize(
	w http.ResponseWriter, r *http.Request,
) (*models.Repository, *models.User, *gogit.Repository, bool) {
	ownerName := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")

	ownerUser, err := h.store.GetUserByUsername(r.Context(), ownerName)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.NotFound(w)
		} else {
			response.InternalError(w, h.logger, err)
		}
		return nil, nil, nil, false
	}

	dbRepo, err := h.store.GetRepoByOwnerAndName(r.Context(), ownerUser.ID, repoName)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.NotFound(w)
		} else {
			response.InternalError(w, h.logger, err)
		}
		return nil, nil, nil, false
	}

	if dbRepo.IsPrivate {
		caller, ok := middleware.UserFromContext(r.Context())
		if !ok || caller.ID != dbRepo.OwnerID {
			response.Forbidden(w)
			return nil, nil, nil, false
		}
	}

	gitRepo, err := gogit.PlainOpen(dbRepo.DiskPath)
	if err != nil {
		h.logger.Error("opening git repo", zap.String("path", dbRepo.DiskPath), zap.Error(err))
		response.InternalError(w, h.logger, err)
		return nil, nil, nil, false
	}

	return dbRepo, ownerUser, gitRepo, true
}

// isTextContent reports whether b looks like UTF-8 text by checking for
// null bytes (the simplest binary-content heuristic).
func isTextContent(b []byte) bool {
	const sampleSize = 512
	n := len(b)
	if n > sampleSize {
		n = sampleSize
	}
	for _, c := range b[:n] {
		if c == 0 {
			return false
		}
	}
	return true
}

// resolveCommit resolves a ref string (branch, tag, or full SHA) to a commit object.
func resolveCommit(repo *gogit.Repository, ref string) (*object.Commit, error) {
	hash, err := repo.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		return nil, fmt.Errorf("resolving ref %q: %w", ref, err)
	}
	return repo.CommitObject(*hash)
}
