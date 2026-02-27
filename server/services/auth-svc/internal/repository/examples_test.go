package repository_test

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/domain"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/repository"
)

// ExampleNewPool 展示如何创建优化的数据库连接池
func ExampleNewPool() {
	ctx := context.Background()

	// 使用默认配置
	cfg := repository.DefaultDBConfig()

	// 或自定义配置
	cfg = &repository.DBConfig{
		Host:              "localhost",
		Port:              5432,
		User:              "auth_user",
		Password:          "secure_password",
		Database:          "auth",
		MaxConns:          20,               // 最大20个连接
		MinConns:          5,                // 最少保持5个连接
		MaxConnLifetime:   time.Hour,        // 连接最多存活1小时
		MaxConnIdleTime:   30 * time.Minute, // 空闲连接30分钟后关闭
		HealthCheckPeriod: time.Minute,      // 每分钟健康检查
	}

	// 创建连接池
	pool, err := repository.NewPool(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create pool: %v", err)
	}
	defer repository.ClosePool(pool)

	log.Println("Database pool created successfully")
}

// ExampleTransaction 展示如何使用事务
func ExampleTransaction() {
	ctx := context.Background()

	cfg := repository.DefaultDBConfig()
	pool, _ := repository.NewPool(ctx, cfg)
	defer repository.ClosePool(pool)

	// 创建仓储
	userRepo := repository.NewUserRepository(pool)
	deviceRepo := repository.NewDeviceRepository(pool)

	// 创建事务执行器
	txExecutor := repository.NewTransaction(pool)

	// 在事务中执行多个操作
	err := txExecutor.ExecTx(ctx, func(tx pgx.Tx) error {
		// 注意：这里需要修改仓储接口以支持传入tx
		// 这是一个示例，展示事务的概念

		user := domain.NewUser("13800138000")
		if err := userRepo.Create(ctx, user); err != nil {
			return err
		}

		device := domain.NewDevice(
			user.ID,
			"iPhone 13 Pro",
			"iOS",
			"1.0.0",
			"192.168.1.1",
			"fingerprint123",
		)
		if err := deviceRepo.Create(ctx, device); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Printf("Transaction failed: %v", err)
	} else {
		log.Println("Transaction completed successfully")
	}
}

// ExampleBatchOperations 展示如何使用批量操作
func ExampleBatchOperations() {
	ctx := context.Background()

	cfg := repository.DefaultDBConfig()
	pool, _ := repository.NewPool(ctx, cfg)
	defer repository.ClosePool(pool)

	// 创建批量操作实例
	batchOps := repository.NewBatchOperations(pool)

	// 批量创建用户
	users := []*domain.User{
		domain.NewUser("13800138001"),
		domain.NewUser("13800138002"),
		domain.NewUser("13800138003"),
	}

	err := batchOps.BatchCreateUsers(ctx, users)
	if err != nil {
		log.Printf("Batch create users failed: %v", err)
	} else {
		log.Printf("Successfully created %d users", len(users))
	}

	// 批量删除设备
	deviceIDs := []string{"device-id-1", "device-id-2", "device-id-3"}
	err = batchOps.BatchDeleteDevices(ctx, deviceIDs)
	if err != nil {
		log.Printf("Batch delete devices failed: %v", err)
	} else {
		log.Printf("Successfully deleted %d devices", len(deviceIDs))
	}
}

// ExampleCleanupService 展示如何使用清理服务
func ExampleCleanupService() {
	ctx := context.Background()

	cfg := repository.DefaultDBConfig()
	pool, _ := repository.NewPool(ctx, cfg)
	defer repository.ClosePool(pool)

	// 创建仓储
	smsVerifyRepo := repository.NewSMSVerificationRepository(pool)
	deviceRepo := repository.NewDeviceRepository(pool)
	smsRecordRepo := repository.NewSMSRecordRepository(pool)

	// 创建清理服务
	cleanupService := repository.NewCleanupService(
		smsVerifyRepo,
		deviceRepo,
		smsRecordRepo,
	)

	// 手动执行清理任务
	if err := cleanupService.CleanExpiredSMSVerifications(ctx); err != nil {
		log.Printf("Cleanup failed: %v", err)
	}

	// 或启动定期清理（在goroutine中运行）
	go cleanupService.StartScheduledCleanup(ctx)
	log.Println("Cleanup service started")
}

// ExampleHealthChecker 展示如何使用健康检查
func ExampleHealthChecker() {
	ctx := context.Background()

	cfg := repository.DefaultDBConfig()
	pool, _ := repository.NewPool(ctx, cfg)
	defer repository.ClosePool(pool)

	// 创建健康检查器
	healthChecker := repository.NewHealthChecker(pool)

	// 执行健康检查
	status := healthChecker.Check(ctx)

	if status.Healthy {
		log.Printf("Database is healthy (response time: %v)", status.ResponseTime)
		log.Printf("Pool utilization: %.2f%%", status.PoolStats.UtilizationRate*100)
	} else {
		log.Printf("Database is unhealthy: %s", status.Error)
	}

	// 带超时的健康检查
	status = healthChecker.CheckWithTimeout(5 * time.Second)
	log.Printf("Health check result: %+v", status)
}

// ExampleMonitor 展示如何使用监控器
func ExampleMonitor() {
	ctx := context.Background()

	cfg := repository.DefaultDBConfig()
	pool, _ := repository.NewPool(ctx, cfg)
	defer repository.ClosePool(pool)

	// 创建监控器
	monitor := repository.NewMonitor(pool)

	// 获取连接池指标
	metrics := monitor.GetPoolMetrics()
	log.Printf("Pool metrics: %+v", metrics)

	// 获取表大小统计
	tableSizes, err := monitor.GetTableSizes(ctx)
	if err != nil {
		log.Printf("Failed to get table sizes: %v", err)
	} else {
		log.Println("Table sizes:")
		for _, ts := range tableSizes {
			log.Printf("  %s: %s", ts.TableName, ts.Size)
		}
	}

	// 检查慢查询（需要启用pg_stat_statements）
	slowQueries, err := monitor.CheckSlowQueries(ctx, 100) // 查询平均执行时间>100ms的SQL
	if err != nil {
		log.Printf("Failed to check slow queries: %v", err)
	} else if len(slowQueries) > 0 {
		log.Println("Slow queries detected:")
		for _, sq := range slowQueries {
			log.Printf("  Query: %s, Mean time: %.2fms", sq.Query[:50], sq.MeanExecTime)
		}
	}
}

// ExamplePoolStats 展示如何监控连接池状态
func ExamplePoolStats() {
	ctx := context.Background()

	cfg := repository.DefaultDBConfig()
	pool, _ := repository.NewPool(ctx, cfg)
	defer repository.ClosePool(pool)

	// 定期打印连接池统计
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := repository.PoolStats(pool)
			log.Printf("Pool stats: Acquired=%d, Idle=%d, Total=%d/%d",
				stats.AcquiredConns(),
				stats.IdleConns(),
				stats.TotalConns(),
				stats.MaxConns(),
			)
		}
	}
}

