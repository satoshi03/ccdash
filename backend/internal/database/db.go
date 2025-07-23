package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	
	"claudeee-backend/internal/services"
	_ "github.com/marcboeker/go-duckdb"
)

func Initialize() (*sql.DB, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	
	dbPath := filepath.Join(homeDir, ".claudeee", "claudeee.db")
	
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}
	
	db, err := sql.Open("duckdb", dbPath)
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
		
		`CREATE INDEX IF NOT EXISTS idx_sessions_project_name ON sessions (project_name)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_start_time ON sessions (start_time)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions (status)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_session_id ON messages (session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages (timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_message_role ON messages (message_role)`,
	}
	
	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %s, error: %w", query, err)
		}
	}
	
	return nil
}