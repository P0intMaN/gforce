// Package sshserver implements the GForce SSH daemon for git-over-SSH.
package sshserver

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

// LoadOrGenerateHostKey loads an ED25519 host key from path.
// If the file does not exist it generates a new key, writes it at path
// with mode 0600, and returns it. The key is NEVER regenerated if the
// file already exists — regenerating would cause "host key changed"
// warnings on every developer's machine.
func LoadOrGenerateHostKey(path string) (ssh.Signer, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return parseHostKey(data)
	}

	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("sshserver: reading host key %q: %w", path, err)
	}

	// Key file doesn't exist — generate and persist.
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("sshserver: generating host key: %w", err)
	}

	privBytes, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return nil, fmt.Errorf("sshserver: marshaling host key: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("sshserver: creating host key directory: %w", err)
	}
	if err := os.WriteFile(path, pem.EncodeToMemory(privBytes), 0o600); err != nil {
		return nil, fmt.Errorf("sshserver: writing host key: %w", err)
	}

	return ssh.NewSignerFromKey(priv)
}

func parseHostKey(data []byte) (ssh.Signer, error) {
	signer, err := ssh.ParsePrivateKey(data)
	if err != nil {
		return nil, fmt.Errorf("sshserver: parsing host key: %w", err)
	}
	return signer, nil
}
