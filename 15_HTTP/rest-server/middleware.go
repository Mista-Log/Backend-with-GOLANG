package main

import (
	"log"
	"net/http"
	"time"
)

// chain applies middlewares in the order LISTED, so the first one in the
// list is OUTERMOST (runs first on the way in, last on the way out) —
// matching the guide's Section 4 diagram exactly.
func chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// requestIDMiddleware tags every request with a unique ID, both as a
// response header (so a browser dev-tools user or client library can see
// it) and available to later middleware/handlers for correlated logging.
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := randomToken()[:8]
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware times the ENTIRE remaining chain — everything inside
// it, including CORS and auth checks, counts toward the logged duration.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%v)", r.Method, r.URL.Path, time.Since(start).Round(time.Microsecond))
	})
}

// recoverMiddleware is OUTERMOST in main.go's chain — it must wrap
// EVERYTHING else, since a panic anywhere inside any inner middleware or
// handler needs to be caught here, exactly like Module 08's panic/recover
// guidance: this is a genuine system boundary.
func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC recovered: %v", err)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware implements Section 8's preflight handling. allowedOrigin
// is configurable so a real deployment can restrict it to a known frontend
// domain instead of "*".
func corsMiddleware(allowedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == http.MethodOptions {
				// Answer the PREFLIGHT and stop — the guide's Section 8
				// warning about forgetting this early return, made explicit.
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// authMiddleware protects mutating routes (POST/PUT/DELETE) by requiring a
// valid session cookie — GET requests pass through untouched, so the API
// stays browsable without logging in first.
func authMiddleware(sessions *SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			cookie, err := r.Cookie("session")
			if err != nil {
				writeError(w, http.StatusUnauthorized, "login required")
				return
			}
			username, ok := sessions.Valid(cookie.Value)
			if !ok {
				writeError(w, http.StatusUnauthorized, "invalid or expired session")
				return
			}
			log.Printf("authenticated request from %s", username)
			next.ServeHTTP(w, r)
		})
	}
}
