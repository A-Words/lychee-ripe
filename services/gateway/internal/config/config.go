package config

import (
	"bytes"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/lychee-ripe/gateway/internal/domain"
	"golang.org/x/net/publicsuffix"
	"gopkg.in/yaml.v3"
)

// Config is the top-level gateway configuration.
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Upstream  UpstreamConfig  `yaml:"upstream"`
	DB        DBConfig        `yaml:"db"`
	Seed      SeedConfig      `yaml:"seed"`
	Trace     TraceConfig     `yaml:"trace"`
	Chain     ChainConfig     `yaml:"chain"`
	Auth      AuthConfig      `yaml:"auth"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
	CORS      CORSConfig      `yaml:"cors"`
	Logging   LoggingConfig   `yaml:"logging"`
}

// ServerConfig defines the gateway listener settings.
type ServerConfig struct {
	Host          string `yaml:"host"`
	Port          int    `yaml:"port"`
	ReadTimeoutS  int    `yaml:"read_timeout_s"`
	WriteTimeoutS int    `yaml:"write_timeout_s"`
}

// UpstreamConfig defines the backend FastAPI service connection.
type UpstreamConfig struct {
	BaseURL  string `yaml:"base_url"`
	TimeoutS int    `yaml:"timeout_s"`
}

// DBConfig defines database connection settings.
type DBConfig struct {
	Driver           string           `yaml:"driver"`
	DSN              string           `yaml:"dsn"`
	MaxOpenConns     int              `yaml:"max_open_conns"`
	MaxIdleConns     int              `yaml:"max_idle_conns"`
	ConnMaxLifetimeS int              `yaml:"conn_max_lifetime_s"`
	SQLite           SQLiteDBConfig   `yaml:"sqlite"`
	Postgres         PostgresDBConfig `yaml:"postgres"`
}

type SQLiteDBConfig struct {
	JournalMode   string `yaml:"journal_mode"`
	BusyTimeoutMS int    `yaml:"busy_timeout_ms"`
}

type PostgresDBConfig struct {
	SSLMode string `yaml:"ssl_mode"`
	Schema  string `yaml:"schema"`
}

type SeedConfig struct {
	DefaultResourcesEnabled bool `yaml:"default_resources_enabled"`
}

// TraceConfig defines the gateway trace runtime mode.
type TraceConfig struct {
	Mode domain.TraceMode `yaml:"mode"`
}

// ChainConfig defines EVM chain adapter settings.
type ChainConfig struct {
	RPCURL                string `yaml:"rpc_url"`
	ChainID               string `yaml:"chain_id"`
	ContractAddress       string `yaml:"contract_address"`
	PrivateKey            string `yaml:"private_key"`
	TxTimeoutS            int    `yaml:"tx_timeout_s"`
	ReceiptPollIntervalMS int    `yaml:"receipt_poll_interval_ms"`
}

// AuthConfig defines authentication settings.
type AuthConfig struct {
	Mode                AuthModeConfig `yaml:"mode"`
	BootstrapAdminEmail string         `yaml:"bootstrap_admin_email"`
	OIDC                OIDCConfig     `yaml:"oidc"`
	Web                 WebAuthConfig  `yaml:"web"`
}

type AuthModeConfig string

const (
	AuthModeDisabled AuthModeConfig = "disabled"
	AuthModeOIDC     AuthModeConfig = "oidc"
)

type OIDCConfig struct {
	IssuerURL   string `yaml:"issuer_url"`
	Audience    string `yaml:"audience"`
	WebClientID string `yaml:"web_client_id"`
}

type WebAuthConfig struct {
	PublicBaseURL  string `yaml:"public_base_url"`
	AppBaseURL     string `yaml:"app_base_url"`
	CookieName     string `yaml:"cookie_name"`
	CookieSecure   bool   `yaml:"cookie_secure"`
	CookieSameSite string `yaml:"cookie_same_site"`
	// SessionTTLS is the fallback session duration in seconds when the OIDC
	// token does not carry an explicit expiry (identity.ExpiresAt or
	// token.ExpiresIn). Defaults to 3600 (1 hour).
	SessionTTLS int `yaml:"session_ttl_s"`
}

// RateLimitConfig defines token-bucket rate limiting settings.
type RateLimitConfig struct {
	Enabled           bool     `yaml:"enabled"`
	RequestsPerSecond float64  `yaml:"requests_per_second"`
	Burst             int      `yaml:"burst"`
	ExcludePaths      []string `yaml:"exclude_paths"`
}

// CORSConfig defines cross-origin request settings.
type CORSConfig struct {
	Enabled          bool     `yaml:"enabled"`
	AllowedOrigins   []string `yaml:"allowed_origins"`
	AllowedMethods   []string `yaml:"allowed_methods"`
	AllowedHeaders   []string `yaml:"allowed_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
	MaxAgeS          int      `yaml:"max_age_s"`
}

