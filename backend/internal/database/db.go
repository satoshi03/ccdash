package database

import (
	"database/sql"
	"fmt"

	"ccdash-backend/internal/config"
	"ccdash-backend/internal/services"

	_ "github.com/marcboeker/go-duckdb"
)

func Initialize() (*sql.DB, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	return InitializeWithConfig(cfg)
}

func InitializeWithConfig(cfg *config.Config) (*sql.DB, error) {
	if err := cfg.EnsureDatabaseDir(); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("duckdb", cfg.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	// Initialize differential sync schema
	stateManager := services.NewFileSyncStateManager(db)
	if err := stateManager.InitializeSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize sync schema: %w", err)
	}

	return db, nil
}

func createTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS sessions (
			id VARCHAR PRIMARY KEY,
			project_name VARCHAR NOT NULL,
			project_path VARCHAR NOT NULL,
			start_time TIMESTAMP NOT NULL,
			end_time TIMESTAMP,
			total_input_tokens INTEGER DEFAULT 0,
			total_output_tokens INTEGER DEFAULT 0,
			total_tokens INTEGER DEFAULT 0,
			message_count INTEGER DEFAULT 0,
			total_cost DOUBLE DEFAULT 0.0,
			status VARCHAR DEFAULT 'active',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS messages (
			id VARCHAR PRIMARY KEY,
			session_id VARCHAR NOT NULL,
			parent_uuid VARCHAR,
			is_sidechain BOOLEAN DEFAULT false,
			user_type VARCHAR,
			message_type VARCHAR,
			message_role VARCHAR,
			model VARCHAR,
			content TEXT,
			input_tokens INTEGER DEFAULT 0,
			cache_creation_input_tokens INTEGER DEFAULT 0,
			cache_read_input_tokens INTEGER DEFAULT 0,
			output_tokens INTEGER DEFAULT 0,
			service_tier VARCHAR,
			request_id VARCHAR,
			timestamp TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (session_id) REFERENCES sessions (id)
		)`,

		`CREATE TABLE IF NOT EXISTS session_windows (
			id TEXT PRIMARY KEY,
			window_start TIMESTAMP NOT NULL,
			window_end TIMESTAMP NOT NULL,
			reset_time TIMESTAMP NOT NULL,
			total_input_tokens INTEGER DEFAULT 0,
			total_output_tokens INTEGER DEFAULT 0,
			total_tokens INTEGER DEFAULT 0,
			message_count INTEGER DEFAULT 0,
			session_count INTEGER DEFAULT 0,
			total_cost DOUBLE DEFAULT 0.0,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS session_window_messages (
			id VARCHAR PRIMARY KEY,
			session_window_id TEXT NOT NULL,
			message_id VARCHAR NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (session_window_id) REFERENCES session_windows (id),
			FOREIGN KEY (message_id) REFERENCES messages (id),
			UNIQUE(session_window_id, message_id)
		)`,

		// Add total_cost column to existing sessions table if it doesn't exist
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS total_cost DOUBLE DEFAULT 0.0`,

		// Add project_id column to sessions table for Project integration (Phase 2)
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS project_id VARCHAR`,

		// Add total_cost column to existing session_windows table if it doesn't exist
		`ALTER TABLE session_windows ADD COLUMN IF NOT EXISTS total_cost DOUBLE DEFAULT 0.0`,

		`CREATE INDEX IF NOT EXISTS idx_sessions_project_name ON sessions (project_name)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_project_id ON sessions (project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_start_time ON sessions (start_time)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions (status)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_session_id ON messages (session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages (timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_message_role ON messages (message_role)`,

		`CREATE INDEX IF NOT EXISTS idx_session_windows_times ON session_windows(window_start, window_end)`,
		`CREATE INDEX IF NOT EXISTS idx_session_windows_active ON session_windows(is_active)`,
		`CREATE INDEX IF NOT EXISTS idx_session_windows_reset_time ON session_windows(reset_time)`,

		`CREATE INDEX IF NOT EXISTS idx_session_window_messages_window_id ON session_window_messages(session_window_id)`,
		`CREATE INDEX IF NOT EXISTS idx_session_window_messages_message_id ON session_window_messages(message_id)`,

		// Projects table
		`CREATE TABLE IF NOT EXISTS projects (
			id VARCHAR PRIMARY KEY,
			name VARCHAR NOT NULL,
			path VARCHAR NOT NULL,
			description TEXT,
			repository_url VARCHAR,
			language VARCHAR,
			framework VARCHAR,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(name, path)
		)`,

		// Projects table indexes
		`CREATE INDEX IF NOT EXISTS idx_projects_name ON projects (name)`,
		`CREATE INDEX IF NOT EXISTS idx_projects_active ON projects (is_active)`,
		`CREATE INDEX IF NOT EXISTS idx_projects_path ON projects (path)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %s, error: %w", query, err)
		}
	}

	return nil
}
