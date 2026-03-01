package handler

import (
	"fmt"
	"net/http"
	"time"

	"admin-svc/internal/service"

	"github.com/gin-gonic/gin"
)

// ConfigHandler 配置管理处理器
type ConfigHandler struct {
	configSvc *service.ConfigService
	auditSvc  *service.AuditService
}

// NewConfigHandler 创建配置处理器
func NewConfigHandler(
	configSvc *service.ConfigService,
	auditSvc *service.AuditService,
) *ConfigHandler {
	return &ConfigHandler{
		configSvc: configSvc,
		auditSvc:  auditSvc,
	}
}

// GetConfig 获取配置
func (h *ConfigHandler) GetConfig(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "config key is required"})
		return
	}

	config, err := h.configSvc.GetWithVersion(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("config not found: %v", err)})
		return
	}

	c.JSON(http.StatusOK, config)
}

// ListConfigs 列出配置
func (h *ConfigHandler) ListConfigs(c *gin.Context) {
	prefix := c.Query("prefix")

	configs, err := h.configSvc.List(c.Request.Context(), prefix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to list configs: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"configs": configs,
		"count":   len(configs),
	})
}

// UpdateConfig 更新配置
func (h *ConfigHandler) UpdateConfig(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "config key is required"})
		return
	}

	var req struct {
		Value  string `json:"value" binding:"required"`
		Reason string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// 获取旧值
	oldConfig, _ := h.configSvc.GetWithVersion(c.Request.Context(), key)
	oldValue := ""
	if oldConfig != nil {
		oldValue = oldConfig.Value
	}

	// 更新配置
	if err := h.configSvc.Set(c.Request.Context(), key, req.Value); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to update config: %v", err)})
		return
	}

	// TODO: 保存历史记录到数据库
	adminID := c.GetString("admin_id")
	adminName := c.GetString("admin_name")
	
	history := h.configSvc.CreateHistory(
		key, oldValue, req.Value,
		adminID, adminName, req.Reason,
	)
	_ = history // TODO: 保存到数据库

	c.JSON(http.StatusOK, gin.H{
		"message": "config updated successfully",
		"key":     key,
		"value":   req.Value,
	})
}

// DeleteConfig 删除配置
func (h *ConfigHandler) DeleteConfig(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "config key is required"})
		return
	}

	if err := h.configSvc.Delete(c.Request.Context(), key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to delete config: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "config deleted successfully",
	})
}

// GetConfigHistory 获取配置历史
func (h *ConfigHandler) GetConfigHistory(c *gin.Context) {
	key := c.Param("key")

	// TODO: 从数据库查询历史记录 (key)
	_ = key

	c.JSON(http.StatusOK, gin.H{
		"histories": []interface{}{},
		"count":     0,
	})
}

// ClearConfigCache 清除配置缓存
func (h *ConfigHandler) ClearConfigCache(c *gin.Context) {
	if err := h.configSvc.ClearCache(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to clear cache: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "cache cleared successfully",
		"cleared_at": time.Now(),
	})
}
