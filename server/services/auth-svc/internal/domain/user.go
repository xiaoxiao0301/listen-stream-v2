package domain

import (
	"time"

	"github.com/google/uuid"
)

// User 用户实体
type User struct {
	ID           string    // UUID
	Phone        string    // 手机号（唯一）
	TokenVersion int       // Token版本号，用于密钥轮换时全局撤销旧Token
	IsActive     bool      // 账号是否激活
	CreatedAt    time.Time // 创建时间
	UpdatedAt    time.Time // 更新时间
}

// NewUser 创建新用户
func NewUser(phone string) *User {
	now := time.Now()
	return &User{
		ID:           uuid.New().String(),
		Phone:        phone,
		TokenVersion: 1, // 初始版本号为1
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// IncrementTokenVersion 递增Token版本号（用于撤销所有旧Token）
func (u *User) IncrementTokenVersion() {
	u.TokenVersion++
	u.UpdatedAt = time.Now()
}

// Deactivate 停用账号
func (u *User) Deactivate() {
	u.IsActive = false
	u.UpdatedAt = time.Now()
}

// Activate 激活账号
func (u *User) Activate() {
	u.IsActive = true
	u.UpdatedAt = time.Now()
}

// Validate 验证用户数据
func (u *User) Validate() error {
	if u.ID == "" {
		return ErrInvalidUserID
	}
	if u.Phone == "" {
		return ErrInvalidPhone
	}
	if len(u.Phone) != 11 {
		return ErrInvalidPhoneFormat
	}
	if u.TokenVersion < 1 {
		return ErrInvalidTokenVersion
	}
	return nil
}

// IsTokenVersionValid 检查Token版本是否有效
func (u *User) IsTokenVersionValid(tokenVersion int) bool {
	return u.TokenVersion == tokenVersion
}

// CanLogin 检查用户是否可以登录
func (u *User) CanLogin() bool {
	return u.IsActive
}
