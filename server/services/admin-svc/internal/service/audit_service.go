package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"admin-svc/internal/domain"

	"github.com/redis/go-redis/v9"
)

// AuditService 操作审计服务
type AuditService struct {
	redis *redis.Client
}

func NewAuditService(redis *redis.Client) *AuditService {
	return &AuditService{redis: redis}
}

// LogOperation 记录操作日志
func (s *AuditService) LogOperation(ctx context.Context, log *domain.OperationLog) error {
	// TODO: 实际应该写入数据库
	// 这里简化实现，仅记录到Redis用于审计
	logJSON, err := json.Marshal(log)
	if err != nil {
		return fmt.Errorf("marshal log: %w", err)
	}

	// 存储到Redis列表（最近1000条）
	key := "audit:logs"
	if err := s.redis.LPush(ctx, key, logJSON).Err(); err != nil {
		return fmt.Errorf("lpush log: %w", err)
	}

	// 限制列表长度
	if err := s.redis.LTrim(ctx, key, 0, 999).Err(); err != nil {
		return fmt.Errorf("ltrim log: %w", err)
	}

	// 异步检查异常活动
	go s.CheckAnomalousActivity(ctx, log)

	return nil
}

// CheckAnomalousActivity 检查异常活动
func (s *AuditService) CheckAnomalousActivity(ctx context.Context, log *domain.OperationLog) (*domain.AnomalousActivity, error) {
	// 1. 检查批量禁用用户
	if log.Operation == "user_management" && log.Action == "disable" {
		// 统计1小时内该管理员禁用的用户数
		count, err := s.countRecentOperations(ctx, log.AdminID, "user_management:disable", 1*time.Hour)
		if err == nil && count >= 20 {
			return s.createAnomaly(
				domain.AnomalousTypeBulkDisable,
				domain.SeverityHigh,
				fmt.Sprintf("管理员 %s 在1小时内禁用了 %d 个用户", log.AdminName, count),
				log.AdminID,
				log.AdminName,
			), nil
		}
	}

	// 2. 检查敏感操作（非工作时间）
	if s.isSensitiveOperation(log.Operation) && !s.isWorkingHours(log.CreatedAt) {
		return s.createAnomaly(
			domain.AnomalousSensitiveOp,
			domain.SeverityMedium,
			fmt.Sprintf("管理员 %s 在非工作时间(%s)执行敏感操作: %s %s",
				log.AdminName,
				log.CreatedAt.Format("2006-01-02 15:04:05"),
				log.Operation,
				log.Action,
			),
			log.AdminID,
			log.AdminName,
		), nil
	}

	// 3. 检查连续登录失败
	if log.Operation == "login" && log.Status == "failure" {
		count, err := s.countRecentOperations(ctx, log.AdminID, "login:failure", 10*time.Minute)
		if err == nil && count >= 5 {
			return s.createAnomaly(
				domain.AnomalousTypeLoginFailure,
				domain.SeverityMedium,
				fmt.Sprintf("管理员 %s 在10分钟内连续登录失败 %d 次", log.AdminName, count),
				log.AdminID,
				log.AdminName,
			), nil
		}
	}

	// 4. 检查大量数据导出
	if log.Operation == "export" {
		count, err := s.countRecentOperations(ctx, log.AdminID, "export", 1*time.Hour)
		if err == nil && count >= 10 {
			return s.createAnomaly(
				domain.AnomalousTypeDataLeak,
				domain.SeverityCritical,
				fmt.Sprintf("管理员 %s 在1小时内导出数据 %d 次", log.AdminName, count),
				log.AdminID,
				log.AdminName,
			), nil
		}
	}

	return nil, nil
}

// countRecentOperations 统计最近时间窗口内的操作次数
func (s *AuditService) countRecentOperations(ctx context.Context, adminID, operationType string, window time.Duration) (int64, error) {
	key := fmt.Sprintf("audit:ops:%s:%s", adminID, operationType)
	now := time.Now()
	minScore := float64(now.Add(-window).Unix())

	// 使用Sorted Set，score为时间戳
	// 1. 添加当前操作
	if err := s.redis.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.Unix()),
		Member: now.Format(time.RFC3339Nano),
	}).Err(); err != nil {
		return 0, err
	}

	// 2. 清理过期数据
	if err := s.redis.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%f", minScore)).Err(); err != nil {
		return 0, err
	}

	// 3. 统计数量
	count, err := s.redis.ZCard(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	// 4. 设置TTL（防止内存泄漏）
	s.redis.Expire(ctx, key, window+1*time.Hour)

	return count, nil
}

// createAnomaly 创建异常活动记录
func (s *AuditService) createAnomaly(
	anomalyType, severity, description, adminID, adminName string,
) *domain.AnomalousActivity {
	anomaly := &domain.AnomalousActivity{
		ID:          fmt.Sprintf("anomaly-%d", time.Now().UnixNano()),
		Type:        anomalyType,
		Severity:    severity,
		Description: description,
		AdminID:     adminID,
		AdminName:   adminName,
		Resolved:    false,
		CreatedAt:   time.Now(),
	}

	// TODO: 实际应该写入数据库
	// 这里简化实现，发布到Redis频道用于告警
	ctx := context.Background()
	anomalyJSON, _ := json.Marshal(anomaly)
	s.redis.Publish(ctx, "admin:alerts", anomalyJSON)

	return anomaly
}

// isSensitiveOperation 判断是否为敏感操作
func (s *AuditService) isSensitiveOperation(operation string) bool {
	sensitiveOps := map[string]bool{
		"user_management": true,
		"config_update":   true,
		"permission":      true,
		"export":          true,
	}
	return sensitiveOps[operation]
}

// isWorkingHours 判断是否为工作时间
func (s *AuditService) isWorkingHours(t time.Time) bool {
	// 工作时间: 周一至周五 8:00-22:00
	weekday := t.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}

	hour := t.Hour()
	return hour >= 8 && hour < 22
}

// SendAlert 发送告警（可以集成到钉钉、Slack等）
func (s *AuditService) SendAlert(ctx context.Context, alert *domain.AnomalousActivity) error {
	// 发布到Redis频道
	alertJSON, err := json.Marshal(alert)
	if err != nil {
		return err
	}

	return s.redis.Publish(ctx, "admin:alerts", alertJSON).Err()
}
