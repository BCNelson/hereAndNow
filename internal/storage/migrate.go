package storage

import (
	"bufio"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Migration represents a database migration
type Migration struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	UpSQL       string    `json:"-"`
	DownSQL     string    `json:"-"`
	AppliedAt   time.Time `json:"applied_at"`
	Filename    string    `json:"filename"`
}

// Migrator handles database migrations
type Migrator struct {
	db            *DB
	migrationsDir string
}

// NewMigrator creates a new migration manager
func NewMigrator(db *DB, migrationsDir string) *Migrator {
	return &Migrator{
		db:            db,
		migrationsDir: migrationsDir,
	}
}

// Init creates the migrations tracking table
func (m *Migrator) Init() error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS migrations (
		id INTEGER PRIMARY KEY NOT NULL,
		name TEXT NOT NULL,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		filename TEXT NOT NULL,
		
		UNIQUE(id),
		UNIQUE(name),
		UNIQUE(filename)
	)`

	_, err := m.db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	return nil
}

// Up runs all pending migrations
func (m *Migrator) Up() error {
	if err := m.Init(); err != nil {
		return err
	}

	migrations, err := m.loadMigrationFiles()
	if err != nil {
		return err
	}

	appliedMigrations, err := m.getAppliedMigrations()
	if err != nil {
		return err
	}

	// Create a map of applied migrations for quick lookup
	appliedMap := make(map[int]bool)
	for _, applied := range appliedMigrations {
		appliedMap[applied.ID] = true
	}

	// Apply pending migrations
	for _, migration := range migrations {
		if !appliedMap[migration.ID] {
			if err := m.applyMigration(migration); err != nil {
				return fmt.Errorf("failed to apply migration %03d_%s: %w", migration.ID, migration.Name, err)
			}
			fmt.Printf("Applied migration %03d_%s\n", migration.ID, migration.Name)
		}
	}

	return nil
}

// Down rolls back the last migration
func (m *Migrator) Down() error {
	if err := m.Init(); err != nil {
		return err
	}

	appliedMigrations, err := m.getAppliedMigrations()
	if err != nil {
		return err
	}

	if len(appliedMigrations) == 0 {
		return fmt.Errorf("no migrations to rollback")
	}

	// Get the last applied migration
	lastMigration := appliedMigrations[len(appliedMigrations)-1]

	// Load the migration file to get the down SQL
	migrationFile, err := m.loadMigrationFile(lastMigration.Filename)
	if err != nil {
		return fmt.Errorf("failed to load migration file for rollback: %w", err)
	}

	if migrationFile.DownSQL == "" {
		return fmt.Errorf("migration %03d_%s has no down migration", lastMigration.ID, lastMigration.Name)
	}

	// Apply the down migration
	if err := m.rollbackMigration(migrationFile); err != nil {
		return fmt.Errorf("failed to rollback migration %03d_%s: %w", lastMigration.ID, lastMigration.Name, err)
	}

	fmt.Printf("Rolled back migration %03d_%s\n", lastMigration.ID, lastMigration.Name)
	return nil
}

// Reset rolls back all migrations
func (m *Migrator) Reset() error {
	if err := m.Init(); err != nil {
		return err
	}

	appliedMigrations, err := m.getAppliedMigrations()
	if err != nil {
		return err
	}

	// Rollback migrations in reverse order
	for i := len(appliedMigrations) - 1; i >= 0; i-- {
		migration := appliedMigrations[i]

		// Load the migration file to get the down SQL
		migrationFile, err := m.loadMigrationFile(migration.Filename)
		if err != nil {
			return fmt.Errorf("failed to load migration file for rollback: %w", err)
		}

		if migrationFile.DownSQL == "" {
			return fmt.Errorf("migration %03d_%s has no down migration", migration.ID, migration.Name)
		}

		// Apply the down migration
		if err := m.rollbackMigration(migrationFile); err != nil {
			return fmt.Errorf("failed to rollback migration %03d_%s: %w", migration.ID, migration.Name, err)
		}

		fmt.Printf("Rolled back migration %03d_%s\n", migration.ID, migration.Name)
	}

	return nil
}

// Status shows the current migration status
func (m *Migrator) Status() error {
	if err := m.Init(); err != nil {
		return err
	}

	migrations, err := m.loadMigrationFiles()
	if err != nil {
		return err
	}

	appliedMigrations, err := m.getAppliedMigrations()
	if err != nil {
		return err
	}

	// Create a map of applied migrations for quick lookup
	appliedMap := make(map[int]Migration)
	for _, applied := range appliedMigrations {
		appliedMap[applied.ID] = applied
	}

	fmt.Println("Migration Status:")
	fmt.Println("================")

	for _, migration := range migrations {
		if applied, exists := appliedMap[migration.ID]; exists {
			fmt.Printf("✓ %03d_%s (applied at %s)\n", migration.ID, migration.Name, applied.AppliedAt.Format(time.RFC3339))
		} else {
			fmt.Printf("✗ %03d_%s (pending)\n", migration.ID, migration.Name)
		}
	}

	return nil
}

// applyMigration applies a single migration within a transaction
func (m *Migrator) applyMigration(migration Migration) error {
	tx, err := m.db.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Apply the migration SQL
	if _, err := tx.Exec(migration.UpSQL); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record the migration
	insertSQL := `INSERT INTO migrations (id, name, filename) VALUES (?, ?, ?)`
	if _, err := tx.Exec(insertSQL, migration.ID, migration.Name, migration.Filename); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}

// rollbackMigration rolls back a single migration within a transaction
func (m *Migrator) rollbackMigration(migration Migration) error {
	tx, err := m.db.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Apply the down migration SQL
	if _, err := tx.Exec(migration.DownSQL); err != nil {
		return fmt.Errorf("failed to execute down migration SQL: %w", err)
	}

	// Remove the migration record
	deleteSQL := `DELETE FROM migrations WHERE id = ?`
	if _, err := tx.Exec(deleteSQL, migration.ID); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	return tx.Commit()
}

// loadMigrationFiles loads all migration files from the migrations directory
func (m *Migrator) loadMigrationFiles() ([]Migration, error) {
	if _, err := os.Stat(m.migrationsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("migrations directory does not exist: %s", m.migrationsDir)
	}

	var migrations []Migration

	err := filepath.WalkDir(m.migrationsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".sql") {
			return nil
		}

		migration, err := m.loadMigrationFile(filepath.Base(path))
		if err != nil {
			return fmt.Errorf("failed to load migration file %s: %w", path, err)
		}

		migrations = append(migrations, migration)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort migrations by ID
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].ID < migrations[j].ID
	})

	return migrations, nil
}

// loadMigrationFile loads a single migration file
func (m *Migrator) loadMigrationFile(filename string) (Migration, error) {
	// Parse migration ID and name from filename
	// Expected format: 001_initial_schema.sql
	re := regexp.MustCompile(`^(\d+)_(.+)\.sql$`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) != 3 {
		return Migration{}, fmt.Errorf("invalid migration filename format: %s", filename)
	}

	id, err := strconv.Atoi(matches[1])
	if err != nil {
		return Migration{}, fmt.Errorf("invalid migration ID in filename: %s", filename)
	}

	name := strings.ReplaceAll(matches[2], "_", " ")

	// Read the migration file
	filePath := filepath.Join(m.migrationsDir, filename)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return Migration{}, fmt.Errorf("failed to read migration file: %w", err)
	}

	upSQL, downSQL := m.parseMigrationContent(string(content))

	return Migration{
		ID:       id,
		Name:     name,
		UpSQL:    upSQL,
		DownSQL:  downSQL,
		Filename: filename,
	}, nil
}

// parseMigrationContent separates up and down migrations from file content
func (m *Migrator) parseMigrationContent(content string) (upSQL, downSQL string) {
	lines := strings.Split(content, "\n")
	var upLines, downLines []string
	var inDownSection bool

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Check for down migration marker
		if strings.HasPrefix(trimmed, "-- +migrate down") || 
		   strings.HasPrefix(trimmed, "-- +DOWN") ||
		   strings.HasPrefix(trimmed, "-- DOWN") {
			inDownSection = true
			continue
		}

		// Check for up migration marker (optional, everything before down is up by default)
		if strings.HasPrefix(trimmed, "-- +migrate up") ||
		   strings.HasPrefix(trimmed, "-- +UP") ||
		   strings.HasPrefix(trimmed, "-- UP") {
			inDownSection = false
			continue
		}

		if inDownSection {
			downLines = append(downLines, line)
		} else {
			upLines = append(upLines, line)
		}
	}

	upSQL = strings.TrimSpace(strings.Join(upLines, "\n"))
	downSQL = strings.TrimSpace(strings.Join(downLines, "\n"))

	return upSQL, downSQL
}

// getAppliedMigrations returns all applied migrations
func (m *Migrator) getAppliedMigrations() ([]Migration, error) {
	query := `SELECT id, name, applied_at, filename FROM migrations ORDER BY id`
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	var migrations []Migration
	for rows.Next() {
		var migration Migration
		if err := rows.Scan(&migration.ID, &migration.Name, &migration.AppliedAt, &migration.Filename); err != nil {
			return nil, fmt.Errorf("failed to scan migration row: %w", err)
		}
		migrations = append(migrations, migration)
	}

	return migrations, rows.Err()
}

// Create creates a new migration file
func (m *Migrator) Create(name string) error {
	// Find the next migration ID
	migrations, err := m.loadMigrationFiles()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	nextID := 1
	if len(migrations) > 0 {
		nextID = migrations[len(migrations)-1].ID + 1
	}

	// Create filename
	safeName := strings.ReplaceAll(strings.ToLower(name), " ", "_")
	filename := fmt.Sprintf("%03d_%s.sql", nextID, safeName)
	filePath := filepath.Join(m.migrationsDir, filename)

	// Ensure migrations directory exists
	if err := os.MkdirAll(m.migrationsDir, 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	// Create migration file template
	template := fmt.Sprintf(`-- Migration: %s
-- Created: %s

-- +migrate up
-- Add your up migration here


-- +migrate down
-- Add your down migration here

`, name, time.Now().Format(time.RFC3339))

	if err := os.WriteFile(filePath, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to create migration file: %w", err)
	}

	fmt.Printf("Created migration file: %s\n", filePath)
	return nil
}