// Package db provides database connection and management utilities.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// PostgresConfig holds PostgreSQL connection configuration.
type PostgresConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	
	// Read replicas for read-write splitting
	ReadReplicas []Replica
}

// Replica represents a read replica configuration.
type Replica struct {
	Host string
	Port int
}

// PostgresDB wraps a PostgreSQL database connection pool.
type PostgresDB struct {
	master   *sql.DB
	replicas []*sql.DB
	nextIdx  int // Round-robin index for replicas
}

// NewPostgresDB creates a new PostgreSQL database connection.
func NewPostgresDB(cfg *PostgresConfig) (*PostgresDB, error) {
	// Connect to master
	master, err := connectPostgres(cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to master: %w", err)
	}
	
	// Configure connection pool
	if cfg.MaxOpenConns > 0 {
		master.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		master.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		master.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	
	// Test master connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := master.PingContext(ctx); err != nil {
		master.Close()
		return nil, fmt.Errorf("failed to ping master: %w", err)
	}
	
	db := &PostgresDB{
		master:   master,
		replicas: make([]*sql.DB, 0),
	}
	
	// Connect to replicas if configured
	for _, replica := range cfg.ReadReplicas {
		replicaDB, err := connectPostgres(replica.Host, replica.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode)
		if err != nil {
			// Log error but don't fail - can still use master for reads
			fmt.Printf("Warning: failed to connect to replica %s:%d: %v\n", replica.Host, replica.Port, err)
			continue
		}
		
		// Configure replica pool
		replicaDB.SetMaxOpenConns(cfg.MaxOpenConns)
		replicaDB.SetMaxIdleConns(cfg.MaxIdleConns)
		replicaDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
		
		db.replicas = append(db.replicas, replicaDB)
	}
	
	return db, nil
}

// connectPostgres creates a PostgreSQL connection.
func connectPostgres(host string, port int, user, password, database, sslMode string) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, database, sslMode)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	return db, nil
}

// Master returns the master database connection for writes.
func (db *PostgresDB) Master() *sql.DB {
	return db.master
}

// Replica returns a read replica database connection.
// If no replicas are available, returns the master.
// Uses round-robin load balancing across replicas.
func (db *PostgresDB) Replica() *sql.DB {
	if len(db.replicas) == 0 {
		return db.master
	}
	
	// Round-robin selection
	replica := db.replicas[db.nextIdx%len(db.replicas)]
	db.nextIdx++
	return replica
}

// BeginTx starts a new transaction on the master database.
func (db *PostgresDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return db.master.BeginTx(ctx, opts)
}

// ExecContext executes a query on the master without returning any rows.
func (db *PostgresDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return db.master.ExecContext(ctx, query, args...)
}

// QueryContext executes a query on a replica that returns rows.
func (db *PostgresDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return db.Replica().QueryContext(ctx, query, args...)
}

// QueryRowContext executes a query on a replica that returns at most one row.
func (db *PostgresDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return db.Replica().QueryRowContext(ctx, query, args...)
}

// Ping checks if all database connections are alive.
func (db *PostgresDB) Ping(ctx context.Context) error {
	// Ping master
	if err := db.master.PingContext(ctx); err != nil {
		return fmt.Errorf("master ping failed: %w", err)
	}
	
	// Ping replicas
	for i, replica := range db.replicas {
		if err := replica.PingContext(ctx); err != nil {
			return fmt.Errorf("replica %d ping failed: %w", i, err)
		}
	}
	
	return nil
}

// Close closes all database connections.
func (db *PostgresDB) Close() error {
	// Close master
	if err := db.master.Close(); err != nil {
		return fmt.Errorf("failed to close master: %w", err)
	}
	
	// Close replicas
	for i, replica := range db.replicas {
		if err := replica.Close(); err != nil {
			return fmt.Errorf("failed to close replica %d: %w", i, err)
		}
	}
	
	return nil
}

// Stats returns database statistics.
func (db *PostgresDB) Stats() DBStats {
	masterStats := db.master.Stats()
	stats := DBStats{
		Master: ConnectionStats{
			OpenConnections:   masterStats.OpenConnections,
			InUse:             masterStats.InUse,
			Idle:              masterStats.Idle,
			WaitCount:         masterStats.WaitCount,
			WaitDuration:      masterStats.WaitDuration,
			MaxIdleClosed:     masterStats.MaxIdleClosed,
			MaxLifetimeClosed: masterStats.MaxLifetimeClosed,
		},
		Replicas: make([]ConnectionStats, len(db.replicas)),
	}
	
	for i, replica := range db.replicas {
		replicaStats := replica.Stats()
		stats.Replicas[i] = ConnectionStats{
			OpenConnections:   replicaStats.OpenConnections,
			InUse:             replicaStats.InUse,
			Idle:              replicaStats.Idle,
			WaitCount:         replicaStats.WaitCount,
			WaitDuration:      replicaStats.WaitDuration,
			MaxIdleClosed:     replicaStats.MaxIdleClosed,
			MaxLifetimeClosed: replicaStats.MaxLifetimeClosed,
		}
	}
	
	return stats
}

// DBStats holds database connection statistics.
type DBStats struct {
	Master   ConnectionStats
	Replicas []ConnectionStats
}

// ConnectionStats holds connection pool statistics.
type ConnectionStats struct {
	OpenConnections   int
	InUse             int
	Idle              int
	WaitCount         int64
	WaitDuration      time.Duration
	MaxIdleClosed     int64
	MaxLifetimeClosed int64
}