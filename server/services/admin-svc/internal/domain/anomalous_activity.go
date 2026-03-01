package domain

import "time"

// AnomalousActivity 异常活动告警实体
type AnomalousActivity struct {
	ID          string     `json:"id" db:"id"`
	Type        string     `json:"type" db:"type"`                       // 异常类型
	Severity    string     `json:"severity" db:"severity"`               // low, medium, high, critical
	Description string     `json:"description" db:"description"`         // 异常描述
	AdminID     string     `json:"admin_id" db:"admin_id"`               // 触发异常的管理员ID
	AdminName   string     `json:"admin_name" db:"admin_name"`           // 管理员姓名
	Details     string     `json:"details" db:"details"`                 // 详细信息（JSON）
	Resolved    bool       `json:"resolved" db:"resolved"`               // 是否已处理
	ResolvedBy  string     `json:"resolved_by,omitempty" db:"resolved_by"` // 处理人
	ResolvedAt  *time.Time `json:"resolved_at,omitempty" db:"resolved_at"` // 处理时间
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// AnomalousType 异常类型常量
const (
	// 批量操作异常
	AnomalousTypeBulkDisable = "bulk_disable" // 短时间内禁用多个用户
	AnomalousTypeBulkDelete  = "bulk_delete"  // 短时间内删除多个资源

	// 敏感操作异常
	AnomalousSensitiveOp = "sensitive_op" // 非工作时间敏感操作

	// 登录异常
	AnomalousTypeLoginFailure = "login_failure" // 连续登录失败
	AnomalousTypeUnusualIP    = "unusual_ip"    // 异常IP登录

	// 数据异常
	AnomalousTypeDataLeak = "data_leak" // 大量数据导出
)

// Severity 严重程度常量
const (
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)

// Resolve 标记为已处理
func (a *AnomalousActivity) Resolve(adminID string) {
	a.Resolved = true
	a.ResolvedBy = adminID
	now := time.Now()
	a.ResolvedAt = &now
}
