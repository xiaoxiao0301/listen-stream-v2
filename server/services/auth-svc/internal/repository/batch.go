package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/domain"
)

// BatchOperations 批量操作接口
type BatchOperations interface {
	// BatchCreateUsers 批量创建用户
	BatchCreateUsers(ctx context.Context, users []*domain.User) error
	// BatchCreateDevices 批量创建设备
	BatchCreateDevices(ctx context.Context, devices []*domain.Device) error
	// BatchDeleteDevices 批量删除设备
	BatchDeleteDevices(ctx context.Context, deviceIDs []string) error
}

// batchOperations 批量操作实现
type batchOperations struct {
	pool *pgxpool.Pool
}

// NewBatchOperations 创建批量操作实例
func NewBatchOperations(pool *pgxpool.Pool) BatchOperations {
	return &batchOperations{pool: pool}
}

// BatchCreateUsers 批量创建用户（使用事务）
func (b *batchOperations) BatchCreateUsers(ctx context.Context, users []*domain.User) error {
	if len(users) == 0 {
		return nil
	}

	// 使用事务确保原子性
	tx, err := b.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 使用批量插入
	batch := &pgx.Batch{}
	query := `
		INSERT INTO users (id, phone, token_version, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	for _, user := range users {
		batch.Queue(query,
			user.ID,
			user.Phone,
			user.TokenVersion,
			user.IsActive,
			user.CreatedAt,
			user.UpdatedAt,
		)
	}

	// 执行批量操作
	results := tx.SendBatch(ctx, batch)
	defer results.Close()

	// 检查每个插入的结果
	for i := 0; i < len(users); i++ {
		_, err := results.Exec()
		if err != nil {
			return fmt.Errorf("insert user %d: %w", i, err)
		}
	}

	// 提交事务
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// BatchCreateDevices 批量创建设备（使用事务）
func (b *batchOperations) BatchCreateDevices(ctx context.Context, devices []*domain.Device) error {
	if len(devices) == 0 {
		return nil
	}

	// 使用事务确保原子性
	tx, err := b.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 使用批量插入
	batch := &pgx.Batch{}
	query := `
		INSERT INTO devices (
			id, user_id, device_name, fingerprint, platform,
			app_version, last_ip, last_login_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	for _, device := range devices {
		batch.Queue(query,
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
	}

	// 执行批量操作
	results := tx.SendBatch(ctx, batch)
	defer results.Close()

	// 检查每个插入的结果
	for i := 0; i < len(devices); i++ {
		_, err := results.Exec()
		if err != nil {
			return fmt.Errorf("insert device %d: %w", i, err)
		}
	}

	// 提交事务
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// BatchDeleteDevices 批量删除设备（使用事务）
func (b *batchOperations) BatchDeleteDevices(ctx context.Context, deviceIDs []string) error {
	if len(deviceIDs) == 0 {
		return nil
	}

	// 使用事务确保原子性
	tx, err := b.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 使用 IN 子句批量删除
	query := `DELETE FROM devices WHERE id = ANY($1)`
	_, err = tx.Exec(ctx, query, deviceIDs)
	if err != nil {
		return fmt.Errorf("batch delete devices: %w", err)
	}

	// 提交事务
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
