package gitserver

import (
	"fmt"
	"io"
)

// ServiceCapabilities returns the capability list advertised for a given git service.
// These strings are written into the first ref advertisement line after the NUL byte.
func ServiceCapabilities(service string) []string {
	switch service {
	case "git-upload-pack":
		return []string{
			"multi_ack",
			"thin-pack",
			"side-band",
			"side-band-64k",
			"ofs-delta",
			"shallow",
			"no-progress",
			"include-tag",
			"multi_ack_detailed",
			"symref=HEAD:refs/heads/main",
			"agent=git/gforce",
		}
	case "git-receive-pack":
		return []string{
			"report-status",
			"delete-refs",
			"side-band-64k",
			"quiet",
			"atomic",
			"ofs-delta",
			"agent=git/gforce",
		}
	default:
		return nil
	}
}

// WriteServiceHeader writes the smart-HTTP service discovery header:
//
//	pkt-line("# service=<service>\n")
//	flush-pkt
//
// This is the first thing written in the info/refs response body, before
// the git subprocess writes the actual ref advertisement.
func WriteServiceHeader(w io.Writer, service string) error {
	if err := PacketWrite(w, fmt.Sprintf("# service=%s\n", service)); err != nil {
		return fmt.Errorf("gitserver: writing service header: %w", err)
	}
	if err := PacketFlush(w); err != nil {
		return fmt.Errorf("gitserver: writing service flush: %w", err)
	}
	return nil
}
