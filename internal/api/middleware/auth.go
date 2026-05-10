package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gforce/gforce/internal/auth"
)

type contextKey string

// ClaimsKey is the context key under which validated JWT claims are stored.
const ClaimsKey contextKey = "jwt_claims"

// Authenticator validates Bearer tokens on incoming requests.
type Authenticator struct {
	svc *auth.Service
}

// NewAuthenticator creates an Authenticator backed by the given auth.Service.
func NewAuthenticator(svc *auth.Service) *Authenticator {
	return &Authenticator{svc: svc}
}

// Require is HTTP middleware that rejects requests without a valid Bearer token.
// On success it stores the parsed *auth.Claims in the request context under ClaimsKey.
func (a *Authenticator) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get("Authorization")
		if raw == "" {
			http.Error(w, "authorization header required", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(raw, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			http.Error(w, "malformed authorization header", http.StatusUnauthorized)
			return
		}

		claims, err := a.svc.Validate(parts[1])
		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ClaimsFromContext retrieves the authenticated claims from ctx.
// Returns nil if the context carries no claims (unauthenticated request).
func ClaimsFromContext(ctx context.Context) *auth.Claims {
	c, _ := ctx.Value(ClaimsKey).(*auth.Claims)
	return c
}
