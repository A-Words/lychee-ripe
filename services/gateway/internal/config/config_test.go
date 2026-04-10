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
	if cfg.DB.DSN != filepath.Join("mlops", "artifacts", "data", "gateway.db") {
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
	if cfg.Seed.DefaultResourcesEnabled {
		t.Error("seed.default_resources_enabled should default to false")
	}
	if cfg.Trace.Mode != "database" {
		t.Errorf("unexpected default trace.mode: %s", cfg.Trace.Mode)
	}
	if cfg.Chain.TxTimeoutS != 30 {
		t.Errorf("unexpected default chain.tx_timeout_s: %d", cfg.Chain.TxTimeoutS)
	}
	if cfg.Chain.ReceiptPollIntervalMS != 500 {
		t.Errorf("unexpected default chain.receipt_poll_interval_ms: %d", cfg.Chain.ReceiptPollIntervalMS)
	}
	if got := cfg.CORS.AllowedMethods; len(got) != 5 || got[0] != "GET" || got[1] != "POST" || got[2] != "PATCH" || got[3] != "DELETE" || got[4] != "OPTIONS" {
		t.Errorf("unexpected default cors.allowed_methods: %#v", got)
	}
	if cfg.Auth.Web.CookieName != "lychee_session" {
		t.Errorf("unexpected default auth.web.cookie_name: %q", cfg.Auth.Web.CookieName)
	}
	if cfg.Auth.Web.CookieSameSite != "lax" {
		t.Errorf("unexpected default auth.web.cookie_same_site: %q", cfg.Auth.Web.CookieSameSite)
	}
	if cfg.CORS.AllowCredentials {
		t.Error("cors.allow_credentials should default to false")
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
seed:
  default_resources_enabled: true
auth:
  mode: "oidc"
  bootstrap_admin_email: "admin@example.com"
  oidc:
    issuer_url: "https://issuer.example.com"
    audience: "lychee-ripe"
    web_client_id: "orchard-console-web"
  web:
    public_base_url: "http://127.0.0.1:9000"
    app_base_url: "http://127.0.0.1:3000"
    cookie_name: "lychee_session"
    cookie_secure: false
    cookie_same_site: "lax"
cors:
  allowed_origins:
    - "http://127.0.0.1:3000"
  allow_credentials: true
rate_limit:
  enabled: false
trace:
  mode: "database"
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
	if !cfg.Seed.DefaultResourcesEnabled {
		t.Error("seed.default_resources_enabled = false, want true")
	}
	if cfg.Auth.Mode != AuthModeOIDC {
		t.Errorf("auth.mode = %q, want oidc", cfg.Auth.Mode)
	}
	if cfg.Auth.BootstrapAdminEmail != "admin@example.com" {
		t.Errorf("auth.bootstrap_admin_email = %q, want admin@example.com", cfg.Auth.BootstrapAdminEmail)
	}
	if cfg.Auth.OIDC.IssuerURL != "https://issuer.example.com" {
		t.Errorf("auth.oidc.issuer_url = %q", cfg.Auth.OIDC.IssuerURL)
	}
	if cfg.Auth.OIDC.Audience != "lychee-ripe" {
		t.Errorf("auth.oidc.audience = %q", cfg.Auth.OIDC.Audience)
	}
	if cfg.Auth.OIDC.WebClientID != "orchard-console-web" {
		t.Errorf("auth.oidc.web_client_id = %q", cfg.Auth.OIDC.WebClientID)
	}
	if cfg.Auth.Web.PublicBaseURL != "http://127.0.0.1:9000" {
		t.Errorf("auth.web.public_base_url = %q", cfg.Auth.Web.PublicBaseURL)
	}
	if cfg.Auth.Web.AppBaseURL != "http://127.0.0.1:3000" {
		t.Errorf("auth.web.app_base_url = %q", cfg.Auth.Web.AppBaseURL)
	}
	if cfg.Auth.Web.CookieSameSite != "lax" {
		t.Errorf("auth.web.cookie_same_site = %q, want lax", cfg.Auth.Web.CookieSameSite)
	}
	if cfg.RateLimit.Enabled {
		t.Error("rate_limit should be disabled")
	}
	if cfg.Trace.Mode != "database" {
		t.Errorf("trace.mode = %q, want database", cfg.Trace.Mode)
	}
	// Defaults should be preserved for unset fields.
	if cfg.CORS.MaxAgeS != 3600 {
		t.Errorf("cors max_age_s = %d, want 3600 (default)", cfg.CORS.MaxAgeS)
	}
	if got := cfg.CORS.AllowedMethods; len(got) != 5 || got[0] != "GET" || got[1] != "POST" || got[2] != "PATCH" || got[3] != "DELETE" || got[4] != "OPTIONS" {
		t.Errorf("unexpected default cors.allowed_methods: %#v", got)
	}
}

func TestLoadBlockchainModeValid(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "gateway.yaml")
	content := `
trace:
  mode: "blockchain"
chain:
  rpc_url: "http://127.0.0.1:8545"
  chain_id: "31337"
  contract_address: "0x1234567890abcdef1234567890abcdef12345678"
  private_key: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
  tx_timeout_s: 45
  receipt_poll_interval_ms: 200
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Trace.Mode != "blockchain" {
		t.Fatalf("trace.mode = %q, want blockchain", cfg.Trace.Mode)
	}
	if cfg.Chain.ChainID != "31337" {
		t.Fatalf("chain.chain_id = %q, want 31337", cfg.Chain.ChainID)
	}
	if cfg.Chain.TxTimeoutS != 45 {
		t.Fatalf("chain.tx_timeout_s = %d, want 45", cfg.Chain.TxTimeoutS)
	}
	if cfg.Chain.ReceiptPollIntervalMS != 200 {
		t.Fatalf("chain.receipt_poll_interval_ms = %d, want 200", cfg.Chain.ReceiptPollIntervalMS)
	}
}

func TestLoadBlockchainModeMissingField(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "gateway.yaml")
	content := `
trace:
  mode: "blockchain"
chain:
  chain_id: "31337"
  contract_address: "0x1234567890abcdef1234567890abcdef12345678"
  private_key: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected validation error when blockchain rpc_url is missing")
	}
}

