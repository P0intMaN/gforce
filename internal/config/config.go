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
	// SSHPort is the TCP port the SSH server listens on (default 2222, not 22).
	SSHPort int `mapstructure:"ssh_port"`
	// SSHHostKeyPath is where the ED25519 host key is stored across restarts.
	// If the file doesn't exist it is generated once and saved here.
	SSHHostKeyPath string `mapstructure:"ssh_host_key_path"`
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

	// AutomaticEnv does not resolve nested struct keys during Unmarshal.
	// BindEnv pre-registers each mapping so it is evaluated deterministically.
	bindEnvs(v)

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

// bindEnvs explicitly maps every config key to its GFORCE_* env var.
// This is required because Viper's AutomaticEnv does not work reliably
// with nested structs during Unmarshal.
func bindEnvs(v *viper.Viper) {
	pairs := [][2]string{
		{"server.port", "GFORCE_SERVER_PORT"},
		{"server.base_url", "GFORCE_SERVER_BASE_URL"},
		{"server.allowed_origins", "GFORCE_SERVER_ALLOWED_ORIGINS"},
		{"server.read_timeout_secs", "GFORCE_SERVER_READ_TIMEOUT_SECS"},
		{"server.write_timeout_secs", "GFORCE_SERVER_WRITE_TIMEOUT_SECS"},
		{"db.dsn", "GFORCE_DB_DSN"},
		{"db.max_open_conns", "GFORCE_DB_MAX_OPEN_CONNS"},
		{"db.max_idle_conns", "GFORCE_DB_MAX_IDLE_CONNS"},
		{"git.storage_path", "GFORCE_GIT_STORAGE_PATH"},
		{"git.ssh_port", "GFORCE_GIT_SSH_PORT"},
		{"git.ssh_host_key_path", "GFORCE_GIT_SSH_HOST_KEY_PATH"},
		{"auth.jwt_secret", "GFORCE_AUTH_JWT_SECRET"},
		{"auth.token_ttl_minutes", "GFORCE_AUTH_TOKEN_TTL_MINUTES"},
		{"log.level", "GFORCE_LOG_LEVEL"},
		{"kubernetes.namespace", "GFORCE_KUBERNETES_NAMESPACE"},
	}
	for _, p := range pairs {
		_ = v.BindEnv(p[0], p[1])
	}
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
	v.SetDefault("git.ssh_port", 2222)
	v.SetDefault("git.ssh_host_key_path", "./data/host_key")
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
	if len(missing) > 0 {
		return fmt.Errorf("required environment variables not set: %s", strings.Join(missing, ", "))
	}

	return nil
}
