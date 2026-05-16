package sshserver

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/ssh"
)

// generateEphemeralKey creates a one-time ED25519 host key.
// Used only in tests — production always loads from disk.
func generateEphemeralKey() (ssh.Signer, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating ephemeral key: %w", err)
	}
	return ssh.NewSignerFromKey(priv)
}
