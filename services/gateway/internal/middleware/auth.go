package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/lychee-ripe/gateway/internal/config"
	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/oidc"
	"github.com/lychee-ripe/gateway/internal/service"
)

type principalContextKey struct{}

type AuthResolver interface {
	ResolvePrincipal(ctx context.Context, identity domain.IdentityClaims, mode domain.AuthMode) (domain.Principal, error)
}

type TokenValidator interface {
	Validate(ctx context.Context, rawToken string) (domain.IdentityClaims, error)
}

func Auth(
	cfg config.AuthConfig,
	validator TokenValidator,
	resolver AuthResolver,
	logger *slog.Logger,
) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	if cfg.Mode == config.AuthModeDisabled {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if isPublicPath(r.URL.Path) {
					next.ServeHTTP(w, r)
					return
				}
				principal := domain.Principal{
					Subject:     "dev-admin",
					Email:       "dev-admin@local",
					DisplayName: "Dev Admin",
					Role:        domain.UserRoleAdmin,
					Status:      domain.UserStatusActive,
					AuthMode:    domain.AuthModeDisabled,
				}
				next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
			})
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isPublicPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			token := bearerToken(r.Header.Get("Authorization"))
			if token == "" && strings.HasPrefix(r.URL.Path, "/v1/infer/stream") {
				token = strings.TrimSpace(r.URL.Query().Get("access_token"))
			}
			if token == "" {
				writeAuthError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
				return
			}
			if validator == nil || resolver == nil {
				writeAuthError(w, http.StatusServiceUnavailable, "auth_unavailable", "auth unavailable")
				return
			}
			identity, err := validator.Validate(r.Context(), token)
			if err != nil {
				if errors.Is(err, oidc.ErrUnavailable) {
					logger.Error("auth: validate token failed", "error", err)
					writeAuthError(w, http.StatusServiceUnavailable, "auth_unavailable", "auth unavailable")
					return
				}
				writeAuthError(w, http.StatusUnauthorized, "unauthorized", "invalid bearer token")
				return
			}
			principal, err := resolver.ResolvePrincipal(r.Context(), identity, domain.AuthModeOIDC)
			if err != nil {
				switch {
				case errors.Is(err, service.ErrNotFound):
					writeAuthError(w, http.StatusForbidden, "forbidden", "user is not provisioned")
				case errors.Is(err, service.ErrInvalidRequest):
					writeAuthError(w, http.StatusForbidden, "forbidden", "user is disabled")
				default:
					logger.Error("auth: resolve principal failed", "error", err)
					writeAuthError(w, http.StatusServiceUnavailable, "auth_unavailable", "auth unavailable")
				}
				return
			}
			if !isAuthorized(r.URL.Path, r.Method, principal.Role) {
				writeAuthError(w, http.StatusForbidden, "forbidden", "insufficient role")
				return
			}
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
		})
	}
}

func WithPrincipal(ctx context.Context, principal domain.Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func GetPrincipal(ctx context.Context) (domain.Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(domain.Principal)
	return principal, ok
}

func isPublicPath(path string) bool {
	switch path {
	case "/healthz", "/v1/health":
		return true
	}
	return strings.HasPrefix(path, "/v1/trace/")
}

func isAuthorized(path, method string, role domain.UserRole) bool {
	if role == domain.UserRoleAdmin {
		return true
	}
	if method == http.MethodGet && (path == "/v1/orchards" || path == "/v1/plots" || path == "/v1/auth/me") {
		return true
	}
	switch {
	case strings.HasPrefix(path, "/v1/infer/"),
		path == "/v1/models/current",
		(method == http.MethodPost && path == "/v1/batches"),
		(method == http.MethodGet && strings.HasPrefix(path, "/v1/batches/") && path != "/v1/batches/reconcile"),
		path == "/v1/dashboard/overview":
		return true
	default:
		return false
	}
}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func writeAuthError(w http.ResponseWriter, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error":   code,
		"message": message,
	})
}
