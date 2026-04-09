package service

import "testing"

func TestResolveAppRedirectPreservesBasePathPrefix(t *testing.T) {
	t.Parallel()

	got := resolveAppRedirect("https://example.com/console", "/admin")
	want := "https://example.com/console/admin"
	if got != want {
		t.Fatalf("resolveAppRedirect() = %q, want %q", got, want)
	}
}

func TestResolveAppRedirectPreservesBasePathPrefixWithQuery(t *testing.T) {
	t.Parallel()

	got := resolveAppRedirect("https://example.com/console", "/dashboard?tab=recent")
	want := "https://example.com/console/dashboard?tab=recent"
	if got != want {
		t.Fatalf("resolveAppRedirect() = %q, want %q", got, want)
	}
}

func TestResolveAppRedirectKeepsGatewayCallbackUnderPrefixedBase(t *testing.T) {
	t.Parallel()

	svc := &WebAuthService{}
	svc.cfg.Web.PublicBaseURL = "https://example.com/gateway"

	got := svc.callbackURL()
	want := "https://example.com/gateway/v1/auth/callback"
	if got != want {
		t.Fatalf("callbackURL() = %q, want %q", got, want)
	}
}

func TestLoginBindingCookiePathKeepsGatewayCallbackUnderPrefixedBase(t *testing.T) {
	t.Parallel()

	svc := &WebAuthService{}
	svc.cfg.Web.PublicBaseURL = "https://example.com/gateway"

	got := svc.LoginBindingCookiePath()
	want := "/gateway/v1/auth/callback"
	if got != want {
		t.Fatalf("LoginBindingCookiePath() = %q, want %q", got, want)
	}
}

func TestLoginFailureRedirectURLKeepsAppBasePathAndErrorCode(t *testing.T) {
	t.Parallel()

	svc := &WebAuthService{}
	svc.cfg.Web.AppBaseURL = "https://example.com/console"

	got := svc.LoginFailureRedirectURL("invalid_request", "/admin?tab=users")
	want := "https://example.com/console/login?auth_error=invalid_request&redirect=%2Fadmin%3Ftab%3Dusers"
	if got != want {
		t.Fatalf("LoginFailureRedirectURL() = %q, want %q", got, want)
	}
}
