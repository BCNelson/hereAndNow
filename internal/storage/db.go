package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the database connection with additional functionality
type DB struct {
	*sql.DB
	path string
}

// Config holds database configuration
type Config struct {
	Path     string
	InMemory bool
}

// NewDB creates a new database connection with WAL mode enabled
func NewDB(config Config) (*DB, error) {
	var dsn string
	var dbPath string

	if config.InMemory {
		dsn = ":memory:"
		dbPath = ":memory:"
	} else {
		if config.Path == "" {
			return nil, fmt.Errorf("database path cannot be empty for file-based database")
		}

		// Ensure the directory exists
		dir := filepath.Dir(config.Path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}

		// Add SQLite connection parameters for WAL mode and foreign keys
		dsn = fmt.Sprintf("%s?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000", config.Path)
		dbPath = config.Path
	}

	sqlDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{
		DB:   sqlDB,
		path: dbPath,
	}

	// Verify WAL mode is enabled (only for file-based databases)
	if !config.InMemory {
		if err := db.verifyWALMode(); err != nil {
			sqlDB.Close()
			return nil, fmt.Errorf("failed to verify WAL mode: %w", err)
		}
	}

	return db, nil
}

// verifyWALMode ensures WAL mode is properly enabled
func (db *DB) verifyWALMode() error {
	var journalMode string
	err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		return fmt.Errorf("failed to check journal mode: %w", err)
	}

	if journalMode != "wal" {
		return fmt.Errorf("WAL mode not enabled, current mode: %s", journalMode)
	}

	// Verify foreign keys are enabled
	var foreignKeys int
	err = db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys)
	if err != nil {
		return fmt.Errorf("failed to check foreign keys: %w", err)
	}

	if foreignKeys != 1 {
		return fmt.Errorf("foreign keys not enabled")
	}

	return nil
}

// Path returns the database file path
func (db *DB) Path() string {
	return db.path
}

// BeginTx starts a new transaction with the given options
func (db *DB) BeginTx() (*sql.Tx, error) {
	return db.DB.Begin()
}

// Health checks the database connection health
func (db *DB) Health() error {
	// Test basic connectivity
	if err := db.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Test a simple query
	var result int
	if err := db.QueryRow("SELECT 1").Scan(&result); err != nil {
		return fmt.Errorf("test query failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("unexpected test query result: %d", result)
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// GetVersion returns the SQLite version
func (db *DB) GetVersion() (string, error) {
	var version string
	err := db.QueryRow("SELECT sqlite_version()").Scan(&version)
	if err != nil {
		return "", fmt.Errorf("failed to get SQLite version: %w", err)
	}
	return version, nil
}

// Stats returns database statistics
type DBStats struct {
	MaxOpenConnections int `json:"max_open_connections"`
	OpenConnections    int `json:"open_connections"`
	InUse              int `json:"in_use"`
	Idle               int `json:"idle"`
}

// GetStats returns database connection statistics
func (db *DB) GetStats() DBStats {
	stats := db.DB.Stats()
	return DBStats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
	}
}

// Vacuum optimizes the database (should be run periodically)
func (db *DB) Vacuum() error {
	_, err := db.Exec("VACUUM")
	if err != nil {
		return fmt.Errorf("failed to vacuum database: %w", err)
	}
	return nil
}

// WALCheckpoint forces a WAL checkpoint
func (db *DB) WALCheckpoint() error {
	_, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	if err != nil {
		return fmt.Errorf("failed to checkpoint WAL: %w", err)
	}
	return nil
}