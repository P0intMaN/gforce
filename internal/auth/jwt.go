// Package auth handles JWT token issuance and validation.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenValidator is the interface for validating bearer tokens.
// *Service satisfies it automatically.
type TokenValidator interface {
	Validate(token string) (*Claims, error)
}

// Claims are the custom JWT claims embedded in every gforce token.
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Service issues and validates JWTs using a shared HMAC secret.
type Service struct {
	secret []byte
	ttl    time.Duration
}

// NewService creates an auth.Service. secret must be non-empty.
func NewService(secret string, ttl time.Duration) (*Service, error) {
	if secret == "" {
		return nil, errors.New("jwt secret must not be empty")
	}
	return &Service{secret: []byte(secret), ttl: ttl}, nil
}

// Issue creates a signed JWT for the given user. The token is valid for the
// duration specified at construction.
func (s *Service) Issue(userID, username string) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
			Issuer:    "gforce",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("signing jwt: %w", err)
	}
	return signed, nil
}

// Validate parses and verifies the token string, returning the embedded Claims.
// It returns an error if the token is malformed, expired, or has an invalid signature.
func (s *Service) Validate(tokenString string) (*Claims, error) {
	var claims Claims
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))

	if err != nil {
		return nil, fmt.Errorf("parsing jwt: %w", err)
	}
	if !token.Valid {
		return nil, errors.New("token is not valid")
	}
	return &claims, nil
}
