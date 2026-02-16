package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/lychee-ripe/gateway/internal/config"
)

// Auth returns an API key authentication middleware.
// Keys are checked against the X-API-Key header or the Authorization bearer token.
func Auth(cfg config.AuthConfig, logger *slog.Logger) func(http.Handler) http.Handler {
	if !cfg.Enabled {
		return func(next http.Handler) http.Handler { return next }
	}

	if len(cfg.APIKeys) == 0 {
		logger.Error("auth: enabled with no api_keys configured, rejecting all requests")
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, `{"error":"auth misconfigured"}`, http.StatusUnauthorized)
			})
		}
	}

	keys := make(map[string]struct{}, len(cfg.APIKeys))
	for _, k := range cfg.APIKeys {
		keys[k] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-API-Key")
			if key == "" {
				if bearer := r.Header.Get("Authorization"); bearer != "" {
					const prefix = "Bearer "
					if strings.HasPrefix(bearer, prefix) {
						key = strings.TrimPrefix(bearer, prefix)
					}
				}
			}

			if _, ok := keys[key]; !ok {
				logger.Warn("auth: unauthorized request",
					"remote", r.RemoteAddr,
					"path", r.URL.Path,
				)
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
