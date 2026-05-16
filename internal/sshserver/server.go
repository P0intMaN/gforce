package sshserver

import (
	"fmt"
	"net"

	"github.com/gforce/gforce/internal/store"
	"golang.org/x/crypto/ssh"
	"go.uber.org/zap"
)

const defaultMaxConns = 100

// SSHServer is a git-over-SSH server that authenticates via public keys
// stored in the GForce database and hands off to system git binaries.
type SSHServer struct {
	store     store.Store
	repoRoot  string
	hostKey   ssh.Signer
	logger    *zap.Logger
	config    *ssh.ServerConfig
	semaphore chan struct{} // limits concurrent connections
}

// NewSSHServer creates an SSHServer. It loads (or generates) the host key
// from hostKeyPath and configures key-only authentication.
func NewSSHServer(
	st store.Store,
	repoRoot string,
	hostKeyPath string,
	logger *zap.Logger,
) (*SSHServer, error) {
	hostKey, err := LoadOrGenerateHostKey(hostKeyPath)
	if err != nil {
		return nil, fmt.Errorf("sshserver.NewSSHServer: %w", err)
	}

	logger.Info("SSH host key loaded", zap.String("path", hostKeyPath),
		zap.String("fingerprint", ssh.FingerprintSHA256(hostKey.PublicKey())))

	srv := &SSHServer{
		store:     st,
		repoRoot:  repoRoot,
		hostKey:   hostKey,
		logger:    logger,
		semaphore: make(chan struct{}, defaultMaxConns),
	}

	cfg := &ssh.ServerConfig{
		PublicKeyCallback: srv.publicKeyCallback,
		AuthLogCallback:   srv.authLogCallback,
		// Password auth is explicitly disabled — key-only.
		// NoClientAuth: false (default) so clients must authenticate.
		BannerCallback: func(conn ssh.ConnMetadata) string {
			return "GForce Git Server — key auth only\n"
		},
	}
	cfg.AddHostKey(hostKey)
	srv.config = cfg

	return srv, nil
}

// ListenAndServe starts the TCP listener and accepts connections.
// It blocks until the listener fails.
func (s *SSHServer) ListenAndServe(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("sshserver: listening on %s: %w", addr, err)
	}
	defer ln.Close()

	s.logger.Info("SSH server listening", zap.String("addr", addr))

	for {
		conn, err := ln.Accept()
		if err != nil {
			return fmt.Errorf("sshserver: accepting connection: %w", err)
		}

		// Enforce max-concurrency via semaphore — drop connections beyond limit.
		select {
		case s.semaphore <- struct{}{}:
			go func() {
				defer func() { <-s.semaphore }()
				s.handleConn(conn)
			}()
		default:
			s.logger.Warn("SSH max connections reached — dropping connection",
				zap.String("remote_addr", conn.RemoteAddr().String()))
			_ = conn.Close()
		}
	}
}

// handleConn performs the SSH handshake and dispatches channel requests.
func (s *SSHServer) handleConn(conn net.Conn) {
	defer conn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.config)
	if err != nil {
		s.logger.Debug("SSH handshake failed",
			zap.String("remote_addr", conn.RemoteAddr().String()),
			zap.Error(err))
		return
	}
	defer sshConn.Close()

	s.logger.Info("SSH connection established",
		zap.String("user", sshConn.User()),
		zap.String("remote_addr", sshConn.RemoteAddr().String()),
		zap.String("client_version", string(sshConn.ClientVersion())),
	)

	// Discard all global requests (keepalive, tcpip-forward, etc.).
	go ssh.DiscardRequests(reqs)

	// Handle channels — each git operation is one session channel.
	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			_ = newChan.Reject(ssh.UnknownChannelType,
				"only session channels are supported")
			continue
		}

		ch, requests, err := newChan.Accept()
		if err != nil {
			s.logger.Warn("SSH channel accept failed", zap.Error(err))
			continue
		}

		// Each channel runs in its own goroutine; a connection may open
		// multiple channels (rare for git, but valid SSH protocol).
		perms := sshConn.Permissions
		go s.handleSession(ch, requests, perms)
	}
}
