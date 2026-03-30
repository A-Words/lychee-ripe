package config

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the top-level gateway configuration.
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Upstream  UpstreamConfig  `yaml:"upstream"`
	DB        DBConfig        `yaml:"db"`
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

// ChainConfig defines EVM chain adapter settings.
type ChainConfig struct {
	Enabled               bool   `yaml:"enabled"`
	RPCURL                string `yaml:"rpc_url"`
	ChainID               string `yaml:"chain_id"`
	ContractAddress       string `yaml:"contract_address"`
	PrivateKey            string `yaml:"private_key"`
	TxTimeoutS            int    `yaml:"tx_timeout_s"`
	ReceiptPollIntervalMS int    `yaml:"receipt_poll_interval_ms"`
}

// AuthConfig defines API key authentication settings.
type AuthConfig struct {
	Enabled bool     `yaml:"enabled"`
	APIKeys []string `yaml:"api_keys"`
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
	Enabled        bool     `yaml:"enabled"`
	AllowedOrigins []string `yaml:"allowed_origins"`
	AllowedMethods []string `yaml:"allowed_methods"`
	AllowedHeaders []string `yaml:"allowed_headers"`
	MaxAgeS        int      `yaml:"max_age_s"`
}

// LoggingConfig defines structured logging settings.
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func resolveWorkspacePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	candidates := []string{
		path,
		filepath.Join("..", "..", path),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Dir(candidate)); err == nil {
			return candidate
		}
	}

	return path
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
		Chain: ChainConfig{
			Enabled:               false,
			TxTimeoutS:            30,
			ReceiptPollIntervalMS: 500,
		},
		Auth: AuthConfig{
			Enabled: false,
		},
		RateLimit: RateLimitConfig{
			Enabled:           true,
			RequestsPerSecond: 20,
			Burst:             40,
			ExcludePaths:      []string{"/healthz"},
		},
		CORS: CORSConfig{
			Enabled:        true,
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST", "OPTIONS"},
			AllowedHeaders: []string{"Content-Type", "Authorization", "X-API-Key"},
			MaxAgeS:        3600,
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

	if c.Chain.TxTimeoutS <= 0 {
		c.Chain.TxTimeoutS = 30
	}
	if c.Chain.ReceiptPollIntervalMS <= 0 {
		c.Chain.ReceiptPollIntervalMS = 500
	}

	if !c.Chain.Enabled {
		return nil
	}

	if strings.TrimSpace(c.Chain.RPCURL) == "" {
		return fmt.Errorf("chain.rpc_url is required when chain.enabled=true")
	}
	parsedURL, err := url.Parse(strings.TrimSpace(c.Chain.RPCURL))
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("chain.rpc_url must be a valid absolute url")
	}

	if strings.TrimSpace(c.Chain.ChainID) == "" {
		return fmt.Errorf("chain.chain_id is required when chain.enabled=true")
	}
	chainID, err := strconv.ParseInt(strings.TrimSpace(c.Chain.ChainID), 10, 64)
	if err != nil || chainID <= 0 {
		return fmt.Errorf("chain.chain_id must be a positive integer string")
	}

	if strings.TrimSpace(c.Chain.ContractAddress) == "" {
		return fmt.Errorf("chain.contract_address is required when chain.enabled=true")
	}
	if ok, _ := regexp.MatchString(`^0x[0-9a-fA-F]{40}$`, strings.TrimSpace(c.Chain.ContractAddress)); !ok {
		return fmt.Errorf("chain.contract_address must be a 20-byte hex address")
	}

	if strings.TrimSpace(c.Chain.PrivateKey) == "" {
		return fmt.Errorf("chain.private_key is required when chain.enabled=true")
	}
	key := strings.TrimPrefix(strings.TrimSpace(c.Chain.PrivateKey), "0x")
	if ok, _ := regexp.MatchString(`^[0-9a-fA-F]{64}$`, key); !ok {
		return fmt.Errorf("chain.private_key must be a 32-byte hex private key")
	}

	return nil
}
