package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// FileLoader loads configuration from YAML files and environment variables.
type FileLoader struct {
	configPath string
	validator  *Validator
}

// NewFileLoader creates a new file loader.
func NewFileLoader(configPath string) *FileLoader {
	return &FileLoader{
		configPath: configPath,
		validator:  NewValidator(),
	}
}

// Load loads infrastructure configuration from file and environment variables.
func (l *FileLoader) Load() (*Config, error) {
	v := viper.New()
	
	// Set config file
	if l.configPath != "" {
		v.SetConfigFile(l.configPath)
	} else {
		// Default config paths
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./config")
		v.AddConfigPath(".")
	}
	
	// Environment variable support
	v.SetEnvPrefix("LS") // Listen Stream prefix
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	
	// Set defaults
	l.setDefaults(v)
	
	// Read config file
	if err := v.ReadInConfig(); err != nil {
		// Config file is optional if all required values are in env
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}
	
	// Unmarshal config
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	// Validate infrastructure config
	if err := l.validator.ValidateInfrastructure(&config.Infrastructure); err != nil {
		return nil, fmt.Errorf("invalid infrastructure config: %w", err)
	}
	
	return &config, nil
}

// setDefaults sets default configuration values.
func (l *FileLoader) setDefaults(v *viper.Viper) {
	// PostgreSQL defaults
	v.SetDefault("infrastructure.postgres.host", "localhost")
	v.SetDefault("infrastructure.postgres.port", 5432)
	v.SetDefault("infrastructure.postgres.user", "postgres")
	v.SetDefault("infrastructure.postgres.ssl_mode", "disable")
	v.SetDefault("infrastructure.postgres.max_open_conns", 25)
	v.SetDefault("infrastructure.postgres.max_idle_conns", 5)
	v.SetDefault("infrastructure.postgres.conn_max_lifetime", "5m")
	
	// Redis defaults
	v.SetDefault("infrastructure.redis.host", "localhost")
	v.SetDefault("infrastructure.redis.port", 6379)
	v.SetDefault("infrastructure.redis.db", 0)
	v.SetDefault("infrastructure.redis.cluster", false)
	v.SetDefault("infrastructure.redis.pool_size", 10)
	v.SetDefault("infrastructure.redis.min_idle_conns", 5)
	v.SetDefault("infrastructure.redis.dial_timeout", 5*time.Second)
	v.SetDefault("infrastructure.redis.read_timeout", 3*time.Second)
	v.SetDefault("infrastructure.redis.write_timeout", 3*time.Second)
	v.SetDefault("infrastructure.redis.pool_timeout", 4*time.Second)
	
	// Server defaults
	v.SetDefault("infrastructure.server.http_port", 8080)
	v.SetDefault("infrastructure.server.grpc_port", 9090)
	v.SetDefault("infrastructure.server.read_timeout", 30*time.Second)
	v.SetDefault("infrastructure.server.write_timeout", 30*time.Second)
	v.SetDefault("infrastructure.server.shutdown_timeout", 30*time.Second)
	v.SetDefault("infrastructure.server.enable_pprof", false)
}

// LoadFromEnv loads configuration from environment variables only.
// Useful for containerized deployments.
func LoadFromEnv() (*Config, error) {
	loader := NewFileLoader("")
	
	v := viper.New()
	v.SetEnvPrefix("LS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	
	loader.setDefaults(v)
	
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config from env: %w", err)
	}
	
	if err := loader.validator.ValidateInfrastructure(&config.Infrastructure); err != nil {
		return nil, fmt.Errorf("invalid infrastructure config: %w", err)
	}
	
	return &config, nil
}

// CreateExampleConfig creates an example configuration file.
func CreateExampleConfig(outputPath string) error {
	exampleYAML := `# Listen Stream Configuration Example
infrastructure:
  postgres:
    host: localhost
    port: 5432
    user: postgres
    password: your_password
    database: listen_stream
    ssl_mode: disable
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: 5m
    
    # Optional: Read replicas
    # read_replicas:
    #   - host: replica1.example.com
    #     port: 5432
    #   - host: replica2.example.com
    #     port: 5432

  redis:
    host: localhost
    port: 6379
    password: ""
    db: 0
    pool_size: 10
    min_idle_conns: 5
    dial_timeout: 5s
    read_timeout: 3s
    write_timeout: 3s
    pool_timeout: 4s
    
    # Cluster mode (optional)
    # cluster: true
    # cluster_addrs:
    #   - redis1.example.com:6379
    #   - redis2.example.com:6379
    #   - redis3.example.com:6379

  server:
    http_port: 8080
    grpc_port: 9090
    read_timeout: 30s
    write_timeout: 30s
    shutdown_timeout: 30s
    enable_pprof: false

# Business configuration is loaded from Consul KV
# See docs/config-management-strategy.md for details
`
	
	// Create directory if not exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Write example config
	if err := os.WriteFile(outputPath, []byte(exampleYAML), 0644); err != nil {
		return fmt.Errorf("failed to write example config: %w", err)
	}
	
	return nil
}
