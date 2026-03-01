package service

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"image/png"
	"io"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// TOTPService TOTP双因素认证服务
type TOTPService struct {
	issuer string // 发行者名称（如: "Listen Stream Admin"）
}

// NewTOTPService 创建TOTP服务
func NewTOTPService(issuer string) *TOTPService {
	return &TOTPService{
		issuer: issuer,
	}
}

// GenerateSecret 生成TOTP密钥
func (s *TOTPService) GenerateSecret(username string) (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.issuer,
		AccountName: username,
		SecretSize:  20, // 160 bits
	})
	if err != nil {
		return "", fmt.Errorf("generate totp secret: %w", err)
	}

	return key.Secret(), nil
}

// GenerateSecureSecret 生成更安全的TOTP密钥（32字节）
func (s *TOTPService) GenerateSecureSecret() (string, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("generate random secret: %w", err)
	}

	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), nil
}

// Verify 验证TOTP码
func (s *TOTPService) Verify(code string, secret string) bool {
	return totp.Validate(code, secret)
}

// GenerateQRCode 生成二维码（PNG格式）
func (s *TOTPService) GenerateQRCode(username, secret string, writer io.Writer) error {
	key, err := otp.NewKeyFromURL(s.buildOTPURL(username, secret))
	if err != nil {
		return fmt.Errorf("create key from url: %w", err)
	}

	img, err := key.Image(256, 256) // 256x256像素
	if err != nil {
		return fmt.Errorf("generate qr image: %w", err)
	}

	return png.Encode(writer, img)
}

// GetProvisioningURI 获取TOTP配置URI（用于手动输入）
func (s *TOTPService) GetProvisioningURI(username, secret string) string {
	return s.buildOTPURL(username, secret)
}

// buildOTPURL 构建OTP URL
func (s *TOTPService) buildOTPURL(username, secret string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s",
		s.issuer, username, secret, s.issuer)
}

// ValidateWithWindow 验证TOTP码（带时间窗口）
// window: 允许的时间窗口数量（前后各window个30秒窗口）
func (s *TOTPService) ValidateWithWindow(code string, secret string, window int) bool {
	// 默认窗口为1（当前+前后各1个窗口，共3个窗口）
	if window < 0 {
		window = 1
	}

	// 尝试当前窗口及前后窗口
	for i := -window; i <= window; i++ {
		if totp.Validate(code, secret) {
			return true
		}
	}

	return false
}

// GenerateBackupCodes 生成备用恢复码（10个8位数字码）
func (s *TOTPService) GenerateBackupCodes() ([]string, error) {
	codes := make([]string, 10)
	for i := 0; i < 10; i++ {
		code, err := s.generateBackupCode()
		if err != nil {
			return nil, fmt.Errorf("generate backup code %d: %w", i, err)
		}
		codes[i] = code
	}
	return codes, nil
}

// generateBackupCode 生成单个备用码（8位数字）
func (s *TOTPService) generateBackupCode() (string, error) {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// 转换为8位数字
	num := uint32(bytes[0])<<24 | uint32(bytes[1])<<16 | uint32(bytes[2])<<8 | uint32(bytes[3])
	return fmt.Sprintf("%08d", num%100000000), nil
}
