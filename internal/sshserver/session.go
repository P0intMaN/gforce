package sshserver

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/gforce/gforce/internal/gitserver"
	"github.com/gforce/gforce/internal/models"
	"github.com/gforce/gforce/internal/store"
	"golang.org/x/crypto/ssh"
	"go.uber.org/zap"
)

const gitSubprocessTimeout = 5 * time.Minute

// handleSession processes the requests on an accepted SSH channel.
// Only "exec" requests are honoured — shell, subsystem, and all others are
// rejected. This is a git server, not a general-purpose SSH server.
func (s *SSHServer) handleSession(
	ch ssh.Channel,
	requests <-chan *ssh.Request,
	perms *ssh.Permissions,
) {
	defer ch.Close()

	username := perms.Extensions["username"]

	for req := range requests {
		switch req.Type {
		case "exec":
			s.handleExec(ch, req, username)
			return // one exec per session
		default:
			// Explicitly reject everything else: shell, pty, subsystem,
			// port-forward, x11, auth-agent-req, etc.
			if req.WantReply {
				_ = req.Reply(false, nil)
			}
		}
	}
}

// handleExec parses and executes a git-upload-pack or git-receive-pack command.
func (s *SSHServer) handleExec(
	ch ssh.Channel,
	req *ssh.Request,
	username string,
) {
	command, rawPath, err := parseGitCommand(req)
	if err != nil {
		s.logger.Warn("SSH exec parse error", zap.String("user", username), zap.Error(err))
		_ = req.Reply(false, nil)
		sendError(ch, "invalid command")
		return
	}

	// Allow only the two git pack commands — nothing else.
	if command != "git-upload-pack" && command != "git-receive-pack" {
		s.logger.Warn("SSH exec rejected — non-git command",
			zap.String("user", username),
			zap.String("command", command),
		)
		_ = req.Reply(false, nil)
		sendError(ch, fmt.Sprintf("command not allowed: %q", command))
		return
	}

	owner, repoName := parseRepoPath(rawPath)
	if owner == "" || repoName == "" {
		_ = req.Reply(false, nil)
		sendError(ch, "invalid repository path")
		return
	}

	repo, err := s.resolveRepo(owner, repoName)
	if err != nil {
		_ = req.Reply(false, nil)
		sendError(ch, "repository not found")
		return
	}

	if !s.isAuthorized(username, command, repo) {
		s.logger.Info("SSH exec denied — insufficient permissions",
			zap.String("user", username),
			zap.String("command", command),
			zap.String("repo", owner+"/"+repoName),
		)
		_ = req.Reply(false, nil)
		sendError(ch, "access denied")
		return
	}

	// Signal success to the client before starting the subprocess.
	_ = req.Reply(true, nil)

	diskPath := gitserver.GetRepoPath(s.repoRoot, owner, repoName)

	ctx, cancel := context.WithTimeout(context.Background(), gitSubprocessTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, diskPath) //nolint:gosec
	cmd.Env = []string{
		"GIT_DIR=" + diskPath,
		"GIT_HTTP_EXPORT_ALL=1",
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
	}
	cmd.Stdin = ch          // SSH channel carries pack data
	cmd.Stdout = ch         // pack-protocol output back to client
	cmd.Stderr = ch.Stderr() // git errors must reach the client or pushes fail silently

	s.logger.Info("SSH git command starting",
		zap.String("user", username),
		zap.String("command", command),
		zap.String("repo", owner+"/"+repoName),
	)

	runErr := cmd.Run()

	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	if runErr != nil {
		s.logger.Warn("SSH git command finished with error",
			zap.String("user", username),
			zap.String("command", command),
			zap.Int("exit_code", exitCode),
			zap.Error(runErr),
		)
	} else if command == "git-receive-pack" && exitCode == 0 {
		// Fire-and-forget: record the push event. Mirrors the HTTP git server's
		// behaviour in gitserver/handler.go. Never blocks the SSH session.
		actorUsername := username
		repoID := repo.ID
		repoName := repo.Name
		go func() {
			user, err := s.store.GetUserByUsername(context.Background(), actorUsername)
			if err != nil {
				return
			}
			_ = s.store.RecordEvent(context.Background(), store.RecordEventParams{
				ActorID:   user.ID,
				EventType: "git.push",
				RepoID:    &repoID,
				Payload:   map[string]interface{}{"repo": repoName},
			})
		}()
	}

	sendExitStatus(ch, uint32(exitCode))
}

// resolveRepo looks up a repository by owner username and repository name.
func (s *SSHServer) resolveRepo(owner, repoName string) (*models.Repository, error) {
	ctx := context.Background()

	ownerUser, err := s.store.GetUserByUsername(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("owner %q not found: %w", owner, err)
	}

	repo, err := s.store.GetRepoByOwnerAndName(ctx, ownerUser.ID, repoName)
	if err != nil {
		return nil, fmt.Errorf("repo %q/%q not found: %w", owner, repoName, err)
	}

	return repo, nil
}

// isAuthorized checks whether username may perform command on repo.
//
//   - git-upload-pack  on a public repo  → anyone with a registered key
//   - git-upload-pack  on a private repo → owner only
//   - git-receive-pack (push)            → owner only
func (s *SSHServer) isAuthorized(username, command string, repo *models.Repository) bool {
	ctx := context.Background()

	user, err := s.store.GetUserByUsername(ctx, username)
	if err != nil {
		s.logger.Warn("SSH auth: failed to load user", zap.String("username", username), zap.Error(err))
		return false
	}

	isOwner := user.ID == repo.OwnerID

	switch {
	case command == "git-receive-pack":
		return isOwner
	case repo.IsPrivate:
		return isOwner
	default:
		return true // public repo, read-only
	}
}

// keep errors import used
