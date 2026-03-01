package handler

import (
	"fmt"
	"net/http"
	"time"

	"admin-svc/internal/service"

	"github.com/gin-gonic/gin"
)

// StatsHandler 统计处理器
type StatsHandler struct {
	statsSvc *service.StatsService
	exportSvc *service.ExportService
}

// NewStatsHandler 创建统计处理器
func NewStatsHandler(
	statsSvc *service.StatsService,
	exportSvc *service.ExportService,
) *StatsHandler {
	return &StatsHandler{
		statsSvc:  statsSvc,
		exportSvc: exportSvc,
	}
}

// GetRealtimeStats 获取实时统计
func (h *StatsHandler) GetRealtimeStats(c *gin.Context) {
	stats, err := h.statsSvc.GetRealtimeStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get stats: %v", err)})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetDailyStats 获取每日统计
func (h *StatsHandler) GetDailyStats(c *gin.Context) {
	var query struct {
		StartDate string `form:"start_date" binding:"required"`
		EndDate   string `form:"end_date" binding:"required"`
	}

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query"})
		return
	}

	// 解析日期
	startDate, err := time.Parse("2006-01-02", query.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date format"})
		return
	}

	endDate, err := time.Parse("2006-01-02", query.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date format"})
		return
	}

	// TODO: 从数据库查询每日统计 (startDate to endDate)
	_, _ = startDate, endDate

	c.JSON(http.StatusOK, gin.H{
		"stats": []interface{}{},
		"count": 0,
	})
}

// ExportDailyStats 导出每日统计
func (h *StatsHandler) ExportDailyStats(c *gin.Context) {
	format := c.Query("format") // excel or csv
	if format != "excel" && format != "csv" {
		format = "excel"
	}

	var query struct {
		StartDate string `form:"start_date" binding:"required"`
		EndDate   string `form:"end_date" binding:"required"`
	}

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query"})
		return
	}

	// TODO: 从数据库查询统计数据
	stats := []*struct{}{} // 示例

	// 导出
	if format == "excel" {
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="stats_%s_%s.xlsx"`, query.StartDate, query.EndDate))
		// err := h.exportSvc.ExportStatsToExcel(c.Request.Context(), stats, c.Writer)
		// TODO: 实现导出
		_ = stats
	} else {
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="stats_%s_%s.csv"`, query.StartDate, query.EndDate))
		// TODO: 实现CSV导出
	}
}
