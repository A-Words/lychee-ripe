package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.Server.Port != 9000 {
		t.Errorf("expected default port 9000, got %d", cfg.Server.Port)
	}
	if cfg.Upstream.BaseURL != "http://127.0.0.1:8000" {
		t.Errorf("unexpected default upstream: %s", cfg.Upstream.BaseURL)
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "gateway.yaml")
	content := `
server:
  host: "127.0.0.1"
  port: 8080
upstream:
  base_url: "http://localhost:3000"
  timeout_s: 10
auth:
  enabled: true
  api_keys:
    - "test-key-1"
rate_limit:
  enabled: false
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("host = %q, want 127.0.0.1", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Upstream.BaseURL != "http://localhost:3000" {
		t.Errorf("upstream = %q", cfg.Upstream.BaseURL)
	}
	if !cfg.Auth.Enabled {
		t.Error("auth should be enabled")
	}
	if len(cfg.Auth.APIKeys) != 1 || cfg.Auth.APIKeys[0] != "test-key-1" {
		t.Errorf("api_keys = %v", cfg.Auth.APIKeys)
	}
	if cfg.RateLimit.Enabled {
		t.Error("rate_limit should be disabled")
	}
	// Defaults should be preserved for unset fields.
	if cfg.CORS.MaxAgeS != 3600 {
		t.Errorf("cors max_age_s = %d, want 3600 (default)", cfg.CORS.MaxAgeS)
	}
}

func TestLoadMissing(t *testing.T) {
	_, err := Load("/nonexistent/gateway.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestAddr(t *testing.T) {
	cfg := Defaults()
	addr := cfg.Addr()
	if addr != "0.0.0.0:9000" {
		t.Errorf("addr = %q, want 0.0.0.0:9000", addr)
	}
}
