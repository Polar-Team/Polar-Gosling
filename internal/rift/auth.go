package rift

import (
	"crypto/subtle"
	"errors"
	"net/http"
)

// ErrUnauthorized is returned when a request fails authentication.
var ErrUnauthorized = errors.New("rift: unauthorized")

// Authenticator validates runner requests against the configured shared secret.
type Authenticator struct {
	// token is the expected bearer token value (resolved from secret URI at startup).
	token []byte
}

// NewAuthenticator creates an Authenticator with the given resolved token value.
func NewAuthenticator(token string) *Authenticator {
	return &Authenticator{token: []byte(token)}
}

// Authenticate checks the Authorization header of r.
// It expects the format: "Bearer <token>"
// Uses constant-time comparison to prevent timing attacks.
func (a *Authenticator) Authenticate(r *http.Request) error {
	const prefix = "Bearer "
	header := r.Header.Get("Authorization")
	if len(header) <= len(prefix) {
		return ErrUnauthorized
	}
	provided := []byte(header[len(prefix):])
	if subtle.ConstantTimeCompare(provided, a.token) != 1 {
		return ErrUnauthorized
	}
	return nil
}

// Middleware returns an http.Handler that enforces authentication before
// delegating to next. Unauthenticated requests receive 401 Unauthorized.
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := a.Authenticate(r); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
