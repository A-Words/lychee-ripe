package oidc

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
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
)

var (
	ErrMissingBearerToken = errors.New("missing bearer token")
	ErrInvalidToken       = errors.New("invalid token")
)

type Validator struct {
	issuer   string
	audience string
	client   *http.Client

	mu         sync.RWMutex
	keys       map[string]*rsa.PublicKey
	lastSynced time.Time
	jwksURI    string
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
	if err != nil || token == nil || !token.Valid {
		return domain.IdentityClaims{}, ErrInvalidToken
	}
	if strings.TrimSpace(claims.Subject) == "" {
		return domain.IdentityClaims{}, ErrInvalidToken
	}
	return domain.IdentityClaims{
		Subject:     claims.Subject,
		Email:       strings.ToLower(strings.TrimSpace(claims.Email)),
		DisplayName: strings.TrimSpace(claims.Name),
	}, nil
}

func (v *Validator) Discover(ctx context.Context) (discoveryDocument, error) {
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
	if key, ok := v.keys[kid]; ok && time.Since(v.lastSynced) < jwksTTL {
		v.mu.RUnlock()
		return key, nil
	}
	v.mu.RUnlock()

	forceRefresh := false
	v.mu.RLock()
	_, kidKnown := v.keys[kid]
	v.mu.RUnlock()
	if !kidKnown {
		forceRefresh = true
	}

	if err := v.refreshKeys(ctx, forceRefresh); err != nil {
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
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, doc.JWKSURI, nil)
	if err != nil {
		return err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks endpoint returned %d", resp.StatusCode)
	}

	var jwks jwksDocument
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return err
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
		return fmt.Errorf("no rsa keys available in jwks")
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
