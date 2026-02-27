package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// HealthChecker performs health checks on database connections.
type HealthChecker struct {
	db *PostgresDB
}

// NewHealthChecker creates a new health checker.
func NewHealthChecker(db *PostgresDB) *HealthChecker {
	return &HealthChecker{db: db}
}

// HealthStatus represents the health status of the database.
type HealthStatus struct {
	Healthy       bool              `json:"healthy"`
	Master        ConnectionHealth  `json:"master"`
	Replicas      []ConnectionHealth `json:"replicas,omitempty"`
	ResponseTime  time.Duration     `json:"response_time_ms"`
	Error         string            `json:"error,omitempty"`
}

// ConnectionHealth represents the health status of a single connection.
type ConnectionHealth struct {
	Healthy      bool          `json:"healthy"`
	ResponseTime time.Duration `json:"response_time_ms"`
	Error        string        `json:"error,omitempty"`
}

// Check performs a health check on all database connections.
func (h *HealthChecker) Check(ctx context.Context) HealthStatus {
	startTime := time.Now()
	
	status := HealthStatus{
		Healthy:  true,
		Replicas: make([]ConnectionHealth, 0),
	}
	
	// Check master
	status.Master = h.checkConnection(ctx, h.db.master, "master")
	if !status.Master.Healthy {
		status.Healthy = false
	}
	
	// Check replicas
	for i, replica := range h.db.replicas {
		replicaHealth := h.checkConnection(ctx, replica, fmt.Sprintf("replica_%d", i))
		status.Replicas = append(status.Replicas, replicaHealth)
		
		// Replicas being unhealthy doesn't make the whole system unhealthy
		// (can still use master for reads)
	}
	
	status.ResponseTime = time.Since(startTime)
	
	return status
}

// checkConnection checks the health of a single database connection.
func (h *HealthChecker) checkConnection(ctx context.Context, db DBExecutor, name string) ConnectionHealth {
	startTime := time.Now()
	
	health := ConnectionHealth{
		Healthy: true,
	}
	
	// Simple ping
	if err := db.PingContext(ctx); err != nil {
		health.Healthy = false
		health.Error = fmt.Sprintf("ping failed: %v", err)
		health.ResponseTime = time.Since(startTime)
		return health
	}
	
	// Execute a simple query
	var result int
	err := db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		health.Healthy = false
		health.Error = fmt.Sprintf("query failed: %v", err)
		health.ResponseTime = time.Since(startTime)
		return health
	}
	
	if result != 1 {
		health.Healthy = false
		health.Error = "unexpected query result"
	}
	
	health.ResponseTime = time.Since(startTime)
	return health
}

// DeepCheck performs a more thorough health check.
// This includes checking connection pool stats and query performance.
func (h *HealthChecker) DeepCheck(ctx context.Context) HealthDeepStatus {
	basicStatus := h.Check(ctx)
	stats := h.db.Stats()
	
	deepStatus := HealthDeepStatus{
		HealthStatus: basicStatus,
		MasterStats:  stats.Master,
		ReplicaStats: stats.Replicas,
		Warnings:     make([]string, 0),
	}
	
	// Check for potential issues
	
	// High wait count indicates connection pool exhaustion
	if stats.Master.WaitCount > 1000 {
		deepStatus.Warnings = append(deepStatus.Warnings,
			fmt.Sprintf("High master wait count: %d", stats.Master.WaitCount))
	}
	
	// High wait duration indicates slow queries or connection pool issues
	if stats.Master.WaitDuration > 10*time.Second {
		deepStatus.Warnings = append(deepStatus.Warnings,
			fmt.Sprintf("High master wait duration: %v", stats.Master.WaitDuration))
	}
	
	// All connections in use might indicate insufficient pool size
	if stats.Master.InUse == stats.Master.OpenConnections && stats.Master.OpenConnections > 0 {
		deepStatus.Warnings = append(deepStatus.Warnings,
			"All master connections in use - consider increasing pool size")
	}
	
	return deepStatus
}

// HealthDeepStatus represents a deep health check status.
type HealthDeepStatus struct {
	HealthStatus
	MasterStats  ConnectionStats   `json:"master_stats"`
	ReplicaStats []ConnectionStats `json:"replica_stats,omitempty"`
	Warnings     []string          `json:"warnings,omitempty"`
}

// DBExecutor is an interface for database connections that can execute queries.
type DBExecutor interface {
	PingContext(ctx context.Context) error
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}
