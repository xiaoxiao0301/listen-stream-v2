package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBConfig 数据库配置
type DBConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	MaxConns        int32         // 最大连接数
	MinConns        int32         // 最小连接数
	MaxConnLifetime time.Duration // 连接最大生命周期
	MaxConnIdleTime time.Duration // 空闲连接超时
	HealthCheckPeriod time.Duration // 健康检查周期
}

// DefaultDBConfig 返回默认数据库配置
func DefaultDBConfig() *DBConfig {
	return &DBConfig{
		Host:              "localhost",
		Port:              5432,
		User:              "postgres",
		Password:          "postgres",
		Database:          "auth",
		MaxConns:          20,                    // 最大20个连接
		MinConns:          5,                     // 最少保持5个连接
		MaxConnLifetime:   time.Hour,             // 连接最多存活1小时
		MaxConnIdleTime:   30 * time.Minute,      // 空闲连接30分钟后关闭
		HealthCheckPeriod: time.Minute,           // 每分钟健康检查
	}
}

// NewPool 创建优化的数据库连接池
func NewPool(ctx context.Context, cfg *DBConfig) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database,
	)

	// 解析配置
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// 应用性能优化配置
	config.MaxConns = cfg.MaxConns
	config.MinConns = cfg.MinConns
	config.MaxConnLifetime = cfg.MaxConnLifetime
	config.MaxConnIdleTime = cfg.MaxConnIdleTime
	config.HealthCheckPeriod = cfg.HealthCheckPeriod

	// 创建连接池
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	// 验证连接
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}

// Transaction 事务执行器接口
type Transaction interface {
	ExecTx(ctx context.Context, fn func(pgx.Tx) error) error
}

// txExecutor 事务执行器实现
type txExecutor struct {
	pool *pgxpool.Pool
}

// NewTransaction 创建事务执行器
func NewTransaction(pool *pgxpool.Pool) Transaction {
	return &txExecutor{pool: pool}
}

// ExecTx 在事务中执行函数
func (e *txExecutor) ExecTx(ctx context.Context, fn func(pgx.Tx) error) error {
	// 开始事务
	tx, err := e.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// 确保事务会被提交或回滚
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				err = fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
			}
		}
	}()

	// 执行业务逻辑
	if err = fn(tx); err != nil {
		return err
	}

	// 提交事务
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// PoolStats 获取连接池统计信息
func PoolStats(pool *pgxpool.Pool) *pgxpool.Stat {
	return pool.Stat()
}

// ClosePool 关闭连接池
func ClosePool(pool *pgxpool.Pool) {
	if pool != nil {
		pool.Close()
	}
}
