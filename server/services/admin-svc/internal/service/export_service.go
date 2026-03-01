package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"time"

	"admin-svc/internal/domain"

	"github.com/xuri/excelize/v2"
)

// ExportService 数据导出服务
type ExportService struct{}

func NewExportService() *ExportService {
	return &ExportService{}
}

// ExportToCSV 导出操作日志为CSV
func (s *ExportService) ExportToCSV(ctx context.Context, logs []domain.OperationLog, output string) error {
	file, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头
	headers := []string{
		"ID", "管理员ID", "管理员姓名", "操作", "资源", "动作",
		"状态", "IP地址", "User Agent", "Request ID",
		"耗时(ms)", "创建时间",
	}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	// 写入数据
	for _, log := range logs {
		row := []string{
			log.ID,
			log.AdminID,
			log.AdminName,
			log.Operation,
			log.Resource,
			log.Action,
			log.Status,
			log.IP,
			log.UserAgent,
			log.RequestID,
			fmt.Sprintf("%d", log.Duration),
			log.CreatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("write row: %w", err)
		}
	}

	return nil
}

// ExportToExcel 导出操作日志为Excel
func (s *ExportService) ExportToExcel(ctx context.Context, logs []domain.OperationLog, output string) error {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "操作日志"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("new sheet: %w", err)
	}
	f.SetActiveSheet(index)

	// 设置表头
	headers := []string{
		"ID", "管理员ID", "管理员姓名", "操作", "资源", "动作",
		"状态", "IP地址", "User Agent", "Request ID",
		"耗时(ms)", "错误信息", "创建时间",
	}

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// 写入数据
	for i, log := range logs {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), log.ID)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), log.AdminID)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), log.AdminName)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), log.Operation)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), log.Resource)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), log.Action)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), log.Status)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), log.IP)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), log.UserAgent)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), log.RequestID)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), log.Duration)
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", row), log.ErrorMsg)
		f.SetCellValue(sheetName, fmt.Sprintf("M%d", row), log.CreatedAt.Format(time.RFC3339))
	}

	// 自动调整列宽
	for i := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheetName, col, col, 15)
	}

	// 保存文件
	if err := f.SaveAs(output); err != nil {
		return fmt.Errorf("save file: %w", err)
	}

	return nil
}

// ExportStatsToExcel 导出统计数据为Excel
func (s *ExportService) ExportStatsToExcel(ctx context.Context, stats []domain.DailyStats, output string) error {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "每日统计"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("new sheet: %w", err)
	}
	f.SetActiveSheet(index)

	// 设置表头
	headers := []string{
		"日期", "总用户数", "新增用户", "活跃用户",
		"总请求数", "成功请求", "失败请求", "错误率(%)",
		"平均响应时间(ms)", "总收藏数", "总歌单数", "总播放次数",
	}

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// 写入数据
	for i, stat := range stats {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), stat.Date.Format("2006-01-02"))
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), stat.TotalUsers)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), stat.NewUsers)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), stat.ActiveUsers)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), stat.TotalRequests)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), stat.SuccessRequests)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), stat.FailedRequests)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), fmt.Sprintf("%.2f", stat.ErrorRate))
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), stat.AvgResponseTime)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), stat.TotalFavorites)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), stat.TotalPlaylists)
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", row), stat.TotalPlays)
	}

	// 自动调整列宽
	for i := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheetName, col, col, 15)
	}

	// 设置表头样式
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
	})
	f.SetCellStyle(sheetName, "A1", fmt.Sprintf("%s1", string(rune('A'+len(headers)-1))), headerStyle)

	// 保存文件
	if err := f.SaveAs(output); err != nil {
		return fmt.Errorf("save file: %w", err)
	}

	return nil
}
