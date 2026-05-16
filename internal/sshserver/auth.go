package sshserver

import (
	"context"
	"fmt"

	"golang.org/x/crypto/ssh"
	"go.uber.org/zap"
)

// publicKeyCallback is the ssh.ServerConfig.PublicKeyCallback implementation.
// It looks up the offered public key's fingerprint in the ssh_keys table,
// resolves the owning user, and stores the username in the connection
// permissions so the session handler can use it without another DB round-trip.

func remoteAddr(conn ssh.ConnMetadata) string {
	if conn == nil {
		return "<test>"
	}
	return conn.RemoteAddr().String()
}

func connUser(conn ssh.ConnMetadata) string {
	if conn == nil {
		return "<test>"
	}
	return conn.User()
}

func (s *SSHServer) publicKeyCallback(
	conn ssh.ConnMetadata,
	key ssh.PublicKey,
) (*ssh.Permissions, error) {
	fingerprint := ssh.FingerprintSHA256(key)

	sshKey, err := s.store.GetSSHKeyByFingerprint(context.Background(), fingerprint)
	if err != nil {
		s.logger.Info("SSH auth rejected — unknown key",
			zap.String("fingerprint", fingerprint),
			zap.String("remote_addr", remoteAddr(conn)),
		)
		return nil, fmt.Errorf("unknown key")
	}

	user, err := s.store.GetUserByID(context.Background(), sshKey.UserID)
	if err != nil {
		s.logger.Error("SSH auth — user not found for key",
			zap.String("fingerprint", fingerprint),
			zap.String("user_id", sshKey.UserID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("user not found")
	}

	s.logger.Info("SSH auth accepted",
		zap.String("username", user.Username),
		zap.String("fingerprint", fingerprint),
		zap.String("remote_addr", remoteAddr(conn)),
	)

	return &ssh.Permissions{
		Extensions: map[string]string{
			"username": user.Username,
			"user_id":  user.ID.String(),
		},
	}, nil
}

// authLogCallback logs every authentication attempt for audit purposes.
func (s *SSHServer) authLogCallback(
	conn ssh.ConnMetadata,
	method string,
	err error,
) {
	if err != nil {
		s.logger.Debug("SSH auth attempt failed",
			zap.String("method", method),
			zap.String("user", connUser(conn)),
			zap.String("remote_addr", remoteAddr(conn)),
			zap.Error(err),
		)
	}
}
