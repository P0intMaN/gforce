// Package gitutil provides pure git helper functions backed by go-git.
package gitutil

import (
	"fmt"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/gforce/gforce/internal/models"
)

// InitBare initialises a bare git repository at the given path.
// It is idempotent: if the repository already exists it is opened, not re-created.
func InitBare(path string) error {
	_, err := gogit.PlainInit(path, true)
	if err != nil && err != gogit.ErrRepositoryAlreadyExists {
		return fmt.Errorf("init bare repo at %s: %w", path, err)
	}
	return nil
}

// ListCommits returns up to limit commits reachable from branch in the repository
// at diskPath. The commits are returned in reverse-chronological order.
func ListCommits(diskPath, branch string, limit int) ([]*models.Commit, error) {
	repo, err := gogit.PlainOpen(diskPath)
	if err != nil {
		return nil, fmt.Errorf("opening repo at %s: %w", diskPath, err)
	}

	ref, err := repo.Reference(plumbing.NewBranchReferenceName(branch), true)
	if err != nil {
		return nil, fmt.Errorf("resolving branch %q: %w", branch, err)
	}

	iter, err := repo.Log(&gogit.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, fmt.Errorf("getting commit log: %w", err)
	}
	defer iter.Close()

	var commits []*models.Commit
	err = iter.ForEach(func(c *object.Commit) error {
		if len(commits) >= limit {
			return fmt.Errorf("stop") // sentinel to break early
		}
		commits = append(commits, &models.Commit{
			SHA:       c.Hash.String(),
			Message:   c.Message,
			Author:    c.Author.Name,
			Email:     c.Author.Email,
			Timestamp: c.Author.When,
		})
		return nil
	})
	// The "stop" sentinel is expected; filter it out.
	if err != nil && err.Error() != "stop" {
		return nil, fmt.Errorf("iterating commits: %w", err)
	}

	return commits, nil
}

// ListBranches returns the names of all local branches in the repository at diskPath.
func ListBranches(diskPath string) ([]string, error) {
	repo, err := gogit.PlainOpen(diskPath)
	if err != nil {
		return nil, fmt.Errorf("opening repo at %s: %w", diskPath, err)
	}

	iter, err := repo.Branches()
	if err != nil {
		return nil, fmt.Errorf("listing branches: %w", err)
	}
	defer iter.Close()

	var names []string
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		names = append(names, ref.Name().Short())
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterating branches: %w", err)
	}

	return names, nil
}

// DefaultBranch returns the symbolic HEAD reference target (i.e. the default branch name).
func DefaultBranch(diskPath string) (string, error) {
	repo, err := gogit.PlainOpen(diskPath)
	if err != nil {
		return "", fmt.Errorf("opening repo at %s: %w", diskPath, err)
	}

	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("reading HEAD: %w", err)
	}

	return head.Name().Short(), nil
}
