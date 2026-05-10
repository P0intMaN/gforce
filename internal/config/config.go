// Package config provides typed configuration loading for the gforce server.
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all runtime configuration for the gforce server.
type Config struct {
	Server     ServerConfig
	DB         DBConfig
	Git        GitConfig
	Auth       AuthConfig
	Log        LogConfig
	Kubernetes KubernetesConfig
}

// ServerConfig contains HTTP server settings.
type ServerConfig struct {
	// Port is the TCP port the HTTP server listens on.
	Port int `mapstructure:"port"`
	// BaseURL is the externally reachable URL of this server (used for clone URLs).
	BaseURL string `mapstructure:"base_url"`
	// AllowedOrigins lists origins allowed by the CORS middleware. Use ["*"] for open access.
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	// ReadTimeoutSecs is the maximum duration for reading an entire request.
	ReadTimeoutSecs int `mapstructure:"read_timeout_secs"`
	// WriteTimeoutSecs is the maximum duration for writing a response.
	WriteTimeoutSecs int `mapstructure:"write_timeout_secs"`
}

// DBConfig contains database connection settings.
type DBConfig struct {
	// DSN is the PostgreSQL connection string.
	DSN string `mapstructure:"dsn"`
	// MaxOpenConns is the maximum number of open connections to the database.
	MaxOpenConns int `mapstructure:"max_open_conns"`
	// MaxIdleConns is the maximum number of idle connections in the pool.
	MaxIdleConns int `mapstructure:"max_idle_conns"`
}

// GitConfig contains git storage settings.
type GitConfig struct {
	// StoragePath is the root directory where git repositories are stored on disk.
	StoragePath string `mapstructure:"storage_path"`
}

// AuthConfig contains authentication settings.
type AuthConfig struct {
	// JWTSecret is the HMAC secret used to sign and verify JWT tokens.
	JWTSecret string `mapstructure:"jwt_secret"`
	// TokenTTLMinutes is the number of minutes a JWT token remains valid.
	TokenTTLMinutes int `mapstructure:"token_ttl_minutes"`
}

// LogConfig contains structured logging settings.
type LogConfig struct {
	// Level is the minimum log level: debug, info, warn, error.
	Level string `mapstructure:"level"`
}

// KubernetesConfig contains Kubernetes operator settings.
type KubernetesConfig struct {
	// Namespace is the Kubernetes namespace the operator watches.
	Namespace string `mapstructure:"namespace"`
}

// Load reads configuration from environment variables (prefix: GFORCE_) and
// an optional config file, applies defaults, and validates required fields.
func Load() (*Config, error) {
	v := viper.New()

	v.SetEnvPrefix("gforce")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling config: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.base_url", "http://localhost:8080")
	v.SetDefault("server.allowed_origins", []string{"*"})
	v.SetDefault("server.read_timeout_secs", 30)
	v.SetDefault("server.write_timeout_secs", 30)
	v.SetDefault("db.max_open_conns", 25)
	v.SetDefault("db.max_idle_conns", 5)
	v.SetDefault("git.storage_path", "/var/lib/gforce/repos")
	v.SetDefault("auth.token_ttl_minutes", 60)
	v.SetDefault("log.level", "info")
	v.SetDefault("kubernetes.namespace", "gforce-system")
}

func validate(cfg *Config) error {
	var missing []string

	if cfg.DB.DSN == "" {
		missing = append(missing, "GFORCE_DB_DSN")
	}
	if cfg.Auth.JWTSecret == "" {
		missing = append(missing, "GFORCE_AUTH_JWT_SECRET")
	}
	if cfg.Git.StoragePath == "" {
		missing = append(missing, "GFORCE_GIT_STORAGE_PATH")
	}

	if len(missing) > 0 {
		return fmt.Errorf("required environment variables not set: %s", strings.Join(missing, ", "))
	}

	return nil
}
