package sms

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/domain"
)

// TwilioProvider Twilio SMS提供商
type TwilioProvider struct {
	config TwilioConfig
	client *http.Client
}

// NewTwilioProvider 创建Twilio SMS提供商
func NewTwilioProvider(config TwilioConfig) *TwilioProvider {
	return &TwilioProvider{
		config: config,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name 返回提供商名称
func (p *TwilioProvider) Name() string {
	return domain.ProviderTwilio
}

// IsAvailable 检查提供商是否可用
func (p *TwilioProvider) IsAvailable() bool {
	return p.config.Enabled &&
		p.config.AccountSID != "" &&
		p.config.AuthToken != "" &&
		p.config.FromNumber != ""
}

// Send 发送短信
func (p *TwilioProvider) Send(ctx context.Context, phone, code string) error {
	if !p.IsAvailable() {
		return fmt.Errorf("twilio sms provider is not available")
	}

	// 构建API URL
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json",
		p.config.AccountSID)

	// 构建请求体
	message := fmt.Sprintf("Your verification code is: %s", code)
	data := url.Values{}
	data.Set("To", phone)
	data.Set("From", p.config.FromNumber)
	data.Set("Body", message)

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// 设置Headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", p.buildAuthHeader())

	// 发送请求
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	// 检查响应状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResult struct {
			Code     int    `json:"code"`
			Message  string `json:"message"`
			MoreInfo string `json:"more_info"`
		}
		if err := json.Unmarshal(body, &errResult); err == nil {
			return fmt.Errorf("twilio sms error [%d]: %s", errResult.Code, errResult.Message)
		}
		return fmt.Errorf("twilio sms error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	// 解析响应
	var result struct {
		SID         string `json:"sid"`
		Status      string `json:"status"`
		ErrorCode   *int   `json:"error_code"`
		ErrorMessage string `json:"error_message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	// 检查错误
	if result.ErrorCode != nil && *result.ErrorCode != 0 {
		return fmt.Errorf("twilio sms send failed [%d]: %s", *result.ErrorCode, result.ErrorMessage)
	}

	return nil
}

// buildAuthHeader 构建Basic Auth Header
func (p *TwilioProvider) buildAuthHeader() string {
	auth := fmt.Sprintf("%s:%s", p.config.AccountSID, p.config.AuthToken)
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))
	return fmt.Sprintf("Basic %s", encoded)
}
