package handlers

import (
	"github.com/gforce/gforce/internal/store"
	"go.uber.org/zap"
)

// GitHandler is kept for backward compatibility.
// New code should use GitContentHandler for repository content endpoints.
type GitHandler struct {
	store  store.RepoStore
	logger *zap.Logger
}

// NewGitHandler creates a GitHandler.
func NewGitHandler(s store.RepoStore, logger *zap.Logger) *GitHandler {
	return &GitHandler{store: s, logger: logger}
}
