package domain

import (
	"encoding/json"
	"time"
)

// OperationLog 操作日志实体（结构化）
type OperationLog struct {
	ID          string          `json:"id" db:"id"`
	AdminID     string          `json:"admin_id" db:"admin_id"`
	AdminName   string          `json:"admin_name" db:"admin_name"` // 冗余存储，便于查询
	Operation   string          `json:"operation" db:"operation"` // 操作类型
	Resource    string          `json:"resource" db:"resource"` // 操作资源
	ResourceID  string          `json:"resource_id" db:"resource_id"` // 资源ID
	Action      string          `json:"action" db:"action"` // create, update, delete, view
	Details     json.RawMessage `json:"details" db:"details"` // 结构化详情（JSON）
	RequestID   string          `json:"request_id" db:"request_id"` // 链路追踪ID
	IP          string          `json:"ip" db:"ip"`
	UserAgent   string          `json:"user_agent" db:"user_agent"`
	Status      string          `json:"status" db:"status"` // success, failed
	ErrorMsg    string          `json:"error_msg,omitempty" db:"error_msg"` // 错误信息
	Duration    int64           `json:"duration" db:"duration"` // 执行时长（毫秒）
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
}

// Operation 操作类型常量
const (
	OpLogin          = "login"
	OpLogout         = "logout"
	OpCreateUser     = "create_user"
	OpUpdateUser     = "update_user"
	OpDisableUser    = "disable_user"
	OpUpdateConfig   = "update_config"
	OpRollbackConfig = "rollback_config"
	OpExportData     = "export_data"
)

// Resource 资源类型常量
const (
	ResourceAdminUser = "admin_user"
	ResourceConfig    = "config"
	ResourceStats     = "stats"
	ResourceAuditLog  = "audit_log"
)

// Action 动作常量
const (
	ActionCreate = "create"
	ActionUpdate = "update"
	ActionDelete = "delete"
	ActionView   = "view"
)

// Status 状态常量
const (
	StatusSuccess = "success"
	StatusFailed  = "failed"
)

// OperationDetails 操作详情结构（用于序列化到Details字段）
type OperationDetails struct {
	Before map[string]interface{} `json:"before,omitempty"` // 操作前的值
	After  map[string]interface{} `json:"after,omitempty"`  // 操作后的值
	Reason string                 `json:"reason,omitempty"` // 操作原因
	Extra  map[string]interface{} `json:"extra,omitempty"`  // 额外信息
}

// MarshalDetails 序列化详情
func MarshalDetails(details *OperationDetails) (json.RawMessage, error) {
	return json.Marshal(details)
}

// UnmarshalDetails 反序列化详情
func UnmarshalDetails(raw json.RawMessage) (*OperationDetails, error) {
	var details OperationDetails
	err := json.Unmarshal(raw, &details)
	return &details, err
}
