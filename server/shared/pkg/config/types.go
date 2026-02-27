// Package config provides configuration management for the Listen Stream system.
package config

import "time"

// Config represents the application configuration.
type Config struct {
	// Infrastructure configuration (from file/env)
	Infrastructure InfrastructureConfig `mapstructure:"infrastructure"`
	
	// Business configuration (from Consul KV)
	Business BusinessConfig
}

// InfrastructureConfig holds basic infrastructure settings.
type InfrastructureConfig struct {
	Postgres PostgresConfig `mapstructure:"postgres"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Consul   ConsulConfig   `mapstructure:"consul"`
	Server   ServerConfig   `mapstructure:"server"`
}

// PostgresConfig holds PostgreSQL connection settings.
type PostgresConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime string        `mapstructure:"conn_max_lifetime"`
	
	// Read replicas (optional)
	ReadReplicas []PostgresReplicaConfig `mapstructure:"read_replicas"`
}

// PostgresReplicaConfig holds replica connection settings.
type PostgresReplicaConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	// Single instance or sentinel mode
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	
	// Cluster mode
	Cluster       bool     `mapstructure:"cluster"`
	ClusterAddrs  []string `mapstructure:"cluster_addrs"`
	
	// Connection pool
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	
	// Timeouts
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	PoolTimeout  time.Duration `mapstructure:"pool_timeout"`
}

// ServerConfig holds server settings.
type ServerConfig struct {
	HTTPPort         int           `mapstructure:"http_port"`
	GRPCPort         int           `mapstructure:"grpc_port"`
	ReadTimeout      time.Duration `mapstructure:"read_timeout"`
	WriteTimeout     time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout  time.Duration `mapstructure:"shutdown_timeout"`
	EnablePprof      bool          `mapstructure:"enable_pprof"`
}

// ConsulConfig holds Consul connection settings.
type ConsulConfig struct {
	Address    string        `mapstructure:"address"`
	Scheme     string        `mapstructure:"scheme"`
	Token      string        `mapstructure:"token"`
	Datacenter string        `mapstructure:"datacenter"`
	Timeout    time.Duration `mapstructure:"timeout"`
	
	// Service registration
	ServiceName string   `mapstructure:"service_name"`
	ServiceTags []string `mapstructure:"service_tags"`
	
	// Health check
	HealthCheckInterval time.Duration `mapstructure:"health_check_interval"`
	HealthCheckTimeout  time.Duration `mapstructure:"health_check_timeout"`
	
	// KV settings
	KVPrefix      string        `mapstructure:"kv_prefix"`
	CacheTTL      time.Duration `mapstructure:"cache_ttl"`
	WatchInterval time.Duration `mapstructure:"watch_interval"`
}

// BusinessConfig holds business configuration from Consul KV.
type BusinessConfig struct {
	Common   CommonConfig
	API      APIConfig
	SMS      SMSConfig
	Features FeatureFlags
}

// CommonConfig holds common business settings.
type CommonConfig struct {
	JWTSecret     string `json:"jwt_secret"`
	JWTVersion    int    `json:"jwt_version"`
	AESKey        string `json:"aes_key"`
	JWTExpiry     int    `json:"jwt_expiry"`      // seconds
	RefreshExpiry int    `json:"refresh_expiry"`  // seconds
}

// APIConfig holds third-party API configurations.
type APIConfig struct {
	QQMusic QQMusicConfig `json:"qq_music"`
	Joox    JooxConfig    `json:"joox"`
	NetEase NetEaseConfig `json:"netease"`
	Kugou   KugouConfig   `json:"kugou"`
}

// QQMusicConfig holds QQ Music API settings.
type QQMusicConfig struct {
	BaseURL    string `json:"base_url"`
	APIKey     string `json:"api_key"`      // encrypted
	RateLimit  int    `json:"rate_limit"`   // requests per second
	Timeout    int    `json:"timeout"`      // seconds
	Enabled    bool   `json:"enabled"`
}

// JooxConfig holds Joox API settings.
type JooxConfig struct {
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key"`
	RateLimit int    `json:"rate_limit"`
	Timeout   int    `json:"timeout"`
	Enabled   bool   `json:"enabled"`
}

// NetEaseConfig holds NetEase Cloud Music API settings.
type NetEaseConfig struct {
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key"`
	RateLimit int    `json:"rate_limit"`
	Timeout   int    `json:"timeout"`
	Enabled   bool   `json:"enabled"`
}

// KugouConfig holds Kugou Music API settings.
type KugouConfig struct {
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key"`
	RateLimit int    `json:"rate_limit"`
	Timeout   int    `json:"timeout"`
	Enabled   bool   `json:"enabled"`
}

// SMSConfig holds SMS provider configurations.
type SMSConfig struct {
	Aliyun  AliyunSMSConfig  `json:"aliyun"`
	Tencent TencentSMSConfig `json:"tencent"`
	Twilio  TwilioSMSConfig  `json:"twilio"`
}

// AliyunSMSConfig holds Aliyun SMS settings.
type AliyunSMSConfig struct {
	AccessKeyID     string `json:"access_key_id"`      // encrypted
	AccessKeySecret string `json:"access_key_secret"`  // encrypted
	SignName        string `json:"sign_name"`
	TemplateCode    string `json:"template_code"`
	Enabled         bool   `json:"enabled"`
}

// TencentSMSConfig holds Tencent Cloud SMS settings.
type TencentSMSConfig struct {
	SecretID     string `json:"secret_id"`      // encrypted
	SecretKey    string `json:"secret_key"`     // encrypted
	AppID        string `json:"app_id"`
	SignName     string `json:"sign_name"`
	TemplateID   string `json:"template_id"`
	Enabled      bool   `json:"enabled"`
}

// TwilioSMSConfig holds Twilio SMS settings.
type TwilioSMSConfig struct {
	AccountSID string `json:"account_sid"`  // encrypted
	AuthToken  string `json:"auth_token"`   // encrypted
	FromNumber string `json:"from_number"`
	Enabled    bool   `json:"enabled"`
}

// FeatureFlags holds feature toggle settings.
type FeatureFlags struct {
	TokenIPBinding     bool `json:"token_ip_binding"`
	DeviceFingerprint  bool `json:"device_fingerprint"`
	TwoFactorAuth      bool `json:"two_factor_auth"`
	RateLimitEnabled   bool `json:"rate_limit_enabled"`
	CacheWarmup        bool `json:"cache_warmup"`
	OpenTelemetry      bool `json:"open_telemetry"`
}