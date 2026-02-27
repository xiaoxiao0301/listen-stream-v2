package config

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
)

// ConsulLoader loads business configuration from Consul KV.
type ConsulLoader struct {
	client    *api.Client
	kvPrefix  string
	cache     *Cache
	validator *Validator
	mu        sync.RWMutex
	config    *BusinessConfig
}

// NewConsulLoader creates a new Consul KV loader.
func NewConsulLoader(cfg *ConsulConfig) (*ConsulLoader, error) {
	// Validate Consul config
	validator := NewValidator()
	if err := validator.ValidateConsul(cfg); err != nil {
		return nil, fmt.Errorf("invalid consul config: %w", err)
	}
	
	// Create Consul client
	config := api.DefaultConfig()
	config.Address = cfg.Address
	config.Scheme = cfg.Scheme
	config.Token = cfg.Token
	config.Datacenter = cfg.Datacenter
	
	if cfg.Timeout > 0 {
		config.HttpClient.Timeout = cfg.Timeout
	}
	
	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consul client: %w", err)
	}
	
	// Create cache
	cacheTTL := cfg.CacheTTL
	if cacheTTL == 0 {
		cacheTTL = 30 * time.Second // Default 30s cache
	}
	
	return &ConsulLoader{
		client:    client,
		kvPrefix:  cfg.KVPrefix,
		cache:     NewCache(cacheTTL),
		validator: validator,
	}, nil
}

// Load loads business configuration from Consul KV.
func (l *ConsulLoader) Load(ctx context.Context) (*BusinessConfig, error) {
	// Check cache first
	if cached, ok := l.cache.Get("business_config"); ok {
		if cfg, ok := cached.(*BusinessConfig); ok {
			return cfg, nil
		}
	}
	
	// Load from Consul
	cfg := &BusinessConfig{}
	
	// Load common config
	common, err := l.loadCommon(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load common config: %w", err)
	}
	cfg.Common = *common
	
	// Load API config
	apiCfg, err := l.loadAPI(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load api config: %w", err)
	}
	cfg.API = *apiCfg
	
	// Load SMS config
	smsCfg, err := l.loadSMS(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load sms config: %w", err)
	}
	cfg.SMS = *smsCfg
	
	// Load feature flags
	features, err := l.loadFeatures(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load feature flags: %w", err)
	}
	cfg.Features = *features
	
	// Validate business config
	if err := l.validator.ValidateBusiness(cfg); err != nil {
		return nil, fmt.Errorf("invalid business config: %w", err)
	}
	
	// Cache the config
	l.cache.Set("business_config", cfg)
	
	// Store in memory
	l.mu.Lock()
	l.config = cfg
	l.mu.Unlock()
	
	return cfg, nil
}

// GetCached returns the cached configuration without querying Consul.
func (l *ConsulLoader) GetCached() *BusinessConfig {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	if l.config != nil {
		return l.config
	}
	
	// Try cache
	if cached, ok := l.cache.Get("business_config"); ok {
		if cfg, ok := cached.(*BusinessConfig); ok {
			return cfg
		}
	}
	
	return nil
}

// InvalidateCache clears the configuration cache.
func (l *ConsulLoader) InvalidateCache() {
	l.cache.Clear()
}

// loadCommon loads common configuration from Consul KV.
func (l *ConsulLoader) loadCommon(ctx context.Context) (*CommonConfig, error) {
	prefix := path.Join(l.kvPrefix, "common")
	
	cfg := &CommonConfig{}
	
	// Load each key
	jwtSecret, err := l.getKV(ctx, path.Join(prefix, "jwt_secret"))
	if err != nil {
		return nil, err
	}
	cfg.JWTSecret = jwtSecret
	
	jwtVersion, err := l.getKV(ctx, path.Join(prefix, "jwt_version"))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(jwtVersion), &cfg.JWTVersion); err != nil {
		return nil, fmt.Errorf("invalid jwt_version: %w", err)
	}
	
	aesKey, err := l.getKV(ctx, path.Join(prefix, "aes_key"))
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return nil, err
	}
	cfg.AESKey = aesKey
	
	jwtExpiry, err := l.getKV(ctx, path.Join(prefix, "jwt_expiry"))
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return nil, err
	}
	if jwtExpiry != "" {
		if err := json.Unmarshal([]byte(jwtExpiry), &cfg.JWTExpiry); err != nil {
			return nil, fmt.Errorf("invalid jwt_expiry: %w", err)
		}
	} else {
		cfg.JWTExpiry = 3600 // Default 1 hour
	}
	
	refreshExpiry, err := l.getKV(ctx, path.Join(prefix, "refresh_expiry"))
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return nil, err
	}
	if refreshExpiry != "" {
		if err := json.Unmarshal([]byte(refreshExpiry), &cfg.RefreshExpiry); err != nil {
			return nil, fmt.Errorf("invalid refresh_expiry: %w", err)
		}
	} else {
		cfg.RefreshExpiry = 604800 // Default 7 days
	}
	
	return cfg, nil
}

