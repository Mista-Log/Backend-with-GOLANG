package main

import (
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// IPRateLimiter holds one token-bucket limiter PER CLIENT IP — the guide's
// note that per-client rate limiting needs one limiter per identity, not
// one shared globally. Protected by a Mutex (Module 12): many request
// goroutines look up/create limiters concurrently.
type IPRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	r        rate.Limit
	burst    int
}

func NewIPRateLimiter(requestsPerSecond float64, burst int) *IPRateLimiter {
	return &IPRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		r:        rate.Limit(requestsPerSecond),
		burst:    burst,
	}
}

func (rl *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	limiter, exists := rl.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.r, rl.burst)
		rl.limiters[ip] = limiter
	}
	return limiter
}

func rateLimitMiddleware(rl *IPRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr // fall back to the raw value if parsing fails
			}

			limiter := rl.getLimiter(ip)
			if !limiter.Allow() {
				w.Header().Set("Retry-After", "1")
				http.Error(w, `{"error":"rate limit exceeded, slow down"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// loggingMiddleware uses log/slog (standard library, Go 1.21+) for
// STRUCTURED (JSON) logs, per the guide's Logging section — real log
// aggregation tools can parse this directly, unlike free-text log lines.
func loggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"query", r.URL.RawQuery,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}
