package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gforce/gforce/internal/api/middleware"
	"github.com/gforce/gforce/internal/auth"
)

type errorResponse struct {
	Error string `json:"error"`
}

// respondJSON writes v as JSON with the given HTTP status code.
func respondJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// respondError writes a JSON error body with the given status and message.
func respondError(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, errorResponse{Error: msg})
}

// claimsFromRequest extracts the authenticated JWT claims from the request context.
func claimsFromRequest(r *http.Request) (*auth.Claims, bool) {
	c := middleware.ClaimsFromContext(r.Context())
	return c, c != nil
}
