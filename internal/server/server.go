// Package server manages the lifecycle of the gforce HTTP server.
package server

import (
	"context"
	"fmt"
	"net/http"
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
