package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Logging  LoggingConfig  `yaml:"logging"`
	Features FeaturesConfig `yaml:"features"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
	Path  string `yaml:"path"`
}

type FeaturesConfig struct {
	NaturalLanguage    bool `yaml:"natural_language"`
	CalendarSync       bool `yaml:"calendar_sync"`
	WeatherIntegration bool `yaml:"weather_integration"`
}

func getConfigPath() string {
	if globalConfig.ConfigPath != "" {
		return globalConfig.ConfigPath
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".hereandnow/config.yaml"
	}

	return filepath.Join(homeDir, ".hereandnow", "config.yaml")
}

func LoadConfig() (*Config, error) {
	configPath := getConfigPath()
	
	// If config doesn't exist, return default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return GetDefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Expand paths
	config.Database.Path = expandPath(config.Database.Path)
	config.Logging.Path = expandPath(config.Logging.Path)

	return &config, nil
}

func SaveConfig(config *Config) error {
	configPath := getConfigPath()
	configDir := filepath.Dir(configPath)

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func GetDefaultConfig() *Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	baseDir := filepath.Join(homeDir, ".hereandnow")

	return &Config{
		Server: ServerConfig{
			Host: "127.0.0.1",
			Port: 8080,
		},
		Database: DatabaseConfig{
			Path: filepath.Join(baseDir, "data.db"),
		},
		Logging: LoggingConfig{
			Level: "info",
			Path:  filepath.Join(baseDir, "logs"),
		},
		Features: FeaturesConfig{
			NaturalLanguage:    true,
			CalendarSync:       false,
			WeatherIntegration: false,
		},
	}
}

func expandPath(path string) string {
	if path == "" {
		return path
	}

	// Expand ~ to home directory
	if path[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		if len(path) == 1 {
			return homeDir
		}
		return filepath.Join(homeDir, path[1:])
	}

	// Make relative paths absolute
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return path
		}
		return absPath
	}

	return path
}

func InitDatabase(dbPath string) (*sql.DB, error) {
	// Ensure directory exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create tables if they don't exist
	if err := createTables(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return db, nil
}

func createTables(db *sql.DB) error {
	schema := `
	-- Users table
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		salt TEXT NOT NULL,
		is_admin BOOLEAN DEFAULT 0,
		timezone TEXT DEFAULT 'UTC',
		preferences TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Tasks table
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		description TEXT,
		creator_id TEXT NOT NULL REFERENCES users(id),
		assignee_id TEXT REFERENCES users(id),
		list_id TEXT REFERENCES task_lists(id),
		status TEXT NOT NULL DEFAULT 'pending',
		priority INTEGER DEFAULT 3,
		estimated_minutes INTEGER,
		due_at DATETIME,
		completed_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		metadata TEXT,
		recurrence_rule TEXT,
		parent_task_id TEXT REFERENCES tasks(id)
	);

	-- Task Lists table
	CREATE TABLE IF NOT EXISTS task_lists (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		owner_id TEXT NOT NULL REFERENCES users(id),
		is_shared BOOLEAN DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Locations table
	CREATE TABLE IF NOT EXISTS locations (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		latitude REAL NOT NULL,
		longitude REAL NOT NULL,
		radius INTEGER NOT NULL DEFAULT 100,
		user_id TEXT NOT NULL REFERENCES users(id),
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Task Dependencies table
	CREATE TABLE IF NOT EXISTS task_dependencies (
		id TEXT PRIMARY KEY,
		task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		depends_on_task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		dependency_type TEXT NOT NULL DEFAULT 'blocks',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Task Locations table (many-to-many)
	CREATE TABLE IF NOT EXISTS task_locations (
		id TEXT PRIMARY KEY,
		task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		location_id TEXT NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Context table
	CREATE TABLE IF NOT EXISTS contexts (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id),
		timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		current_latitude REAL,
		current_longitude REAL,
		current_location_id TEXT REFERENCES locations(id),
		available_minutes INTEGER DEFAULT 0,
		social_context TEXT DEFAULT 'alone',
		energy_level INTEGER DEFAULT 3,
		weather_condition TEXT,
		traffic_level TEXT,
		metadata TEXT
	);

	-- Calendar Events table
	CREATE TABLE IF NOT EXISTS calendar_events (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id),
		title TEXT NOT NULL,
		description TEXT,
		start_at DATETIME NOT NULL,
		end_at DATETIME NOT NULL,
		location TEXT,
		all_day BOOLEAN DEFAULT 0,
		calendar_source TEXT,
		external_id TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- List Members table
	CREATE TABLE IF NOT EXISTS list_members (
		id TEXT PRIMARY KEY,
		list_id TEXT NOT NULL REFERENCES task_lists(id) ON DELETE CASCADE,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		role TEXT NOT NULL DEFAULT 'viewer',
		joined_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Task Assignments table
	CREATE TABLE IF NOT EXISTS task_assignments (
		id TEXT PRIMARY KEY,
		task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		assigner_id TEXT NOT NULL REFERENCES users(id),
		assignee_id TEXT NOT NULL REFERENCES users(id),
		assigned_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		accepted_at DATETIME,
		status TEXT DEFAULT 'pending'
	);

	-- Filter Audit table
	CREATE TABLE IF NOT EXISTS filter_audit (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id),
		task_id TEXT NOT NULL REFERENCES tasks(id),
		context_id TEXT NOT NULL REFERENCES contexts(id),
		filter_name TEXT NOT NULL,
		filter_result TEXT NOT NULL,
		reason TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Analytics table
	CREATE TABLE IF NOT EXISTS analytics (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id),
		metric_name TEXT NOT NULL,
		metric_value REAL NOT NULL,
		metadata TEXT,
		recorded_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Create indexes for performance
	CREATE INDEX IF NOT EXISTS idx_tasks_creator_id ON tasks(creator_id);
	CREATE INDEX IF NOT EXISTS idx_tasks_assignee_id ON tasks(assignee_id);
	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_contexts_user_id ON contexts(user_id);
	CREATE INDEX IF NOT EXISTS idx_contexts_timestamp ON contexts(timestamp);
	CREATE INDEX IF NOT EXISTS idx_locations_user_id ON locations(user_id);
	CREATE INDEX IF NOT EXISTS idx_calendar_events_user_id ON calendar_events(user_id);
	CREATE INDEX IF NOT EXISTS idx_calendar_events_start_at ON calendar_events(start_at);
	CREATE INDEX IF NOT EXISTS idx_filter_audit_user_id ON filter_audit(user_id);
	CREATE INDEX IF NOT EXISTS idx_filter_audit_task_id ON filter_audit(task_id);
	CREATE INDEX IF NOT EXISTS idx_analytics_user_id ON analytics(user_id);

	-- Create full-text search indexes
	CREATE VIRTUAL TABLE IF NOT EXISTS tasks_fts USING fts5(
		title, description, content='tasks', content_rowid='rowid'
	);

	-- Create triggers to keep FTS in sync
	CREATE TRIGGER IF NOT EXISTS tasks_fts_insert AFTER INSERT ON tasks BEGIN
		INSERT INTO tasks_fts(rowid, title, description) VALUES (new.rowid, new.title, new.description);
	END;

	CREATE TRIGGER IF NOT EXISTS tasks_fts_delete AFTER DELETE ON tasks BEGIN
		INSERT INTO tasks_fts(tasks_fts, rowid, title, description) VALUES('delete', old.rowid, old.title, old.description);
	END;

	CREATE TRIGGER IF NOT EXISTS tasks_fts_update AFTER UPDATE ON tasks BEGIN
		INSERT INTO tasks_fts(tasks_fts, rowid, title, description) VALUES('delete', old.rowid, old.title, old.description);
		INSERT INTO tasks_fts(rowid, title, description) VALUES (new.rowid, new.title, new.description);
	END;
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create database schema: %w", err)
	}

	return nil
}

func ValidateConfig(config *Config) error {
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	if config.Server.Host == "" {
		return fmt.Errorf("server host cannot be empty")
	}

	if config.Database.Path == "" {
		return fmt.Errorf("database path cannot be empty")
	}

	if config.Logging.Level != "debug" && config.Logging.Level != "info" && 
	   config.Logging.Level != "warn" && config.Logging.Level != "error" {
		return fmt.Errorf("invalid logging level: %s", config.Logging.Level)
	}

	return nil
}