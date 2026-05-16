package sshserver

import (
	"encoding/binary"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

// parseGitCommand decodes an SSH "exec" request payload.
//
// Git encodes the command as a 4-byte big-endian length followed by the
// command string:
//
//	\x00\x00\x00\x1fgit-upload-pack '/alice/repo.git'
//
// Returns the command (e.g. "git-upload-pack") and the raw repo path
// argument (e.g. "/alice/repo.git", quotes stripped).
func parseGitCommand(req *ssh.Request) (command, repoPath string, err error) {
	if len(req.Payload) < 4 {
		return "", "", fmt.Errorf("exec payload too short (%d bytes)", len(req.Payload))
	}

	length := binary.BigEndian.Uint32(req.Payload[:4])
	if int(length) > len(req.Payload)-4 {
		return "", "", fmt.Errorf("exec payload length mismatch: declared %d, have %d",
			length, len(req.Payload)-4)
	}

	cmdStr := string(req.Payload[4 : 4+length])
	if cmdStr == "" {
		return "", "", fmt.Errorf("empty exec command")
	}

	// Split on the first space: "git-upload-pack '/alice/repo.git'"
	parts := strings.SplitN(cmdStr, " ", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected exec format: %q", cmdStr)
	}

	command = parts[0]
	repoPath = strings.Trim(parts[1], "'\"")
	return command, repoPath, nil
}

// parseRepoPath normalises the repo path sent by git and returns the
// owner and repository name.
//
// Examples:
//
//	"/alice/repo.git"   → ("alice", "repo")
//	"alice/repo.git"    → ("alice", "repo")
//	"alice/repo"        → ("alice", "repo")
//	"/alice/my-repo.git" → ("alice", "my-repo")
func parseRepoPath(raw string) (owner, repo string) {
	path := strings.TrimPrefix(raw, "/")
	path = strings.TrimSuffix(path, ".git")

	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return path, ""
	}
	return parts[0], parts[1]
}

// sendError writes msg to the channel's stderr and closes the channel.
func sendError(ch ssh.Channel, msg string) {
	_, _ = ch.Stderr().Write([]byte("error: " + msg + "\n"))
	_ = ch.Close()
}

// sendExitStatus sends an "exit-status" SSH request on the channel so
// the client's git process receives the subprocess exit code.
func sendExitStatus(ch ssh.Channel, code uint32) {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, code)
	_, _ = ch.SendRequest("exit-status", false, payload)
}
