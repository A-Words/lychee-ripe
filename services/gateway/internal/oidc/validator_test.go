package oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lychee-ripe/gateway/internal/config"
)

func TestValidatorGetKeyReturnsCachedKeyWithoutRefresh(t *testing.T) {
	t.Parallel()

	key := mustGenerateRSAKey(t)
	validator := &Validator{
		client:     &http.Client{Timeout: defaultTimeout},
		keys:       map[string]*rsa.PublicKey{"known": &key.PublicKey},
		lastSynced: time.Now(),
	}

	got, err := validator.getKey(context.Background(), "known")
	if err != nil {
		t.Fatalf("getKey returned error: %v", err)
	}
	if got != &key.PublicKey {
		t.Fatal("expected cached key to be returned")
	}
}

func TestValidatorGetKeyForcesRefreshForUnknownKidWithinTTL(t *testing.T) {
	t.Parallel()

	oldKey := mustGenerateRSAKey(t)
	newKey := mustGenerateRSAKey(t)
	var discoveryHits atomic.Int32
	var jwksHits atomic.Int32
	var currentKeys atomic.Value
	currentKeys.Store(jwksDocument{Keys: []jwkKey{toJWK(t, "new", &newKey.PublicKey)}})
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case discoveryPath:
			discoveryHits.Add(1)
			_ = json.NewEncoder(w).Encode(discoveryDocument{
				Issuer:  r.Host,
				JWKSURI: serverURL + "/jwks",
			})
		case "/jwks":
			jwksHits.Add(1)
			_ = json.NewEncoder(w).Encode(currentKeys.Load().(jwksDocument))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	validator, err := NewValidator(config.OIDCConfig{
		IssuerURL: server.URL,
		Audience:  "orchard-console",
	})
	if err != nil {
		t.Fatalf("NewValidator returned error: %v", err)
	}
	validator.keys["old"] = &oldKey.PublicKey
	validator.lastSynced = time.Now()

	got, err := validator.getKey(context.Background(), "new")
	if err != nil {
		t.Fatalf("getKey returned error: %v", err)
	}
	if got == nil || got.N.Cmp(newKey.N) != 0 || got.E != newKey.E {
		t.Fatal("expected refreshed key for new kid")
	}
	if discoveryHits.Load() != 1 {
		t.Fatalf("discovery hits = %d, want 1", discoveryHits.Load())
	}
	if jwksHits.Load() != 1 {
		t.Fatalf("jwks hits = %d, want 1", jwksHits.Load())
	}
}

func TestValidatorGetKeyReturnsInvalidTokenWhenUnknownKidRemainsMissing(t *testing.T) {
	t.Parallel()

	oldKey := mustGenerateRSAKey(t)
	var currentKeys atomic.Value
	currentKeys.Store(jwksDocument{Keys: []jwkKey{toJWK(t, "old", &oldKey.PublicKey)}})
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case discoveryPath:
			_ = json.NewEncoder(w).Encode(discoveryDocument{
				Issuer:  r.Host,
				JWKSURI: serverURL + "/jwks",
			})
		case "/jwks":
			_ = json.NewEncoder(w).Encode(currentKeys.Load().(jwksDocument))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	validator, err := NewValidator(config.OIDCConfig{
		IssuerURL: server.URL,
		Audience:  "orchard-console",
	})
	if err != nil {
		t.Fatalf("NewValidator returned error: %v", err)
	}
	validator.keys["old"] = &oldKey.PublicKey
	validator.lastSynced = time.Now()

	if _, err := validator.getKey(context.Background(), "new"); err != ErrInvalidToken {
		t.Fatalf("error = %v, want ErrInvalidToken", err)
	}
}

func TestValidatorRefreshKeysSkipsNetworkWhenCacheFreshAndNotForced(t *testing.T) {
	t.Parallel()

	key := mustGenerateRSAKey(t)
	var hits atomic.Int32
	validator := &Validator{
		client: &http.Client{
			Timeout: defaultTimeout,
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				hits.Add(1)
				t.Fatal("unexpected network call during non-forced refresh")
				return nil, nil
			}),
		},
		keys:       map[string]*rsa.PublicKey{"known": &key.PublicKey},
		lastSynced: time.Now(),
	}

	if err := validator.refreshKeys(context.Background(), false); err != nil {
		t.Fatalf("refreshKeys returned error: %v", err)
	}
	if hits.Load() != 0 {
		t.Fatalf("network hits = %d, want 0", hits.Load())
	}
}

func mustGenerateRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	return key
}

func toJWK(t *testing.T, kid string, key *rsa.PublicKey) jwkKey {
	t.Helper()
	return jwkKey{
		Kid: kid,
		Kty: "RSA",
		N:   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		E:   encodeExponent(t, key.E),
		Alg: "RS256",
		Use: "sig",
	}
}

func encodeExponent(t *testing.T, exponent int) string {
	t.Helper()
	if exponent <= 0 {
		t.Fatal("expected positive exponent")
	}
	bytes := make([]byte, 0, 4)
	for exponent > 0 {
		bytes = append([]byte{byte(exponent & 0xff)}, bytes...)
		exponent >>= 8
	}
	return base64.RawURLEncoding.EncodeToString(bytes)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
