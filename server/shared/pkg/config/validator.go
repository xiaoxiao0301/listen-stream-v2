package config

import (
	"fmt"
	"net/url"
	"time"
)

// Validator validates configuration values.
type Validator struct{}

// NewValidator creates a new configuration validator.
func NewValidator() *Validator {
	return &Validator{}
}

// Validate validates the entire configuration.
func (v *Validator) Validate(cfg *Config) error {
	if err := v.ValidateInfrastructure(&cfg.Infrastructure); err != nil {
		return fmt.Errorf("infrastructure config: %w", err)
	}
	
	if err := v.ValidateBusiness(&cfg.Business); err != nil {
		return fmt.Errorf("business config: %w", err)
	}
	
	return nil
}

// ValidateInfrastructure validates infrastructure configuration.
func (v *Validator) ValidateInfrastructure(cfg *InfrastructureConfig) error {
	if err := v.ValidatePostgres(&cfg.Postgres); err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	
	if err := v.ValidateRedis(&cfg.Redis); err != nil {
		return fmt.Errorf("redis: %w", err)
	}
	
	if err := v.ValidateServer(&cfg.Server); err != nil {
		return fmt.Errorf("server: %w", err)
	}
	
	return nil
}

// ValidatePostgres validates PostgreSQL configuration.
func (v *Validator) ValidatePostgres(cfg *PostgresConfig) error {
	if cfg.Host == "" {
		return fmt.Errorf("host is required")
	}
	
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return fmt.Errorf("invalid port: %d", cfg.Port)
	}
	
	if cfg.User == "" {
		return fmt.Errorf("user is required")
	}
	
	if cfg.Database == "" {
		return fmt.Errorf("database is required")
	}
	
	// Validate connection pool settings
	if cfg.MaxOpenConns < 0 {
		return fmt.Errorf("max_open_conns cannot be negative")
	}
	
	if cfg.MaxIdleConns < 0 {
		return fmt.Errorf("max_idle_conns cannot be negative")
	}
	
	if cfg.MaxIdleConns > cfg.MaxOpenConns && cfg.MaxOpenConns > 0 {
		return fmt.Errorf("max_idle_conns cannot exceed max_open_conns")
	}
	
	// Validate conn_max_lifetime if provided
	if cfg.ConnMaxLifetime != "" {
		if _, err := time.ParseDuration(cfg.ConnMaxLifetime); err != nil {
			return fmt.Errorf("invalid conn_max_lifetime: %w", err)
		}
	}
	
	return nil
}

// ValidateRedis validates Redis configuration.
func (v *Validator) ValidateRedis(cfg *RedisConfig) error {
	if cfg.Cluster {
		// Cluster mode
		if len(cfg.ClusterAddrs) == 0 {
			return fmt.Errorf("cluster_addrs is required in cluster mode")
		}
	} else {
		// Single instance mode
		if cfg.Host == "" {
			return fmt.Errorf("host is required")
		}
		
		if cfg.Port <= 0 || cfg.Port > 65535 {
			return fmt.Errorf("invalid port: %d", cfg.Port)
		}
	}
	
	// Validate pool settings
	if cfg.PoolSize < 0 {
		return fmt.Errorf("pool_size cannot be negative")
	}
	
	if cfg.MinIdleConns < 0 {
		return fmt.Errorf("min_idle_conns cannot be negative")
	}
	
	if cfg.MinIdleConns > cfg.PoolSize && cfg.PoolSize > 0 {
		return fmt.Errorf("min_idle_conns cannot exceed pool_size")
	}
	
	return nil
}

// ValidateServer validates server configuration.
func (v *Validator) ValidateServer(cfg *ServerConfig) error {
	if cfg.HTTPPort <= 0 || cfg.HTTPPort > 65535 {
		return fmt.Errorf("invalid http_port: %d", cfg.HTTPPort)
	}
	
	if cfg.GRPCPort > 0 {
		if cfg.GRPCPort > 65535 {
			return fmt.Errorf("invalid grpc_port: %d", cfg.GRPCPort)
		}
		
		if cfg.GRPCPort == cfg.HTTPPort {
			return fmt.Errorf("grpc_port cannot be the same as http_port")
		}
	}
	
	if cfg.ReadTimeout < 0 {
		return fmt.Errorf("read_timeout cannot be negative")
	}
	
	if cfg.WriteTimeout < 0 {
		return fmt.Errorf("write_timeout cannot be negative")
	}
	
	if cfg.ShutdownTimeout < 0 {
		return fmt.Errorf("shutdown_timeout cannot be negative")
	}
	
	return nil
}

// ValidateBusiness validates business configuration.
func (v *Validator) ValidateBusiness(cfg *BusinessConfig) error {
	if err := v.ValidateCommon(&cfg.Common); err != nil {
		return fmt.Errorf("common: %w", err)
	}
	
	// API and SMS configs are optional, but validate if present
	if err := v.ValidateAPI(&cfg.API); err != nil {
		return fmt.Errorf("api: %w", err)
	}
	
	if err := v.ValidateSMS(&cfg.SMS); err != nil {
		return fmt.Errorf("sms: %w", err)
	}
	
	return nil
}

