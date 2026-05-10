// Package gitserver implements the Git smart-HTTP protocol for push and fetch operations.
package gitserver

import (
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/gforce/gforce/internal/store"
	"go.uber.org/zap"
)

// Handler serves Git smart-HTTP endpoints (info/refs, git-upload-pack, git-receive-pack).
// It delegates actual pack computation to the system git binary via CGI-style stdio pipes.
type Handler struct {
	repos       store.RepoStore
	gitRootPath string
	logger      *zap.Logger
}

// New creates a Handler that resolves repository disk paths using repos and gitRootPath.
func New(repos store.RepoStore, gitRootPath string, logger *zap.Logger) *Handler {
	return &Handler{repos: repos, gitRootPath: gitRootPath, logger: logger}
}

// InfoRefs handles GET /{owner}/{repo}.git/info/refs — the first handshake in a clone/fetch/push.
func (h *Handler) InfoRefs(w http.ResponseWriter, r *http.Request) {
	service := r.URL.Query().Get("service")
	if service != "git-upload-pack" && service != "git-receive-pack" {
		http.Error(w, "unsupported service", http.StatusForbidden)
		return
	}

	diskPath, err := h.resolveDiskPath(r)
	if err != nil {
		h.logger.Warn("resolving disk path", zap.Error(err))
		http.Error(w, "repository not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/x-%s-advertisement", service))
	w.Header().Set("Cache-Control", "no-cache")

	pkt := fmt.Sprintf("# service=%s\n", service)
	_, _ = fmt.Fprintf(w, "%04x%s0000", len(pkt)+4, pkt)

	cmd := exec.CommandContext(r.Context(), service, "--stateless-rpc", "--advertise-refs", diskPath) //nolint:gosec
	cmd.Stdout = w
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		h.logger.Error("git info-refs", zap.String("service", service), zap.Error(err))
	}
}

// UploadPack handles POST /{owner}/{repo}.git/git-upload-pack — serves fetch/clone data.
func (h *Handler) UploadPack(w http.ResponseWriter, r *http.Request) {
	h.runService(w, r, "git-upload-pack")
}

// ReceivePack handles POST /{owner}/{repo}.git/git-receive-pack — accepts pushed objects.
func (h *Handler) ReceivePack(w http.ResponseWriter, r *http.Request) {
	h.runService(w, r, "git-receive-pack")
}

func (h *Handler) runService(w http.ResponseWriter, r *http.Request, service string) {
	diskPath, err := h.resolveDiskPath(r)
	if err != nil {
		h.logger.Warn("resolving disk path", zap.Error(err))
		http.Error(w, "repository not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/x-%s-result", service))
	w.Header().Set("Cache-Control", "no-cache")

	cmd := exec.CommandContext(r.Context(), service, "--stateless-rpc", diskPath) //nolint:gosec
	cmd.Stdin = r.Body
	cmd.Stdout = w
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		h.logger.Error("git service", zap.String("service", service), zap.Error(err))
	}
}

// resolveDiskPath extracts the owner/repo path params and looks up the on-disk path.
func (h *Handler) resolveDiskPath(r *http.Request) (string, error) {
	owner := chi.URLParam(r, "owner")
	repoName := strings.TrimSuffix(chi.URLParam(r, "repo"), ".git")

	if owner == "" || repoName == "" {
		return "", fmt.Errorf("missing owner or repo path params")
	}

	return filepath.Join(h.gitRootPath, owner, repoName+".git"), nil
}
