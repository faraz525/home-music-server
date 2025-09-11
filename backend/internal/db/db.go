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
    // Simple one-shot migration: apply schema.sql if users table missing
    var tableCount int
    _ = d.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users'").Scan(&tableCount)
    if tableCount > 0 {
        return nil
    }
    b, err := migrationsFS.ReadFile("migrations/schema.sql")
    if err != nil {
        return fmt.Errorf("failed to read schema: %w", err)
    }
    if _, err := d.Exec(string(b)); err != nil {
        return fmt.Errorf("failed to execute schema: %w", err)
    }
    return nil
}

