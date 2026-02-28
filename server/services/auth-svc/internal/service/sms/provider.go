package sms

import (
	"context"
)

// Provider SMS提供商接口
// 所有SMS服务商必须实现此接口
type Provider interface {
	// Send 发送短信
	// phone: 手机号（国际格式，如+8613800138000）
	// code: 验证码
	// 返回: error（如果发送失败）
	Send(ctx context.Context, phone, code string) error

	// Name 返回提供商名称
	// 返回: aliyun/tencent/twilio
	Name() string

	// IsAvailable 检查提供商是否可用
	// 返回: true表示可用，false表示不可用（如配置未设置）
	IsAvailable() bool
}

// Config SMS配置（从Consul KV加载）
type Config struct {
	// Aliyun 阿里云配置
	Aliyun AliyunConfig `json:"aliyun"`
	// Tencent 腾讯云配置
	Tencent TencentConfig `json:"tencent"`
	// Twilio Twilio配置
	Twilio TwilioConfig `json:"twilio"`
	// FallbackEnabled 是否启用Fallback（默认true）
	FallbackEnabled bool `json:"fallback_enabled"`
}

// AliyunConfig 阿里云SMS配置
type AliyunConfig struct {
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	SignName        string `json:"sign_name"`
	TemplateCode    string `json:"template_code"`
	Endpoint        string `json:"endpoint"`
	Enabled         bool   `json:"enabled"`
}

// TencentConfig 腾讯云SMS配置
type TencentConfig struct {
	SecretID   string `json:"secret_id"`
	SecretKey  string `json:"secret_key"`
	AppID      string `json:"app_id"`
	SignName   string `json:"sign_name"`
	TemplateID string `json:"template_id"`
	Region     string `json:"region"`
	Enabled    bool   `json:"enabled"`
}

// TwilioConfig Twilio SMS配置
type TwilioConfig struct {
	AccountSID string `json:"account_sid"`
	AuthToken  string `json:"auth_token"`
	FromNumber string `json:"from_number"`
	Enabled    bool   `json:"enabled"`
}

// NewConfig 创建默认配置
func NewConfig() *Config {
	return &Config{
		FallbackEnabled: true,
		Aliyun: AliyunConfig{
			Endpoint: "dysmsapi.aliyuncs.com",
			Enabled:  false,
		},
		Tencent: TencentConfig{
			Region:  "ap-guangzhou",
			Enabled: false,
		},
		Twilio: TwilioConfig{
			Enabled: false,
		},
	}
}
