// Authentication Service — ties together every section of Module 18's
// guide: bcrypt password hashing, JWT access tokens, rotating refresh
// tokens with reuse detection, RBAC and permission-based middleware, and a
// full OAuth2 Authorization Code flow against a locally simulated
// provider (fakeprovider/).
//
// Run with: go mod tidy && go run .
// Then see README.md for a full curl walkthrough.
package main

import (
	"fmt"
	"log"
	"net/http"

	"authservice/fakeprovider"
)

func main() {
	users := NewUserStore()
	refreshStore := NewRefreshTokenStore()

	// Seed one admin account so the RBAC/permission routes are reachable
	// without extra setup — see README.md for the exact curl commands.
	adminUser, _ := users.Register("admin", "adminpass123", "admin@example.com")
	adminUser.Role = RoleAdmin

	// The fake OAuth provider — see fakeprovider/provider.go. In a real
	// deployment, oauthConfig would point at Google's actual endpoints
	// instead (golang.org/x/oauth2/google.Endpoint) — everything else in
	// oauth.go is unchanged either way.
	provider := fakeprovider.NewServer()
	defer provider.Close()
	oauthConfig := newOAuthConfig(provider.URL)
	states := newOAuthStateStore()

	mux := http.NewServeMux()

	mux.HandleFunc("POST /register", registerHandler(users))
	mux.HandleFunc("POST /login", loginHandler(users, refreshStore))
	mux.HandleFunc("POST /refresh", refreshHandler(users, refreshStore))
	mux.HandleFunc("POST /logout", logoutHandler(refreshStore))

	mux.HandleFunc("GET /oauth/login", oauthLoginHandler(oauthConfig, states))
	mux.HandleFunc("GET /oauth/callback", oauthCallbackHandler(oauthConfig, provider.URL, states, users, refreshStore))

	// Protected routes — authMiddleware first (identity), then a SECOND,
	// route-specific gate (role or permission) layered on top.
	mux.Handle("GET /profile", authMiddleware(profileHandler(users)))
	mux.Handle("GET /admin/users", authMiddleware(requireRole(RoleAdmin)(adminUsersHandler(users))))
	mux.Handle("POST /admin/products", authMiddleware(requirePermission("products:write")(createProductHandler())))

	fmt.Println("Authentication Service listening on http://localhost:8080")
	fmt.Println("Fake OAuth provider running at", provider.URL, "(fully offline)")
	fmt.Println("Seeded admin account: username=admin password=adminpass123")
	fmt.Println("See README.md for a full curl walkthrough.")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
