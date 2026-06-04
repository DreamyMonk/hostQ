// Package middleware wires auth + RBAC into chi. Use RequireAuth on any
// authenticated route, RequireRole on any route that needs a specific role.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/DreamyMonk/hostQ-cloud/backend/internal/auth"
	"github.com/google/uuid"
)

type ctxKey int

const (
	ctxUserID ctxKey = iota
	ctxRole
)

func UserID(r *http.Request) (uuid.UUID, bool) {
	v, ok := r.Context().Value(ctxUserID).(uuid.UUID)
	return v, ok
}

func Role(r *http.Request) string {
	v, _ := r.Context().Value(ctxRole).(string)
	return v
}

// RequireAuth verifies the bearer token, attaches uid + role to the request
// context. Without a valid token returns 401.
func RequireAuth(svc *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tok := bearerFrom(r)
			if tok == "" {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}
			claims, err := svc.ParseAccess(tok)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ctxUserID, claims.UserID)
			ctx = context.WithValue(ctx, ctxRole, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns a middleware that 403s if the caller doesn't have one
// of the listed roles. superadmin always passes.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := Role(r)
			if role == "superadmin" {
				next.ServeHTTP(w, r)
				return
			}
			for _, want := range roles {
				if role == want {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, "forbidden", http.StatusForbidden)
		})
	}
}

func bearerFrom(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return ""
	}
	return strings.TrimSpace(h[7:])
}
