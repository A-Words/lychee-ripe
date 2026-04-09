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
