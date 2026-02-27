package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/domain"
)

// UserRepository 用户仓储接口
type UserRepository interface {
	// Create 创建用户
	Create(ctx context.Context, user *domain.User) error
	// GetByID 根据ID获取用户
	GetByID(ctx context.Context, id string) (*domain.User, error)
	// GetByPhone 根据手机号获取用户
	GetByPhone(ctx context.Context, phone string) (*domain.User, error)
	// UpdateTokenVersion 更新Token版本号
	UpdateTokenVersion(ctx context.Context, id string, version int) error
	// UpdateActive 更新激活状态
	UpdateActive(ctx context.Context, id string, isActive bool) error
	// Delete 删除用户
	Delete(ctx context.Context, id string) error
	// List 分页查询用户列表
	List(ctx context.Context, limit, offset int) ([]*domain.User, error)
	// Count 统计用户总数
	Count(ctx context.Context) (int64, error)
	// CountActive 统计激活用户数
	CountActive(ctx context.Context) (int64, error)
}

// userRepository PostgreSQL用户仓储实现
type userRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository 创建用户仓储
func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepository{db: db}
}

// Create 创建用户
func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, phone, token_version, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(ctx, query,
		user.ID,
		user.Phone,
		user.TokenVersion,
		user.IsActive,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		return err
	}
	return nil
}

// GetByID 根据ID获取用户
func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT id, phone, token_version, is_active, created_at, updated_at FROM users WHERE id = $1`

	var user domain.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Phone,
		&user.TokenVersion,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetByPhone 根据手机号获取用户
func (r *userRepository) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
	query := `SELECT id, phone, token_version, is_active, created_at, updated_at FROM users WHERE phone = $1`

	var user domain.User
	err := r.db.QueryRow(ctx, query, phone).Scan(
		&user.ID,
		&user.Phone,
		&user.TokenVersion,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// UpdateTokenVersion 更新Token版本号
func (r *userRepository) UpdateTokenVersion(ctx context.Context, id string, version int) error {
	query := `UPDATE users SET token_version = $2, updated_at = $3 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, version, time.Now())
	return err
}

// UpdateActive 更新激活状态
func (r *userRepository) UpdateActive(ctx context.Context, id string, isActive bool) error {
	query := `UPDATE users SET is_active = $2, updated_at = $3 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, isActive, time.Now())
	return err
}

// Delete 删除用户
func (r *userRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// List 分页查询用户列表
func (r *userRepository) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	query := `
		SELECT id, phone, token_version, is_active, created_at, updated_at 
		FROM users 
		ORDER BY created_at DESC 
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var user domain.User
		err := rows.Scan(
			&user.ID,
			&user.Phone,
			&user.TokenVersion,
			&user.IsActive,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

// Count 统计用户总数
func (r *userRepository) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM users`
	var count int64
	err := r.db.QueryRow(ctx, query).Scan(&count)
	return count, err
}

// CountActive 统计激活用户数
func (r *userRepository) CountActive(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM users WHERE is_active = true`
	var count int64
	err := r.db.QueryRow(ctx, query).Scan(&count)
	return count, err
}
