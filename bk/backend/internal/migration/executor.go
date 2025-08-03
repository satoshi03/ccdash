package migration

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Executor executes migrations
type Executor struct {
	db *sql.DB
	vm *VersionManager
}

// NewExecutor creates a new migration executor
func NewExecutor(db *sql.DB, vm *VersionManager) *Executor {
	return &Executor{
		db: db,
		vm: vm,
	}
}

// ExecuteUp executes an up migration
func (e *Executor) ExecuteUp(migration MigrationPair) error {
	if migration.Up == nil {
		return fmt.Errorf("no up migration found for version %s", migration.Version)
	}
	
	// Check if already applied
	applied, err := e.vm.IsApplied(migration.Version)
	if err != nil {
		return err
	}
	if applied {
		return fmt.Errorf("migration %s already applied", migration.Version)
	}
	
	// Start timing
	start := time.Now()
	
	// Mark as dirty
	if err := e.vm.SetVersion(migration.Version, true); err != nil {
		return fmt.Errorf("failed to set dirty flag: %w", err)
	}
	
	// Execute in transaction
	tx, err := e.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	// Execute migration
	statements := splitStatements(migration.Up.Content)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		
		if _, err := tx.Exec(stmt); err != nil {
			tx.Rollback()
			execTime := int(time.Since(start).Milliseconds())
			errMsg := err.Error()
			e.vm.RecordMigration(migration.Version, migration.Name, migration.Up.Content, getDownContent(migration), execTime, "failed", &errMsg)
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		execTime := int(time.Since(start).Milliseconds())
		errMsg := err.Error()
		e.vm.RecordMigration(migration.Version, migration.Name, migration.Up.Content, getDownContent(migration), execTime, "failed", &errMsg)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	// Record success
	execTime := int(time.Since(start).Milliseconds())
	if err := e.vm.RecordMigration(migration.Version, migration.Name, migration.Up.Content, getDownContent(migration), execTime, "success", nil); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}
	
	// Clear dirty flag
	if err := e.vm.SetVersion(migration.Version, false); err != nil {
		return fmt.Errorf("failed to clear dirty flag: %w", err)
	}
	
	return nil
}

// ExecuteDown executes a down migration
func (e *Executor) ExecuteDown(migration MigrationPair) error {
	if migration.Down == nil {
		return fmt.Errorf("no down migration found for version %s", migration.Version)
	}
	
	// Check if applied
	applied, err := e.vm.IsApplied(migration.Version)
	if err != nil {
		return err
	}
	if !applied {
		return fmt.Errorf("migration %s not applied", migration.Version)
	}
	
	// Start timing
	start := time.Now()
	
	// Mark as dirty
	prevVersion := getPreviousVersion(migration.Version)
	if err := e.vm.SetVersion(prevVersion, true); err != nil {
		return fmt.Errorf("failed to set dirty flag: %w", err)
	}
	
	// Execute in transaction
	tx, err := e.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	// Execute migration
	statements := splitStatements(migration.Down.Content)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		
		if _, err := tx.Exec(stmt); err != nil {
			tx.Rollback()
			return fmt.Errorf("rollback failed: %w", err)
		}
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	// Update migration history
	execTime := int(time.Since(start).Milliseconds())
	_, err = e.db.Exec(`
		UPDATE migration_history 
		SET status = 'rolled_back', 
		    execution_time_ms = ?,
		    applied_at = CURRENT_TIMESTAMP
		WHERE version = ?
	`, execTime, migration.Version)
	if err != nil {
		return fmt.Errorf("failed to update migration history: %w", err)
	}
	
	// Clear dirty flag
	if err := e.vm.SetVersion(prevVersion, false); err != nil {
		return fmt.Errorf("failed to clear dirty flag: %w", err)
	}
	
	return nil
}

// splitStatements splits SQL content into individual statements
func splitStatements(content string) []string {
	// Simple split by semicolon - can be enhanced for more complex cases
	statements := strings.Split(content, ";")
	var result []string
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" {
			result = append(result, stmt)
		}
	}
	return result
}

// getDownContent gets the down migration content if available
func getDownContent(migration MigrationPair) string {
	if migration.Down != nil {
		return migration.Down.Content
	}
	return ""
}

// getPreviousVersion calculates the previous version (simplified)
func getPreviousVersion(version string) string {
	// This is a simplified implementation
	// In a real system, you'd query the previous applied migration
	return "0"
}