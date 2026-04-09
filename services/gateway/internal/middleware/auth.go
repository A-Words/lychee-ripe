package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
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

type SessionResolver interface {
	ResolveSessionPrincipal(ctx context.Context, sessionID string) (domain.Principal, error)
}

func Auth(
	cfg config.AuthConfig,
	corsCfg config.CORSConfig,
	validator TokenValidator,
	resolver AuthResolver,
	sessionResolver SessionResolver,
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
			if sessionID := sessionCookieValue(r, cfg.Web.CookieName); sessionID != "" {
				if err := enforceCookieOrigin(r, corsCfg); err != nil {
					logger.Warn("auth: rejected cookie-authenticated request due to origin policy",
						"path", r.URL.Path,
						"method", r.Method,
						"origin", strings.TrimSpace(r.Header.Get("Origin")),
					)
					writeAuthError(w, http.StatusForbidden, "forbidden", err.Error())
					return
				}
				if sessionResolver == nil {
					writeAuthError(w, http.StatusServiceUnavailable, "auth_unavailable", "auth unavailable")
					return
				}
				principal, err := sessionResolver.ResolveSessionPrincipal(r.Context(), sessionID)
				if err != nil {
					switch {
					case errors.Is(err, service.ErrNotFound):
						writeAuthError(w, http.StatusUnauthorized, "unauthorized", "invalid session")
					case errors.Is(err, service.ErrInvalidRequest):
						writeAuthError(w, http.StatusForbidden, "forbidden", "user is disabled")
					default:
						logger.Error("auth: resolve session principal failed", "error", err)
						writeAuthError(w, http.StatusServiceUnavailable, "auth_unavailable", "auth unavailable")
					}
					return
				}
				if !isAuthorized(r, principal.Role) {
					writeAuthError(w, http.StatusForbidden, "forbidden", "insufficient role")
					return
				}
				next.ServeHTTP(w, stripCookie(r.WithContext(WithPrincipal(r.Context(), principal)), cfg.Web.CookieName))
				return
			}
			token := bearerToken(r.Header.Get("Authorization"))
			if token == "" && strings.HasPrefix(r.URL.Path, "/v1/infer/stream") {
				token = strings.TrimSpace(r.URL.Query().Get("access_token"))
				if token != "" {
					r = stripQueryToken(r, "access_token")
				}
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
			if !isAuthorized(r, principal.Role) {
				writeAuthError(w, http.StatusForbidden, "forbidden", "insufficient role")
				return
			}
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
		})
	}
}

func enforceCookieOrigin(r *http.Request, corsCfg config.CORSConfig) error {
	if !requiresCookieOriginCheck(r) {
		return nil
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return errors.New("origin header required for cookie-authenticated unsafe requests")
	}
	if !corsCfg.AllowsOrigin(origin) {
		return errors.New("origin is not allowed for cookie-authenticated request")
	}
	return nil
}

func requiresCookieOriginCheck(r *http.Request) bool {
	if r == nil {
		return false
	}
	if isWebSocketRequest(r) {
		return true
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

func isWebSocketRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	return headerContainsToken(r.Header, "Connection", "upgrade") &&
		headerContainsToken(r.Header, "Upgrade", "websocket")
}

func headerContainsToken(header http.Header, key string, want string) bool {
	values := header.Values(key)
	for _, value := range values {
		for _, token := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(token), want) {
				return true
			}
		}
	}
	return false
}

func stripQueryToken(r *http.Request, key string) *http.Request {
	if r == nil || r.URL == nil {
		return r
	}
	query := r.URL.Query()
	if strings.TrimSpace(query.Get(key)) == "" {
		return r
	}

	cloned := r.Clone(r.Context())
	nextURL := *r.URL
	query.Del(key)
	nextURL.RawQuery = query.Encode()
	cloned.URL = &nextURL
	cloned.RequestURI = nextURL.RequestURI()
	return cloned
}

func stripCookie(r *http.Request, cookieName string) *http.Request {
	if r == nil {
		return r
	}
	cookieName = strings.TrimSpace(cookieName)
	if cookieName == "" {
		return r
	}
	header := strings.TrimSpace(r.Header.Get("Cookie"))
	if header == "" {
		return r
	}
	parts := strings.Split(header, ";")
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		piece := strings.TrimSpace(part)
		if piece == "" {
			continue
		}
		if strings.HasPrefix(piece, cookieName+"=") {
			continue
		}
		filtered = append(filtered, piece)
	}

	cloned := r.Clone(r.Context())
	if len(filtered) == 0 {
		cloned.Header.Del("Cookie")
		return cloned
	}
	cloned.Header.Set("Cookie", strings.Join(filtered, "; "))
	return cloned
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
	case "/healthz", "/v1/health", "/v1/auth/login", "/v1/auth/callback", "/v1/auth/logout":
		return true
	}
	return strings.HasPrefix(path, "/v1/trace/")
}

func isAuthorized(r *http.Request, role domain.UserRole) bool {
	path := r.URL.Path
	method := r.Method
	if role == domain.UserRoleAdmin {
		return true
	}
	if method == http.MethodGet && path == "/v1/auth/me" {
		return true
	}
	if method == http.MethodGet && (path == "/v1/orchards" || path == "/v1/plots") {
		return !queryBool(r, "include_archived")
	}
	switch {
	case strings.HasPrefix(path, "/v1/infer/"),
		path == "/v1/models/current",
		(method == http.MethodPost && path == "/v1/batches"),
		(method == http.MethodGet && isBatchItemPath(path)),
		path == "/v1/dashboard/overview":
		return true
	default:
		return false
	}
}

func queryBool(r *http.Request, key string) bool {
	if r == nil || r.URL == nil {
		return false
	}
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return false
	}
	parsed, err := strconv.ParseBool(value)
	return err == nil && parsed
}

func isBatchItemPath(path string) bool {
	const prefix = "/v1/batches/"
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(path, prefix))
	if rest == "" || rest == "reconcile" || strings.Contains(rest, "/") {
		return false
	}
	return true
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

func sessionCookieValue(r *http.Request, cookieName string) string {
	cookieName = strings.TrimSpace(cookieName)
	if r == nil || cookieName == "" {
		return ""
	}
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

func writeAuthError(w http.ResponseWriter, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error":   code,
		"message": message,
	})
}
