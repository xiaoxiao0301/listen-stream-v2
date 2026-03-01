package handler

import (
	"bytes"
	"net/http"
	"time"

	"admin-svc/internal/domain"
	"admin-svc/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminHandler 管理员处理器
type AdminHandler struct {
	totpSvc   *service.TOTPService
	auditSvc  *service.AuditService
	// 实际项目中需要Repository
}

// NewAdminHandler 创建管理员处理器
func NewAdminHandler(
	totpSvc *service.TOTPService,
	auditSvc *service.AuditService,
) *AdminHandler {
	return &AdminHandler{
		totpSvc:  totpSvc,
		auditSvc: auditSvc,
	}
}

// Login 管理员登录
func (h *AdminHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		TOTPCode string `json:"totp_code"` // 可选，启用2FA时必须
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// TODO: 从数据库查询用户，验证密码
	// TODO: 如果启用2FA，验证TOTP码
	// TODO: 生成JWT Token
	// TODO: 记录登录日志

	// 记录操作日志
	log := &domain.OperationLog{
		ID:        uuid.New().String(),
		AdminID:   "admin-id",
		AdminName: req.Username,
		Operation: domain.OpLogin,
		Resource:  domain.ResourceAdminUser,
		Action:    domain.ActionView,
		RequestID: c.GetString("request_id"),
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Status:    domain.StatusSuccess,
		Duration:  100,
		CreatedAt: time.Now(),
	}
	
	_ = h.auditSvc.LogOperation(c.Request.Context(), log)

	c.JSON(http.StatusOK, gin.H{
		"message": "login successful",
		"token":   "jwt-token",
	})
}

// Enable2FA 启用双因素认证
func (h *AdminHandler) Enable2FA(c *gin.Context) {
	adminID := c.GetString("admin_id")
	if adminID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 生成TOTP密钥
	secret, err := h.totpSvc.GenerateSecret(adminID)
	_ = adminID // TODO: save to database
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate secret"})
		return
	}

	// TODO: 保存密钥到数据库（但标记为未启用）

	// 生成二维码
	var buf bytes.Buffer
	if err := h.totpSvc.GenerateQRCode(adminID, secret, &buf); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate qr code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"secret":           secret,
		"provisioning_uri": h.totpSvc.GetProvisioningURI(adminID, secret),
		"qr_code":          buf.Bytes(),
	})
}

// Verify2FA 验证并启用双因素认证
func (h *AdminHandler) Verify2FA(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required,len=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid code"})
		return
	}

	adminID := c.GetString("admin_id")
	_ = adminID // TODO: 用于数据库操作

	// TODO: 从数据库获取密钥
	secret := "user-secret"

	// 验证TOTP码
	if !h.totpSvc.Verify(req.Code, secret) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid totp code"})
		return
	}

	// TODO: 更新数据库，标记2FA为启用状态

	c.JSON(http.StatusOK, gin.H{
		"message": "2fa enabled successfully",
	})
}

// Disable2FA 禁用双因素认证
func (h *AdminHandler) Disable2FA(c *gin.Context) {
	var req struct {
		Password string `json:"password" binding:"required"`
		Code     string `json:"code" binding:"required,len=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	adminID := c.GetString("admin_id")
	_ = adminID // TODO: use for database operation

	// TODO: 验证密码
	// TODO: 验证TOTP码
	// TODO: 更新数据库，禁用2FA

	c.JSON(http.StatusOK, gin.H{
		"message": "2fa disabled successfully",
	})
}

// ListAdmins 列出管理员
func (h *AdminHandler) ListAdmins(c *gin.Context) {
	type query struct {
		Page   int    `form:"page" binding:"min=1"`
		Size   int    `form:"size" binding:"min=1,max=100"`
		Status string `form:"status"`
	}

	var q query
	q.Page = 1
	q.Size = 20

	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query"})
		return
	}

	// TODO: 从数据库查询管理员列表

	c.JSON(http.StatusOK, gin.H{
		"admins": []interface{}{},
		"total":  0,
		"page":   q.Page,
		"size":   q.Size,
	})
}
