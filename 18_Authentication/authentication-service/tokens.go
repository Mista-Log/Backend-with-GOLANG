package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// In a real deployment this comes from an environment variable / secrets
// manager — hardcoded here only because this is a self-contained demo.
var jwtSecret = []byte("demo-secret-change-me-in-real-life")

const accessTokenTTL = 15 * time.Minute

type AccessClaims struct {
	UserID int  `json:"sub"`
	Role   Role `json:"role"`
	jwt.RegisteredClaims
}

// GenerateAccessToken issues a short-lived, signed JWT — the guide's
// Section 1, applied. No database lookup is needed to verify this later;
// the signature alone proves it's authentic and unmodified.
func GenerateAccessToken(u *User) (string, error) {
	claims := AccessClaims{
		UserID: u.ID,
		Role:   u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ParseAccessToken(tokenString string) (*AccessClaims, error) {
	claims := &AccessClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("token is not valid")
	}
	return claims, nil
}

// --- Refresh tokens: opaque, server-tracked, rotating -----------------

type refreshRecord struct {
	userID    int
	familyID  string // every token descended from one login shares a family
	used      bool
	expiresAt time.Time
}

// RefreshTokenStore implements rotation + reuse detection exactly as
// diagrammed in the guide: each token works ONCE, and reusing an
// already-used token revokes its ENTIRE family, not just itself.
type RefreshTokenStore struct {
	mu      sync.Mutex
	records map[string]*refreshRecord
}

func NewRefreshTokenStore() *RefreshTokenStore {
	return &RefreshTokenStore{records: make(map[string]*refreshRecord)}
}

func randomToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// IssueNewFamily is called at LOGIN — starts a brand new token family.
func (s *RefreshTokenStore) IssueNewFamily(userID int) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	token := randomToken()
	familyID := randomToken()
	s.records[token] = &refreshRecord{
		userID:    userID,
		familyID:  familyID,
		expiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	return token
}

// Rotate is called at REFRESH time: validates the presented token, and —
// if it's genuinely unused and unexpired — marks it used and issues a new
// token in the SAME family. If it's ALREADY used, that's reuse: the whole
// family is revoked immediately (the guide's theft-detection diagram).
func (s *RefreshTokenStore) Rotate(oldToken string) (newToken string, userID int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.records[oldToken]
	if !ok {
		return "", 0, fmt.Errorf("unknown refresh token")
	}
	if time.Now().After(record.expiresAt) {
		return "", 0, fmt.Errorf("refresh token expired")
	}
	if record.used {
		// REUSE DETECTED — this exact scenario is what the guide's
		// "theft scenario" diagram describes. Revoke every token in this
		// family, forcing a fresh login even for the legitimate user.
		s.revokeFamilyLocked(record.familyID)
		return "", 0, fmt.Errorf("refresh token reuse detected — entire session family revoked")
	}

	record.used = true
	newTok := randomToken()
	s.records[newTok] = &refreshRecord{
		userID:    record.userID,
		familyID:  record.familyID, // SAME family — rotation, not a new login
		expiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	return newTok, record.userID, nil
}

func (s *RefreshTokenStore) revokeFamilyLocked(familyID string) {
	for token, record := range s.records {
		if record.familyID == familyID {
			delete(s.records, token)
		}
	}
}

// RevokeByToken is called at LOGOUT — revokes just this one token's family
// (in practice, usually just itself, since logout typically happens right
// after a normal rotation, before any further refresh has occurred).
func (s *RefreshTokenStore) RevokeByToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if record, ok := s.records[token]; ok {
		s.revokeFamilyLocked(record.familyID)
	}
}
