package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/lychee-ripe/gateway/internal/config"
	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/oidc"
	"github.com/lychee-ripe/gateway/internal/repository"
)

const (
	defaultWebOIDCScope = "openid profile email"
	defaultStateTTL     = 10 * time.Minute
)

type WebAuthRepository interface {
	CreateWebAuthState(ctx context.Context, state domain.WebAuthStateRecord) (domain.WebAuthStateRecord, error)
	ConsumeWebAuthState(ctx context.Context, state string, now time.Time) (domain.WebAuthStateRecord, error)
	CreateWebSession(ctx context.Context, session domain.WebSessionRecord) (domain.WebSessionRecord, error)
	GetWebSession(ctx context.Context, sessionIDHash string, now time.Time) (domain.WebSessionRecord, error)
	DeleteWebSession(ctx context.Context, sessionIDHash string) error
	GetPrincipalByID(ctx context.Context, userID string) (domain.UserRecord, error)
	GetUserByOIDCSubject(ctx context.Context, subject string) (domain.UserRecord, error)
}

type WebAuthService struct {
	repo      WebAuthRepository
	validator *oidc.Validator
	auth      *AuthService
	cfg       config.AuthConfig
	nowFn     func() time.Time
	randomFn  func(int) (string, error)
	stateTTL  time.Duration
}

type CompleteWebLoginResult struct {
	SessionID   string
	RedirectURL string
}

type LogoutResult struct {
	RedirectURL string
}

func NewWebAuthService(repo WebAuthRepository, validator *oidc.Validator, auth *AuthService, cfg config.AuthConfig) *WebAuthService {
	return &WebAuthService{
		repo:      repo,
		validator: validator,
		auth:      auth,
		cfg:       cfg,
		nowFn:     func() time.Time { return time.Now().UTC() },
		randomFn:  randomToken,
		stateTTL:  defaultStateTTL,
	}
}

func (s *WebAuthService) CookieName() string {
	name := strings.TrimSpace(s.cfg.Web.CookieName)
	if name == "" {
		return "lychee_session"
	}
	return name
}

func (s *WebAuthService) CookieSecure() bool {
	return s.cfg.Web.CookieSecure
}

func (s *WebAuthService) BeginLogin(ctx context.Context, redirectPath string) (string, error) {
	if s.repo == nil || s.validator == nil || s.auth == nil {
		return "", ErrServiceUnavailable
	}

	state, err := s.randomFn(32)
	if err != nil {
		return "", ErrServiceUnavailable
	}
	codeVerifier, err := s.randomFn(48)
	if err != nil {
		return "", ErrServiceUnavailable
	}
	codeChallenge, err := oidcCodeChallenge(codeVerifier)
	if err != nil {
		return "", ErrServiceUnavailable
	}

	now := s.nowFn()
	if _, err := s.repo.CreateWebAuthState(ctx, domain.WebAuthStateRecord{
		State:        state,
		CodeVerifier: codeVerifier,
		RedirectPath: normalizeRedirectPath(redirectPath),
		ExpiresAt:    now.Add(s.stateTTL),
		CreatedAt:    now,
		UpdatedAt:    now,
	}); err != nil {
		return "", ErrServiceUnavailable
	}

	discovery, err := s.validator.Discover(ctx)
	if err != nil {
		return "", ErrServiceUnavailable
	}
	authURL, err := url.Parse(discovery.AuthorizationEndpoint)
	if err != nil || authURL.Scheme == "" || authURL.Host == "" {
		return "", ErrServiceUnavailable
	}
	query := authURL.Query()
	query.Set("client_id", strings.TrimSpace(s.cfg.OIDC.WebClientID))
	query.Set("response_type", "code")
	query.Set("scope", defaultWebOIDCScope)
	query.Set("redirect_uri", s.callbackURL())
	query.Set("state", state)
	query.Set("code_challenge", codeChallenge)
	query.Set("code_challenge_method", "S256")
	authURL.RawQuery = query.Encode()
	return authURL.String(), nil
}

