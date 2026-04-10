package oidc

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lychee-ripe/gateway/internal/config"
	"github.com/lychee-ripe/gateway/internal/domain"
)

const (
	discoveryPath  = "/.well-known/openid-configuration"
	defaultTimeout = 10 * time.Second
	jwksTTL        = 5 * time.Minute
	maxStaleJWKS   = 15 * time.Minute
)

var (
	ErrMissingBearerToken = errors.New("missing bearer token")
	ErrInvalidToken       = errors.New("invalid token")
	ErrUnavailable        = errors.New("oidc unavailable")
)

type Validator struct {
	issuer   string
	audience string
	client   *http.Client

	mu         sync.RWMutex
	keys       map[string]*rsa.PublicKey
	lastSynced time.Time
	jwksURI    string

	discoveryMu         sync.RWMutex
	cachedDiscovery     *discoveryDocument
	discoveryLastSynced time.Time
}

type Claims struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	jwt.RegisteredClaims
}

type discoveryDocument struct {
	Issuer                string `json:"issuer"`
	JWKSURI               string `json:"jwks_uri"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	EndSessionEndpoint    string `json:"end_session_endpoint"`
}

type DiscoveryDocument = discoveryDocument

type AuthorizationCodeExchange struct {
	ClientID     string
	Code         string
	CodeVerifier string
	RedirectURI  string
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type jwksDocument struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
	Alg string `json:"alg"`
	Use string `json:"use"`
}

func NewValidator(cfg config.OIDCConfig) (*Validator, error) {
	issuer := strings.TrimRight(strings.TrimSpace(cfg.IssuerURL), "/")
	if issuer == "" {
		return nil, fmt.Errorf("issuer url is required")
	}
	audience := strings.TrimSpace(cfg.Audience)
	if audience == "" {
		return nil, fmt.Errorf("audience is required")
	}
	return &Validator{
		issuer:   issuer,
		audience: audience,
		client:   &http.Client{Timeout: defaultTimeout},
		keys:     make(map[string]*rsa.PublicKey),
	}, nil
}

func (v *Validator) Validate(ctx context.Context, rawToken string) (domain.IdentityClaims, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return domain.IdentityClaims{}, ErrMissingBearerToken
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(rawToken, claims, func(token *jwt.Token) (any, error) {
		kid, _ := token.Header["kid"].(string)
		if kid == "" {
			return nil, ErrInvalidToken
		}
		key, err := v.getKey(ctx, kid)
		if err != nil {
			return nil, err
		}
		return key, nil
	}, jwt.WithAudience(v.audience), jwt.WithIssuer(v.issuer), jwt.WithValidMethods([]string{"RS256", "RS384", "RS512"}))
	if err != nil {
		if errors.Is(err, ErrUnavailable) {
			return domain.IdentityClaims{}, err
		}
		return domain.IdentityClaims{}, ErrInvalidToken
	}
	if token == nil || !token.Valid {
		return domain.IdentityClaims{}, ErrInvalidToken
	}
	if strings.TrimSpace(claims.Subject) == "" {
		return domain.IdentityClaims{}, ErrInvalidToken
	}
	return domain.IdentityClaims{
		Subject:     claims.Subject,
		Email:       strings.ToLower(strings.TrimSpace(claims.Email)),
		DisplayName: strings.TrimSpace(claims.Name),
		ExpiresAt:   claimsExpiry(claims),
	}, nil
}

func (v *Validator) ExchangeAuthorizationCode(ctx context.Context, input AuthorizationCodeExchange) (TokenResponse, error) {
	doc, err := v.Discover(ctx)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("%w: discover oidc config: %v", ErrUnavailable, err)
	}
	if strings.TrimSpace(doc.TokenEndpoint) == "" {
		return TokenResponse{}, fmt.Errorf("%w: token endpoint missing from discovery document", ErrUnavailable)
	}

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {strings.TrimSpace(input.ClientID)},
		"code":          {strings.TrimSpace(input.Code)},
		"code_verifier": {strings.TrimSpace(input.CodeVerifier)},
		"redirect_uri":  {strings.TrimSpace(input.RedirectURI)},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, doc.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return TokenResponse{}, fmt.Errorf("%w: build token request: %v", ErrUnavailable, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := v.client.Do(req)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("%w: exchange authorization code: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return TokenResponse{}, fmt.Errorf("%w: token endpoint returned %d: %s", ErrUnavailable, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var token TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return TokenResponse{}, fmt.Errorf("%w: decode token response: %v", ErrUnavailable, err)
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return TokenResponse{}, fmt.Errorf("%w: token response missing access_token", ErrUnavailable)
	}
	return token, nil
}

// Discover returns the OIDC discovery document, using a cached copy when it is
// still within the jwksTTL window. On cache miss or expiry it fetches a fresh
// copy; on fetch failure it falls back to the stale cache when available.
func (v *Validator) Discover(ctx context.Context) (DiscoveryDocument, error) {
	v.discoveryMu.RLock()
	cached := v.cachedDiscovery
	fresh := cached != nil && time.Since(v.discoveryLastSynced) < jwksTTL
	v.discoveryMu.RUnlock()
	if fresh {
		return *cached, nil
	}

	doc, err := v.fetchDiscovery(ctx)
	if err != nil {
		// Fall back to stale cache when the upstream is temporarily unavailable.
		if cached != nil {
			return *cached, nil
		}
		return discoveryDocument{}, err
	}

	v.discoveryMu.Lock()
	v.cachedDiscovery = &doc
	v.discoveryLastSynced = time.Now()
	v.discoveryMu.Unlock()
	return doc, nil
}

func (v *Validator) fetchDiscovery(ctx context.Context) (discoveryDocument, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.issuer+discoveryPath, nil)
	if err != nil {
		return discoveryDocument{}, err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return discoveryDocument{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return discoveryDocument{}, fmt.Errorf("oidc discovery returned %d", resp.StatusCode)
	}
	var doc discoveryDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return discoveryDocument{}, err
	}
	if strings.TrimSpace(doc.JWKSURI) == "" {
		return discoveryDocument{}, fmt.Errorf("jwks_uri missing from discovery document")
	}
	return doc, nil
}

func (v *Validator) getKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	cachedKey, kidKnown := v.keys[kid]
	cacheFresh := time.Since(v.lastSynced) < jwksTTL
	cacheAcceptableStale := time.Since(v.lastSynced) < maxStaleJWKS
	v.mu.RUnlock()
	if kidKnown && cacheFresh {
		return cachedKey, nil
	}

	forceRefresh := !kidKnown
	if err := v.refreshKeys(ctx, forceRefresh); err != nil {
		if kidKnown && cacheAcceptableStale {
			return cachedKey, nil
		}
		return nil, err
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	key, ok := v.keys[kid]
	if !ok {
		return nil, ErrInvalidToken
	}
	return key, nil
}

func (v *Validator) refreshKeys(ctx context.Context, force bool) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !force && time.Since(v.lastSynced) < jwksTTL && len(v.keys) > 0 {
		return nil
	}
	doc, err := v.Discover(ctx)
	if err != nil {
		return fmt.Errorf("%w: discover oidc config: %v", ErrUnavailable, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, doc.JWKSURI, nil)
	if err != nil {
		return fmt.Errorf("%w: build jwks request: %v", ErrUnavailable, err)
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: fetch jwks: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: jwks endpoint returned %d", ErrUnavailable, resp.StatusCode)
	}

	var jwks jwksDocument
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("%w: decode jwks: %v", ErrUnavailable, err)
	}

	keys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, item := range jwks.Keys {
		if strings.TrimSpace(item.Kid) == "" || strings.ToUpper(item.Kty) != "RSA" {
			continue
		}
		key, err := jwkToPublicKey(item)
		if err != nil {
			continue
		}
		keys[item.Kid] = key
	}
	if len(keys) == 0 {
		return fmt.Errorf("%w: no rsa keys available in jwks", ErrUnavailable)
	}
	v.keys = keys
	v.jwksURI = doc.JWKSURI
	v.lastSynced = time.Now()
	return nil
}

func jwkToPublicKey(key jwkKey) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, err
	}
	n := new(big.Int).SetBytes(nBytes)
	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}
	if n.Sign() <= 0 || e <= 0 {
		return nil, fmt.Errorf("invalid rsa jwk")
	}
	return &rsa.PublicKey{N: n, E: e}, nil
}

func claimsExpiry(claims *Claims) *time.Time {
	if claims == nil || claims.ExpiresAt == nil {
		return nil
	}
	expiresAt := claims.ExpiresAt.Time.UTC()
	return &expiresAt
}
