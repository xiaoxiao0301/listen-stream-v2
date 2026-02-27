package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/domain"
)

// DeviceRepository 设备仓储接口
type DeviceRepository interface {
	// Create 创建设备
	Create(ctx context.Context, device *domain.Device) error
	// GetByID 根据ID获取设备
	GetByID(ctx context.Context, id string) (*domain.Device, error)
	// GetByFingerprint 根据用户ID和指纹获取设备
	GetByFingerprint(ctx context.Context, userID, fingerprint string) (*domain.Device, error)
	// ListByUserID 获取用户的所有设备
	ListByUserID(ctx context.Context, userID string) ([]*domain.Device, error)
	// CountByUserID 统计用户设备数量
	CountByUserID(ctx context.Context, userID string) (int64, error)
	// UpdateLoginInfo 更新登录信息
	UpdateLoginInfo(ctx context.Context, id string, ip string, loginAt time.Time) error
	// Delete 删除设备
	Delete(ctx context.Context, id string) error
	// DeleteByUserID 删除用户的所有设备
	DeleteByUserID(ctx context.Context, userID string) error
	// DeleteInactive 删除不活跃设备（超过90天未登录）
	DeleteInactive(ctx context.Context, before time.Time) error
}

// deviceRepository PostgreSQL设备仓储实现
type deviceRepository struct {
	db *pgxpool.Pool
}

// NewDeviceRepository 创建设备仓储
func NewDeviceRepository(db *pgxpool.Pool) DeviceRepository {
	return &deviceRepository{db: db}
}

// Create 创建设备
func (r *deviceRepository) Create(ctx context.Context, device *domain.Device) error {
	query := `
		INSERT INTO devices (
			id, user_id, device_name, fingerprint, platform,
			app_version, last_ip, last_login_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.Exec(ctx, query,
		device.ID,
		device.UserID,
		device.DeviceName,
		device.Fingerprint,
		device.Platform,
		device.AppVersion,
		device.LastIP,
		device.LastLoginAt,
		device.CreatedAt,
	)
	return err
}

// GetByID 根据ID获取设备
func (r *deviceRepository) GetByID(ctx context.Context, id string) (*domain.Device, error) {
	query := `
		SELECT id, user_id, device_name, fingerprint, platform,
		       app_version, last_ip, last_login_at, created_at
		FROM devices
		WHERE id = $1
	`

	var device domain.Device
	err := r.db.QueryRow(ctx, query, id).Scan(
		&device.ID,
		&device.UserID,
		&device.DeviceName,
		&device.Fingerprint,
		&device.Platform,
		&device.AppVersion,
		&device.LastIP,
		&device.LastLoginAt,
		&device.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrDeviceNotFound
		}
		return nil, err
	}
	return &device, nil
}

// GetByFingerprint 根据用户ID和指纹获取设备
func (r *deviceRepository) GetByFingerprint(ctx context.Context, userID, fingerprint string) (*domain.Device, error) {
	query := `
		SELECT id, user_id, device_name, fingerprint, platform,
		       app_version, last_ip, last_login_at, created_at
		FROM devices
		WHERE user_id = $1 AND fingerprint = $2
		LIMIT 1
	`

	var device domain.Device
	err := r.db.QueryRow(ctx, query, userID, fingerprint).Scan(
		&device.ID,
		&device.UserID,
		&device.DeviceName,
		&device.Fingerprint,
		&device.Platform,
		&device.AppVersion,
		&device.LastIP,
		&device.LastLoginAt,
		&device.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrDeviceNotFound
		}
		return nil, err
	}
	return &device, nil
}

// ListByUserID 获取用户的所有设备
func (r *deviceRepository) ListByUserID(ctx context.Context, userID string) ([]*domain.Device, error) {
	query := `
		SELECT id, user_id, device_name, fingerprint, platform,
		       app_version, last_ip, last_login_at, created_at
		FROM devices
		WHERE user_id = $1
		ORDER BY last_login_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []*domain.Device
	for rows.Next() {
		var device domain.Device
		err := rows.Scan(
			&device.ID,
			&device.UserID,
			&device.DeviceName,
			&device.Fingerprint,
			&device.Platform,
			&device.AppVersion,
			&device.LastIP,
			&device.LastLoginAt,
			&device.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		devices = append(devices, &device)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return devices, nil
}

// CountByUserID 统计用户设备数量
func (r *deviceRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	query := `SELECT COUNT(*) FROM devices WHERE user_id = $1`
	var count int64
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

// UpdateLoginInfo 更新登录信息
func (r *deviceRepository) UpdateLoginInfo(ctx context.Context, id string, ip string, loginAt time.Time) error {
	query := `UPDATE devices SET last_ip = $2, last_login_at = $3 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, ip, loginAt)
	return err
}

// Delete 删除设备
func (r *deviceRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM devices WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// DeleteByUserID 删除用户的所有设备
func (r *deviceRepository) DeleteByUserID(ctx context.Context, userID string) error {
	query := `DELETE FROM devices WHERE user_id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

// DeleteInactive 删除不活跃设备（超过90天未登录）
func (r *deviceRepository) DeleteInactive(ctx context.Context, before time.Time) error {
	query := `DELETE FROM devices WHERE last_login_at < $1`
	_, err := r.db.Exec(ctx, query, before)
	return err
}
