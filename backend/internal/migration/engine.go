package migration

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"os"
)

// Engine is the main migration engine
type Engine struct {
	db       *sql.DB
	scanner  *FileScanner
	executor *Executor
	version  *VersionManager
	logger   *log.Logger
}

// NewEngine creates a new migration engine
func NewEngine(db *sql.DB, fsys fs.FS) (*Engine, error) {
	logger := log.New(os.Stdout, "[migration] ", log.LstdFlags)
	
	version := NewVersionManager(db)
	if err := version.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize version manager: %w", err)
	}
	
	scanner := NewFileScanner(fsys)
	executor := NewExecutor(db, version, logger)
	
	return &Engine{
		db:       db,
		scanner:  scanner,
		executor: executor,
		version:  version,
		logger:   logger,
	}, nil
}

// Up runs all pending migrations
func (e *Engine) Up() error {
	// Get all available migrations
	migrations, err := e.scanner.ScanMigrations()
	if err != nil {
		return fmt.Errorf("failed to scan migrations: %w", err)
	}
	
	// Get applied migrations
	applied, err := e.version.GetAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}
	
	// Create a set of applied migrations for quick lookup
	appliedSet := make(map[string]bool)
	for _, version := range applied {
		appliedSet[version] = true
	}
	
	// Filter pending migrations
	var pending []Migration
	for _, m := range migrations {
		if !appliedSet[m.Version] {
			pending = append(pending, m)
		}
	}
	
	if len(pending) == 0 {
		e.logger.Println("No pending migrations")
		return nil
	}
	
	e.logger.Printf("Found %d pending migrations", len(pending))
	
	// Execute pending migrations
	for _, m := range pending {
		e.logger.Printf("Migrating %s: %s", m.Version, FormatName(m.Name))
		if err := e.executor.Execute(m, Up); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	
	e.logger.Println("All migrations completed successfully")
	return nil
}

// Down rolls back the last migration
func (e *Engine) Down() error {
	// Get current version
	currentVersion, err := e.version.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}
	
	if currentVersion == nil {
		e.logger.Println("No migrations to roll back")
		return nil
	}
	
	// Get all migrations
	migrations, err := e.scanner.ScanMigrations()
	if err != nil {
		return fmt.Errorf("failed to scan migrations: %w", err)
	}
	
	// Find the current migration
	var currentMigration *Migration
	for _, m := range migrations {
		if m.Version == currentVersion.Version {
			currentMigration = &m
			break
		}
	}
	
	if currentMigration == nil {
		return fmt.Errorf("migration %s not found", currentVersion.Version)
	}
	
	e.logger.Printf("Rolling back %s: %s", currentMigration.Version, FormatName(currentMigration.Name))
	
	// Execute down migration
	if err := e.executor.Execute(*currentMigration, Down); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}
	
	e.logger.Println("Rollback completed successfully")
	return nil
}

// Status returns the current migration status
func (e *Engine) Status() (*Status, error) {
	// Get current version
	currentVersion, err := e.version.GetCurrentVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get current version: %w", err)
	}
	
	// Get all migrations
	migrations, err := e.scanner.ScanMigrations()
	if err != nil {
		return nil, fmt.Errorf("failed to scan migrations: %w", err)
	}
	
	// Get applied migrations
	applied, err := e.version.GetAppliedMigrations()
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}
	
	status := &Status{
		Applied:        applied,
		Pending:        []Migration{},
		TotalAvailable: len(migrations),
	}
	
	if currentVersion != nil {
		status.CurrentVersion = currentVersion.Version
		status.Dirty = currentVersion.Dirty
	}
	
	// Create a set of applied migrations for quick lookup
	appliedSet := make(map[string]bool)
	for _, version := range applied {
		appliedSet[version] = true
	}
	
	// Find pending migrations
	for _, m := range migrations {
		if !appliedSet[m.Version] {
			status.Pending = append(status.Pending, m)
		}
	}
	
	return status, nil
}

// Status represents the migration status
type Status struct {
	CurrentVersion string
	Dirty          bool
	Applied        []string
	Pending        []Migration
	TotalAvailable int
}