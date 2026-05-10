// Package gitutil provides pure git helper functions backed by go-git.
package gitutil

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	gogit "github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
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

// CreateInitialCommit creates a README.md and makes the first commit in the
// bare repository at bareRepoPath. The commit is pushed to refs/heads/<branch>.
func CreateInitialCommit(bareRepoPath, repoName, branch string) error {
	tmpDir, err := os.MkdirTemp("", "gforce-init-*")
	if err != nil {
		return fmt.Errorf("gitutil.CreateInitialCommit: creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	localRepo, err := gogit.PlainInit(tmpDir, false)
	if err != nil {
		return fmt.Errorf("gitutil.CreateInitialCommit: init local repo: %w", err)
	}

	readme := fmt.Sprintf("# %s\n\nCreated with [gforce](https://github.com/gforce/gforce).\n", repoName)
	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte(readme), 0o644); err != nil {
		return fmt.Errorf("gitutil.CreateInitialCommit: writing README: %w", err)
	}

	wt, err := localRepo.Worktree()
	if err != nil {
		return fmt.Errorf("gitutil.CreateInitialCommit: getting worktree: %w", err)
	}
	if _, err := wt.Add("README.md"); err != nil {
		return fmt.Errorf("gitutil.CreateInitialCommit: staging README: %w", err)
	}

	sig := &object.Signature{Name: "gforce", Email: "noreply@gforce.dev", When: time.Now()}
	if _, err := wt.Commit("Initial commit", &gogit.CommitOptions{Author: sig, Committer: sig}); err != nil {
		return fmt.Errorf("gitutil.CreateInitialCommit: committing: %w", err)
	}

	if _, err := localRepo.CreateRemote(&gitconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{bareRepoPath},
	}); err != nil {
		return fmt.Errorf("gitutil.CreateInitialCommit: creating remote: %w", err)
	}

	refSpec := gitconfig.RefSpec(fmt.Sprintf("HEAD:refs/heads/%s", branch))
	if err := localRepo.Push(&gogit.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []gitconfig.RefSpec{refSpec},
	}); err != nil {
		return fmt.Errorf("gitutil.CreateInitialCommit: pushing: %w", err)
	}

	// Update the bare repo's HEAD to point to the new branch.
	bareRepo, err := gogit.PlainOpen(bareRepoPath)
	if err != nil {
		return fmt.Errorf("gitutil.CreateInitialCommit: opening bare repo: %w", err)
	}
	headRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName(branch))
	return bareRepo.Storer.SetReference(headRef)
}

// ListCommits returns up to limit commits starting at offset, reachable from
// the given ref in the repository at diskPath.
func ListCommits(diskPath, ref string, limit, offset int) ([]*models.Commit, error) {
	repo, err := gogit.PlainOpen(diskPath)
	if err != nil {
		return nil, fmt.Errorf("opening repo at %s: %w", diskPath, err)
	}

	hash, err := repo.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		return nil, fmt.Errorf("resolving ref %q: %w", ref, err)
	}

	iter, err := repo.Log(&gogit.LogOptions{From: *hash})
	if err != nil {
		return nil, fmt.Errorf("getting commit log: %w", err)
	}
	defer iter.Close()

	var commits []*models.Commit
	i := 0
	err = iter.ForEach(func(c *object.Commit) error {
		if i < offset {
			i++
			return nil
		}
		if len(commits) >= limit {
			return fmt.Errorf("stop")
		}
		var parents []string
		for _, p := range c.ParentHashes {
			parents = append(parents, p.String())
		}
		commits = append(commits, &models.Commit{
			SHA:       c.Hash.String(),
			Message:   c.Message,
			Author:    c.Author.Name,
			Email:     c.Author.Email,
			Timestamp: c.Author.When,
		})
		i++
		return nil
	})
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

// DefaultBranch returns the symbolic HEAD reference target.
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
