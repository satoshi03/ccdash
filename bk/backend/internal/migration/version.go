package migration

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Version represents a migration version
type Version struct {
	Version       string
	Name          string
	UpScript      string
	DownScript    string
	AppliedAt     *time.Time
	ExecutionTime *int
	Status        string
	ErrorMessage  *string
}

// VersionManager manages migration versions
type VersionManager struct {
	db *sql.DB
}

// NewVersionManager creates a new version manager
func NewVersionManager(db *sql.DB) *VersionManager {
	return &VersionManager{db: db}
}

// CreateTables creates the migration tracking tables
func (vm *VersionManager) CreateTables() error {
	tx, err := vm.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create migration_history table
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS migration_history (
			id INTEGER PRIMARY KEY,
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

	// Create schema_version table
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

// GetAppliedVersions returns all applied migrations
func (vm *VersionManager) GetAppliedVersions() ([]Version, error) {
	rows, err := vm.db.Query(`
		SELECT version, name, applied_at, execution_time_ms, status, error_message, up_script, down_script
		FROM migration_history
		ORDER BY version ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query migration history: %w", err)
	}
	defer rows.Close()

	var versions []Version
	for rows.Next() {
		var v Version
		err := rows.Scan(&v.Version, &v.Name, &v.AppliedAt, &v.ExecutionTime, &v.Status, &v.ErrorMessage, &v.UpScript, &v.DownScript)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		versions = append(versions, v)
	}

	return versions, nil
}

// GetCurrentVersion returns the current schema version
func (vm *VersionManager) GetCurrentVersion() (string, bool, error) {
	var version string
	var dirty bool
	err := vm.db.QueryRow("SELECT version, dirty FROM schema_version ORDER BY updated_at DESC LIMIT 1").Scan(&version, &dirty)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("failed to get current version: %w", err)
	}
	return version, dirty, nil
}

// SetVersion sets the current schema version
func (vm *VersionManager) SetVersion(version string, dirty bool) error {
	_, err := vm.db.Exec(`
		INSERT INTO schema_version (version, dirty, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`, version, dirty)
	if err != nil {
		return fmt.Errorf("failed to set version: %w", err)
	}
	return nil
}

// RecordMigration records a migration execution
func (vm *VersionManager) RecordMigration(version, name, upScript, downScript string, executionTime int, status string, errorMsg *string) error {
	_, err := vm.db.Exec(`
		INSERT INTO migration_history (version, name, up_script, down_script, execution_time_ms, status, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, version, name, upScript, downScript, executionTime, status, errorMsg)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}
	return nil
}

// IsApplied checks if a migration version has been applied
func (vm *VersionManager) IsApplied(version string) (bool, error) {
	var count int
	err := vm.db.QueryRow("SELECT COUNT(*) FROM migration_history WHERE version = ? AND status = 'success'", version).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if migration is applied: %w", err)
	}
	return count > 0, nil
}

// ParseVersionFromFilename extracts version from migration filename
func ParseVersionFromFilename(filename string) (string, string, error) {
	// Pattern: {timestamp}_{description}.{up|down}.sql
	re := regexp.MustCompile(`^(\d{14})_([^.]+)\.(up|down)\.sql$`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) != 4 {
		return "", "", fmt.Errorf("invalid migration filename format: %s", filename)
	}
	return matches[1], strings.ReplaceAll(matches[2], "_", " "), nil
}