// loadAPI loads API configuration from Consul KV.
func (l *ConsulLoader) loadAPI(ctx context.Context) (*APIConfig, error) {
	prefix := path.Join(l.kvPrefix, "api")
	
	cfg := &APIConfig{}
	
	// Load QQ Music config
	qqMusic, err := l.loadAPIEndpoint(ctx, path.Join(prefix, "qq_music"))
	if err != nil {
		return nil, fmt.Errorf("qq_music: %w", err)
	}
	cfg.QQMusic = QQMusicConfig{
		BaseURL:   qqMusic["base_url"],
		APIKey:    qqMusic["api_key"],
		Enabled:   qqMusic["enabled"] == "true",
	}
	if rateLimit := qqMusic["rate_limit"]; rateLimit != "" {
		json.Unmarshal([]byte(rateLimit), &cfg.QQMusic.RateLimit)
	}
	if timeout := qqMusic["timeout"]; timeout != "" {
		json.Unmarshal([]byte(timeout), &cfg.QQMusic.Timeout)
	}
	
	// Load Joox config
	joox, err := l.loadAPIEndpoint(ctx, path.Join(prefix, "joox"))
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return nil, fmt.Errorf("joox: %w", err)
	}
	cfg.Joox = JooxConfig{
		BaseURL:   joox["base_url"],
		APIKey:    joox["api_key"],
		Enabled:   joox["enabled"] == "true",
	}
	
	// Load NetEase config
	netease, err := l.loadAPIEndpoint(ctx, path.Join(prefix, "netease"))
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return nil, fmt.Errorf("netease: %w", err)
	}
	cfg.NetEase = NetEaseConfig{
		BaseURL:   netease["base_url"],
		APIKey:    netease["api_key"],
		Enabled:   netease["enabled"] == "true",
	}
	
	// Load Kugou config
	kugou, err := l.loadAPIEndpoint(ctx, path.Join(prefix, "kugou"))
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return nil, fmt.Errorf("kugou: %w", err)
	}
	cfg.Kugou = KugouConfig{
		BaseURL:   kugou["base_url"],
		APIKey:    kugou["api_key"],
		Enabled:   kugou["enabled"] == "true",
	}
	
	return cfg, nil
}

// loadAPIEndpoint loads an API endpoint configuration.
func (l *ConsulLoader) loadAPIEndpoint(ctx context.Context, prefix string) (map[string]string, error) {
	result := make(map[string]string)
	
	baseURL, err := l.getKV(ctx, path.Join(prefix, "base_url"))
	if err != nil {
		return nil, err
	}
	result["base_url"] = baseURL
	
	// Optional keys
	apiKey, _ := l.getKV(ctx, path.Join(prefix, "api_key"))
	result["api_key"] = apiKey
	
	enabled, _ := l.getKV(ctx, path.Join(prefix, "enabled"))
	result["enabled"] = enabled
	
	rateLimit, _ := l.getKV(ctx, path.Join(prefix, "rate_limit"))
	result["rate_limit"] = rateLimit
	
	timeout, _ := l.getKV(ctx, path.Join(prefix, "timeout"))
	result["timeout"] = timeout
	
	return result, nil
}