func TestLoadBlockchainModeInvalidFormat(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "gateway.yaml")
	content := `
trace:
  mode: "blockchain"
chain:
  rpc_url: "http://127.0.0.1:8545"
  chain_id: "31337"
  contract_address: "0xinvalid"
  private_key: "abc"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected validation error for invalid chain contract/key format")
	}
}

func TestLoadInvalidTraceMode(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "gateway.yaml")
	content := `
trace:
  mode: "legacy"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected validation error for invalid trace.mode")
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

func TestLoadAppliesAuthEnvOverrides(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "gateway.yaml")
	content := `
auth:
  mode: "disabled"
cors:
  allowed_origins:
    - "https://app.example.com"
  allow_credentials: true
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("LYCHEE_AUTH_MODE", "oidc")
	t.Setenv("LYCHEE_AUTH_OIDC_ISSUER_URL", "https://issuer.override.example.com")
	t.Setenv("LYCHEE_AUTH_OIDC_AUDIENCE", "lychee-ripe-override")
	t.Setenv("LYCHEE_AUTH_OIDC_WEB_CLIENT_ID", "web-client-override")
	t.Setenv("LYCHEE_AUTH_BOOTSTRAP_ADMIN_EMAIL", "bootstrap@example.com")
	t.Setenv("LYCHEE_AUTH_WEB_PUBLIC_BASE_URL", "https://gateway.example.com")
	t.Setenv("LYCHEE_AUTH_WEB_APP_BASE_URL", "https://app.example.com")
	t.Setenv("LYCHEE_AUTH_WEB_COOKIE_NAME", "session_cookie")
	t.Setenv("LYCHEE_AUTH_WEB_COOKIE_SECURE", "true")
	t.Setenv("LYCHEE_AUTH_WEB_COOKIE_SAME_SITE", "none")

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Auth.Mode != AuthModeOIDC {
		t.Fatalf("auth.mode = %q, want oidc", cfg.Auth.Mode)
	}
	if cfg.Auth.OIDC.IssuerURL != "https://issuer.override.example.com" {
		t.Fatalf("auth.oidc.issuer_url = %q", cfg.Auth.OIDC.IssuerURL)
	}
	if cfg.Auth.OIDC.Audience != "lychee-ripe-override" {
		t.Fatalf("auth.oidc.audience = %q", cfg.Auth.OIDC.Audience)
	}
	if cfg.Auth.OIDC.WebClientID != "web-client-override" {
		t.Fatalf("auth.oidc.web_client_id = %q", cfg.Auth.OIDC.WebClientID)
	}
	if cfg.Auth.BootstrapAdminEmail != "bootstrap@example.com" {
		t.Fatalf("auth.bootstrap_admin_email = %q", cfg.Auth.BootstrapAdminEmail)
	}
	if cfg.Auth.Web.PublicBaseURL != "https://gateway.example.com" {
		t.Fatalf("auth.web.public_base_url = %q", cfg.Auth.Web.PublicBaseURL)
	}
	if cfg.Auth.Web.AppBaseURL != "https://app.example.com" {
		t.Fatalf("auth.web.app_base_url = %q", cfg.Auth.Web.AppBaseURL)
	}
	if cfg.Auth.Web.CookieName != "session_cookie" {
		t.Fatalf("auth.web.cookie_name = %q", cfg.Auth.Web.CookieName)
	}
	if !cfg.Auth.Web.CookieSecure {
		t.Fatal("auth.web.cookie_secure = false, want true")
	}
	if cfg.Auth.Web.CookieSameSite != "none" {
		t.Fatalf("auth.web.cookie_same_site = %q, want none", cfg.Auth.Web.CookieSameSite)
	}
}

func TestLoadAppliesSeedEnvOverride(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "gateway.yaml")
	content := `
