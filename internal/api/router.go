// Package api wires together the Chi router, middleware, and all HTTP handlers.
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/gforce/gforce/internal/api/handlers"
	"github.com/gforce/gforce/internal/api/middleware"
	"github.com/gforce/gforce/internal/auth"
	"github.com/gforce/gforce/internal/gitserver"
	"github.com/gforce/gforce/internal/server"
	"github.com/gforce/gforce/internal/store"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"go.uber.org/zap"
)

// RouterConfig bundles every dependency required to build the router.
type RouterConfig struct {
	Store          store.Store
	AuthService    *auth.Service
	GitRootPath    string
	BaseURL        string
	AllowedOrigins []string
	Logger         *zap.Logger
	// K8sClient is optional — when non-nil, Repository CRs are created on repo creation.
	K8sClient    k8sclient.Client
	K8sNamespace string
	// UIDistDir is the path to the built React UI (ui/dist).
	// When set, the UI is served as a catch-all fallback after all API routes.
	UIDistDir string
}

// NewRouter constructs the full Chi router with all middleware and routes registered.
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	// Token validator interface satisfied by *auth.Service.
	tv := cfg.AuthService

	// ── Global middleware ────────────────────────────────────────────────────
	origins := cfg.AllowedOrigins
	if len(origins) == 0 {
		origins = []string{"*"}
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID", "X-RateLimit-Limit"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.RequestLogger(cfg.Logger))
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.RateLimit(100)) // 100 req/min per IP

	// ── Handlers ─────────────────────────────────────────────────────────────
	userH := handlers.NewUserHandler(cfg.Store, cfg.AuthService, cfg.Logger)
	repoH := handlers.NewRepoHandler(cfg.Store, cfg.GitRootPath, cfg.BaseURL, cfg.Logger, cfg.K8sClient, cfg.K8sNamespace)
	gitContentH := handlers.NewGitContentHandler(cfg.Store, cfg.BaseURL, cfg.Logger)
	gitSmartHTTP := gitserver.NewGitHandler(cfg.Store, tv, cfg.GitRootPath, cfg.Logger)

	// ── Infra routes ─────────────────────────────────────────────────────────
	r.Get("/healthz", healthz)
	r.Handle("/metrics", promhttp.Handler())

	// ── API v1 ───────────────────────────────────────────────────────────────
	r.Route("/api/v1", func(r chi.Router) {
		// Public auth
		r.Post("/auth/register", userH.Register)
		r.Post("/auth/login", userH.Login)

		// Public user endpoints
		r.Get("/users/{username}", userH.GetUser)
		r.With(middleware.OptionalAuth(tv, cfg.Store)).Get("/users/{username}/repos", repoH.ListByUser)

		// Public repo endpoint — optional auth so private repos return 403 not 404
		r.With(middleware.OptionalAuth(tv, cfg.Store)).Get("/repos/{owner}/{repo}", repoH.Get)

		// Authenticated user endpoints
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(tv, cfg.Store))

			r.Get("/user", userH.GetCurrentUser)
			r.Patch("/user", userH.UpdateProfile)
			r.Post("/user/repos", repoH.Create)
			r.Post("/user/keys", userH.AddSSHKey)
			r.Get("/user/keys", userH.ListSSHKeys)
			r.Delete("/user/keys/{id}", userH.DeleteSSHKey)
		})

		// Repo mutation — owner only
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(tv, cfg.Store))

			r.Patch("/repos/{owner}/{repo}", repoH.Update)
			r.Delete("/repos/{owner}/{repo}", repoH.Delete)
		})

		// Git content — optional auth (private repo access check inside handler)
		r.Group(func(r chi.Router) {
			r.Use(middleware.OptionalAuth(tv, cfg.Store))

			r.Get("/repos/{owner}/{repo}/tree/{ref}", gitContentH.GetTree)
			r.Get("/repos/{owner}/{repo}/blob/{ref}/*", gitContentH.GetBlob)
			r.Get("/repos/{owner}/{repo}/commits/{ref}", gitContentH.ListCommits)
			r.Get("/repos/{owner}/{repo}/branches", gitContentH.ListBranches)
			r.Get("/repos/{owner}/{repo}/refs/{ref}/exists", gitContentH.RefExists)
		})
	})

	// Catch-all: git smart-HTTP for /{owner}/{repo}.git/... paths;
	// React SPA for everything else (when UIDistDir is configured).
	if uiH := server.UIHandler(cfg.UIDistDir); uiH != nil {
		r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isGitPath(r.URL.Path) {
				gitSmartHTTP.ServeHTTP(w, r)
				return
			}
			uiH.ServeHTTP(w, r)
		}))
	} else {
		r.Handle("/*", gitSmartHTTP)
	}

	return r
}

// isGitPath returns true for paths that belong to the git smart-HTTP protocol.
func isGitPath(path string) bool {
	return len(path) > 4 &&
		(containsSuffix(path, ".git/info/refs") ||
			containsSuffix(path, ".git/git-upload-pack") ||
			containsSuffix(path, ".git/git-receive-pack"))
}

func containsSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
