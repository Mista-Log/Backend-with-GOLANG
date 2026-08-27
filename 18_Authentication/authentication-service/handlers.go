package main

import (
	"encoding/json"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

type authResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// issueTokenPair is shared by login, OAuth callback, and refresh — one
// place that always produces a matched access+refresh pair.
func issueTokenPair(user *User, refreshStore *RefreshTokenStore) (authResponse, error) {
	access, err := GenerateAccessToken(user)
	if err != nil {
		return authResponse{}, err
	}
	refresh := refreshStore.IssueNewFamily(user.ID)
	return authResponse{AccessToken: access, RefreshToken: refresh}, nil
}

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

func registerHandler(users *UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req registerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Username == "" || req.Password == "" {
			writeError(w, http.StatusBadRequest, "username and password are required")
			return
		}
		user, err := users.Register(req.Username, req.Password, req.Email)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"id": user.ID, "username": user.Username})
	}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func loginHandler(users *UserStore, refreshStore *RefreshTokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		user, err := users.Authenticate(req.Username, req.Password)
		if err != nil {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		tokens, err := issueTokenPair(user, refreshStore)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to issue tokens")
			return
		}
		writeJSON(w, http.StatusOK, tokens)
	}
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

func refreshHandler(users *UserStore, refreshStore *RefreshTokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req refreshRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		newRefresh, userID, err := refreshStore.Rotate(req.RefreshToken)
		if err != nil {
			// Includes the reuse-detection case from tokens.go — either way,
			// the client must log in again.
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		user, err := users.GetByID(userID)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "user no longer exists")
			return
		}
		newAccess, err := GenerateAccessToken(user)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to issue token")
			return
		}
		writeJSON(w, http.StatusOK, authResponse{AccessToken: newAccess, RefreshToken: newRefresh})
	}
}

func logoutHandler(refreshStore *RefreshTokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req refreshRequest
		json.NewDecoder(r.Body).Decode(&req)
		refreshStore.RevokeByToken(req.RefreshToken)
		w.WriteHeader(http.StatusNoContent)
	}
}

// profileHandler needs ONLY authMiddleware — any logged-in user, any role.
func profileHandler(users *UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := claimsFromContext(r.Context())
		user, err := users.GetByID(claims.UserID)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": user.ID, "username": user.Username, "role": user.Role})
	}
}

// adminUsersHandler is protected by requireRole(RoleAdmin) — RBAC: a
// coarse "must be an admin" check.
func adminUsersHandler(users *UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"message": "here is the full user list (admin only)"})
	}
}

// createProductHandler is protected by requirePermission("products:write")
// instead — the guide's finer-grained alternative, demonstrated on a
// DIFFERENT route than the role-based one above so both are visible.
func createProductHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusCreated, map[string]string{"message": "product created (requires products:write permission)"})
	}
}
