// Package api wires together the Chi router and all HTTP handlers.
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/gforce/gforce/internal/api/handlers"
	"github.com/gforce/gforce/internal/api/middleware"
	"github.com/gforce/gforce/internal/auth"
	"github.com/gforce/gforce/internal/gitserver"
	"github.com/gforce/gforce/internal/store"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// RouterConfig bundles the dependencies required to build the router.
type RouterConfig struct {
	Store       store.Store
	AuthService *auth.Service
	GitRootPath string
	Logger      *zap.Logger
}

// NewRouter constructs the full Chi router with all middleware and routes registered.
// The git smart-HTTP handler is mounted last as a catch-all for /{owner}/{repo}.git/... paths.
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	authn := middleware.NewAuthenticator(cfg.AuthService)

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.RequestLogger(cfg.Logger))
	r.Use(chimiddleware.Recoverer)

	userH := handlers.NewUserHandler(cfg.Store, cfg.AuthService, cfg.Logger)
	repoH := handlers.NewRepoHandler(cfg.Store, cfg.GitRootPath, cfg.Logger)
	gitH := handlers.NewGitHandler(cfg.Store, cfg.Logger)
	gitSmartHTTP := gitserver.NewGitHandler(cfg.Store, cfg.AuthService, cfg.GitRootPath, cfg.Logger)

	r.Get("/healthz", healthz)
	r.Handle("/metrics", promhttp.Handler())

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", userH.Login)
		r.Post("/users", userH.Register)

		r.Get("/explore/repos", repoH.ListPublic)

		r.Group(func(r chi.Router) {
			r.Use(authn.Require)

			r.Get("/users/me", userH.GetCurrentUser)

			r.Route("/repos", func(r chi.Router) {
				r.Get("/", repoH.List)
				r.Post("/", repoH.Create)

				r.Route("/{repoID}", func(r chi.Router) {
					r.Get("/", repoH.Get)
					r.Patch("/", repoH.Update)
					r.Delete("/", repoH.Delete)

					r.Get("/commits", gitH.ListCommits)
					r.Get("/branches", gitH.ListBranches)
				})
			})
		})
	})

	// Git smart-HTTP: catch-all after all API routes.
	// GitHandler.ServeHTTP returns 404 for any path that is not a git protocol URL.
	r.Handle("/*", gitSmartHTTP)

	return r
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
