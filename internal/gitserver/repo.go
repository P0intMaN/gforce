package gitserver

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	gogit "github.com/go-git/go-git/v5"
)

// GetRepoPath returns the canonical on-disk path for a repository.
// The path takes the form <repoRoot>/<owner>/<repo>.git.
func GetRepoPath(repoRoot, owner, repo string) string {
	return filepath.Join(repoRoot, owner, repo+".git")
}

// EnsureOwnerDir creates the per-owner subdirectory under repoRoot if it does
// not already exist.
func EnsureOwnerDir(repoRoot, owner string) error {
	dir := filepath.Join(repoRoot, owner)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("gitserver.EnsureOwnerDir: creating %q: %w", dir, err)
	}
	return nil
}

// InitBareRepo initialises a bare git repository at path using go-git.
// It is idempotent: if a valid bare repository already exists there, it is a no-op.
func InitBareRepo(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("gitserver.InitBareRepo: creating directory %q: %w", path, err)
	}

	_, err := gogit.PlainInit(path, true)
	if err != nil && !errors.Is(err, gogit.ErrRepositoryAlreadyExists) {
		return fmt.Errorf("gitserver.InitBareRepo: %w", err)
	}
	return nil
}

// RepoExists reports whether a valid bare git repository exists at path.
func RepoExists(path string) bool {
	_, err := gogit.PlainOpen(path)
	return err == nil
}
