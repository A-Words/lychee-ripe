package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/lychee-ripe/gateway/internal/config"
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
	LoginBindingCookiePath() string
	LoginBindingCookieSameSite() service.HTTPSameSite
	LoginFailureRedirectURL(errorCode string) string
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
			Path:     svc.LoginBindingCookiePath(),
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
		browserBindingToken := ""
		if cookie, err := r.Cookie(svc.LoginBindingCookieName()); err == nil {
			browserBindingToken = strings.TrimSpace(cookie.Value)
		}

		http.SetCookie(w, &http.Cookie{
			Name:     svc.LoginBindingCookieName(),
			Value:    "",
			Path:     svc.LoginBindingCookiePath(),
			HttpOnly: true,
			SameSite: toHTTPSameSite(svc.LoginBindingCookieSameSite()),
			Secure:   svc.CookieSecure(),
			MaxAge:   -1,
		})
		if callbackError := strings.TrimSpace(r.URL.Query().Get("error")); callbackError != "" {
			redirectLoginFailure(w, r, svc, callbackError)
			return
		}
		if code == "" || state == "" {
			redirectLoginFailure(w, r, svc, "invalid_request")
			return
		}

		result, err := svc.CompleteLogin(r.Context(), code, state, browserBindingToken)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrInvalidRequest), errors.Is(err, service.ErrNotFound):
				redirectLoginFailure(w, r, svc, "invalid_request")
			default:
				redirectLoginFailure(w, r, svc, "auth_unavailable")
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

func redirectLoginFailure(w http.ResponseWriter, r *http.Request, svc webAuthService, errorCode string) {
	http.Redirect(w, r, svc.LoginFailureRedirectURL(errorCode), http.StatusFound)
}

func PostLogout(svc webAuthService, corsCfg config.CORSConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := ""
		if cookie, err := r.Cookie(svc.CookieName()); err == nil {
			sessionID = strings.TrimSpace(cookie.Value)
		}
		if sessionID != "" {
			if err := enforceLogoutOrigin(r, corsCfg); err != nil {
				writeError(w, r, http.StatusForbidden, "forbidden", err.Error(), nil)
				return
			}
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

func enforceLogoutOrigin(r *http.Request, corsCfg config.CORSConfig) error {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return errors.New("origin header required for cookie-authenticated unsafe requests")
	}
	if !corsCfg.AllowsOrigin(origin) {
		return errors.New("origin is not allowed for cookie-authenticated request")
	}
	return nil
}

func toHTTPSameSite(value service.HTTPSameSite) http.SameSite {
	switch value {
	case service.HTTPSameSiteNone:
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
