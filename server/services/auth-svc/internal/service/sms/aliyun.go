package sms

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/domain"
)

// AliyunProvider 阿里云SMS提供商
type AliyunProvider struct {
	config AliyunConfig
	client *http.Client
}

// NewAliyunProvider 创建阿里云SMS提供商
func NewAliyunProvider(config AliyunConfig) *AliyunProvider {
	return &AliyunProvider{
		config: config,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name 返回提供商名称
func (p *AliyunProvider) Name() string {
	return domain.ProviderAliyun
}

// IsAvailable 检查提供商是否可用
func (p *AliyunProvider) IsAvailable() bool {
	return p.config.Enabled &&
		p.config.AccessKeyID != "" &&
		p.config.AccessKeySecret != "" &&
		p.config.SignName != "" &&
		p.config.TemplateCode != ""
}

// Send 发送短信
func (p *AliyunProvider) Send(ctx context.Context, phone, code string) error {
	if !p.IsAvailable() {
		return fmt.Errorf("aliyun sms provider is not available")
	}

	// 构建请求参数
	params := p.buildParams(phone, code)

	// 签名
	signature := p.sign(params)
	params["Signature"] = signature

	// 构建URL
	endpoint := p.config.Endpoint
	if endpoint == "" {
		endpoint = "dysmsapi.aliyuncs.com"
	}
	apiURL := fmt.Sprintf("https://%s/?%s", endpoint, p.encodeParams(params))

	// 发送请求
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

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
		Code      string `json:"Code"`
		Message   string `json:"Message"`
		RequestID string `json:"RequestId"`
		BizID     string `json:"BizId"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	// 检查结果
	if result.Code != "OK" {
		return fmt.Errorf("aliyun sms error: %s - %s", result.Code, result.Message)
	}

	return nil
}

// buildParams 构建请求参数
func (p *AliyunProvider) buildParams(phone, code string) map[string]string {
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	nonce := uuid.New().String()

	// 模板参数
	templateParam := fmt.Sprintf(`{"code":"%s"}`, code)

	params := map[string]string{
		"AccessKeyId":      p.config.AccessKeyID,
		"Action":           "SendSms",
		"Format":           "JSON",
		"PhoneNumbers":     phone,
		"SignName":         p.config.SignName,
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureNonce":   nonce,
		"SignatureVersion": "1.0",
		"TemplateCode":     p.config.TemplateCode,
		"TemplateParam":    templateParam,
		"Timestamp":        timestamp,
		"Version":          "2017-05-25",
	}

	return params
}

// sign 签名
func (p *AliyunProvider) sign(params map[string]string) string {
	// 1. 按参数名排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 2. 构建查询字符串
	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, p.percentEncode(k)+"="+p.percentEncode(params[k]))
	}
	canonicalizedQueryString := strings.Join(pairs, "&")

	// 3. 构建待签名字符串
	stringToSign := "GET&" + p.percentEncode("/") + "&" + p.percentEncode(canonicalizedQueryString)

	// 4. 计算签名
	h := hmac.New(sha1.New, []byte(p.config.AccessKeySecret+"&"))
	h.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return signature
}

// percentEncode URL编码
func (p *AliyunProvider) percentEncode(s string) string {
	encoded := url.QueryEscape(s)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}

// encodeParams 编码参数
func (p *AliyunProvider) encodeParams(params map[string]string) string {
	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}
	return values.Encode()
}