// LoggingConfig defines structured logging settings.
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func (c CORSConfig) AllowsOrigin(origin string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return false
	}
	for _, allowedOrigin := range c.AllowedOrigins {
		allowedOrigin = strings.TrimSpace(allowedOrigin)
		if allowedOrigin == "*" {
			return true
		}
		if strings.EqualFold(allowedOrigin, origin) {
			return true
		}
	}
	return false
}

func isExplicitRelativePath(path string) bool {
	path = filepath.ToSlash(strings.TrimSpace(path))
	return path == "." || path == ".." || strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../")
}

func isWorkspaceRelativePath(path string) bool {
	if isExplicitRelativePath(path) {
		return false
	}

	path = filepath.ToSlash(filepath.Clean(path))
	head := path
	if idx := strings.Index(path, "/"); idx >= 0 {
		head = path[:idx]
	}

	switch head {
	case "clients", "services", "shared", "mlops", "tooling", "tests", "docs":
		return true
	default:
		return false
	}
}

func resolveWorkspacePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	path = filepath.Clean(path)
	if isExplicitRelativePath(path) || !isWorkspaceRelativePath(path) {
		return path
	}

	wd, err := os.Getwd()
	if err != nil {
		return path
	}

	if strings.HasSuffix(filepath.ToSlash(filepath.Clean(wd)), "/services/gateway") {
		return filepath.Join("..", "..", path)
	}

	return path
}

func normalizeSQLiteDSN(configPath, dsn string) string {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" || filepath.IsAbs(dsn) {
		return dsn
	}
	if dsn == ":memory:" {
		return dsn
	}

	prefix := ""
	if strings.HasPrefix(dsn, "file:") {
		prefix = "file:"
		dsn = strings.TrimPrefix(dsn, prefix)
	}

	query := ""
	if idx := strings.Index(dsn, "?"); idx >= 0 {
		query = dsn[idx:]
		dsn = dsn[:idx]
	}

	if dsn == "" {
		return prefix + query
	}

	if isExplicitRelativePath(dsn) {
		return prefix + filepath.Clean(filepath.Join(filepath.Dir(configPath), dsn)) + query
	}

	return prefix + resolveWorkspacePath(dsn) + query
}