func (s *WebAuthService) CompleteLogin(ctx context.Context, code string, state string) (CompleteWebLoginResult, error) {
	if s.repo == nil || s.validator == nil || s.auth == nil {
		return CompleteWebLoginResult{}, ErrServiceUnavailable
	}
	now := s.nowFn()
	authState, err := s.repo.ConsumeWebAuthState(ctx, strings.TrimSpace(state), now)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return CompleteWebLoginResult{}, ErrInvalidRequest
		}
		return CompleteWebLoginResult{}, ErrServiceUnavailable
	}

	token, err := s.validator.ExchangeAuthorizationCode(ctx, oidc.AuthorizationCodeExchange{
		ClientID:     s.cfg.OIDC.WebClientID,
		Code:         code,
		CodeVerifier: authState.CodeVerifier,
		RedirectURI:  s.callbackURL(),
	})
	if err != nil {
		return CompleteWebLoginResult{}, ErrServiceUnavailable
	}

	identity, err := s.validator.Validate(ctx, token.AccessToken)
	if err != nil {
		if errors.Is(err, oidc.ErrInvalidToken) || errors.Is(err, oidc.ErrMissingBearerToken) {
			return CompleteWebLoginResult{}, ErrInvalidRequest
		}
		return CompleteWebLoginResult{}, ErrServiceUnavailable
	}

	principal, err := s.auth.ResolvePrincipal(ctx, identity, domain.AuthModeOIDC)
	if err != nil {
		return CompleteWebLoginResult{}, err
	}
	user, err := s.auth.GetUserByOIDCSubject(ctx, principal.Subject)
	if err != nil {
		return CompleteWebLoginResult{}, err
	}

	sessionID, err := s.randomFn(32)
	if err != nil {
		return CompleteWebLoginResult{}, ErrServiceUnavailable
	}

	expiresAt := now.Add(time.Hour)
	if identity.ExpiresAt != nil {
		expiresAt = identity.ExpiresAt.UTC()
	} else if token.ExpiresIn > 0 {
		expiresAt = now.Add(time.Duration(token.ExpiresIn) * time.Second)
	}
	if expiresAt.Before(now) {
		return CompleteWebLoginResult{}, ErrInvalidRequest
	}

	var idToken *string
	if trimmed := strings.TrimSpace(token.IDToken); trimmed != "" {
		idToken = &trimmed
	}
	if _, err := s.repo.CreateWebSession(ctx, domain.WebSessionRecord{
		SessionIDHash: hashOpaqueToken(sessionID),
		UserID:        user.ID,
		IDToken:       idToken,
		ExpiresAt:     expiresAt,
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		return CompleteWebLoginResult{}, ErrServiceUnavailable
	}

	return CompleteWebLoginResult{
		SessionID:   sessionID,
		RedirectURL: resolveAppRedirect(s.cfg.Web.AppBaseURL, authState.RedirectPath),
	}, nil
}

func (s *WebAuthService) ResolveSessionPrincipal(ctx context.Context, sessionID string) (domain.Principal, error) {
	if s.repo == nil {
		return domain.Principal{}, ErrServiceUnavailable
	}
	session, err := s.repo.GetWebSession(ctx, hashOpaqueToken(sessionID), s.nowFn())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return domain.Principal{}, ErrNotFound
		}
		return domain.Principal{}, ErrServiceUnavailable
	}

	user, err := s.repo.GetPrincipalByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			_ = s.repo.DeleteWebSession(ctx, session.SessionIDHash)
			return domain.Principal{}, ErrNotFound
		}
		return domain.Principal{}, ErrServiceUnavailable
	}
	if user.Status != domain.UserStatusActive {
		return domain.Principal{}, ErrInvalidRequest
	}
	return principalFromUser(user, domain.AuthModeOIDC), nil
}

func (s *WebAuthService) Logout(ctx context.Context, sessionID string) (LogoutResult, error) {
	if s.repo == nil {
		return LogoutResult{}, ErrServiceUnavailable
	}
	if strings.TrimSpace(sessionID) == "" {
		return LogoutResult{}, nil
	}

	session, err := s.repo.GetWebSession(ctx, hashOpaqueToken(sessionID), s.nowFn())
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return LogoutResult{}, ErrServiceUnavailable
	}
	if err := s.repo.DeleteWebSession(ctx, hashOpaqueToken(sessionID)); err != nil {
		return LogoutResult{}, ErrServiceUnavailable
	}

	redirectURL := ""
	if err == nil && session.IDToken != nil && strings.TrimSpace(*session.IDToken) != "" && s.validator != nil {
		if discovery, discoverErr := s.validator.Discover(ctx); discoverErr == nil && strings.TrimSpace(discovery.EndSessionEndpoint) != "" {
			if endSessionURL, parseErr := url.Parse(discovery.EndSessionEndpoint); parseErr == nil {
				query := endSessionURL.Query()
				query.Set("id_token_hint", strings.TrimSpace(*session.IDToken))
				query.Set("post_logout_redirect_uri", resolveAppRedirect(s.cfg.Web.AppBaseURL, "/login"))
				endSessionURL.RawQuery = query.Encode()
				redirectURL = endSessionURL.String()
			}
		}
	}

	return LogoutResult{RedirectURL: redirectURL}, nil
}

func principalFromUser(user domain.UserRecord, mode domain.AuthMode) domain.Principal {
	subject := strings.TrimSpace(user.ID)
	if user.OIDCSubject != nil && strings.TrimSpace(*user.OIDCSubject) != "" {
		subject = strings.TrimSpace(*user.OIDCSubject)
	}
	return domain.Principal{
		Subject:     subject,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		Status:      user.Status,
		AuthMode:    mode,
	}
}

func normalizeRedirectPath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || !strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "//") {
		return "/dashboard"
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed == nil || parsed.IsAbs() {
		return "/dashboard"
	}
	if !strings.HasPrefix(path.Clean(parsed.Path), "/") {
		return "/dashboard"
	}
	return trimmed
}

func resolveAppRedirect(appBaseURL string, redirectPath string) string {
	base, err := url.Parse(strings.TrimSpace(appBaseURL))
	if err != nil || base == nil {
		return redirectPath
	}
	target, err := base.Parse(normalizeRedirectPath(redirectPath))
	if err != nil {
		return base.String()
	}
	return target.String()
}

func (s *WebAuthService) callbackURL() string {
	return resolveAppRedirect(s.cfg.Web.PublicBaseURL, "/v1/auth/callback")
}

func hashOpaqueToken(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func randomToken(length int) (string, error) {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func oidcCodeChallenge(codeVerifier string) (string, error) {
	sum := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(sum[:]), nil
}
