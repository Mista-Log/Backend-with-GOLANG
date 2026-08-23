package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
)

// SessionStore tracks valid session tokens — a plain map + Mutex (Module 12),
// since sessions are created/checked from many concurrent request goroutines.
type SessionStore struct {
	mu       sync.Mutex
	sessions map[string]string // token -> username
}

func NewSessionStore() *SessionStore {
	return &SessionStore{sessions: make(map[string]string)}
}

func (s *SessionStore) Create(username string) string {
	token := randomToken()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[token] = username
	return token
}

func (s *SessionStore) Valid(token string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	username, ok := s.sessions[token]
	return username, ok
}

func randomToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// loginHandler is deliberately simplified — any non-empty username/password
// "succeeds," since the point of this project is the HTTP/cookie mechanics,
// not real credential verification.
func loginHandler(sessions *SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
			writeError(w, http.StatusBadRequest, "username and password are required")
			return
		}

		token := sessions.Create(req.Username)

		// Section 6's Cookie diagram, applied for real: the browser will
		// store this and automatically resend it on every future request
		// to this server.
		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    token,
			Path:     "/",
			HttpOnly: true,                 // JavaScript cannot read this cookie
			SameSite: http.SameSiteLaxMode,   // mitigates cross-site request forgery
			MaxAge:   3600,
		})

		writeJSON(w, http.StatusOK, map[string]string{"message": "logged in as " + req.Username})
	}
}
