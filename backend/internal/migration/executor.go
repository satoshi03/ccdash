package migration

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

// Executor executes migrations
type Executor struct {
	db      *sql.DB
	version *VersionManager
	logger  *log.Logger
}

// NewExecutor creates a new migration executor
func NewExecutor(db *sql.DB, version *VersionManager, logger *log.Logger) *Executor {
	return &Executor{
		db:      db,
		version: version,
		logger:  logger,
	}
}

// Execute runs a migration in the specified direction
func (e *Executor) Execute(migration Migration, direction Direction) error {
	startTime := time.Now()
	
	script := migration.UpScript
	if direction == Down {
		script = migration.DownScript
	}
	
	if script == "" {
		return fmt.Errorf("no %s script for migration %s", direction, migration.Version)
	}
	
	// Start transaction
	tx, err := e.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	// Set version as dirty
	if err := e.version.SetVersion(migration.Version, true); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to set dirty version: %w", err)
	}
	
	// Execute migration
	statements := splitStatements(script)
	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		
		e.logger.Printf("Executing statement %d/%d for migration %s %s", i+1, len(statements), migration.Version, direction)
		
		if _, err := tx.Exec(stmt); err != nil {
			tx.Rollback()
			errMsg := fmt.Sprintf("failed to execute statement %d: %v", i+1, err)
			
			// Record failed migration
			e.version.RecordMigration(
				migration.Version,
				migration.Name,
				migration.UpScript,
				migration.DownScript,
				migration.Checksum,
				time.Since(startTime).Milliseconds(),
				"failed",
				errMsg,
			)
			
			return fmt.Errorf("%s", errMsg)
		}
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	// Record successful migration
	status := "success"
	if direction == Up {
		err = e.version.RecordMigration(
			migration.Version,
			migration.Name,
			migration.UpScript,
			migration.DownScript,
			migration.Checksum,
			time.Since(startTime).Milliseconds(),
			status,
			"",
		)
	}
	
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}
	
	// Set version as clean
	if err := e.version.SetVersion(migration.Version, false); err != nil {
		return fmt.Errorf("failed to set clean version: %w", err)
	}
	
	e.logger.Printf("Successfully executed migration %s %s in %v", migration.Version, direction, time.Since(startTime))
	
	return nil
}

// ExecuteBatch executes multiple migrations
func (e *Executor) ExecuteBatch(migrations []Migration, direction Direction) error {
	for _, migration := range migrations {
		if err := e.Execute(migration, direction); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", migration.Version, err)
		}
	}
	return nil
}

// splitStatements splits SQL script into individual statements
func splitStatements(script string) []string {
	// Simple implementation - split by semicolon
	// In production, this should handle:
	// - Semicolons within strings
	// - Stored procedures/functions
	// - Comments
	statements := strings.Split(script, ";")
	
	// Filter out empty statements
	var result []string
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" {
			result = append(result, stmt)
		}
	}
	
	return result
}

// ValidateMigration checks if a migration can be safely executed
func (e *Executor) ValidateMigration(migration Migration, direction Direction) error {
	script := migration.UpScript
	if direction == Down {
		script = migration.DownScript
	}
	
	if script == "" {
		return fmt.Errorf("no %s script for migration %s", direction, migration.Version)
	}
	
	// Basic validation - check for dangerous operations
	dangerousKeywords := []string{
		"DROP DATABASE",
		"TRUNCATE",
	}
	
	upperScript := strings.ToUpper(script)
	for _, keyword := range dangerousKeywords {
		if strings.Contains(upperScript, keyword) {
			return fmt.Errorf("migration contains dangerous operation: %s", keyword)
		}
	}
	
	return nil
}