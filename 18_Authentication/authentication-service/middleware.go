package main

import (
	"context"
	"net/http"
	"strings"
)

// rolePermissions is the "roles as named bundles of permissions" idea from
// the guide — RoleAdmin gets everything RoleUser has, plus more, without
// duplicating logic anywhere authorization is actually checked.
var rolePermissions = map[Role][]string{
	RoleUser:  {"orders:read"},
	RoleAdmin: {"orders:read", "orders:write", "products:read", "products:write", "users:read"},
}

func hasPermission(role Role, permission string) bool {
	for _, p := range rolePermissions[role] {
		if p == permission {
			return true
		}
	}
	return false
}

// contextKey follows Module 13's dedicated-key-type rule — prevents any
// other package's context.WithValue calls from colliding with this one.
type contextKey string

const claimsContextKey contextKey = "claims"

func claimsFromContext(ctx context.Context) *AccessClaims {
	claims, _ := ctx.Value(claimsContextKey).(*AccessClaims)
	return claims
}

// authMiddleware is the FIRST gate: is there a valid, signed, unexpired
// access token at all? It doesn't check ROLE or PERMISSION — just identity.
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		tokenString := strings.TrimPrefix(header, "Bearer ")

		claims, err := ParseAccessToken(tokenString)
		if err != nil {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}

		// Module 13's propagation habit: derive a new context carrying the
		// claims, pass THAT down — never a struct field, never a global.
		ctx := context.WithValue(r.Context(), claimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireRole is a SECOND gate, layered on top of authMiddleware — RBAC,
// the coarse "is this user in the right role at all" check.
func requireRole(role Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := claimsFromContext(r.Context())
			if claims == nil || claims.Role != role {
				writeError(w, http.StatusForbidden, "requires role: "+string(role))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// requirePermission is the FINER-GRAINED alternative — the guide's
// contrast between RBAC and permissions, both implemented and both used
// in handlers.go so the difference is visible in real routes, not just
// described.
func requirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := claimsFromContext(r.Context())
			if claims == nil || !hasPermission(claims.Role, permission) {
				writeError(w, http.StatusForbidden, "requires permission: "+permission)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
