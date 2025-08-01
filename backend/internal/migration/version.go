package migration

import (
	"database/sql"
	"fmt"
	"time"
)

// Version represents the current schema version
type Version struct {
	Version   string    `json:"version"`
	Dirty     bool      `json:"dirty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// VersionManager manages database schema versions
type VersionManager struct {
	db *sql.DB
}

// NewVersionManager creates a new version manager
func NewVersionManager(db *sql.DB) *VersionManager {
	return &VersionManager{db: db}
}

// Initialize creates the necessary tables for migration tracking
func (vm *VersionManager) Initialize() error {
	tx, err := vm.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Create sequence for migration history if not exists
	_, err = tx.Exec(`CREATE SEQUENCE IF NOT EXISTS migration_history_id_seq`)
	if err != nil {
		return fmt.Errorf("failed to create migration_history sequence: %w", err)
	}
	
	// Create migration history table
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS migration_history (
			id INTEGER PRIMARY KEY DEFAULT nextval('migration_history_id_seq'),
			version VARCHAR NOT NULL UNIQUE,
			name VARCHAR NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			execution_time_ms INTEGER,
			checksum VARCHAR,
			up_script TEXT,
			down_script TEXT,
			status VARCHAR DEFAULT 'success',
			error_message TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migration_history table: %w", err)
	}

	// Create schema version table
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS schema_version (
			version VARCHAR PRIMARY KEY,
			dirty BOOLEAN DEFAULT false,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_version table: %w", err)
	}

	return tx.Commit()
}

// GetCurrentVersion returns the current schema version
func (vm *VersionManager) GetCurrentVersion() (*Version, error) {
	var v Version
	err := vm.db.QueryRow(`
		SELECT version, dirty, updated_at 
		FROM schema_version 
		ORDER BY updated_at DESC 
		LIMIT 1
	`).Scan(&v.Version, &v.Dirty, &v.UpdatedAt)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get current version: %w", err)
	}
	
	return &v, nil
}

// SetVersion sets the current schema version
func (vm *VersionManager) SetVersion(version string, dirty bool) error {
	_, err := vm.db.Exec(`
		INSERT OR REPLACE INTO schema_version (version, dirty, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`, version, dirty)
	
	if err != nil {
		return fmt.Errorf("failed to set version: %w", err)
	}
	
	return nil
}

// GetAppliedMigrations returns a list of applied migrations
func (vm *VersionManager) GetAppliedMigrations() ([]string, error) {
	rows, err := vm.db.Query(`
		SELECT version 
		FROM migration_history 
		WHERE status = 'success'
		ORDER BY version ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	var versions []string
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}
		versions = append(versions, version)
	}

	return versions, nil
}

// RecordMigration records a migration execution
func (vm *VersionManager) RecordMigration(version, name, upScript, downScript, checksum string, executionTimeMs int64, status string, errorMessage string) error {
	_, err := vm.db.Exec(`
		INSERT INTO migration_history (version, name, up_script, down_script, checksum, execution_time_ms, status, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, version, name, upScript, downScript, checksum, executionTimeMs, status, errorMessage)
	
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}
	
	return nil
}