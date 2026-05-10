package handlers

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

// parseUUID parses s as a UUID.
func parseUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid UUID %q: %w", s, err)
	}
	return id, nil
}

// writeNoContent sends a 204 with no body.
func writeNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
