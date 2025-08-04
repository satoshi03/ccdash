package migration

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
)

// Engine is the main migration engine
type Engine struct {
	db       *sql.DB
	scanner  *Scanner
	executor *Executor
	vm       *VersionManager
}

// NewEngine creates a new migration engine from a directory
func NewEngine(db *sql.DB, dir string) (*Engine, error) {
	vm := NewVersionManager(db)
	
	// Create migration tables
	if err := vm.CreateTables(); err != nil {
		return nil, fmt.Errorf("failed to create migration tables: %w", err)
	}
	
	scanner := NewScanner(os.DirFS(dir))
	executor := NewExecutor(db, vm)
	
	return &Engine{
		db:       db,
		scanner:  scanner,
		executor: executor,
		vm:       vm,
	}, nil
}

// NewEngineWithEmbed creates a new migration engine with embedded files
func NewEngineWithEmbed(db *sql.DB, embedFS embed.FS) (*Engine, error) {
	vm := NewVersionManager(db)
	
	// Create migration tables
	if err := vm.CreateTables(); err != nil {
		return nil, fmt.Errorf("failed to create migration tables: %w", err)
	}
	
	scanner := NewEmbedScanner(embedFS)
	executor := NewExecutor(db, vm)
	
	return &Engine{
		db:       db,
		scanner:  scanner,
		executor: executor,
		vm:       vm,
	}, nil
}

// Up runs all pending migrations
func (e *Engine) Up() error {
	pending, err := e.scanner.ScanPending(e.vm)
	if err != nil {
		return fmt.Errorf("failed to scan pending migrations: %w", err)
	}
	
	if len(pending) == 0 {
		log.Println("No pending migrations")
		return nil
	}
	
	log.Printf("Found %d pending migrations", len(pending))
	
	for _, migration := range pending {
		log.Printf("Applying migration %s: %s", migration.Version, migration.Name)
		if err := e.executor.ExecuteUp(migration); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migration.Version, err)
		}
		log.Printf("Successfully applied migration %s", migration.Version)
	}
	
	return nil
}

// Down rolls back the last migration
func (e *Engine) Down() error {
	// Get the last applied migration
	applied, err := e.vm.GetAppliedVersions()
	if err != nil {
		return fmt.Errorf("failed to get applied versions: %w", err)
	}
	
	if len(applied) == 0 {
		log.Println("No migrations to roll back")
		return nil
	}
	
	// Get the last successful migration
	var lastMigration *Version
	for i := len(applied) - 1; i >= 0; i-- {
		if applied[i].Status == "success" {
			lastMigration = &applied[i]
			break
		}
	}
	
	if lastMigration == nil {
		log.Println("No successful migrations to roll back")
		return nil
	}
	
	// Find the migration pair
	allMigrations, err := e.scanner.Scan()
	if err != nil {
		return fmt.Errorf("failed to scan migrations: %w", err)
	}
	
	var migrationPair *MigrationPair
	for _, m := range allMigrations {
		if m.Version == lastMigration.Version {
			migrationPair = &m
			break
		}
	}
	
	if migrationPair == nil {
		return fmt.Errorf("migration file not found for version %s", lastMigration.Version)
	}
	
	log.Printf("Rolling back migration %s: %s", migrationPair.Version, migrationPair.Name)
	if err := e.executor.ExecuteDown(*migrationPair); err != nil {
		return fmt.Errorf("failed to roll back migration %s: %w", migrationPair.Version, err)
	}
	log.Printf("Successfully rolled back migration %s", migrationPair.Version)
	
	return nil
}

// Status returns the current migration status
func (e *Engine) Status() (*Status, error) {
	currentVersion, dirty, err := e.vm.GetCurrentVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get current version: %w", err)
	}
	
	applied, err := e.vm.GetAppliedVersions()
	if err != nil {
		return nil, fmt.Errorf("failed to get applied versions: %w", err)
	}
	
	pending, err := e.scanner.ScanPending(e.vm)
	if err != nil {
		return nil, fmt.Errorf("failed to scan pending migrations: %w", err)
	}
	
	return &Status{
		CurrentVersion: currentVersion,
		Dirty:          dirty,
		Applied:        applied,
		Pending:        pending,
	}, nil
}

// Status represents the migration status
type Status struct {
	CurrentVersion string
	Dirty          bool
	Applied        []Version
	Pending        []MigrationPair
}