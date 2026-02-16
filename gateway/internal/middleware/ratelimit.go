package middleware

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/lychee-ripe/gateway/internal/config"
)

// tokenBucket implements a simple token-bucket rate limiter.
type tokenBucket struct {
	mu       sync.Mutex
	tokens   float64
	capacity float64
	rate     float64
	last     time.Time
}

func newTokenBucket(rate float64, burst int) *tokenBucket {
	return &tokenBucket{
		tokens:   float64(burst),
		capacity: float64(burst),
		rate:     rate,
		last:     time.Now(),
	}
}

func (tb *tokenBucket) allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.last).Seconds()
	tb.last = now
	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}

	if tb.tokens < 1 {
		return false
	}
	tb.tokens--
	return true
}

// RateLimit returns a global token-bucket rate limiting middleware.
// Paths listed in cfg.ExcludePaths are forwarded without consuming tokens.
func RateLimit(cfg config.RateLimitConfig, logger *slog.Logger) func(http.Handler) http.Handler {
	if !cfg.Enabled {
		return func(next http.Handler) http.Handler { return next }
	}

	bucket := newTokenBucket(cfg.RequestsPerSecond, cfg.Burst)

	excluded := make(map[string]struct{}, len(cfg.ExcludePaths))
	for _, p := range cfg.ExcludePaths {
		excluded[p] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := excluded[r.URL.Path]; ok {
				next.ServeHTTP(w, r)
				return
			}
			if !bucket.allow() {
				logger.Warn("rate_limit: request throttled",
					"remote", r.RemoteAddr,
					"path", r.URL.Path,
				)
				w.Header().Set("Retry-After", "1")
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