// ValidateCommon validates common business configuration.
func (v *Validator) ValidateCommon(cfg *CommonConfig) error {
	if cfg.JWTSecret == "" {
		return fmt.Errorf("jwt_secret is required")
	}
	
	if len(cfg.JWTSecret) < 32 {
		return fmt.Errorf("jwt_secret must be at least 32 characters")
	}
	
	if cfg.JWTVersion < 1 {
		return fmt.Errorf("jwt_version must be >= 1")
	}
	
	if cfg.AESKey != "" && len(cfg.AESKey) != 32 {
		return fmt.Errorf("aes_key must be exactly 32 characters (256-bit)")
	}
	
	if cfg.JWTExpiry <= 0 {
		return fmt.Errorf("jwt_expiry must be positive")
	}
	
	if cfg.RefreshExpiry <= 0 {
		return fmt.Errorf("refresh_expiry must be positive")
	}
	
	if cfg.RefreshExpiry <= cfg.JWTExpiry {
		return fmt.Errorf("refresh_expiry must be greater than jwt_expiry")
	}
	
	return nil
}

// ValidateAPI validates API configuration.
func (v *Validator) ValidateAPI(cfg *APIConfig) error {
	if err := v.ValidateAPIEndpoint("qq_music", cfg.QQMusic.BaseURL, cfg.QQMusic.Enabled); err != nil {
		return err
	}
	
	if err := v.ValidateAPIEndpoint("joox", cfg.Joox.BaseURL, cfg.Joox.Enabled); err != nil {
		return err
	}
	
	if err := v.ValidateAPIEndpoint("netease", cfg.NetEase.BaseURL, cfg.NetEase.Enabled); err != nil {
		return err
	}
	
	if err := v.ValidateAPIEndpoint("kugou", cfg.Kugou.BaseURL, cfg.Kugou.Enabled); err != nil {
		return err
	}
	
	return nil
}

// ValidateAPIEndpoint validates an API endpoint.
func (v *Validator) ValidateAPIEndpoint(name, baseURL string, enabled bool) error {
	if !enabled {
		return nil
	}
	
	if baseURL == "" {
		return fmt.Errorf("%s base_url is required when enabled", name)
	}
	
	if _, err := url.Parse(baseURL); err != nil {
		return fmt.Errorf("%s invalid base_url: %w", name, err)
	}
	
	return nil
}

// ValidateSMS validates SMS configuration.
func (v *Validator) ValidateSMS(cfg *SMSConfig) error {
	// At least one SMS provider should be enabled
	hasEnabled := cfg.Aliyun.Enabled || cfg.Tencent.Enabled || cfg.Twilio.Enabled
	if !hasEnabled {
		return fmt.Errorf("at least one SMS provider must be enabled")
	}
	
	if cfg.Aliyun.Enabled {
		if cfg.Aliyun.AccessKeyID == "" {
			return fmt.Errorf("aliyun access_key_id is required")
		}
		if cfg.Aliyun.AccessKeySecret == "" {
			return fmt.Errorf("aliyun access_key_secret is required")
		}
		if cfg.Aliyun.SignName == "" {
			return fmt.Errorf("aliyun sign_name is required")
		}
		if cfg.Aliyun.TemplateCode == "" {
			return fmt.Errorf("aliyun template_code is required")
		}
	}
	
	if cfg.Tencent.Enabled {
		if cfg.Tencent.SecretID == "" {
			return fmt.Errorf("tencent secret_id is required")
		}
		if cfg.Tencent.SecretKey == "" {
			return fmt.Errorf("tencent secret_key is required")
		}
		if cfg.Tencent.AppID == "" {
			return fmt.Errorf("tencent app_id is required")
		}
		if cfg.Tencent.SignName == "" {
			return fmt.Errorf("tencent sign_name is required")
		}
		if cfg.Tencent.TemplateID == "" {
			return fmt.Errorf("tencent template_id is required")
		}
	}
	
	if cfg.Twilio.Enabled {
		if cfg.Twilio.AccountSID == "" {
			return fmt.Errorf("twilio account_sid is required")
		}
		if cfg.Twilio.AuthToken == "" {
			return fmt.Errorf("twilio auth_token is required")
		}
		if cfg.Twilio.FromNumber == "" {
			return fmt.Errorf("twilio from_number is required")
		}
	}
	
	return nil
}

// ValidateConsul validates Consul configuration.
func (v *Validator) ValidateConsul(cfg *ConsulConfig) error {
	if cfg.Address == "" {
		return fmt.Errorf("address is required")
	}
	
	if cfg.Scheme != "http" && cfg.Scheme != "https" {
		return fmt.Errorf("scheme must be http or https")
	}
	
	if cfg.ServiceName == "" {
		return fmt.Errorf("service_name is required")
	}
	
	if cfg.KVPrefix == "" {
		return fmt.Errorf("kv_prefix is required")
	}
	
	if cfg.CacheTTL < 0 {
		return fmt.Errorf("cache_ttl cannot be negative")
	}
	
	if cfg.HealthCheckInterval <= 0 {
		return fmt.Errorf("health_check_interval must be positive")
	}
	
	if cfg.HealthCheckTimeout <= 0 {
		return fmt.Errorf("health_check_timeout must be positive")
	}
	
	if cfg.HealthCheckTimeout >= cfg.HealthCheckInterval {
		return fmt.Errorf("health_check_timeout must be less than health_check_interval")
	}
	
	return nil
}
