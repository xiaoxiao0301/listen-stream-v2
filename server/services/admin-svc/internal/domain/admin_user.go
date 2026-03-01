package domain

import "time"

// AdminUser 管理员实体
type AdminUser struct {
	ID           string    `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	PasswordHash string    `json:"-" db:"password_hash"` // 不在JSON中暴露
	Email        string    `json:"email" db:"email"`
	Role         string    `json:"role" db:"role"` // admin, operator, viewer
	Status       string    `json:"status" db:"status"` // active, disabled
	TOTPSecret   string    `json:"-" db:"totp_secret"` // TOTP密钥（可选）
	TOTPEnabled  bool      `json:"totp_enabled" db:"totp_enabled"` // 是否启用2FA
	LastLoginAt  *time.Time `json:"last_login_at" db:"last_login_at"`
	LastLoginIP  string    `json:"last_login_ip" db:"last_login_ip"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// AdminRole 管理员角色常量
const (
	RoleAdmin    = "admin"    // 超级管理员：所有权限
	RoleOperator = "operator" // 运营：查看+编辑
	RoleViewer   = "viewer"   // 只读：仅查看
)

// AdminStatus 管理员状态常量
const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
)

// HasPermission 检查是否有权限
func (u *AdminUser) HasPermission(permission string) bool {
	switch u.Role {
	case RoleAdmin:
		return true // 管理员有所有权限
	case RoleOperator:
		// 运营可以编辑，但不能管理用户
		return permission != "manage_admins"
	case RoleViewer:
		// 只读用户只能查看
		return permission == "view"
	default:
		return false
	}
}

// UpdateLastLogin 更新最后登录信息
func (u *AdminUser) UpdateLastLogin(ip string) {
	now := time.Now()
	u.LastLoginAt = &now
	u.LastLoginIP = ip
}
