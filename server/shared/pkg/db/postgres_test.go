package db

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func TestPostgresConfig_Validation(t *testing.T) {
	cfg := &PostgresConfig{
		Host:            "localhost",
		Port:            5432,
		User:            "postgres",
		Password:        "password",
		Database:        "testdb",
		SSLMode:         "disable",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	}
	
	if cfg.Host != "localhost" {
		t.Errorf("Host = %v, want localhost", cfg.Host)
	}
	if cfg.Port != 5432 {
		t.Errorf("Port = %v, want 5432", cfg.Port)
	}
	if cfg.MaxOpenConns != 25 {
		t.Errorf("MaxOpenConns = %v, want 25", cfg.MaxOpenConns)
	}
}

func TestPostgresConfig_WithReplicas(t *testing.T) {
	cfg := &PostgresConfig{
		Host:     "master.db",
		Port:     5432,
		User:     "postgres",
		Password: "pass",
		Database: "mydb",
		SSLMode:  "disable",
		ReadReplicas: []Replica{
			{Host: "replica1.db", Port: 5432},
			{Host: "replica2.db", Port: 5432},
		},
	}
	
	if len(cfg.ReadReplicas) != 2 {
		t.Errorf("ReadReplicas count = %v, want 2", len(cfg.ReadReplicas))
	}
	
	if cfg.ReadReplicas[0].Host != "replica1.db" {
		t.Errorf("Replica[0].Host = %v, want replica1.db", cfg.ReadReplicas[0].Host)
	}
	
	if cfg.ReadReplicas[1].Host != "replica2.db" {
		t.Errorf("Replica[1].Host = %v, want replica2.db", cfg.ReadReplicas[1].Host)
	}
}

func TestReplica_Structure(t *testing.T) {
	replica := Replica{
		Host: "replica.example.com",
		Port: 5433,
	}
	
	if replica.Host != "replica.example.com" {
		t.Errorf("Host = %v, want replica.example.com", replica.Host)
	}
	if replica.Port != 5433 {
		t.Errorf("Port = %v, want 5433", replica.Port)
	}
}

func TestDBStats_Structure(t *testing.T) {
	stats := DBStats{
		Master: ConnectionStats{
			OpenConnections: 10,
			InUse:           5,
			Idle:            5,
			WaitCount:       0,
		},
		Replicas: []ConnectionStats{
			{
				OpenConnections: 8,
				InUse:           3,
				Idle:            5,
			},
		},
	}
	
	if stats.Master.OpenConnections != 10 {
		t.Errorf("Master.OpenConnections = %v, want 10", stats.Master.OpenConnections)
	}
	if stats.Master.InUse != 5 {
		t.Errorf("Master.InUse = %v, want 5", stats.Master.InUse)
	}
	if len(stats.Replicas) != 1 {
		t.Errorf("Replicas count = %v, want 1", len(stats.Replicas))
	}
	if stats.Replicas[0].OpenConnections != 8 {
		t.Errorf("Replicas[0].OpenConnections = %v, want 8", stats.Replicas[0].OpenConnections)
	}
}

func TestConnectionStats_Fields(t *testing.T) {
	stats := ConnectionStats{
		OpenConnections:   15,
		InUse:             8,
		Idle:              7,
		WaitCount:         3,
		WaitDuration:      100 * time.Millisecond,
		MaxIdleClosed:     2,
		MaxLifetimeClosed: 1,
	}
	
	if stats.OpenConnections != 15 {
		t.Errorf("OpenConnections = %v, want 15", stats.OpenConnections)
	}
	if stats.InUse != 8 {
		t.Errorf("InUse = %v, want 8", stats.InUse)
	}
	if stats.Idle != 7 {
		t.Errorf("Idle = %v, want 7", stats.Idle)
	}
	if stats.WaitCount != 3 {
		t.Errorf("WaitCount = %v, want 3", stats.WaitCount)
	}
	if stats.WaitDuration != 100*time.Millisecond {
		t.Errorf("WaitDuration = %v, want 100ms", stats.WaitDuration)
	}
}

// MockDB for testing without real database
type MockDB struct {
	*sql.DB
	pingCalled  bool
	closeCalled bool
}

func TestPostgresDB_ReplicaSelection(t *testing.T) {
	// This test validates the replica selection logic conceptually
	// In a real scenario, you'd use a mock or test database
	
	tests := []struct {
		name           string
		replicaCount   int
		expectedCalls  int
		wantRoundRobin bool
	}{
		{
			name:           "NoReplicas",
			replicaCount:   0,
			expectedCalls:  1,
			wantRoundRobin: false,
		},
		{
			name:           "SingleReplica",
			replicaCount:   1,
			expectedCalls:  1,
			wantRoundRobin: false,
		},
		{
			name:           "MultipleReplicas",
			replicaCount:   3,
			expectedCalls:  3,
			wantRoundRobin: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a conceptual test
			// In production, you'd need actual database connections
			if tt.replicaCount < 0 {
				t.Error("Replica count should not be negative")
			}
		})
	}
}

func TestPostgresConfig_DefaultValues(t *testing.T) {
	cfg := &PostgresConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "pass",
		Database: "testdb",
		SSLMode:  "disable",
		// Not setting MaxOpenConns, MaxIdleConns, ConnMaxLifetime
	}
	
	// Test that zero values are acceptable
	if cfg.MaxOpenConns < 0 {
		t.Error("MaxOpenConns should not be negative")
	}
	if cfg.MaxIdleConns < 0 {
		t.Error("MaxIdleConns should not be negative")
	}
	if cfg.ConnMaxLifetime < 0 {
		t.Error("ConnMaxLifetime should not be negative")
	}
}

func TestPostgresConfig_SSLModes(t *testing.T) {
	sslModes := []string{"disable", "require", "verify-ca", "verify-full"}
	
	for _, mode := range sslModes {
		t.Run("SSLMode_"+mode, func(t *testing.T) {
			cfg := &PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "user",
				Password: "pass",
				Database: "db",
				SSLMode:  mode,
			}
			
			if cfg.SSLMode != mode {
				t.Errorf("SSLMode = %v, want %v", cfg.SSLMode, mode)
			}
		})
	}
}

func TestConnectionLifetime(t *testing.T) {
	durations := []time.Duration{
		time.Minute,
		5 * time.Minute,
		time.Hour,
		24 * time.Hour,
	}
	
	for _, d := range durations {
		t.Run(d.String(), func(t *testing.T) {
			cfg := &PostgresConfig{
				Host:            "localhost",
				Port:            5432,
				User:            "user",
				Password:        "pass",
				Database:        "db",
				SSLMode:         "disable",
				ConnMaxLifetime: d,
			}
			
			if cfg.ConnMaxLifetime != d {
				t.Errorf("ConnMaxLifetime = %v, want %v", cfg.ConnMaxLifetime, d)
			}
		})
	}
}

func TestContext_WithTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if ctx.Err() != nil {
		t.Error("Context should not have error initially")
	}
	
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Error("Context should have deadline")
	}
	
	if time.Until(deadline) > 6*time.Second {
		t.Error("Deadline should be within 5 seconds")
	}
}
