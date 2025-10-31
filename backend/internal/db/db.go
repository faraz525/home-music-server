package db

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type DB struct{ *sql.DB }

func New(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}
	dbPath := filepath.Join(dataDir, "db", "cratedrop.sqlite")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}
	sqlDB, err := sql.Open("sqlite3", dbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=cache_size(-64000)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	
	// Set connection limits for Raspberry Pi resource constraints
	sqlDB.SetMaxOpenConns(10)  // Limit concurrent connections
	sqlDB.SetMaxIdleConns(5)   // Reduce idle connections
	sqlDB.SetConnMaxLifetime(0) // Reuse connections indefinitely
	
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	d := &DB{sqlDB}
	if err := d.migrate(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *DB) Close() error { return d.DB.Close() }

func (d *DB) migrate() error {
	// Create migrations tracking table if it doesn't exist
	_, err := d.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// List of migrations to apply in order
	migrations := []string{
		"migrations/schema.sql",
		"migrations/002_sharing_and_trading.sql",
	}

	for _, migrationFile := range migrations {
		// Check if migration already applied
		var count int
		err := d.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", migrationFile).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check migration status: %w", err)
		}

		if count > 0 {
			// Migration already applied, skip
			continue
		}

		// Read and execute migration
		b, err := migrationsFS.ReadFile(migrationFile)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", migrationFile, err)
		}

		if _, err := d.Exec(string(b)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", migrationFile, err)
		}

		// Mark migration as applied
		_, err = d.Exec("INSERT INTO schema_migrations (version) VALUES (?)", migrationFile)
		if err != nil {
			return fmt.Errorf("failed to record migration %s: %w", migrationFile, err)
		}

		fmt.Printf("[CrateDrop] Applied migration: %s\n", migrationFile)
	}

	return nil
}
