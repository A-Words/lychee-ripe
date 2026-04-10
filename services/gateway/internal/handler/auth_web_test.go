package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lychee-ripe/gateway/internal/config"
	"github.com/lychee-ripe/gateway/internal/service"
)

type fakeWebAuthService struct {
	beginResult                service.BeginWebLoginResult
	beginErr                   error
	completeResult             service.CompleteWebLoginResult
	completeErr                error
	logoutResult               service.LogoutResult
	logoutErr                  error
	cookieName                 string
	cookieSecure               bool
	cookieSameSite             service.HTTPSameSite
	loginBindingCookieName     string
	loginRedirectCookieName    string
	loginBindingCookiePath     string
	loginBindingCookieSameSite service.HTTPSameSite
	loginFailureRedirectURL    string
	lastFailureErrorCode       string
	lastFailureRedirectPath    string
	lastBeginRedirectPath      string
	lastCompleteCode           string
	lastCompleteState          string
	lastCompleteBindingToken   string
	lastLogoutSessionID        string
}

func (f *fakeWebAuthService) BeginLogin(_ context.Context, redirectPath string) (service.BeginWebLoginResult, error) {
	f.lastBeginRedirectPath = redirectPath
	if f.beginErr != nil {
		return service.BeginWebLoginResult{}, f.beginErr
	}
	return f.beginResult, nil
}

func (f *fakeWebAuthService) CompleteLogin(_ context.Context, code string, state string, browserBindingToken string) (service.CompleteWebLoginResult, error) {
	f.lastCompleteCode = code
	f.lastCompleteState = state
	f.lastCompleteBindingToken = browserBindingToken
	if f.completeErr != nil {
		return service.CompleteWebLoginResult{}, f.completeErr
	}
	return f.completeResult, nil
}

func (f *fakeWebAuthService) Logout(_ context.Context, sessionID string) (service.LogoutResult, error) {
	f.lastLogoutSessionID = sessionID
	if f.logoutErr != nil {
		return service.LogoutResult{}, f.logoutErr
	}
	return f.logoutResult, nil
}

func (f *fakeWebAuthService) CookieName() string {
	if f.cookieName == "" {
		return "lychee_session"
	}
	return f.cookieName
}

func (f *fakeWebAuthService) CookieSecure() bool {
	return f.cookieSecure
}

func (f *fakeWebAuthService) CookieSameSite() service.HTTPSameSite {
	if f.cookieSameSite == "" {
		return service.HTTPSameSiteLax
	}
	return f.cookieSameSite
}

func (f *fakeWebAuthService) LoginBindingCookieName() string {
	if f.loginBindingCookieName == "" {
		return "lychee_session_login"
	}
	return f.loginBindingCookieName
}

func (f *fakeWebAuthService) LoginRedirectCookieName() string {
	if f.loginRedirectCookieName == "" {
		return "lychee_session_login_redirect"
	}
	return f.loginRedirectCookieName
}

func (f *fakeWebAuthService) LoginBindingCookiePath() string {
	if f.loginBindingCookiePath == "" {
		return "/v1/auth/callback"
	}
	return f.loginBindingCookiePath
}

func (f *fakeWebAuthService) LoginBindingCookieSameSite() service.HTTPSameSite {
	if f.loginBindingCookieSameSite == "" {
		return service.HTTPSameSiteLax
	}
	return f.loginBindingCookieSameSite
}

func (f *fakeWebAuthService) LoginFailureRedirectURL(errorCode string, redirectPath string) string {
	f.lastFailureErrorCode = errorCode
	f.lastFailureRedirectPath = redirectPath
	if f.loginFailureRedirectURL == "" {
		return "/login?auth_error=" + errorCode + "&redirect=" + redirectPath
	}
	return f.loginFailureRedirectURL
}

