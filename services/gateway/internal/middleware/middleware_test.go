package middleware

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lychee-ripe/gateway/internal/config"
	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/service"
)

func TestAuthDisabled(t *testing.T) {
	cfg := config.AuthConfig{Mode: config.AuthModeDisabled}
	mw := Auth(cfg, nil, nil, slog.Default())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := GetPrincipal(r.Context())
		if !ok {
			t.Fatal("expected principal in disabled mode")
		}
		if principal.Role != domain.UserRoleAdmin {
			t.Fatalf("principal role = %q, want admin", principal.Role)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/dashboard/overview", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 with auth disabled, got %d", rec.Code)
	}
}

func TestAuthOIDCMissingBearerRejects(t *testing.T) {
	cfg := config.AuthConfig{Mode: config.AuthModeOIDC}
	mw := Auth(cfg, fakeValidator{}, fakeResolver{}, slog.Default())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be reached")
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/dashboard/overview", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with missing bearer token, got %d", rec.Code)
	}
}

func TestAuthAllowsPublicTracePathWithoutBearer(t *testing.T) {
	cfg := config.AuthConfig{Mode: config.AuthModeOIDC}
	mw := Auth(cfg, fakeValidator{}, fakeResolver{}, slog.Default())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/trace/TRC-ABCD-EFGH", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for public trace path, got %d", rec.Code)
	}
}

func TestAuthAllowsProtectedPathForOperator(t *testing.T) {
	cfg := config.AuthConfig{Mode: config.AuthModeOIDC}
	mw := Auth(cfg, fakeValidator{}, fakeResolver{principal: domain.Principal{Role: domain.UserRoleOperator, Status: domain.UserStatusActive}}, slog.Default())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/dashboard/overview", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for operator on dashboard, got %d", rec.Code)
	}
}

func TestAuthRejectsAdminPathForOperator(t *testing.T) {
	cfg := config.AuthConfig{Mode: config.AuthModeOIDC}
	mw := Auth(cfg, fakeValidator{}, fakeResolver{principal: domain.Principal{Role: domain.UserRoleOperator, Status: domain.UserStatusActive}}, slog.Default())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/users", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 on admin path, got %d", rec.Code)
	}
}

func TestAuthRejectsReconcilePathForOperator(t *testing.T) {
	cfg := config.AuthConfig{Mode: config.AuthModeOIDC}
	mw := Auth(cfg, fakeValidator{}, fakeResolver{principal: domain.Principal{Role: domain.UserRoleOperator, Status: domain.UserStatusActive}}, slog.Default())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/batches/reconcile", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 on reconcile path for operator, got %d", rec.Code)
	}
}

func TestAuthRejectsUnknownProvisionedUser(t *testing.T) {
	cfg := config.AuthConfig{Mode: config.AuthModeOIDC}
	mw := Auth(cfg, fakeValidator{}, fakeResolver{err: service.ErrNotFound}, slog.Default())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/dashboard/overview", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for unknown user, got %d", rec.Code)
	}
}

func TestRateLimitAllows(t *testing.T) {
	cfg := config.RateLimitConfig{Enabled: true, RequestsPerSecond: 100, Burst: 10}
	mw := RateLimit(cfg, slog.Default())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

type fakeValidator struct {
	err error
}

func (f fakeValidator) Validate(_ context.Context, _ string) (domain.IdentityClaims, error) {
	if f.err != nil {
		return domain.IdentityClaims{}, f.err
	}
	return domain.IdentityClaims{
		Subject:     "sub-1",
		Email:       "admin@example.com",
		DisplayName: "Admin",
	}, nil
}

type fakeResolver struct {
	principal domain.Principal
	err       error
}

func (f fakeResolver) ResolvePrincipal(_ context.Context, _ domain.IdentityClaims, _ domain.AuthMode) (domain.Principal, error) {
	if f.err != nil {
		return domain.Principal{}, f.err
	}
	if f.principal.Role == "" {
		return domain.Principal{
			Subject:     "sub-1",
			Email:       "admin@example.com",
			DisplayName: "Admin",
			Role:        domain.UserRoleAdmin,
			Status:      domain.UserStatusActive,
			AuthMode:    domain.AuthModeOIDC,
		}, nil
	}
	if f.principal.AuthMode == "" {
		f.principal.AuthMode = domain.AuthModeOIDC
	}
	return f.principal, nil
}

func TestRateLimitExceeded(t *testing.T) {
	cfg := config.RateLimitConfig{Enabled: true, RequestsPerSecond: 1, Burst: 1}
	mw := RateLimit(cfg, slog.Default())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request should succeed (uses the single burst token).
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("first request: expected 200, got %d", rec.Code)
	}

	// Second request should be throttled.
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("second request: expected 429, got %d", rec2.Code)
	}
}

