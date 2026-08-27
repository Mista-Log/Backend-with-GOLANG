package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"golang.org/x/oauth2"

	"authservice/fakeprovider"
)

// oauthStateStore tracks the "state" value for each in-flight login,
// exactly the CSRF protection the guide calls out — the callback refuses
// to proceed unless the state it receives matches one THIS server itself
// generated moments earlier.
type oauthStateStore struct {
	mu     sync.Mutex
	states map[string]bool
}

func newOAuthStateStore() *oauthStateStore {
	return &oauthStateStore{states: make(map[string]bool)}
}

func (s *oauthStateStore) generate() string {
	b := make([]byte, 16)
	rand.Read(b)
	state := hex.EncodeToString(b)
	s.mu.Lock()
	s.states[state] = true
	s.mu.Unlock()
	return state
}

func (s *oauthStateStore) consume(state string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.states[state] {
		return false
	}
	delete(s.states, state) // single-use, same spirit as the refresh tokens
	return true
}

// newOAuthConfig points a REAL, unmodified golang.org/x/oauth2.Config at
// the fake provider's endpoints instead of Google's real ones — this is
// the ONLY thing that's different from a genuine Google login integration.
func newOAuthConfig(providerURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     fakeprovider.ClientID,
		ClientSecret: fakeprovider.ClientSecret,
		RedirectURL:  "http://localhost:8080/oauth/callback",
		Scopes:       []string{"email", "profile"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  providerURL + "/authorize",
			TokenURL: providerURL + "/token",
		},
	}
}

// oauthLoginHandler is step 1 of the guide's diagram: redirect the user to
// the provider's authorize endpoint.
func oauthLoginHandler(conf *oauth2.Config, states *oauthStateStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := states.generate()
		http.Redirect(w, r, conf.AuthCodeURL(state), http.StatusFound)
	}
}

// oauthCallbackHandler is steps 3-5: verify state, exchange the code
// (step 4), fetch userinfo (step 5), then find-or-create a LOCAL account
// and issue OUR OWN token pair — from this point on, the rest of the
// application never needs to know the user logged in via OAuth at all.
func oauthCallbackHandler(conf *oauth2.Config, providerURL string, states *oauthStateStore, users *UserStore, refreshStore *RefreshTokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		if !states.consume(state) {
			writeError(w, http.StatusBadRequest, "invalid or expired state — possible CSRF attempt")
			return
		}

		code := r.URL.Query().Get("code")
		token, err := conf.Exchange(context.Background(), code) // step 4: server-to-server
		if err != nil {
			writeError(w, http.StatusBadGateway, fmt.Sprintf("token exchange failed: %v", err))
			return
		}

		// step 5: call the provider's userinfo endpoint with the access token
		req, _ := http.NewRequest(http.MethodGet, providerURL+"/userinfo", nil)
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			writeError(w, http.StatusBadGateway, "fetching userinfo failed")
			return
		}
		defer resp.Body.Close()

		var profile struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
			writeError(w, http.StatusBadGateway, "decoding userinfo failed")
			return
		}

		user, err := users.FindOrCreateByEmail(profile.Email, profile.Name)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		tokens, err := issueTokenPair(user, refreshStore)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to issue tokens")
			return
		}

		// A real app would redirect to a frontend URL with these tokens
		// attached; returning them as JSON here keeps the demo curl-able.
		writeJSON(w, http.StatusOK, map[string]any{
			"message": fmt.Sprintf("logged in via OAuth as %s (%s)", user.Username, user.Email),
			"tokens":  tokens,
		})
	}
}