func TestGetLoginSetsBrowserBindingCookieAndRedirects(t *testing.T) {
	svc := &fakeWebAuthService{
		beginResult: service.BeginWebLoginResult{
			AuthorizationURL:    "https://issuer.example.com/auth?state=abc",
			BrowserBindingToken: "binding-token",
		},
		cookieSecure:           true,
		loginBindingCookiePath: "/gateway/v1/auth/callback",
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/login?redirect=/admin", nil)
	rec := httptest.NewRecorder()

	GetLogin(svc).ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if location := rec.Header().Get("Location"); location != "https://issuer.example.com/auth?state=abc" {
		t.Fatalf("Location = %q", location)
	}
	if svc.lastBeginRedirectPath != "/admin" {
		t.Fatalf("redirect path = %q, want /admin", svc.lastBeginRedirectPath)
	}

	resp := rec.Result()
	cookies := resp.Cookies()
	if len(cookies) != 2 {
		t.Fatalf("cookie count = %d, want 2", len(cookies))
	}
	var bindingCookie *http.Cookie
	var redirectCookie *http.Cookie
	for _, cookie := range cookies {
		switch cookie.Name {
		case "lychee_session_login":
			bindingCookie = cookie
		case "lychee_session_login_redirect":
			redirectCookie = cookie
		}
	}
	if bindingCookie == nil {
		t.Fatal("missing binding cookie")
	}
	if bindingCookie.Value != "binding-token" {
		t.Fatalf("cookie value = %q, want binding-token", bindingCookie.Value)
	}
	if bindingCookie.Path != "/gateway/v1/auth/callback" {
		t.Fatalf("cookie path = %q, want /gateway/v1/auth/callback", bindingCookie.Path)
	}
	if !bindingCookie.HttpOnly {
		t.Fatal("expected HttpOnly login binding cookie")
	}
	if !bindingCookie.Secure {
		t.Fatal("expected Secure login binding cookie")
	}
	if bindingCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie SameSite = %v, want Lax", bindingCookie.SameSite)
	}
	if redirectCookie == nil {
		t.Fatal("missing redirect cookie")
	}
	if redirectCookie.Value != "/admin" {
		t.Fatalf("redirect cookie value = %q, want /admin", redirectCookie.Value)
	}
	if redirectCookie.Path != "/gateway/v1/auth/callback" {
		t.Fatalf("redirect cookie path = %q, want /gateway/v1/auth/callback", redirectCookie.Path)
	}
}

