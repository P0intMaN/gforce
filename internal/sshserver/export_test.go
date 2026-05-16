// export_test.go exposes internal functions for testing.
// This file is only compiled during tests.
package sshserver

import (
	"fmt"

	"github.com/gforce/gforce/internal/store"
	"golang.org/x/crypto/ssh"
	"go.uber.org/zap"
)

// ParseGitCommand is exported for testing.
var ParseGitCommand = parseGitCommand

// ParseRepoPath is exported for testing.
var ParseRepoPath = parseRepoPath

// NewSSHServerWithStore creates an SSHServer without loading a host key
// file — for unit tests that don't need a real key file.
func NewSSHServerWithStore(st store.Store, repoRoot string, logger *zap.Logger) (*SSHServer, error) {
	// Generate an ephemeral host key for tests.
	signer, err := generateEphemeralKey()
	if err != nil {
		return nil, fmt.Errorf("generating test host key: %w", err)
	}

	srv := &SSHServer{
		store:     st,
		repoRoot:  repoRoot,
		hostKey:   signer,
		logger:    logger,
		semaphore: make(chan struct{}, defaultMaxConns),
	}

	cfg := &ssh.ServerConfig{
		PublicKeyCallback: srv.publicKeyCallback,
		AuthLogCallback:   srv.authLogCallback,
	}
	cfg.AddHostKey(signer)
	srv.config = cfg

	return srv, nil
}

// TestPublicKeyCallback exposes publicKeyCallback for unit tests.
func (s *SSHServer) TestPublicKeyCallback(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	return s.publicKeyCallback(conn, key)
}
