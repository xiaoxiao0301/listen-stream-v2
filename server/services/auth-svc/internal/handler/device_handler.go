package handler

import (
	"encoding/json"
	"net/http"
	"time"

	deviceservice "github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/service/device"
	jwtservice "github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/service/jwt"
)

// DeviceHandler 设备管理处理器
type DeviceHandler struct {
	deviceService deviceservice.DeviceService
	jwtService    jwtservice.JWTService
}

// NewDeviceHandler 创建设备处理器
func NewDeviceHandler(
	deviceService deviceservice.DeviceService,
	jwtService jwtservice.JWTService,
) *DeviceHandler {
	return &DeviceHandler{
		deviceService: deviceService,
		jwtService:    jwtService,
	}
}

// DeviceInfo 设备信息
type DeviceInfo struct {
	ID          string    `json:"id"`
	DeviceName  string    `json:"device_name"`
	Platform    string    `json:"platform"`
	AppVersion  string    `json:"app_version"`
	LastIP      string    `json:"last_ip"`
	LastLoginAt time.Time `json:"last_login_at"`
	CreatedAt   time.Time `json:"created_at"`
	IsCurrent   bool      `json:"is_current"` // 是否当前设备
}

// ListDevicesResponse 设备列表响应
type ListDevicesResponse struct {
	Success bool         `json:"success"`
	Devices []DeviceInfo `json:"devices"`
	Total   int          `json:"total"`
}

// RemoveDeviceRequest 删除设备请求
type RemoveDeviceRequest struct {
	DeviceID string `json:"device_id"`
}

// RemoveDeviceResponse 删除设备响应
type RemoveDeviceResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// ListDevices 获取设备列表
func (h *DeviceHandler) ListDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// 从请求头获取Token
	token := extractToken(r)
	if token == "" {
		respondError(w, http.StatusUnauthorized, "missing authorization token")
		return
	}

	// 验证Token
	claims, err := h.jwtService.ValidateAccessToken(r.Context(), token, "")
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	// 获取设备列表
	devices, err := h.deviceService.ListDevices(r.Context(), claims.UserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list devices")
		return
	}

	// 转换为响应格式
	deviceInfos := make([]DeviceInfo, 0, len(devices))
	for _, device := range devices {
		deviceInfos = append(deviceInfos, DeviceInfo{
			ID:          device.ID,
			DeviceName:  device.DeviceName,
			Platform:    device.Platform,
			AppVersion:  device.AppVersion,
			LastIP:      device.LastIP,
			LastLoginAt: device.LastLoginAt,
			CreatedAt:   device.CreatedAt,
			IsCurrent:   device.ID == claims.DeviceID,
		})
	}

	respondJSON(w, http.StatusOK, ListDevicesResponse{
		Success: true,
		Devices: deviceInfos,
		Total:   len(deviceInfos),
	})
}

// RemoveDevice 删除设备（踢出设备）
func (h *DeviceHandler) RemoveDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// 从请求头获取Token
	token := extractToken(r)
	if token == "" {
		respondError(w, http.StatusUnauthorized, "missing authorization token")
		return
	}

	// 验证Token
	claims, err := h.jwtService.ValidateAccessToken(r.Context(), token, "")
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	var req RemoveDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.DeviceID == "" {
		respondError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	// 不允许删除当前设备（需要用logout接口）
	if req.DeviceID == claims.DeviceID {
		respondError(w, http.StatusBadRequest, "cannot remove current device, use logout instead")
		return
	}

	// 删除设备
	err = h.deviceService.RemoveDevice(r.Context(), claims.UserID, req.DeviceID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to remove device")
		return
	}

	respondJSON(w, http.StatusOK, RemoveDeviceResponse{
		Success: true,
		Message: "device removed successfully",
	})
}

// extractToken 从请求头提取Token
func extractToken(r *http.Request) string {
	// 从Authorization头获取
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}

	// 支持 "Bearer <token>" 格式
	const prefix = "Bearer "
	if len(auth) > len(prefix) && auth[:len(prefix)] == prefix {
		return auth[len(prefix):]
	}

	return auth
}
