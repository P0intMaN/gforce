// Package response provides the standard JSON envelope used by every API handler.
package response

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// Response is the top-level JSON envelope. Every API response is wrapped in this.
type Response[T any] struct {
	Data  T      `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
	Meta  *Meta  `json:"meta,omitempty"`
}

// Meta carries pagination metadata for list responses.
type Meta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// JSON writes data wrapped in Response[T] with the given status code.
func JSON[T any](w http.ResponseWriter, status int, data T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response[T]{Data: data})
}

// List writes a list with pagination metadata.
func List[T any](w http.ResponseWriter, status int, data T, total, limit, offset int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response[T]{
		Data: data,
		Meta: &Meta{Total: total, Limit: limit, Offset: offset},
	})
}

// Error writes a structured error response. The message is shown to the caller.
func Error(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response[struct{}]{Error: msg})
}

// InternalError logs the real error with zap and returns a generic message to
// the caller. Internal error details are never leaked to the response body.
func InternalError(w http.ResponseWriter, logger *zap.Logger, err error) {
	logger.Error("internal server error", zap.Error(err))
	Error(w, http.StatusInternalServerError, "an internal error occurred")
}

// NotFound writes a 404 response.
func NotFound(w http.ResponseWriter) {
	Error(w, http.StatusNotFound, "resource not found")
}

// Unauthorized writes a 401 response.
func Unauthorized(w http.ResponseWriter) {
	Error(w, http.StatusUnauthorized, "authentication required")
}

// Forbidden writes a 403 response.
func Forbidden(w http.ResponseWriter) {
	Error(w, http.StatusForbidden, "access denied")
}

// BadRequest writes a 400 response with the given message.
func BadRequest(w http.ResponseWriter, msg string) {
	Error(w, http.StatusBadRequest, msg)
}
