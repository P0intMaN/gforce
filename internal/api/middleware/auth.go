// Package middleware provides HTTP middleware for the GForce API.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gforce/gforce/internal/auth"
	"github.com/gforce/gforce/internal/models"
	"github.com/gforce/gforce/internal/store"
)

type contextKey string

const userContextKey contextKey = "user"

// RequireAuth validates the Bearer token in the Authorization header, loads
// the associated user from the store, and stores it in the request context.
// Returns 401 if the token is missing or invalid.
func RequireAuth(tv auth.TokenValidator, users store.UserStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := extractUser(r, tv, users)
			if err != nil || user == nil {
				w.Header().Set("WWW-Authenticate", `Bearer realm="GForce"`)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"authentication required"}`))
				return
			}
			next.ServeHTTP(w, r.WithContext(ContextWithUser(r.Context(), user)))
		})
	}
}

// OptionalAuth runs the same token validation as RequireAuth but continues
// the chain even when no credentials are present. Handlers can call
// UserFromContext to check whether a user is authenticated.
func OptionalAuth(tv auth.TokenValidator, users store.UserStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, _ := extractUser(r, tv, users)
			if user != nil {
				r = r.WithContext(ContextWithUser(r.Context(), user))
			}
			next.ServeHTTP(w, r)
		})
	}
}

// UserFromContext retrieves the authenticated user from ctx.
// Returns (nil, false) for unauthenticated requests.
func UserFromContext(ctx context.Context) (*models.User, bool) {
	u, ok := ctx.Value(userContextKey).(*models.User)
	return u, ok && u != nil
}

// ContextWithUser stores user in ctx.
func ContextWithUser(ctx context.Context, u *models.User) context.Context {
	return context.WithValue(ctx, userContextKey, u)
}

// ClaimsFromContext is kept for backward compatibility with the gitserver package.
// Prefer UserFromContext for new code.
func ClaimsFromContext(ctx context.Context) *auth.Claims {
	u, ok := UserFromContext(ctx)
	if !ok {
		return nil
	}
	return &auth.Claims{UserID: u.ID.String(), Username: u.Username}
}

// extractUser parses the Authorization header and returns the associated user.
func extractUser(r *http.Request, tv auth.TokenValidator, users store.UserStore) (*models.User, error) {
	raw := r.Header.Get("Authorization")
	if raw == "" {
		return nil, nil
	}

	parts := strings.SplitN(raw, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return nil, nil
	}

	claims, err := tv.Validate(parts[1])
	if err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, err
	}

	return users.GetUserByID(r.Context(), userID)
}
