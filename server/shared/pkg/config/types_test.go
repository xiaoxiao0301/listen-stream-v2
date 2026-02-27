package config

import (
	"testing"
	"time"
)

func TestConfig_Structures(t *testing.T) {
	cfg := &Config{
		Infrastructure: InfrastructureConfig{
			Postgres: PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "password",
				Database: "test_db",
				SSLMode:  "disable",
			},
			Redis: RedisConfig{
				Host: "localhost",
				Port: 6379,
			},
			Server: ServerConfig{
				HTTPPort: 8080,
				GRPCPort: 9090,
			},
		},
		Business: BusinessConfig{
			Common: CommonConfig{
				JWTSecret:  "secret-key",
				JWTVersion: 1,
				JWTExpiry:  3600,
			},
		},
	}
	
	if cfg.Infrastructure.Postgres.Host != "localhost" {
		t.Errorf("PostgresHost = %v, want localhost", cfg.Infrastructure.Postgres.Host)
	}
	
	if cfg.Infrastructure.Redis.Port != 6379 {
		t.Errorf("RedisPort = %v, want 6379", cfg.Infrastructure.Redis.Port)
	}
	
	if cfg.Business.Common.JWTVersion != 1 {
		t.Errorf("JWTVersion = %v, want 1", cfg.Business.Common.JWTVersion)
	}
}

func TestPostgresConfig_WithReplicas(t *testing.T) {
	cfg := PostgresConfig{
		Host:     "master.db",
		Port:     5432,
		User:     "postgres",
		Password: "pass",
		Database: "mydb",
		ReadReplicas: []PostgresReplicaConfig{
			{Host: "replica1.db", Port: 5432},
			{Host: "replica2.db", Port: 5432},
		},
	}
	
	if len(cfg.ReadReplicas) != 2 {
		t.Errorf("ReadReplicas count = %v, want 2", len(cfg.ReadReplicas))
	}
	
	if cfg.ReadReplicas[0].Host != "replica1.db" {
		t.Errorf("Replica[0].Host = %v, want replica1.db", cfg.ReadReplicas[0].Host)
	}
}

func TestRedisConfig_SingleInstance(t *testing.T) {
	cfg := RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "pass",
		DB:       0,
	}
	
	if cfg.Cluster {
		t.Error("Cluster should be false for single instance")
	}
	
	if cfg.Host != "localhost" {
		t.Errorf("Host = %v, want localhost", cfg.Host)
	}
}

func TestRedisConfig_Cluster(t *testing.T) {
	cfg := RedisConfig{
		Cluster: true,
		ClusterAddrs: []string{
			"node1:6379",
			"node2:6379",
			"node3:6379",
		},
		PoolSize: 10,
	}
	
	if !cfg.Cluster {
		t.Error("Cluster should be true")
	}
	
	if len(cfg.ClusterAddrs) != 3 {
		t.Errorf("ClusterAddrs count = %v, want 3", len(cfg.ClusterAddrs))
	}
}

func TestConsulConfig(t *testing.T) {
	cfg := ConsulConfig{
		Address:    "localhost:8500",
		Scheme:     "http",
		Datacenter: "dc1",
		KVPrefix:   "listen-stream",
		CacheTTL:   30 * time.Second,
	}
	
	if cfg.Address != "localhost:8500" {
		t.Errorf("Address = %v, want localhost:8500", cfg.Address)
	}
	
	if cfg.CacheTTL != 30*time.Second {
		t.Errorf("CacheTTL = %v, want 30s", cfg.CacheTTL)
	}
}

func TestSMSConfig(t *testing.T) {
	cfg := SMSConfig{
		Aliyun: AliyunSMSConfig{
			AccessKeyID:     "key-id",
			AccessKeySecret: "secret",
			SignName:        "MyApp",
			TemplateCode:    "SMS_12345",
			Enabled:         true,
		},
		Tencent: TencentSMSConfig{
			Enabled: false,
		},
		Twilio: TwilioSMSConfig{
			Enabled: false,
		},
	}
	
	if !cfg.Aliyun.Enabled {
		t.Error("Aliyun SMS should be enabled")
	}
	
	if cfg.Tencent.Enabled {
		t.Error("Tencent SMS should be disabled")
	}
}

func TestAPIConfig(t *testing.T) {
	cfg := APIConfig{
		QQMusic: QQMusicConfig{
			BaseURL:   "https://api.qqmusic.com",
			APIKey:    "key",
			RateLimit: 100,
			Timeout:   30,
			Enabled:   true,
		},
		Joox: JooxConfig{
			Enabled: false,
		},
	}
	
	if !cfg.QQMusic.Enabled {
		t.Error("QQMusic should be enabled")
	}
	
	if cfg.QQMusic.RateLimit != 100 {
		t.Errorf("RateLimit = %v, want 100", cfg.QQMusic.RateLimit)
	}
}

func TestFeatureFlags(t *testing.T) {
	cfg := FeatureFlags{
		TokenIPBinding:    true,
		DeviceFingerprint: true,
		TwoFactorAuth:     false,
		RateLimitEnabled:  true,
		CacheWarmup:       true,
		OpenTelemetry:     false,
	}
	
	if !cfg.TokenIPBinding {
		t.Error("TokenIPBinding should be true")
	}
	
	if cfg.TwoFactorAuth {
		t.Error("TwoFactorAuth should be false")
	}
	
	if !cfg.RateLimitEnabled {
		t.Error("RateLimitEnabled should be true")
	}
}

func TestBusinessConfig(t *testing.T) {
	cfg := BusinessConfig{
		Common: CommonConfig{
			JWTSecret:     "my-jwt-secret-key-here",
			JWTVersion:    2,
			JWTExpiry:     3600,
			RefreshExpiry: 604800,
		},
		Features: FeatureFlags{
			TokenIPBinding: true,
		},
	}
	
	if cfg.Common.JWTVersion != 2 {
		t.Errorf("JWTVersion = %v, want 2", cfg.Common.JWTVersion)
	}
	
	if !cfg.Features.TokenIPBinding {
		t.Error("TokenIPBinding should be enabled")
	}
}
