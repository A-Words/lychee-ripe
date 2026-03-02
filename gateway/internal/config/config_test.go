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
	if cfg.DB.Driver != "sqlite" {
		t.Errorf("unexpected default db driver: %s", cfg.DB.Driver)
	}
	if cfg.DB.DSN != filepath.Join("artifacts", "data", "gateway.db") {
		t.Errorf("unexpected default db dsn: %s", cfg.DB.DSN)
	}
	if cfg.DB.MaxOpenConns != 10 {
		t.Errorf("unexpected default max_open_conns: %d", cfg.DB.MaxOpenConns)
	}
	if cfg.DB.MaxIdleConns != 5 {
		t.Errorf("unexpected default max_idle_conns: %d", cfg.DB.MaxIdleConns)
	}
	if cfg.DB.ConnMaxLifetimeS != 300 {
		t.Errorf("unexpected default conn_max_lifetime_s: %d", cfg.DB.ConnMaxLifetimeS)
	}
	if cfg.DB.SQLite.BusyTimeoutMS != 5000 {
		t.Errorf("unexpected default busy timeout: %d", cfg.DB.SQLite.BusyTimeoutMS)
	}
	if cfg.DB.SQLite.JournalMode != "WAL" {
		t.Errorf("unexpected default journal mode: %s", cfg.DB.SQLite.JournalMode)
	}
	if cfg.DB.Postgres.SSLMode != "disable" {
		t.Errorf("unexpected default postgres ssl_mode: %s", cfg.DB.Postgres.SSLMode)
	}
	if cfg.DB.Postgres.Schema != "public" {
		t.Errorf("unexpected default postgres schema: %s", cfg.DB.Postgres.Schema)
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
db:
  driver: "postgres"
  dsn: "postgres://postgres:postgres@127.0.0.1:5432/lychee_ripe?sslmode=disable"
  max_open_conns: 22
  max_idle_conns: 11
  conn_max_lifetime_s: 600
  sqlite:
    journal_mode: "WAL"
    busy_timeout_ms: 999
  postgres:
    ssl_mode: "disable"
    schema: "public"
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
	if cfg.DB.Driver != "postgres" {
		t.Errorf("db.driver = %q, want postgres", cfg.DB.Driver)
	}
	if cfg.DB.DSN != "postgres://postgres:postgres@127.0.0.1:5432/lychee_ripe?sslmode=disable" {
		t.Errorf("db.dsn = %q", cfg.DB.DSN)
	}
	if cfg.DB.MaxOpenConns != 22 {
		t.Errorf("db.max_open_conns = %d, want 22", cfg.DB.MaxOpenConns)
	}
	if cfg.DB.MaxIdleConns != 11 {
		t.Errorf("db.max_idle_conns = %d, want 11", cfg.DB.MaxIdleConns)
	}
	if cfg.DB.ConnMaxLifetimeS != 600 {
		t.Errorf("db.conn_max_lifetime_s = %d, want 600", cfg.DB.ConnMaxLifetimeS)
	}
	if cfg.DB.SQLite.BusyTimeoutMS != 999 {
		t.Errorf("db.sqlite.busy_timeout_ms = %d, want 999", cfg.DB.SQLite.BusyTimeoutMS)
	}
	if cfg.DB.Postgres.Schema != "public" {
		t.Errorf("db.postgres.schema = %q, want public", cfg.DB.Postgres.Schema)
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

func TestLoadInvalidDBDriver(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "gateway.yaml")
	content := `
db:
  driver: "mysql"
  dsn: "x"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected validation error for invalid db.driver")
	}
}

func TestLoadLegacyDBConfigRejected(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "gateway.yaml")
	content := `
db:
  path: "tmp/test-gateway.db"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected parse error for legacy db.path config")
	}
}

func TestAddr(t *testing.T) {
	cfg := Defaults()
	addr := cfg.Addr()
	if addr != "0.0.0.0:9000" {
		t.Errorf("addr = %q, want 0.0.0.0:9000", addr)
	}
}