func TestGetCallbackRedirectsInvalidRequestBackToLogin(t *testing.T) {
	svc := &fakeWebAuthService{
		loginBindingCookiePath:  "/gateway/v1/auth/callback",
		loginFailureRedirectURL: "https://app.example.com/console/login?auth_error=invalid_request&redirect=%2Fadmin",
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/callback?code=code-1&state=state-1", nil)
	req.AddCookie(&http.Cookie{Name: "lychee_session_login_redirect", Value: "/admin"})
	rec := httptest.NewRecorder()

	svc.completeErr = service.ErrInvalidRequest
	GetCallback(svc).ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if location := rec.Header().Get("Location"); location != "https://app.example.com/console/login?auth_error=invalid_request&redirect=%2Fadmin" {
		t.Fatalf("Location = %q", location)
	}
	if svc.lastCompleteBindingToken != "" {
		t.Fatalf("binding token = %q, want empty", svc.lastCompleteBindingToken)
	}
	if svc.lastFailureRedirectPath != "/admin" {
		t.Fatalf("failure redirect path = %q, want /admin", svc.lastFailureRedirectPath)
	}

	resp := rec.Result()
	cookies := resp.Cookies()
	if len(cookies) != 2 {
		t.Fatalf("cookie count = %d, want 2", len(cookies))
	}
	for _, cookie := range cookies {
		if cookie.Path != "/gateway/v1/auth/callback" {
			t.Fatalf("cleared cookie path = %q, want /gateway/v1/auth/callback", cookie.Path)
		}
		if cookie.Value != "" {
			t.Fatalf("cleared cookie value = %q, want empty", cookie.Value)
		}
	}
	for _, header := range rec.Header().Values("Set-Cookie") {
		if !containsAll(header, "Max-Age=0") {
			t.Fatalf("Set-Cookie = %q, want cleared callback cookie", header)
		}
	}
}

func TestGetCallbackRedirectsProviderErrorsBackToLogin(t *testing.T) {
	svc := &fakeWebAuthService{
		loginFailureRedirectURL: "https://app.example.com/login?auth_error=access_denied&redirect=%2Fadmin",
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/callback?error=access_denied&state=state-1", nil)
	req.AddCookie(&http.Cookie{Name: "lychee_session_login_redirect", Value: "/admin"})
	rec := httptest.NewRecorder()

	GetCallback(svc).ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if location := rec.Header().Get("Location"); location != "https://app.example.com/login?auth_error=access_denied&redirect=%2Fadmin" {
		t.Fatalf("Location = %q", location)
	}
	if svc.lastCompleteCode != "" || svc.lastCompleteState != "" {
		t.Fatalf("complete login should not be called, got code=%q state=%q", svc.lastCompleteCode, svc.lastCompleteState)
	}
	if svc.lastFailureRedirectPath != "/admin" {
		t.Fatalf("failure redirect path = %q, want /admin", svc.lastFailureRedirectPath)
	}
}

func TestGetCallbackSetsSessionCookieUsingConfiguredSameSite(t *testing.T) {
	svc := &fakeWebAuthService{
		completeResult: service.CompleteWebLoginResult{
			SessionID:   "session-1",
			RedirectURL: "https://app.other-example.com/admin",
		},
		cookieSecure:   true,
		cookieSameSite: service.HTTPSameSiteNone,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/callback?code=code-1&state=state-1", nil)
	req.AddCookie(&http.Cookie{Name: "lychee_session_login", Value: "binding-token"})
	rec := httptest.NewRecorder()

	GetCallback(svc).ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if svc.lastCompleteBindingToken != "binding-token" {
		t.Fatalf("binding token = %q, want binding-token", svc.lastCompleteBindingToken)
	}
	if location := rec.Header().Get("Location"); location != "https://app.other-example.com/admin" {
		t.Fatalf("Location = %q", location)
	}

	resp := rec.Result()
	cookies := resp.Cookies()
	if len(cookies) != 3 {
		t.Fatalf("cookie count = %d, want 3", len(cookies))
	}

	var sessionCookie *http.Cookie
	var bindingCookie *http.Cookie
	var redirectCookie *http.Cookie
	for _, cookie := range cookies {
		switch cookie.Name {
		case "lychee_session":
			sessionCookie = cookie
		case "lychee_session_login":
			bindingCookie = cookie
		case "lychee_session_login_redirect":
			redirectCookie = cookie
		}
	}
	if sessionCookie == nil {
		t.Fatal("missing session cookie")
	}
	if sessionCookie.Value != "session-1" {
		t.Fatalf("session cookie value = %q, want session-1", sessionCookie.Value)
	}
	if sessionCookie.SameSite != http.SameSiteNoneMode {
		t.Fatalf("session cookie SameSite = %v, want None", sessionCookie.SameSite)
	}
	if !sessionCookie.Secure {
		t.Fatal("expected Secure session cookie")
	}
	if bindingCookie == nil || bindingCookie.Value != "" {
		t.Fatalf("binding cookie = %#v, want cleared cookie", bindingCookie)
	}
	if redirectCookie == nil || redirectCookie.Value != "" {
		t.Fatalf("redirect cookie = %#v, want cleared cookie", redirectCookie)
	}
}

func TestPostLogoutRejectsDisallowedOriginWhenSessionCookiePresent(t *testing.T) {
	svc := &fakeWebAuthService{}

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	req.AddCookie(&http.Cookie{Name: "lychee_session", Value: "session-1"})
	rec := httptest.NewRecorder()

	PostLogout(svc, config.CORSConfig{AllowedOrigins: []string{"https://app.example.com"}}).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if svc.lastLogoutSessionID != "" {
		t.Fatalf("logout session id = %q, want empty", svc.lastLogoutSessionID)
	}
}

func TestPostLogoutAllowsTrustedOriginAndClearsCookie(t *testing.T) {
	svc := &fakeWebAuthService{
		logoutResult:   service.LogoutResult{RedirectURL: "https://issuer.example.com/logout"},
		cookieSameSite: service.HTTPSameSiteNone,
		cookieSecure:   true,
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.AddCookie(&http.Cookie{Name: "lychee_session", Value: "session-1"})
	rec := httptest.NewRecorder()

	PostLogout(svc, config.CORSConfig{AllowedOrigins: []string{"https://app.example.com"}}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if svc.lastLogoutSessionID != "session-1" {
		t.Fatalf("logout session id = %q, want session-1", svc.lastLogoutSessionID)
	}
	if got := rec.Header().Get("Set-Cookie"); got == "" || !containsAll(got, "lychee_session=", "Max-Age=0") {
		t.Fatalf("Set-Cookie = %q, want cleared session cookie", got)
	}
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}
