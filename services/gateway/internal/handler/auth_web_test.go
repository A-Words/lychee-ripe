package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	loginBindingCookieSameSite service.HTTPSameSite
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

func (f *fakeWebAuthService) LoginBindingCookieSameSite() service.HTTPSameSite {
	if f.loginBindingCookieSameSite == "" {
		return service.HTTPSameSiteLax
	}
	return f.loginBindingCookieSameSite
}

func TestGetLoginSetsBrowserBindingCookieAndRedirects(t *testing.T) {
	svc := &fakeWebAuthService{
		beginResult: service.BeginWebLoginResult{
			AuthorizationURL:    "https://issuer.example.com/auth?state=abc",
			BrowserBindingToken: "binding-token",
		},
		cookieSecure: true,
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
	if len(cookies) != 1 {
		t.Fatalf("cookie count = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != "lychee_session_login" {
		t.Fatalf("cookie name = %q, want lychee_session_login", cookie.Name)
	}
	if cookie.Value != "binding-token" {
		t.Fatalf("cookie value = %q, want binding-token", cookie.Value)
	}
	if cookie.Path != "/v1/auth/callback" {
		t.Fatalf("cookie path = %q, want /v1/auth/callback", cookie.Path)
	}
	if !cookie.HttpOnly {
		t.Fatal("expected HttpOnly login binding cookie")
	}
	if !cookie.Secure {
		t.Fatal("expected Secure login binding cookie")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie SameSite = %v, want Lax", cookie.SameSite)
	}
}

func TestGetCallbackRejectsMissingBrowserBindingCookie(t *testing.T) {
	svc := &fakeWebAuthService{
		completeErr: errors.New("should not be used"),
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/callback?code=code-1&state=state-1", nil)
	rec := httptest.NewRecorder()

	svc.completeErr = service.ErrInvalidRequest
	GetCallback(svc).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if svc.lastCompleteBindingToken != "" {
		t.Fatalf("binding token = %q, want empty", svc.lastCompleteBindingToken)
	}

	resp := rec.Result()
	cookies := resp.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookie count = %d, want 1", len(cookies))
	}
	if cookies[0].Value != "" {
		t.Fatalf("cleared binding cookie value = %q, want empty", cookies[0].Value)
	}
	if got := rec.Header().Get("Set-Cookie"); got == "" || !containsAll(got, "lychee_session_login=", "Max-Age=0") {
		t.Fatalf("Set-Cookie = %q, want cleared login binding cookie", got)
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
	if len(cookies) != 2 {
		t.Fatalf("cookie count = %d, want 2", len(cookies))
	}

	var sessionCookie *http.Cookie
	var bindingCookie *http.Cookie
	for _, cookie := range cookies {
		switch cookie.Name {
		case "lychee_session":
			sessionCookie = cookie
		case "lychee_session_login":
			bindingCookie = cookie
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
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}
