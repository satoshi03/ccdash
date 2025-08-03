package database

import (
	"ccdash-backend/internal/migration"
	"database/sql"
	"embed"
	"fmt"
)

// MigrationsFS is embedded from the migrations package
var MigrationsFS embed.FS

// InitializeMigrationEngine creates and returns a migration engine
func InitializeMigrationEngine(db *sql.DB, migrationsFS embed.FS) (*migration.Engine, error) {
	engine, err := migration.NewEngineWithEmbed(db, migrationsFS)
	if err != nil {
		return nil, fmt.Errorf("failed to create migration engine: %w", err)
	}
	
	return engine, nil
}

// RunMigrations runs all pending migrations
func RunMigrations(db *sql.DB, migrationsFS embed.FS) error {
	engine, err := InitializeMigrationEngine(db, migrationsFS)
	if err != nil {
		return err
	}
	
	return engine.Up()
}