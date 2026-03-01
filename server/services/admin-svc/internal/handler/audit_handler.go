package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"admin-svc/internal/domain"
	"admin-svc/internal/service"

	"github.com/gin-gonic/gin"
)

// AuditHandler 审计日志处理器
type AuditHandler struct {
	auditSvc  *service.AuditService
	exportSvc *service.ExportService
}

func NewAuditHandler(auditSvc *service.AuditService, exportSvc *service.ExportService) *AuditHandler {
	return &AuditHandler{
		auditSvc:  auditSvc,
		exportSvc: exportSvc,
	}
}

// ListOperationLogs 列出操作日志
// GET /api/v1/audit/logs
func (h *AuditHandler) ListOperationLogs(c *gin.Context) {
	var req struct {
		Page      int    `form:"page" binding:"min=1"`
		Size      int    `form:"size" binding:"min=1,max=100"`
		AdminID   string `form:"admin_id"`
		Operation string `form:"operation"`
		Resource  string `form:"resource"`
		Status    string `form:"status"`
		StartDate string `form:"start_date"`
		EndDate   string `form:"end_date"`
	}

	// 设置默认值
	req.Page = 1
	req.Size = 20

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: 实际应该从数据库查询
	// 这里返回模拟数据
	logs := []domain.OperationLog{
		{
			ID:        "log-1",
			AdminID:   "admin-1",
			AdminName: "管理员",
			Operation: "login",
			Resource:  "admin_user",
			Action:    "login",
			Status:    "success",
			IP:        "192.168.1.100",
			UserAgent: "Mozilla/5.0",
			RequestID: "req-1",
			Duration:  50,
			CreatedAt: time.Now().Add(-1 * time.Hour),
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"data": logs,
		"pagination": gin.H{
			"page":  req.Page,
			"size":  req.Size,
			"total": 1,
		},
	})
}

// ExportOperationLogs 导出操作日志
// GET /api/v1/audit/logs/export
func (h *AuditHandler) ExportOperationLogs(c *gin.Context) {
	format := c.DefaultQuery("format", "excel")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// TODO: 根据日期范围查询数据库
	logs := []domain.OperationLog{
		{
			ID:        "log-1",
			AdminID:   "admin-1",
			AdminName: "管理员",
			Operation: "login",
			Resource:  "admin_user",
			Action:    "login",
			Status:    "success",
			IP:        "192.168.1.100",
			UserAgent: "Mozilla/5.0",
			RequestID: "req-1",
			Duration:  50,
			CreatedAt: time.Now(),
		},
	}

	filename := fmt.Sprintf("operation_logs_%s_%s.%s", startDate, endDate, format)
	filepath := fmt.Sprintf("/tmp/%s", filename)

	// 导出
	var err error
	if format == "csv" {
		err = h.exportSvc.ExportToCSV(c.Request.Context(), logs, filepath)
	} else {
		err = h.exportSvc.ExportToExcel(c.Request.Context(), logs, filepath)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("export failed: %v", err)})
		return
	}

	c.FileAttachment(filepath, filename)
}

// ListAnomalousActivities 列出异常活动
// GET /api/v1/audit/anomalies
func (h *AuditHandler) ListAnomalousActivities(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	severity := c.Query("severity")
	resolved := c.Query("resolved")

	// TODO: 实际应该从数据库查询
	activities := []domain.AnomalousActivity{
		{
			ID:          "anomaly-1",
			Type:        domain.AnomalousTypeBulkDisable,
			Severity:    domain.SeverityHigh,
			Description: "管理员在1小时内禁用了25个用户",
			AdminID:     "admin-1",
			AdminName:   "管理员",
			Resolved:    false,
			CreatedAt:   time.Now().Add(-2 * time.Hour),
		},
	}

	// 过滤
	var filtered []domain.AnomalousActivity
	for _, a := range activities {
		if severity != "" && a.Severity != severity {
			continue
		}
		if resolved == "true" && !a.Resolved {
			continue
		}
		if resolved == "false" && a.Resolved {
			continue
		}
		filtered = append(filtered, a)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": filtered,
		"pagination": gin.H{
			"page":  page,
			"size":  size,
			"total": len(filtered),
		},
	})
}

// ResolveAnomalousActivity 标记异常为已处理
// POST /api/v1/audit/anomalies/:id/resolve
func (h *AuditHandler) ResolveAnomalousActivity(c *gin.Context) {
	id := c.Param("id")

	// 获取当前管理员ID（从JWT中解析）
	// TODO: 实际应该从JWT中获取
	adminID := "admin-1"

	// TODO: 实际应该更新数据库
	activity := &domain.AnomalousActivity{
		ID: id,
	}
	activity.Resolve(adminID)

	c.JSON(http.StatusOK, gin.H{
		"message": "异常活动已标记为已处理",
		"data":    activity,
	})
}