// ExampleCompleteSetup 展示完整的设置流程
func ExampleCompleteSetup() {
	ctx := context.Background()

	// 1. 创建优化的连接池
	cfg := repository.DefaultDBConfig()
	pool, err := repository.NewPool(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create pool: %v", err)
	}
	defer repository.ClosePool(pool)

	// 2. 创建所有仓储
	userRepo := repository.NewUserRepository(pool)
	deviceRepo := repository.NewDeviceRepository(pool)
	smsVerifyRepo := repository.NewSMSVerificationRepository(pool)
	smsRecordRepo := repository.NewSMSRecordRepository(pool)

	// 3. 创建批量操作
	batchOps := repository.NewBatchOperations(pool)

	// 4. 创建事务执行器
	txExecutor := repository.NewTransaction(pool)

	// 5. 创建清理服务并启动
	cleanupService := repository.NewCleanupService(
		smsVerifyRepo,
		deviceRepo,
		smsRecordRepo,
	)
	go cleanupService.StartScheduledCleanup(ctx)

	// 6. 创建健康检查器
	healthChecker := repository.NewHealthChecker(pool)

	// 7. 创建监控器
	monitor := repository.NewMonitor(pool)

	// 8. 启动健康检查（每30秒）
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				status := healthChecker.Check(ctx)
				if !status.Healthy {
					log.Printf("ALERT: Database unhealthy: %s", status.Error)
				}
			}
		}
	}()

	// 9. 启动指标采集（每分钟）
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				metrics := monitor.GetPoolMetrics()
				// 这里可以发送到Prometheus或其他监控系统
				log.Printf("Metrics collected: %+v", metrics)
			}
		}
	}()

	log.Println("Repository layer fully initialized with optimizations")

	// 使用仓储...
	_ = userRepo
	_ = deviceRepo
	_ = batchOps
	_ = txExecutor
}