// Defaults returns a Config with sensible defaults.
func Defaults() Config {
	return Config{
		Server: ServerConfig{
			Host:          "0.0.0.0",
			Port:          9000,
			ReadTimeoutS:  30,
			WriteTimeoutS: 60,
		},
		Upstream: UpstreamConfig{
			BaseURL:  "http://127.0.0.1:8000",
			TimeoutS: 30,
		},
		DB: DBConfig{
			Driver:           "sqlite",
			DSN:              resolveWorkspacePath(filepath.Join("mlops", "artifacts", "data", "gateway.db")),
			MaxOpenConns:     10,
			MaxIdleConns:     5,
			ConnMaxLifetimeS: 300,
			SQLite: SQLiteDBConfig{
				JournalMode:   "WAL",
				BusyTimeoutMS: 5000,
			},
			Postgres: PostgresDBConfig{
				SSLMode: "disable",
				Schema:  "public",
			},
		},
		Seed: SeedConfig{
			DefaultResourcesEnabled: false,
		},
		Trace: TraceConfig{
			Mode: domain.TraceModeDatabase,
		},
		Chain: ChainConfig{
			TxTimeoutS:            30,
			ReceiptPollIntervalMS: 500,
		},
		Auth: AuthConfig{
			Mode: AuthModeDisabled,
			Web: WebAuthConfig{
				CookieName:     "lychee_session",
				CookieSecure:   false,
				CookieSameSite: "lax",
				SessionTTLS:    3600,
			},
		},
		RateLimit: RateLimitConfig{
			Enabled:           true,
			RequestsPerSecond: 20,
			Burst:             40,
			ExcludePaths:      []string{"/healthz"},
		},
		CORS: CORSConfig{
			Enabled:          true,
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Content-Type", "Authorization", "X-API-Key"},
			AllowCredentials: false,
			MaxAgeS:          3600,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}
}

// Load reads the gateway config from the given path.
// If the path is empty, it tries the LYCHEE_GATEWAY_CONFIG env var,
// then falls back to tooling/configs/gateway.yaml from the repo root or gateway workspace.
func Load(path string) (Config, error) {
	if path == "" {
		path = os.Getenv("LYCHEE_GATEWAY_CONFIG")
	}
	if path == "" {
		path = resolveWorkspacePath(filepath.Join("tooling", "configs", "gateway.yaml"))
	} else {
		path = resolveWorkspacePath(path)
	}

	cfg := Defaults()

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read gateway config %s: %w", path, err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse gateway config %s: %w", path, err)
	}
	if strings.EqualFold(strings.TrimSpace(cfg.DB.Driver), "sqlite") {
		cfg.DB.DSN = normalizeSQLiteDSN(path, cfg.DB.DSN)
	}
	applyEnvOverrides(&cfg)
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate gateway config %s: %w", path, err)
	}

	return cfg, nil
}

// Addr returns the listen address string "host:port".
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

