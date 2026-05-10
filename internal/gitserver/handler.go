package gitserver

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gforce/gforce/internal/auth"
	"github.com/gforce/gforce/internal/models"
	"github.com/gforce/gforce/internal/store"
	"go.uber.org/zap"
)

// gitProcessTimeout is the maximum time a git subprocess may run.
const gitProcessTimeout = 5 * time.Minute

// URL patterns for the three git smart-HTTP endpoints.
var (
	reInfoRefs    = regexp.MustCompile(`^/([A-Za-z0-9_.-]+)/([A-Za-z0-9_.-]+)\.git/info/refs$`)
	reUploadPack  = regexp.MustCompile(`^/([A-Za-z0-9_.-]+)/([A-Za-z0-9_.-]+)\.git/git-upload-pack$`)
	reReceivePack = regexp.MustCompile(`^/([A-Za-z0-9_.-]+)/([A-Za-z0-9_.-]+)\.git/git-receive-pack$`)
)

// sentinel errors used only within this package for authorization decisions.
var (
	errUnauthorized = errors.New("authentication required")
	errForbidden    = errors.New("access denied")
)

// GitHandler serves Git smart-HTTP endpoints.
// It implements http.Handler and is designed to be registered as a catch-all
// route after all API routes.
type GitHandler struct {
	store    store.Store
	auth     auth.TokenValidator
	repoRoot string
	logger   *zap.Logger
}

// NewGitHandler creates a GitHandler.
func NewGitHandler(st store.Store, tv auth.TokenValidator, repoRoot string, logger *zap.Logger) *GitHandler {
	return &GitHandler{
		store:    st,
		auth:     tv,
		repoRoot: repoRoot,
		logger:   logger,
	}
}

// ServeHTTP routes incoming requests to the correct git sub-handler.
// Paths that do not match a git smart-HTTP pattern receive 404.
func (h *GitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case r.Method == http.MethodGet && reInfoRefs.MatchString(path):
		service := r.URL.Query().Get("service")
		if service != "git-upload-pack" && service != "git-receive-pack" {
			http.Error(w, "invalid service", http.StatusForbidden)
			return
		}
		m := reInfoRefs.FindStringSubmatch(path)
		h.dispatch(w, r, m[1], m[2], service, true)

	case r.Method == http.MethodPost && reUploadPack.MatchString(path):
		m := reUploadPack.FindStringSubmatch(path)
		h.dispatch(w, r, m[1], m[2], "git-upload-pack", false)

	case r.Method == http.MethodPost && reReceivePack.MatchString(path):
		m := reReceivePack.FindStringSubmatch(path)
		h.dispatch(w, r, m[1], m[2], "git-receive-pack", false)

	default:
		http.NotFound(w, r)
	}
}

// dispatch performs auth/authz for the resolved owner/repo, then calls the
// appropriate sub-handler (info/refs or service RPC).
func (h *GitHandler) dispatch(w http.ResponseWriter, r *http.Request, owner, repoName, service string, isInfoRefs bool) {
	repo, err := h.resolveRepo(r.Context(), owner, repoName)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		h.logger.Error("resolving repository",
			zap.String("owner", owner),
			zap.String("repo", repoName),
			zap.Error(err),
		)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	user, err := h.authenticate(r)
	if err != nil {
		h.logger.Debug("authentication failed", zap.Error(err))
		h.requireBasicAuth(w)
		return
	}

	if err := h.authorizeRepo(user, repo, service); err != nil {
		switch {
		case errors.Is(err, errUnauthorized):
			h.requireBasicAuth(w)
		case errors.Is(err, errForbidden):
			http.Error(w, "forbidden", http.StatusForbidden)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	if isInfoRefs {
		h.handleInfoRefs(w, r, repo, service)
	} else {
		h.handleServiceRPC(w, r, repo, service)
	}
}

// handleInfoRefs handles GET /info/refs — the discovery phase of clone/push/fetch.
func (h *GitHandler) handleInfoRefs(w http.ResponseWriter, r *http.Request, repo *models.Repository, service string) {
	ctx, cancel := context.WithTimeout(r.Context(), gitProcessTimeout)
	defer cancel()

	w.Header().Set("Content-Type", "application/x-"+service+"-advertisement")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Pragma", "no-cache")

	if err := WriteServiceHeader(w, service); err != nil {
		h.logger.Error("writing service header", zap.String("service", service), zap.Error(err))
		return
	}

	subcmd := strings.TrimPrefix(service, "git-") // "upload-pack" or "receive-pack"
	var stderrBuf bytes.Buffer

	cmd := exec.CommandContext(ctx, "git", subcmd, "--stateless-rpc", "--advertise-refs", repo.DiskPath) //nolint:gosec
	cmd.Env = gitEnv(repo.DiskPath, r.RemoteAddr)
	cmd.Stdout = &flushWriter{w: w, flusher: httpFlusher(w)}
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		h.logger.Error("git info-refs failed",
			zap.String("service", service),
			zap.String("repo", repo.DiskPath),
			zap.String("stderr", stderrBuf.String()),
			zap.Error(err),
		)
		// Headers already written; can't change status code.
	}
}

// handleServiceRPC handles POST for git-upload-pack and git-receive-pack.
func (h *GitHandler) handleServiceRPC(w http.ResponseWriter, r *http.Request, repo *models.Repository, service string) {
	want := "application/x-" + service + "-request"
	if ct := r.Header.Get("Content-Type"); ct != want {
		http.Error(w, fmt.Sprintf("expected Content-Type %q, got %q", want, ct), http.StatusUnsupportedMediaType)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), gitProcessTimeout)
	defer cancel()

	w.Header().Set("Content-Type", "application/x-"+service+"-result")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Pragma", "no-cache")

	body := r.Body
	if r.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(body)
		if err != nil {
			http.Error(w, "bad gzip body", http.StatusBadRequest)
			return
		}
		defer gz.Close()
		body = gz
	}

	subcmd := strings.TrimPrefix(service, "git-")
	var stderrBuf bytes.Buffer

	cmd := exec.CommandContext(ctx, "git", subcmd, "--stateless-rpc", repo.DiskPath) //nolint:gosec
	cmd.Env = gitEnv(repo.DiskPath, r.RemoteAddr)
	cmd.Stdin = body
	cmd.Stdout = &flushWriter{w: w, flusher: httpFlusher(w)}
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		h.logger.Error("git service rpc failed",
			zap.String("service", service),
			zap.String("repo", repo.DiskPath),
			zap.String("stderr", stderrBuf.String()),
			zap.Error(err),
		)
	}
}

