package main

import (
	"fmt"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type User struct {
	ID           int
	Username     string
	Email        string
	PasswordHash string // empty for OAuth-only accounts — never store a real password for those
	Role         Role
}

type UserStore struct {
	mu       sync.Mutex
	users    map[int]*User
	byName   map[string]int
	byEmail  map[string]int
	nextID   int
}

func NewUserStore() *UserStore {
	return &UserStore{
		users:   make(map[int]*User),
		byName:  make(map[string]int),
		byEmail: make(map[string]int),
		nextID:  1,
	}
}

// Register hashes the password with bcrypt BEFORE storing anything — the
// plaintext password never touches disk or memory beyond this function.
func (s *UserStore) Register(username, password, email string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.byName[username]; exists {
		return nil, fmt.Errorf("username %q already taken", username)
	}

	// bcrypt.DefaultCost (10) balances hashing speed against brute-force
	// resistance — high enough to make guessing passwords via repeated
	// hashing expensive, low enough that a real login doesn't feel slow.
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	u := &User{ID: s.nextID, Username: username, Email: email, PasswordHash: string(hash), Role: RoleUser}
	s.users[u.ID] = u
	s.byName[username] = u.ID
	if email != "" {
		s.byEmail[email] = u.ID
	}
	s.nextID++
	return u, nil
}

// Authenticate verifies a plaintext password against the stored bcrypt
// hash — bcrypt.CompareHashAndPassword does this WITHOUT ever needing to
// reverse the hash; it re-hashes the candidate and compares safely.
func (s *UserStore) Authenticate(username, password string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, ok := s.byName[username]
	if !ok {
		return nil, fmt.Errorf("invalid username or password") // deliberately vague —
	}                                                             // never reveal WHICH part was wrong
	u := s.users[id]

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid username or password")
	}
	return u, nil
}

func (s *UserStore) GetByID(id int) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[id]
	if !ok {
		return nil, fmt.Errorf("no user with id %d", id)
	}
	return u, nil
}

// FindOrCreateByEmail is used by the OAuth callback — a first-time Google
// login creates a local account automatically; subsequent logins match
// the existing one by email, with no password set at all.
func (s *UserStore) FindOrCreateByEmail(email, displayName string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id, ok := s.byEmail[email]; ok {
		return s.users[id], nil
	}

	u := &User{ID: s.nextID, Username: displayName, Email: email, Role: RoleUser} // no PasswordHash — OAuth-only
	s.users[u.ID] = u
	s.byEmail[email] = u.ID
	s.nextID++
	return u, nil
}
