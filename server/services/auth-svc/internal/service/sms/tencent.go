package sms

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/domain"
)

// TencentProvider 腾讯云SMS提供商
type TencentProvider struct {
	config TencentConfig
	client *http.Client
}

// NewTencentProvider 创建腾讯云SMS提供商
func NewTencentProvider(config TencentConfig) *TencentProvider {
	return &TencentProvider{
		config: config,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name 返回提供商名称
func (p *TencentProvider) Name() string {
	return domain.ProviderTencent
}

// IsAvailable 检查提供商是否可用
func (p *TencentProvider) IsAvailable() bool {
	return p.config.Enabled &&
		p.config.SecretID != "" &&
		p.config.SecretKey != "" &&
		p.config.AppID != "" &&
		p.config.SignName != "" &&
		p.config.TemplateID != ""
}

// Send 发送短信
func (p *TencentProvider) Send(ctx context.Context, phone, code string) error {
	if !p.IsAvailable() {
		return fmt.Errorf("tencent sms provider is not available")
	}

	// 构建请求体
	reqBody := p.buildRequestBody(phone, code)
	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}

	// 构建URL
	region := p.config.Region
	if region == "" {
		region = "ap-guangzhou"
	}
	apiURL := "https://sms.tencentcloudapi.com"

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(bodyJSON))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// 设置Headers
	timestamp := time.Now().Unix()
	headers := p.buildHeaders(bodyJSON, timestamp, region)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

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

	// 解析响应
	var result struct {
		Response struct {
			SendStatusSet []struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"SendStatusSet"`
			RequestID string `json:"RequestId"`
			Error     struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	// 检查错误
	if result.Response.Error.Code != "" {
		return fmt.Errorf("tencent sms error: %s - %s",
			result.Response.Error.Code, result.Response.Error.Message)
	}

	// 检查发送状态
	if len(result.Response.SendStatusSet) > 0 {
		status := result.Response.SendStatusSet[0]
		if status.Code != "Ok" {
			return fmt.Errorf("tencent sms send failed: %s - %s", status.Code, status.Message)
		}
	}

	return nil
}

// buildRequestBody 构建请求体
func (p *TencentProvider) buildRequestBody(phone, code string) map[string]interface{} {
	// 去掉+号（腾讯云不需要+号前缀）
	phone = strings.TrimPrefix(phone, "+")

	return map[string]interface{}{
		"SmsSdkAppId": p.config.AppID,
		"SignName":    p.config.SignName,
		"TemplateId":  p.config.TemplateID,
		"TemplateParamSet": []string{code}, // 模板参数数组
		"PhoneNumberSet":   []string{phone},
	}
}

// buildHeaders 构建请求头
func (p *TencentProvider) buildHeaders(payload []byte, timestamp int64, region string) map[string]string {
	// 1. 拼接规范请求串
	canonicalRequest := p.buildCanonicalRequest(payload)

	// 2. 拼接待签名字符串
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	credentialScope := fmt.Sprintf("%s/sms/tc3_request", date)
	hashedCanonicalRequest := p.sha256Hex(canonicalRequest)
	stringToSign := fmt.Sprintf("TC3-HMAC-SHA256\n%d\n%s\n%s",
		timestamp, credentialScope, hashedCanonicalRequest)

	// 3. 计算签名
	secretDate := p.hmacSha256([]byte("TC3"+p.config.SecretKey), date)
	secretService := p.hmacSha256(secretDate, "sms")
	secretSigning := p.hmacSha256(secretService, "tc3_request")
	signature := hex.EncodeToString(p.hmacSha256(secretSigning, stringToSign))

	// 4. 拼接 Authorization
	authorization := fmt.Sprintf("TC3-HMAC-SHA256 Credential=%s/%s, SignedHeaders=content-type;host, Signature=%s",
		p.config.SecretID, credentialScope, signature)

	// 5. 返回Headers
	return map[string]string{
		"Authorization":  authorization,
		"Content-Type":   "application/json",
		"Host":           "sms.tencentcloudapi.com",
		"X-TC-Action":    "SendSms",
		"X-TC-Timestamp": fmt.Sprintf("%d", timestamp),
		"X-TC-Version":   "2021-01-11",
		"X-TC-Region":    region,
	}
}

// buildCanonicalRequest 构建规范请求串
func (p *TencentProvider) buildCanonicalRequest(payload []byte) string {
	httpRequestMethod := "POST"
	canonicalURI := "/"
	canonicalQueryString := ""
	canonicalHeaders := "content-type:application/json\nhost:sms.tencentcloudapi.com\n"
	signedHeaders := "content-type;host"
	hashedPayload := p.sha256Hex(string(payload))

	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		httpRequestMethod,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		hashedPayload,
	)
}

// sha256Hex SHA256哈希
func (p *TencentProvider) sha256Hex(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// hmacSha256 HMAC-SHA256
func (p *TencentProvider) hmacSha256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

// sortKeys 排序map的keys
func sortKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