// loadSMS loads SMS configuration from Consul KV.
func (l *ConsulLoader) loadSMS(ctx context.Context) (*SMSConfig, error) {
	prefix := path.Join(l.kvPrefix, "sms")
	
	cfg := &SMSConfig{}
	
	// Load Aliyun SMS config
	aliyun, err := l.loadSMSProvider(ctx, path.Join(prefix, "aliyun"))
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return nil, fmt.Errorf("aliyun: %w", err)
	}
	cfg.Aliyun = AliyunSMSConfig{
		AccessKeyID:     aliyun["access_key_id"],
		AccessKeySecret: aliyun["access_key_secret"],
		SignName:        aliyun["sign_name"],
		TemplateCode:    aliyun["template_code"],
		Enabled:         aliyun["enabled"] == "true",
	}
	
	// Load Tencent SMS config
	tencent, err := l.loadSMSProvider(ctx, path.Join(prefix, "tencent"))
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return nil, fmt.Errorf("tencent: %w", err)
	}
	cfg.Tencent = TencentSMSConfig{
		SecretID:   tencent["secret_id"],
		SecretKey:  tencent["secret_key"],
		AppID:      tencent["app_id"],
		SignName:   tencent["sign_name"],
		TemplateID: tencent["template_id"],
		Enabled:    tencent["enabled"] == "true",
	}
	
	// Load Twilio SMS config
	twilio, err := l.loadSMSProvider(ctx, path.Join(prefix, "twilio"))
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return nil, fmt.Errorf("twilio: %w", err)
	}
	cfg.Twilio = TwilioSMSConfig{
		AccountSID: twilio["account_sid"],
		AuthToken:  twilio["auth_token"],
		FromNumber: twilio["from_number"],
		Enabled:    twilio["enabled"] == "true",
	}
	
	return cfg, nil
}

// loadSMSProvider loads an SMS provider configuration.
func (l *ConsulLoader) loadSMSProvider(ctx context.Context, prefix string) (map[string]string, error) {
	result := make(map[string]string)
	
	// Load all keys for this provider
	kvPairs, _, err := l.client.KV().List(prefix, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}
	
	if len(kvPairs) == 0 {
		return result, fmt.Errorf("provider not found")
	}
	
	for _, kv := range kvPairs {
		key := strings.TrimPrefix(kv.Key, prefix+"/")
		result[key] = string(kv.Value)
	}
	
	return result, nil
}

// loadFeatures loads feature flags from Consul KV.
func (l *ConsulLoader) loadFeatures(ctx context.Context) (*FeatureFlags, error) {
	prefix := path.Join(l.kvPrefix, "features")
	
	cfg := &FeatureFlags{}
	
	// Load each feature flag (all optional)
	tokenIPBinding, _ := l.getKV(ctx, path.Join(prefix, "token_ip_binding"))
	cfg.TokenIPBinding = tokenIPBinding == "true"
	
	deviceFingerprint, _ := l.getKV(ctx, path.Join(prefix, "device_fingerprint"))
	cfg.DeviceFingerprint = deviceFingerprint == "true"
	
	twoFactorAuth, _ := l.getKV(ctx, path.Join(prefix, "two_factor_auth"))
	cfg.TwoFactorAuth = twoFactorAuth == "true"
	
	rateLimitEnabled, _ := l.getKV(ctx, path.Join(prefix, "rate_limit_enabled"))
	cfg.RateLimitEnabled = rateLimitEnabled == "true"
	
	cacheWarmup, _ := l.getKV(ctx, path.Join(prefix, "cache_warmup"))
	cfg.CacheWarmup = cacheWarmup == "true"
	
	openTelemetry, _ := l.getKV(ctx, path.Join(prefix, "open_telemetry"))
	cfg.OpenTelemetry = openTelemetry == "true"
	
	return cfg, nil
}

// getKV retrieves a value from Consul KV.
func (l *ConsulLoader) getKV(ctx context.Context, key string) (string, error) {
	pair, _, err := l.client.KV().Get(key, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get key %s: %w", key, err)
	}
	
	if pair == nil {
		return "", fmt.Errorf("key %s not found", key)
	}
	
	return string(pair.Value), nil
}

// SetKV sets a value in Consul KV.
func (l *ConsulLoader) SetKV(ctx context.Context, key, value string) error {
	fullKey := path.Join(l.kvPrefix, key)
	
	pair := &api.KVPair{
		Key:   fullKey,
		Value: []byte(value),
	}
	
	_, err := l.client.KV().Put(pair, nil)
	if err != nil {
		return fmt.Errorf("failed to set key %s: %w", fullKey, err)
	}
	
	return nil
}

// DeleteKV deletes a key from Consul KV.
func (l *ConsulLoader) DeleteKV(ctx context.Context, key string) error {
	fullKey := path.Join(l.kvPrefix, key)
	
	_, err := l.client.KV().Delete(fullKey, nil)
	if err != nil {
		return fmt.Errorf("failed to delete key %s: %w", fullKey, err)
	}
	
	return nil
}

// Close closes the Consul loader and cleanup resources.
func (l *ConsulLoader) Close() {
	if l.cache != nil {
		l.cache.Close()
	}
}
