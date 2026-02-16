package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the top-level gateway configuration.
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Upstream  UpstreamConfig  `yaml:"upstream"`
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
// then falls back to configs/gateway.yaml relative to the working directory.
func Load(path string) (Config, error) {
	if path == "" {
		path = os.Getenv("LYCHEE_GATEWAY_CONFIG")
	}
	if path == "" {
		path = filepath.Join("configs", "gateway.yaml")
	}

	cfg := Defaults()

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read gateway config %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse gateway config %s: %w", path, err)
	}

	return cfg, nil
}

// Addr returns the listen address string "host:port".
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
