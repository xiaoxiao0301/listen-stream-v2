package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// Migrator handles database migrations.
type Migrator struct {
	db      *sql.DB
	migrate *migrate.Migrate
}

// NewMigrator creates a new database migrator.
// The migrations parameter should be an embed.FS containing migration files.
//
// Example:
//
//	//go:embed migrations/*.sql
//	var migrationsFS embed.FS
//
//	migrator, err := db.NewMigrator(dbConn.Master(), migrationsFS, "migrations")
func NewMigrator(db *sql.DB, migrations embed.FS, migrationsPath string) (*Migrator, error) {
	// Create source driver from embedded filesystem
	sourceDriver, err := iofs.New(migrations, migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create source driver: %w", err)
	}
	
	// Create database driver
	dbDriver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create database driver: %w", err)
	}
	
	// Create migrate instance
	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", dbDriver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrator: %w", err)
	}
	
	return &Migrator{
		db:      db,
		migrate: m,
	}, nil
}

// Up runs all pending migrations.
func (m *Migrator) Up() error {
	if err := m.migrate.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration up failed: %w", err)
	}
	return nil
}

// Down rolls back the last migration.
func (m *Migrator) Down() error {
	if err := m.migrate.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration down failed: %w", err)
	}
	return nil
}

// Steps runs n migrations. n can be negative to rollback.
func (m *Migrator) Steps(n int) error {
	if err := m.migrate.Steps(n); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration steps failed: %w", err)
	}
	return nil
}

// Migrate migrates to a specific version.
func (m *Migrator) Migrate(version uint) error {
	if err := m.migrate.Migrate(version); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration to version %d failed: %w", version, err)
	}
	return nil
}

// Version returns the current migration version.
func (m *Migrator) Version() (uint, bool, error) {
	version, dirty, err := m.migrate.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return 0, false, fmt.Errorf("failed to get version: %w", err)
	}
	return version, dirty, nil
}

// Force sets the migration version without running migrations.
// This is useful for fixing a dirty migration state.
func (m *Migrator) Force(version int) error {
	if err := m.migrate.Force(version); err != nil {
		return fmt.Errorf("failed to force version: %w", err)
	}
	return nil
}

// Drop drops all tables in the database. Use with caution!
func (m *Migrator) Drop() error {
	if err := m.migrate.Drop(); err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}
	return nil
}

// Close closes the migrator.
func (m *Migrator) Close() error {
	srcErr, dbErr := m.migrate.Close()
	if srcErr != nil {
		return fmt.Errorf("failed to close source: %w", srcErr)
	}
	if dbErr != nil {
		return fmt.Errorf("failed to close database: %w", dbErr)
	}
	return nil
}

// MigrationInfo holds information about a migration.
type MigrationInfo struct {
	Version   uint      `json:"version"`
	Name      string    `json:"name"`
	Applied   bool      `json:"applied"`
	AppliedAt time.Time `json:"applied_at,omitempty"`
}

// GetMigrationHistory returns the migration history.
func (m *Migrator) GetMigrationHistory(ctx context.Context) ([]MigrationInfo, error) {
	query := `
		SELECT version, name, applied_at
		FROM schema_migrations
		ORDER BY version ASC
	`
	
	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		// Table might not exist yet
		return []MigrationInfo{}, nil
	}
	defer rows.Close()
	
	var history []MigrationInfo
	for rows.Next() {
		var info MigrationInfo
		if err := rows.Scan(&info.Version, &info.Name, &info.AppliedAt); err != nil {
			return nil, fmt.Errorf("failed to scan migration info: %w", err)
		}
		info.Applied = true
		history = append(history, info)
	}
	
	return history, nil
}

// Status returns the current migration status.
func (m *Migrator) Status(ctx context.Context) (MigrationStatus, error) {
	version, dirty, err := m.Version()
	if err != nil {
		return MigrationStatus{}, fmt.Errorf("failed to get version: %w", err)
	}
	
	history, err := m.GetMigrationHistory(ctx)
	if err != nil {
		return MigrationStatus{}, fmt.Errorf("failed to get history: %w", err)
	}
	
	return MigrationStatus{
		CurrentVersion:  version,
		Dirty:           dirty,
		MigrationsCount: len(history),
		History:         history,
	}, nil
}

// MigrationStatus represents the current migration status.
type MigrationStatus struct {
	CurrentVersion  uint            `json:"current_version"`
	Dirty           bool            `json:"dirty"`
	MigrationsCount int             `json:"migrations_count"`
	History         []MigrationInfo `json:"history"`
}

// EnsureSchema ensures that the database schema is up to date.
// This is a safe way to run migrations on application startup.
func (m *Migrator) EnsureSchema(ctx context.Context) error {
	// Check if schema_migrations table exists
	var exists bool
	err := m.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'schema_migrations'
		)
	`).Scan(&exists)
	
	if err != nil {
		return fmt.Errorf("failed to check schema_migrations table: %w", err)
	}
	
	if !exists {
		// First time setup - run all migrations
		if err := m.Up(); err != nil {
			return fmt.Errorf("failed to run initial migrations: %w", err)
		}
		return nil
	}
	
	// Check if there are pending migrations
	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get version: %w", err)
	}
	
	if dirty {
		return fmt.Errorf("database is in dirty state (version %d). Please fix manually with 'force' command", version)
	}
	
	// Run pending migrations
	if err := m.Up(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	
	return nil
}