func (c *Config) Validate() error {
	c.DB.Driver = strings.ToLower(strings.TrimSpace(c.DB.Driver))
	switch c.DB.Driver {
	case "sqlite", "postgres":
	default:
		return fmt.Errorf("db.driver must be one of sqlite|postgres, got %q", c.DB.Driver)
	}

	if strings.TrimSpace(c.DB.DSN) == "" {
		return fmt.Errorf("db.dsn is required")
	}

	if c.DB.MaxOpenConns <= 0 {
		return fmt.Errorf("db.max_open_conns must be > 0")
	}
	if c.DB.MaxIdleConns < 0 {
		return fmt.Errorf("db.max_idle_conns must be >= 0")
	}
	if c.DB.ConnMaxLifetimeS < 0 {
		return fmt.Errorf("db.conn_max_lifetime_s must be >= 0")
	}

	if c.DB.SQLite.JournalMode == "" {
		c.DB.SQLite.JournalMode = "WAL"
	}
	if c.DB.Postgres.SSLMode == "" {
		c.DB.Postgres.SSLMode = "disable"
	}
	if c.DB.Postgres.Schema == "" {
		c.DB.Postgres.Schema = "public"
	}
	if c.Trace.Mode == "" {
		c.Trace.Mode = domain.TraceModeDatabase
	}
	switch c.Trace.Mode {
	case domain.TraceModeDatabase, domain.TraceModeBlockchain:
	default:
		return fmt.Errorf("trace.mode must be one of database|blockchain, got %q", c.Trace.Mode)
	}

	if c.Chain.TxTimeoutS <= 0 {
		c.Chain.TxTimeoutS = 30
	}
	if c.Chain.ReceiptPollIntervalMS <= 0 {
		c.Chain.ReceiptPollIntervalMS = 500
	}

	if c.Trace.Mode == domain.TraceModeBlockchain {
		if strings.TrimSpace(c.Chain.RPCURL) == "" {
			return fmt.Errorf("chain.rpc_url is required when trace.mode=blockchain")
		}
		parsedURL, err := url.Parse(strings.TrimSpace(c.Chain.RPCURL))
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			return fmt.Errorf("chain.rpc_url must be a valid absolute url")
		}

		if strings.TrimSpace(c.Chain.ChainID) == "" {
			return fmt.Errorf("chain.chain_id is required when trace.mode=blockchain")
		}
		chainID, err := strconv.ParseInt(strings.TrimSpace(c.Chain.ChainID), 10, 64)
		if err != nil || chainID <= 0 {
			return fmt.Errorf("chain.chain_id must be a positive integer string")
		}

		if strings.TrimSpace(c.Chain.ContractAddress) == "" {
			return fmt.Errorf("chain.contract_address is required when trace.mode=blockchain")
		}
		if ok, _ := regexp.MatchString(`^0x[0-9a-fA-F]{40}$`, strings.TrimSpace(c.Chain.ContractAddress)); !ok {
			return fmt.Errorf("chain.contract_address must be a 20-byte hex address")
		}

		if strings.TrimSpace(c.Chain.PrivateKey) == "" {
			return fmt.Errorf("chain.private_key is required when trace.mode=blockchain")
		}
		key := strings.TrimPrefix(strings.TrimSpace(c.Chain.PrivateKey), "0x")
		if ok, _ := regexp.MatchString(`^[0-9a-fA-F]{64}$`, key); !ok {
			return fmt.Errorf("chain.private_key must be a 32-byte hex private key")
		}
	}

	if c.Auth.Mode == "" {
		c.Auth.Mode = AuthModeDisabled
	}
	switch c.Auth.Mode {
	case AuthModeDisabled:
	case AuthModeOIDC:
		if strings.TrimSpace(c.Auth.OIDC.IssuerURL) == "" {
			return fmt.Errorf("auth.oidc.issuer_url is required when auth.mode=oidc")
		}
		if strings.TrimSpace(c.Auth.OIDC.Audience) == "" {
			return fmt.Errorf("auth.oidc.audience is required when auth.mode=oidc")
		}
		if strings.TrimSpace(c.Auth.OIDC.WebClientID) == "" {
			return fmt.Errorf("auth.oidc.web_client_id is required when auth.mode=oidc")
		}
		if strings.TrimSpace(c.Auth.Web.PublicBaseURL) == "" {
			return fmt.Errorf("auth.web.public_base_url is required when auth.mode=oidc")
		}
		if !isAbsoluteURL(c.Auth.Web.PublicBaseURL) {
			return fmt.Errorf("auth.web.public_base_url must be a valid absolute url")
		}
		if strings.TrimSpace(c.Auth.Web.AppBaseURL) == "" {
			return fmt.Errorf("auth.web.app_base_url is required when auth.mode=oidc")
		}
		if !isAbsoluteURL(c.Auth.Web.AppBaseURL) {
			return fmt.Errorf("auth.web.app_base_url must be a valid absolute url")
		}
	default:
		return fmt.Errorf("auth.mode must be one of disabled|oidc, got %q", c.Auth.Mode)
	}

	if c.Auth.Web.SessionTTLS <= 0 {
		c.Auth.Web.SessionTTLS = 3600
	}

	if strings.TrimSpace(c.Auth.Web.CookieName) == "" {
		c.Auth.Web.CookieName = "lychee_session"
	}
	if strings.TrimSpace(c.Auth.Web.CookieSameSite) == "" {
		c.Auth.Web.CookieSameSite = "lax"
	}
	c.Auth.Web.CookieSameSite = strings.ToLower(strings.TrimSpace(c.Auth.Web.CookieSameSite))
	switch c.Auth.Web.CookieSameSite {
	case "lax", "none":
	default:
		return fmt.Errorf("auth.web.cookie_same_site must be one of lax|none, got %q", c.Auth.Web.CookieSameSite)
	}
	if c.Auth.Web.CookieSameSite == "none" && !c.Auth.Web.CookieSecure {
		return fmt.Errorf("auth.web.cookie_secure must be true when auth.web.cookie_same_site=none")
	}
	if c.Auth.Mode == AuthModeOIDC && c.Auth.Web.CookieSameSite != "none" && !isSameSitePair(c.Auth.Web.PublicBaseURL, c.Auth.Web.AppBaseURL) {
		return fmt.Errorf("auth.web.public_base_url and auth.web.app_base_url must be same-site unless auth.web.cookie_same_site=none")
	}
	if c.Auth.Mode == AuthModeOIDC {
		if !c.CORS.AllowCredentials {
			return fmt.Errorf("cors.allow_credentials must be true when auth.mode=oidc")
		}
		if len(c.CORS.AllowedOrigins) == 0 {
			return fmt.Errorf("cors.allowed_origins must list at least one trusted origin when auth.mode=oidc")
		}
		for _, origin := range c.CORS.AllowedOrigins {
			if strings.TrimSpace(origin) == "*" {
				return fmt.Errorf("cors.allowed_origins cannot contain * when auth.mode=oidc")
			}
		}
	}

	if c.CORS.AllowCredentials {
		for _, origin := range c.CORS.AllowedOrigins {
			if strings.TrimSpace(origin) == "*" {
				return fmt.Errorf("cors.allowed_origins cannot contain * when cors.allow_credentials=true")
			}
		}
	}

	return nil
}

