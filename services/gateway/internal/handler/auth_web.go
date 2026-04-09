package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/lychee-ripe/gateway/internal/service"
)

type webAuthService interface {
	BeginLogin(ctx context.Context, redirectPath string) (service.BeginWebLoginResult, error)
	CompleteLogin(ctx context.Context, code string, state string, browserBindingToken string) (service.CompleteWebLoginResult, error)
	Logout(ctx context.Context, sessionID string) (service.LogoutResult, error)
	CookieName() string
	CookieSecure() bool
	CookieSameSite() service.HTTPSameSite
	LoginBindingCookieName() string
	LoginBindingCookieSameSite() service.HTTPSameSite
}

type logoutResponse struct {
	RedirectURL string `json:"redirect_url,omitempty"`
}

func GetLogin(svc webAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := svc.BeginLogin(r.Context(), strings.TrimSpace(r.URL.Query().Get("redirect")))
		if err != nil {
			writeServiceError(w, r, err)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     svc.LoginBindingCookieName(),
			Value:    result.BrowserBindingToken,
			Path:     "/v1/auth/callback",
			HttpOnly: true,
			SameSite: toHTTPSameSite(svc.LoginBindingCookieSameSite()),
			Secure:   svc.CookieSecure(),
		})
		http.Redirect(w, r, result.AuthorizationURL, http.StatusFound)
	}
}

func GetCallback(svc webAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := strings.TrimSpace(r.URL.Query().Get("code"))
		state := strings.TrimSpace(r.URL.Query().Get("state"))
		if code == "" || state == "" {
			writeError(w, r, http.StatusBadRequest, "invalid_request", "missing authorization code or state", nil)
			return
		}
		browserBindingToken := ""
		if cookie, err := r.Cookie(svc.LoginBindingCookieName()); err == nil {
			browserBindingToken = strings.TrimSpace(cookie.Value)
		}

		result, err := svc.CompleteLogin(r.Context(), code, state, browserBindingToken)
		http.SetCookie(w, &http.Cookie{
			Name:     svc.LoginBindingCookieName(),
			Value:    "",
			Path:     "/v1/auth/callback",
			HttpOnly: true,
			SameSite: toHTTPSameSite(svc.LoginBindingCookieSameSite()),
			Secure:   svc.CookieSecure(),
			MaxAge:   -1,
		})
		if err != nil {
			switch {
			case errors.Is(err, service.ErrInvalidRequest), errors.Is(err, service.ErrNotFound):
				writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), nil)
			default:
				writeError(w, r, http.StatusServiceUnavailable, "auth_unavailable", "auth unavailable", nil)
			}
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     svc.CookieName(),
			Value:    result.SessionID,
			Path:     "/",
			HttpOnly: true,
			SameSite: toHTTPSameSite(svc.CookieSameSite()),
			Secure:   svc.CookieSecure(),
		})
		http.Redirect(w, r, result.RedirectURL, http.StatusFound)
	}
}

func PostLogout(svc webAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := ""
		if cookie, err := r.Cookie(svc.CookieName()); err == nil {
			sessionID = strings.TrimSpace(cookie.Value)
		}

		result, err := svc.Logout(r.Context(), sessionID)
		if err != nil {
			writeServiceError(w, r, err)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     svc.CookieName(),
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			SameSite: toHTTPSameSite(svc.CookieSameSite()),
			Secure:   svc.CookieSecure(),
			MaxAge:   -1,
		})
		writeJSON(w, http.StatusOK, logoutResponse{RedirectURL: result.RedirectURL})
	}
}

func toHTTPSameSite(value service.HTTPSameSite) http.SameSite {
	switch value {
	case service.HTTPSameSiteNone:
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