seed:
  default_resources_enabled: false
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("LYCHEE_SEED_DEFAULT_RESOURCES_ENABLED", "true")

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if !cfg.Seed.DefaultResourcesEnabled {
		t.Fatal("seed.default_resources_enabled = false, want true after env override")
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

func TestLoadRebasesRelativeSQLiteDSNFromWorkspace(t *testing.T) {
	dir := t.TempDir()
	workspaceDir := filepath.Join(dir, "workspace")
	configDir := filepath.Join(workspaceDir, "tooling", "configs")
	serviceDir := filepath.Join(workspaceDir, "services", "gateway")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(configDir, "gateway.yaml")
	content := `
db:
  driver: "sqlite"
  dsn: "mlops/artifacts/data/gateway.db"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})
	if err := os.Chdir(serviceDir); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	want := filepath.Join("..", "..", "mlops", "artifacts", "data", "gateway.db")
	if cfg.DB.DSN != want {
		t.Fatalf("db.dsn = %q, want %q", cfg.DB.DSN, want)
	}
}

func TestDefaultsRebaseWorkspaceDSNWithoutExistingArtifacts(t *testing.T) {
	dir := t.TempDir()
	serviceDir := filepath.Join(dir, "workspace", "services", "gateway")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatal(err)
	}

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})
	if err := os.Chdir(serviceDir); err != nil {
		t.Fatal(err)
	}

	cfg := Defaults()
	want := filepath.Join("..", "..", "mlops", "artifacts", "data", "gateway.db")
	if cfg.DB.DSN != want {
		t.Fatalf("db.dsn = %q, want %q", cfg.DB.DSN, want)
	}
}

func TestLoadRebasesExplicitConfigRelativeSQLiteDSN(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "tooling", "configs")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(configDir, "gateway.yaml")
	content := `
db:
  driver: "sqlite"
  dsn: "./data/gateway.db"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	want := filepath.Join(configDir, "data", "gateway.db")
	if cfg.DB.DSN != want {
		t.Fatalf("db.dsn = %q, want %q", cfg.DB.DSN, want)
	}
}

func TestLoadKeepsAbsoluteSQLiteDSN(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "gateway.yaml")
	absoluteDSN := filepath.Join(dir, "gateway.db")
	content := `
db:
  driver: "sqlite"
  dsn: "` + filepath.ToSlash(absoluteDSN) + `"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.DB.DSN != filepath.ToSlash(absoluteDSN) {
		t.Fatalf("db.dsn = %q, want %q", cfg.DB.DSN, filepath.ToSlash(absoluteDSN))
	}
}

func TestLoadKeepsSQLiteURIStyleDSN(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "gateway.yaml")
	content := `
db:
  driver: "sqlite"
  dsn: "file:gateway.db?cache=shared"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	want := "file:gateway.db?cache=shared"
	if cfg.DB.DSN != want {
		t.Fatalf("db.dsn = %q, want %q", cfg.DB.DSN, want)
	}
}

