// Package gitserver implements the Git smart-HTTP protocol.
package gitserver

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// PacketWrite writes a single pkt-line encoded packet to w.
// The 4-byte hex length prefix includes itself, so the minimum valid length is 4.
func PacketWrite(w io.Writer, data string) error {
	length := 4 + len(data)
	if length > 65520 {
		return fmt.Errorf("pktline: packet too large: %d bytes", length)
	}
	_, err := fmt.Fprintf(w, "%04x%s", length, data)
	return err
}

// PacketFlush writes the flush packet (0000) which terminates a pkt-line stream.
func PacketFlush(w io.Writer) error {
	_, err := io.WriteString(w, "0000")
	return err
}

// PacketRead reads one pkt-line packet from r and returns the payload without
// the 4-byte length prefix. Returns ("", nil) for a flush packet (0000).
func PacketRead(r io.Reader) (string, error) {
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return "", fmt.Errorf("pktline: reading length: %w", err)
	}

	length, err := strconv.ParseInt(string(lenBuf), 16, 32)
	if err != nil {
		return "", fmt.Errorf("pktline: invalid length %q: %w", string(lenBuf), err)
	}

	// Flush packet.
	if length == 0 {
		return "", nil
	}

	if length < 4 {
		return "", fmt.Errorf("pktline: invalid length %d (minimum is 4)", length)
	}

	payload := make([]byte, length-4)
	if _, err := io.ReadFull(r, payload); err != nil {
		return "", fmt.Errorf("pktline: reading payload: %w", err)
	}

	return string(payload), nil
}

// FormatRef formats a git ref advertisement line.
// The first line in a ref list must carry the NUL-separated capability list.
func FormatRef(name, sha string, caps []string, first bool) string {
	if first && len(caps) > 0 {
		return sha + " " + name + "\x00" + strings.Join(caps, " ") + "\n"
	}
	return sha + " " + name + "\n"
}