// resolveRepo looks up a repository by owner username and repo name.
func (h *GitHandler) resolveRepo(ctx context.Context, owner, repoName string) (*models.Repository, error) {
	ownerUser, err := h.store.GetUserByUsername(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("gitserver.resolveRepo: owner %q: %w", owner, err)
	}

	repo, err := h.store.GetRepoByOwnerAndName(ctx, ownerUser.ID, repoName)
	if err != nil {
		return nil, fmt.Errorf("gitserver.resolveRepo: repo %q/%q: %w", owner, repoName, err)
	}

	return repo, nil
}

// authenticate extracts and validates credentials from the request.
//
// Supported schemes:
//   - Bearer <jwt>       — standard API token header
//   - Basic <b64>        — git CLI credential helper; username = GForce username,
//     password = JWT API token. The username in the Basic header must match
//     the subject (username) in the JWT claims.
//
// Returns (nil, nil) when no Authorization header is present — the caller
// decides whether unauthenticated access is acceptable.
func (h *GitHandler) authenticate(r *http.Request) (*models.User, error) {
	raw := r.Header.Get("Authorization")
	if raw == "" {
		return nil, nil
	}

	switch {
	case strings.HasPrefix(raw, "Bearer "):
		token := strings.TrimPrefix(raw, "Bearer ")
		return h.validateToken(r.Context(), token)

	case strings.HasPrefix(raw, "Basic "):
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(raw, "Basic "))
		if err != nil {
			return nil, fmt.Errorf("gitserver: decoding basic auth: %w", err)
		}
		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("gitserver: malformed basic auth credentials")
		}
		basicUsername, passwordToken := parts[0], parts[1]
		if passwordToken == "" {
			// git prompted for credentials but the user left the password blank.
			return nil, fmt.Errorf("gitserver: basic auth password (API token) is empty")
		}

		user, err := h.validateToken(r.Context(), passwordToken)
		if err != nil {
			return nil, err
		}
		// Verify the username supplied in Basic auth matches the token subject.
		// This prevents one user from authenticating as another by reusing a token.
		if basicUsername != "" && basicUsername != user.Username {
			return nil, fmt.Errorf("gitserver: basic auth username %q does not match token subject %q",
				basicUsername, user.Username)
		}
		return user, nil

	default:
		return nil, fmt.Errorf("gitserver: unsupported authorization scheme")
	}
}

// validateToken validates the JWT string and loads the associated user from the store.
func (h *GitHandler) validateToken(ctx context.Context, token string) (*models.User, error) {
	claims, err := h.auth.Validate(token)
	if err != nil {
		return nil, fmt.Errorf("gitserver: invalid token: %w", err)
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("gitserver: invalid user ID in token: %w", err)
	}

	user, err := h.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("gitserver: loading user: %w", err)
	}

	return user, nil
}

// authorizeRepo enforces access control rules:
//   - receive-pack always requires authentication
//   - private repos always require authentication
//   - receive-pack and private-repo access require the user to be the owner
func (h *GitHandler) authorizeRepo(user *models.User, repo *models.Repository, service string) error {
	isWrite := service == "git-receive-pack"

	if isWrite && user == nil {
		return errUnauthorized
	}
	if repo.IsPrivate && user == nil {
		return errUnauthorized
	}
	if user == nil {
		return nil // public read — no further checks needed
	}
	if repo.IsPrivate && user.ID != repo.OwnerID {
		return errForbidden
	}
	if isWrite && user.ID != repo.OwnerID {
		return errForbidden
	}
	return nil
}

// requireBasicAuth writes a 401 with the WWW-Authenticate challenge.
// The realm message tells the git CLI what to put in the password field.
func (h *GitHandler) requireBasicAuth(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="GForce - use your API token as password"`)
	http.Error(w, "authentication required", http.StatusUnauthorized)
}

// --- helpers ----------------------------------------------------------------

// gitEnv builds the minimal environment for git subprocesses.
func gitEnv(repoPath, remoteAddr string) []string {
	return []string{
		"GIT_DIR=" + repoPath,
		"GIT_HTTP_EXPORT_ALL=1",
		"REMOTE_ADDR=" + remoteAddr,
		"PATH=" + os.Getenv("PATH"),
	}
}

// flushWriter wraps an io.Writer and calls Flush after every write, enabling
// streaming responses for large pack transfers.
type flushWriter struct {
	w       interface{ Write([]byte) (int, error) }
	flusher http.Flusher
}

func (fw *flushWriter) Write(p []byte) (int, error) {
	n, err := fw.w.Write(p)
	if fw.flusher != nil {
		fw.flusher.Flush()
	}
	return n, err
}

// httpFlusher extracts an http.Flusher from w, returning nil if unsupported.
func httpFlusher(w http.ResponseWriter) http.Flusher {
	f, _ := w.(http.Flusher)
	return f
}