func TestLoadKeepsWorkspaceRelativeSQLiteDSNIndependentOfConfigMount(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "app")
	configDir := filepath.Join(dir, "etc", "gateway")
	dataDir := filepath.Join(appDir, "mlops", "artifacts", "data")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(configDir, "gateway.yaml")
	content := `
db:
  driver: "sqlite"
  dsn: "mlops/artifacts/data/gateway.db"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})
	if err := os.Chdir(appDir); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	want := filepath.Join("mlops", "artifacts", "data", "gateway.db")
	if filepath.ToSlash(cfg.DB.DSN) != filepath.ToSlash(want) {
		t.Fatalf("db.dsn = %q, want %q", cfg.DB.DSN, filepath.ToSlash(want))
	}
}

func TestLoadRebasesWorkspaceRelativeSQLiteDSNWithQuery(t *testing.T) {
	dir := t.TempDir()
	workspaceDir := filepath.Join(dir, "workspace")
	configDir := filepath.Join(workspaceDir, "tooling", "configs")
	serviceDir := filepath.Join(workspaceDir, "services", "gateway")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(configDir, "gateway.yaml")
	content := `
db:
  driver: "sqlite"
  dsn: "mlops/artifacts/data/gateway.db?cache=shared"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})
	if err := os.Chdir(serviceDir); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	want := filepath.Join("..", "..", "mlops", "artifacts", "data", "gateway.db") + "?cache=shared"
	if filepath.ToSlash(cfg.DB.DSN) != filepath.ToSlash(want) {
		t.Fatalf("db.dsn = %q, want %q", cfg.DB.DSN, filepath.ToSlash(want))
	}
}

func TestLoadRebasesFileSQLiteDSNWithQuery(t *testing.T) {
	dir := t.TempDir()
	workspaceDir := filepath.Join(dir, "workspace")
	configDir := filepath.Join(workspaceDir, "tooling", "configs")
	serviceDir := filepath.Join(workspaceDir, "services", "gateway")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(configDir, "gateway.yaml")
	content := `
db:
  driver: "sqlite"
  dsn: "file:mlops/artifacts/data/gateway.db?cache=shared"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})
	if err := os.Chdir(serviceDir); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	want := "file:" + filepath.Join("..", "..", "mlops", "artifacts", "data", "gateway.db") + "?cache=shared"
	if filepath.ToSlash(cfg.DB.DSN) != filepath.ToSlash(want) {
		t.Fatalf("db.dsn = %q, want %q", cfg.DB.DSN, filepath.ToSlash(want))
	}
}

func TestLoadKeepsWorkspaceRelativeSQLiteDSNWithQueryIndependentOfConfigMount(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "app")
	configDir := filepath.Join(dir, "etc", "gateway")
	dataDir := filepath.Join(appDir, "mlops", "artifacts", "data")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(configDir, "gateway.yaml")
	content := `
db:
  driver: "sqlite"
  dsn: "file:mlops/artifacts/data/gateway.db?cache=shared"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})
	if err := os.Chdir(appDir); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	want := "file:" + filepath.Join("mlops", "artifacts", "data", "gateway.db") + "?cache=shared"
	if filepath.ToSlash(cfg.DB.DSN) != filepath.ToSlash(want) {
		t.Fatalf("db.dsn = %q, want %q", cfg.DB.DSN, filepath.ToSlash(want))
	}
}

