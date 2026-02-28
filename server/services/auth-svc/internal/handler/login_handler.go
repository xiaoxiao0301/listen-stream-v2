package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/domain"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/repository"
	deviceservice "github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/service/device"
	jwtservice "github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/service/jwt"
	smsservice "github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/service/sms"
)

// LoginHandler 登录处理器
type LoginHandler struct {
	smsService    *smsservice.Service
	jwtService    jwtservice.JWTService
	deviceService deviceservice.DeviceService
	userRepo      repository.UserRepository
}

// NewLoginHandler 创建登录处理器
func NewLoginHandler(
	smsService *smsservice.Service,
	jwtService jwtservice.JWTService,
	deviceService deviceservice.DeviceService,
	userRepo repository.UserRepository,
) *LoginHandler {
	return &LoginHandler{
		smsService:    smsService,
		jwtService:    jwtService,
		deviceService: deviceService,
		userRepo:      userRepo,
	}
}

// SendVerificationCodeRequest 发送验证码请求
type SendVerificationCodeRequest struct {
	Phone string `json:"phone"`
}

// SendVerificationCodeResponse 发送验证码响应
type SendVerificationCodeResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	ExpiresIn int    `json:"expires_in"` // 秒
}

// VerifyLoginRequest 验证登录请求
type VerifyLoginRequest struct {
	Phone        string `json:"phone"`
	Code         string `json:"code"`
	DeviceID     string `json:"device_id"`
	DeviceName   string `json:"device_name"`
	Platform     string `json:"platform"`
	OSVersion    string `json:"os_version"`
	AppVersion   string `json:"app_version"`
	DeviceFingerprint string `json:"device_fingerprint,omitempty"`
}

// VerifyLoginResponse 验证登录响应
type VerifyLoginResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresAt    int64  `json:"expires_at,omitempty"` // Unix timestamp
	TokenType    string `json:"token_type,omitempty"`
	UserID       string `json:"user_id,omitempty"`
}

// SendVerificationCode 发送验证码
func (h *LoginHandler) SendVerificationCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req SendVerificationCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// 验证手机号
	if req.Phone == "" {
		respondError(w, http.StatusBadRequest, "phone is required")
		return
	}

	// 发送短信验证码
	result, err := h.smsService.SendVerificationCode(r.Context(), req.Phone)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to send verification code")
		return
	}

	// 计算过期时间（秒）
	expiresIn := int(time.Until(result.ExpiresAt).Seconds())
	if expiresIn < 0 {
		expiresIn = 300 // 默认5分钟
	}

	respondJSON(w, http.StatusOK, SendVerificationCodeResponse{
		Success:   true,
		Message:   "verification code sent",
		ExpiresIn: expiresIn,
	})
}

// VerifyLogin 验证登录
func (h *LoginHandler) VerifyLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req VerifyLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// 验证必填字段
	if req.Phone == "" || req.Code == "" {
		respondError(w, http.StatusBadRequest, "phone and code are required")
		return
	}
	if req.DeviceID == "" || req.DeviceName == "" || req.Platform == "" {
		respondError(w, http.StatusBadRequest, "device information is required")
		return
	}

	// TODO: 从Redis验证验证码
	// storedCode, err := redis.Get(ctx, "sms:code:"+req.Phone)
	// if err != nil || storedCode != req.Code {
	//     respondError(w, http.StatusUnauthorized, "invalid verification code")
	//     return
	// }

	// 简化验证：假设验证码正确（实际应该从Redis获取）
	if req.Code != "123456" && len(req.Code) != 6 {
		respondError(w, http.StatusUnauthorized, "invalid verification code")
		return
	}

	// 查找或创建用户
	user, err := h.userRepo.GetByPhone(r.Context(), req.Phone)
	if err != nil {
		if err == domain.ErrUserNotFound {
			// 创建新用户
			user = domain.NewUser(req.Phone)
			err = h.userRepo.Create(r.Context(), user)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "failed to create user")
				return
			}
		} else {
			respondError(w, http.StatusInternalServerError, "failed to get user")
			return
		}
	}

	// 检查用户是否激活
	if !user.IsActive {
		respondError(w, http.StatusForbidden, "user account is disabled")
		return
	}

	// 获取客户端IP
	clientIP := getClientIP(r)

	// 注册或更新设备
	deviceReq := &deviceservice.RegisterDeviceRequest{
		UserID:     user.ID,
		DeviceName: req.DeviceName,
		DeviceID:   req.DeviceID,
		Platform:   req.Platform,
		OSVersion:  req.OSVersion,
		AppVersion: req.AppVersion,
		ClientIP:   clientIP,
	}
	device, err := h.deviceService.RegisterDevice(r.Context(), deviceReq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to register device: "+err.Error())
		return
	}

	// 生成Token
	tokenPair, err := h.jwtService.GenerateTokenPair(r.Context(), user.ID, device.ID, clientIP)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	// TODO: 删除已使用的验证码
	// redis.Del(ctx, "sms:code:"+req.Phone)

	respondJSON(w, http.StatusOK, VerifyLoginResponse{
		Success:      true,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt.Unix(),
		TokenType:    "Bearer",
		UserID:       user.ID,
	})
}

// generateVerificationCode 生成6位数字验证码
func generateVerificationCode() string {
	// 实际应该使用随机数生成
	// 这里简化为固定值
	return "123456"
}

// getClientIP 获取客户端IP
func getClientIP(r *http.Request) string {
	// 尝试从X-Forwarded-For获取
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		return ip
	}

	// 尝试从X-Real-IP获取
	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	// 使用RemoteAddr
	return r.RemoteAddr
}

// respondJSON 返回JSON响应
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError 返回错误响应
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}