func applyEnvOverrides(cfg *Config) {
	if cfg == nil {
		return
	}
	if value := strings.TrimSpace(os.Getenv("LYCHEE_AUTH_MODE")); value != "" {
		cfg.Auth.Mode = AuthModeConfig(strings.ToLower(value))
	}
	if value := strings.TrimSpace(os.Getenv("LYCHEE_AUTH_OIDC_ISSUER_URL")); value != "" {
		cfg.Auth.OIDC.IssuerURL = value
	}
	if value := strings.TrimSpace(os.Getenv("LYCHEE_AUTH_OIDC_AUDIENCE")); value != "" {
		cfg.Auth.OIDC.Audience = value
	}
	if value := strings.TrimSpace(os.Getenv("LYCHEE_AUTH_OIDC_WEB_CLIENT_ID")); value != "" {
		cfg.Auth.OIDC.WebClientID = value
	}
	if value := strings.TrimSpace(os.Getenv("LYCHEE_AUTH_BOOTSTRAP_ADMIN_EMAIL")); value != "" {
		cfg.Auth.BootstrapAdminEmail = value
	}
	if value := strings.TrimSpace(os.Getenv("LYCHEE_AUTH_WEB_PUBLIC_BASE_URL")); value != "" {
		cfg.Auth.Web.PublicBaseURL = value
	}
	if value := strings.TrimSpace(os.Getenv("LYCHEE_AUTH_WEB_APP_BASE_URL")); value != "" {
		cfg.Auth.Web.AppBaseURL = value
	}
	if value := strings.TrimSpace(os.Getenv("LYCHEE_AUTH_WEB_COOKIE_NAME")); value != "" {
		cfg.Auth.Web.CookieName = value
	}
	if value := strings.TrimSpace(os.Getenv("LYCHEE_AUTH_WEB_COOKIE_SECURE")); value != "" {
		cfg.Auth.Web.CookieSecure = strings.EqualFold(value, "true") || value == "1"
	}
	if value := strings.TrimSpace(os.Getenv("LYCHEE_AUTH_WEB_COOKIE_SAME_SITE")); value != "" {
		cfg.Auth.Web.CookieSameSite = value
	}
	if value := strings.TrimSpace(os.Getenv("LYCHEE_AUTH_WEB_SESSION_TTL_S")); value != "" {
		if ttl, err := strconv.Atoi(value); err == nil && ttl > 0 {
			cfg.Auth.Web.SessionTTLS = ttl
		}
	}
	if value := strings.TrimSpace(os.Getenv("LYCHEE_SEED_DEFAULT_RESOURCES_ENABLED")); value != "" {
		cfg.Seed.DefaultResourcesEnabled = strings.EqualFold(value, "true") || value == "1"
	}
	if value := strings.TrimSpace(os.Getenv("LYCHEE_CORS_ALLOW_CREDENTIALS")); value != "" {
		cfg.CORS.AllowCredentials = strings.EqualFold(value, "true") || value == "1"
	}
}

func isAbsoluteURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	return err == nil && parsed != nil && parsed.Scheme != "" && parsed.Host != ""
}

func isSameSitePair(leftRaw string, rightRaw string) bool {
	left, err := url.Parse(strings.TrimSpace(leftRaw))
	if err != nil || left == nil {
		return false
	}
	right, err := url.Parse(strings.TrimSpace(rightRaw))
	if err != nil || right == nil {
		return false
	}
	return siteKey(left) != "" && siteKey(left) == siteKey(right)
}

func siteKey(parsed *url.URL) string {
	if parsed == nil {
		return ""
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if scheme == "" || host == "" {
		return ""
	}
	if host == "localhost" || net.ParseIP(host) != nil {
		return scheme + "://" + host
	}
	siteHost, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		return scheme + "://" + host
	}
	return scheme + "://" + siteHost
}
