// Package fakeprovider simulates a Google-like OAuth2 provider entirely
// offline, via httptest — the same technique used for external services
// throughout this course (Module 11's API Client, Module 13's API
// Timeout). It implements the real Authorization Code flow's three
// endpoints (authorize, token, userinfo) closely enough that
// golang.org/x/oauth2 (a real, unmodified OAuth client library) can drive
// the whole flow against it without knowing it isn't talking to Google.
package fakeprovider

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

const (
	ClientID     = "fake-client-id"
	ClientSecret = "fake-client-secret"
)

// KnownUser is the one "Google account" this fake provider knows about —
// standing in for whichever real account a person would actually log into
// during a real Google OAuth flow.
var KnownUser = struct {
	Email string
	Name  string
}{Email: "ada@example.com", Name: "Ada Lovelace"}

type Server struct {
	*httptest.Server
	mu           sync.Mutex
	issuedCodes  map[string]bool   // authorization code -> still valid (single-use)
	accessTokens map[string]bool   // issued access tokens -> valid
}

func NewServer() *Server {
	s := &Server{
		issuedCodes:  make(map[string]bool),
		accessTokens: make(map[string]bool),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /authorize", s.handleAuthorize)
	mux.HandleFunc("POST /token", s.handleToken)
	mux.HandleFunc("GET /userinfo", s.handleUserinfo)

	s.Server = httptest.NewServer(mux)
	return s
}

// handleAuthorize stands in for the real provider's login+consent screen.
// A real user would see a login form and an "Allow access?" prompt here;
// since this is a fully offline, non-interactive demo, it immediately
// "approves" and redirects back — the REST of the protocol (the code
// exchange, the state check) still runs for real.
func (s *Server) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	redirectURI := r.URL.Query().Get("redirect_uri")
	state := r.URL.Query().Get("state")
	clientID := r.URL.Query().Get("client_id")

	if clientID != ClientID {
		http.Error(w, "unknown client_id", http.StatusBadRequest)
		return
	}

	code := randomToken()
	s.mu.Lock()
	s.issuedCodes[code] = true
	s.mu.Unlock()

	redirectURL := redirectURI + "?code=" + code + "&state=" + state
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// handleToken exchanges a one-time authorization code for an access
// token — the server-to-server step in the guide's OAuth diagram (step 4).
func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	code := r.Form.Get("code")
	clientID := r.Form.Get("client_id")
	clientSecret := r.Form.Get("client_secret")

	if clientID != ClientID || clientSecret != ClientSecret {
		http.Error(w, "invalid client credentials", http.StatusUnauthorized)
		return
	}

	s.mu.Lock()
	valid := s.issuedCodes[code]
	if valid {
		delete(s.issuedCodes, code) // codes are SINGLE-USE — gone after one exchange
	}
	s.mu.Unlock()

	if !valid {
		http.Error(w, "invalid or already-used code", http.StatusBadRequest)
		return
	}

	accessToken := randomToken()
	s.mu.Lock()
	s.accessTokens[accessToken] = true
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   3600,
	})
}

// handleUserinfo returns the (fake) logged-in user's profile — step 5 in
// the guide's diagram — given a valid access token from step 4.
func (s *Server) handleUserinfo(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")

	s.mu.Lock()
	valid := s.accessTokens[token]
	s.mu.Unlock()

	if !valid {
		http.Error(w, "invalid access token", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"email": KnownUser.Email,
		"name":  KnownUser.Name,
	})
}

func randomToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