func TestAddr(t *testing.T) {
	cfg := Defaults()
	addr := cfg.Addr()
	if addr != "0.0.0.0:9000" {
		t.Errorf("addr = %q, want 0.0.0.0:9000", addr)
	}
}

func TestValidateRejectsCrossSiteWebCookieWithLax(t *testing.T) {
	cfg := Defaults()
	cfg.Auth.Mode = AuthModeOIDC
	cfg.Auth.OIDC.IssuerURL = "https://issuer.example.com"
	cfg.Auth.OIDC.Audience = "lychee-ripe"
	cfg.Auth.OIDC.WebClientID = "orchard-console-web"
	cfg.Auth.Web.PublicBaseURL = "https://gateway.example.com"
	cfg.Auth.Web.AppBaseURL = "https://console.other-example.com"
	cfg.Auth.Web.CookieSameSite = "lax"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for cross-site lax web cookie")
	}
}

func TestValidateRejectsSameSiteNoneWithoutSecure(t *testing.T) {
	cfg := Defaults()
	cfg.Auth.Mode = AuthModeOIDC
	cfg.Auth.OIDC.IssuerURL = "https://issuer.example.com"
	cfg.Auth.OIDC.Audience = "lychee-ripe"
	cfg.Auth.OIDC.WebClientID = "orchard-console-web"
	cfg.Auth.Web.PublicBaseURL = "https://gateway.example.com"
	cfg.Auth.Web.AppBaseURL = "https://console.other-example.com"
	cfg.Auth.Web.CookieSameSite = "none"
	cfg.Auth.Web.CookieSecure = false

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error when cookie_same_site=none without secure cookie")
	}
}

func TestValidateAllowsCrossSiteWebCookieWithNoneAndSecure(t *testing.T) {
	cfg := Defaults()
	cfg.Auth.Mode = AuthModeOIDC
	cfg.Auth.OIDC.IssuerURL = "https://issuer.example.com"
	cfg.Auth.OIDC.Audience = "lychee-ripe"
	cfg.Auth.OIDC.WebClientID = "orchard-console-web"
	cfg.Auth.Web.PublicBaseURL = "https://gateway.example.com"
	cfg.Auth.Web.AppBaseURL = "https://console.other-example.com"
	cfg.Auth.Web.CookieSameSite = "none"
	cfg.Auth.Web.CookieSecure = true
	cfg.CORS.AllowCredentials = true
	cfg.CORS.AllowedOrigins = []string{"https://console.other-example.com"}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestValidateRejectsOIDCWithoutCredentialedCORS(t *testing.T) {
	cfg := Defaults()
	cfg.Auth.Mode = AuthModeOIDC
	cfg.Auth.OIDC.IssuerURL = "https://issuer.example.com"
	cfg.Auth.OIDC.Audience = "lychee-ripe"
	cfg.Auth.OIDC.WebClientID = "orchard-console-web"
	cfg.Auth.Web.PublicBaseURL = "https://gateway.example.com"
	cfg.Auth.Web.AppBaseURL = "https://app.example.com"
	cfg.CORS.AllowCredentials = false
	cfg.CORS.AllowedOrigins = []string{"https://app.example.com"}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error when oidc is enabled without cors.allow_credentials")
	}
}

func TestValidateRejectsOIDCWithWildcardOrigins(t *testing.T) {
	cfg := Defaults()
	cfg.Auth.Mode = AuthModeOIDC
	cfg.Auth.OIDC.IssuerURL = "https://issuer.example.com"
	cfg.Auth.OIDC.Audience = "lychee-ripe"
	cfg.Auth.OIDC.WebClientID = "orchard-console-web"
	cfg.Auth.Web.PublicBaseURL = "https://gateway.example.com"
	cfg.Auth.Web.AppBaseURL = "https://app.example.com"
	cfg.CORS.AllowCredentials = true
	cfg.CORS.AllowedOrigins = []string{"*"}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error when oidc is enabled with wildcard cors origin")
	}
}