func TestRateLimitExcludesHealthz(t *testing.T) {
	cfg := config.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 1,
		Burst:             1,
		ExcludePaths:      []string{"/healthz"},
	}
	mw := RateLimit(cfg, slog.Default())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust the single burst token on a normal path.
	req := httptest.NewRequest(http.MethodGet, "/v1/infer", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("first /v1/infer: expected 200, got %d", rec.Code)
	}

	// Normal path should now be throttled.
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("/v1/infer after exhaust: expected 429, got %d", rec2.Code)
	}

	// /healthz must still pass — it should not consume or be blocked by the bucket.
	healthReq := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, healthReq)
	if rec3.Code != http.StatusOK {
		t.Errorf("/healthz while throttled: expected 200, got %d", rec3.Code)
	}
}

func TestCORSPreflight(t *testing.T) {
	cfg := config.CORSConfig{
		Enabled:        true,
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
		MaxAgeS:        3600,
	}
	mw := CORS(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Errorf("preflight: expected 204, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS origin header")
	}
}

func TestCORSMultipleOriginsMatchesRequest(t *testing.T) {
	cfg := config.CORSConfig{
		Enabled:        true,
		AllowedOrigins: []string{"https://example.com", "https://other.com"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Content-Type"},
		MaxAgeS:        3600,
	}
	mw := CORS(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://other.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	got := rec.Header().Get("Access-Control-Allow-Origin")
	if got != "https://other.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want https://other.com", got)
	}
	if rec.Header().Get("Vary") != "Origin" {
		t.Error("Vary header should be set to Origin for non-wildcard")
	}
}

func TestCORSMultipleOriginsRejectsUnknown(t *testing.T) {
	cfg := config.CORSConfig{
		Enabled:        true,
		AllowedOrigins: []string{"https://example.com"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Content-Type"},
		MaxAgeS:        3600,
	}
	mw := CORS(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin should be empty for unknown origin, got %q", got)
	}
}

func TestRequestID(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		if id == "" {
			t.Error("request ID should not be empty")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("X-Request-ID header should be set")
	}
}

func TestRequestIDPreserved(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		if id != "existing-id" {
			t.Errorf("request ID = %q, want existing-id", id)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "existing-id")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") != "existing-id" {
		t.Error("should preserve existing X-Request-ID")
	}
}

type hijackableRecorder struct {
	*httptest.ResponseRecorder
	hijacked bool
}

func (h *hijackableRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h.hijacked = true
	return nil, nil, nil
}

func TestLoggingPreservesHijacker(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mw := Logging(logger)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("response writer should implement http.Hijacker")
		}
		_, _, err := hj.Hijack()
		if err != nil {
			t.Fatalf("hijack failed: %v", err)
		}
	}))

	rec := &hijackableRecorder{ResponseRecorder: httptest.NewRecorder()}
	req := httptest.NewRequest(http.MethodGet, "/v1/infer/stream", nil)
	handler.ServeHTTP(rec, req)

	if !rec.hijacked {
		t.Fatal("expected underlying writer to be hijacked")
	}
}

func TestLoggingHijackNotSupported(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mw := Logging(logger)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("response writer should implement http.Hijacker")
		}
		_, _, err := hj.Hijack()
		if err == nil {
			t.Fatal("expected hijack to fail when underlying writer does not support it")
		}
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/infer/stream", nil)
	handler.ServeHTTP(rec, req)
}
