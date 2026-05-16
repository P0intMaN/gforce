package sshserver_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gforce/gforce/internal/models"
	"github.com/gforce/gforce/internal/sshserver"
	"github.com/gforce/gforce/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
	"go.uber.org/zap"
)

// --- parseGitCommand tests --------------------------------------------------

func makeExecPayload(cmd string) []byte {
	b := make([]byte, 4+len(cmd))
	binary.BigEndian.PutUint32(b[:4], uint32(len(cmd)))
	copy(b[4:], cmd)
	return b
}

func TestParseGitCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		cmd         string
		wantCommand string
		wantPath    string
		wantErr     bool
	}{
		{
			name:        "upload-pack with quoted path",
			cmd:         "git-upload-pack '/alice/repo.git'",
			wantCommand: "git-upload-pack",
			wantPath:    "/alice/repo.git",
		},
		{
			name:        "receive-pack without leading slash",
			cmd:         "git-receive-pack 'alice/repo'",
			wantCommand: "git-receive-pack",
			wantPath:    "alice/repo",
		},
		{
			name:    "bare shell command",
			cmd:     "bash",
			wantErr: true,
		},
		{
			name:    "empty command",
			cmd:     "",
			wantErr: true,
		},
		{
			name:    "command with no argument",
			cmd:     "git-upload-pack",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := &ssh.Request{
				Type:    "exec",
				Payload: makeExecPayload(tc.cmd),
			}
			cmd, path, err := sshserver.ParseGitCommand(req)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantCommand, cmd)
			assert.Equal(t, tc.wantPath, path)
		})
	}
}

// --- parseRepoPath tests ----------------------------------------------------

func TestParseRepoPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input     string
		wantOwner string
		wantRepo  string
	}{
		{"/alice/repo.git", "alice", "repo"},
		{"alice/repo.git", "alice", "repo"},
		{"alice/repo", "alice", "repo"},
		{"/alice/my-repo.git", "alice", "my-repo"},
		{"/pratheek-first/my-first-repo.git", "pratheek-first", "my-first-repo"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			owner, repo := sshserver.ParseRepoPath(tc.input)
			assert.Equal(t, tc.wantOwner, owner)
			assert.Equal(t, tc.wantRepo, repo)
		})
	}
}

// --- publicKeyCallback tests ------------------------------------------------

// testStore is a minimal store.Store for testing the auth callback.
type testStore struct {
	store.Store // embed to satisfy the interface; unimplemented methods panic
	keys        map[string]*models.SSHKey
	users       map[uuid.UUID]*models.User
}

func (s *testStore) GetSSHKeyByFingerprint(_ context.Context, fp string) (*models.SSHKey, error) {
	if k, ok := s.keys[fp]; ok {
		return k, nil
	}
	return nil, store.ErrNotFound
}

func (s *testStore) GetUserByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	if u, ok := s.users[id]; ok {
		return u, nil
	}
	return nil, store.ErrNotFound
}

func TestPublicKeyAuth(t *testing.T) {
	t.Parallel()

	// Generate a test ED25519 key pair.
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	signer, err := ssh.NewSignerFromKey(priv)
	require.NoError(t, err)
	pubKey := signer.PublicKey()
	fingerprint := ssh.FingerprintSHA256(pubKey)

	userID := uuid.New()
	now := time.Now()

	st := &testStore{
		keys: map[string]*models.SSHKey{
			fingerprint: {
				ID:          uuid.New(),
				UserID:      userID,
				Title:       "test key",
				PublicKey:   string(ssh.MarshalAuthorizedKey(pubKey)),
				Fingerprint: fingerprint,
				CreatedAt:   now,
			},
		},
		users: map[uuid.UUID]*models.User{
			userID: {
				ID:        userID,
				Username:  "alice",
				Email:     "alice@example.com",
				IsActive:  true,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	srv, err := sshserver.NewSSHServerWithStore(st, "/tmp/repos", zap.NewNop())
	require.NoError(t, err)

	t.Run("known fingerprint returns permissions", func(t *testing.T) {
		t.Parallel()
		perms, err := srv.TestPublicKeyCallback(nil, pubKey)
		require.NoError(t, err)
		require.NotNil(t, perms)
		assert.Equal(t, "alice", perms.Extensions["username"])
		assert.Equal(t, userID.String(), perms.Extensions["user_id"])
	})

	t.Run("unknown fingerprint rejected", func(t *testing.T) {
		t.Parallel()
		_, unknownPriv, _ := ed25519.GenerateKey(rand.Reader)
		unknownSigner, _ := ssh.NewSignerFromKey(unknownPriv)
		perms, err := srv.TestPublicKeyCallback(nil, unknownSigner.PublicKey())
		assert.Error(t, err)
		assert.Nil(t, perms)
	})
}
