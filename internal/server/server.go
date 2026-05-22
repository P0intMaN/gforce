// Package server manages the lifecycle of the gforce HTTP server.
package server

import (
	"context"
	"fmt"
	ioFS "io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Server wraps an *http.Server with structured startup and graceful shutdown.
type Server struct {
	httpServer *http.Server
	logger     *zap.Logger
}

// Config holds the parameters needed to construct a Server.
type Config struct {
	Port             int
	Handler          http.Handler
	ReadTimeoutSecs  int
	WriteTimeoutSecs int
	Logger           *zap.Logger
}

// New creates a Server ready to serve on the configured port.
func New(cfg Config) *Server {
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      cfg.Handler,
		ReadTimeout:  time.Duration(cfg.ReadTimeoutSecs) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeoutSecs) * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	return &Server{httpServer: httpServer, logger: cfg.Logger}
}

// Start begins listening for connections. It blocks until the server errors or
// is shut down. Callers should invoke Shutdown to stop it gracefully.
func (s *Server) Start() error {
	s.logger.Info("starting http server", zap.String("addr", s.httpServer.Addr))
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
}

// Shutdown gracefully drains in-flight requests within the deadline imposed by ctx.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down http server")
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutting down http server: %w", err)
	}
	return nil
}

// UIHandlerFS serves the embedded React UI from an fs.FS (embed.FS).
// Non-API, non-git paths fall back to index.html for SPA client-side routing.
// Returns nil if fsys is nil.
func UIHandlerFS(fsys ioFS.FS) http.Handler {
	if fsys == nil {
		return nil
	}
	sub, err := ioFS.Sub(fsys, "dist")
	if err != nil {
		return nil
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.Contains(r.URL.Path, ".git/") {
			http.NotFound(w, r)
			return
		}
		clean := strings.TrimPrefix(r.URL.Path, "/")
		if clean == "" {
			clean = "index.html"
		}
		if _, err := ioFS.Stat(sub, clean); err != nil {
			idx, _ := ioFS.ReadFile(sub, "index.html")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(idx)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

// UIHandler returns an http.Handler that serves the built React UI from distDir.
// Non-API, non-git paths that don't match a file on disk fall back to index.html,
// supporting SPA client-side routing.
//
// Returns nil when distDir is empty or does not exist (UI not built yet).
func UIHandler(distDir string) http.Handler {
	if distDir == "" {
		return nil
	}
	if _, err := os.Stat(distDir); err != nil {
		return nil
	}

	fs := http.FileServer(http.Dir(distDir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Never serve API or git paths — those are handled by the main router.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		if strings.Contains(r.URL.Path, ".git/") {
			http.NotFound(w, r)
			return
		}

		// Check if the file exists in distDir.
		filePath := filepath.Join(distDir, filepath.Clean(r.URL.Path))
		if _, err := os.Stat(filePath); err != nil {
			// SPA fallback: serve index.html so React Router handles the route.
			http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
			return
		}

		fs.ServeHTTP(w, r)
	})
}
