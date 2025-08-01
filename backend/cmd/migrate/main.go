package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"text/tabwriter"

	"ccdash-backend/internal/migration"
	migrations "ccdash-backend/migrations"
	_ "github.com/marcboeker/go-duckdb"
)

func main() {
	var (
		dbPath  = flag.String("db", "", "Database path (default: ~/.ccdash/ccdash.db)")
		command = flag.String("cmd", "status", "Command to run: up, down, status")
		help    = flag.Bool("help", false, "Show help")
	)
	
	flag.Parse()
	
	if *help {
		showHelp()
		return
	}
	
	// Determine database path
	if *dbPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("Failed to get home directory:", err)
		}
		*dbPath = filepath.Join(homeDir, ".ccdash", "ccdash.db")
	}
	
	// Ensure database directory exists
	dbDir := filepath.Dir(*dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatal("Failed to create database directory:", err)
	}
	
	// Connect to database
	db, err := sql.Open("duckdb", *dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()
	
	// Create migration engine with embedded migrations
	engine, err := migration.NewEngine(db, migrations.FS)
	if err != nil {
		log.Fatal("Failed to create migration engine:", err)
	}
	
	// Execute command
	switch *command {
	case "up":
		if err := runUp(engine); err != nil {
			log.Fatal("Migration failed:", err)
		}
	case "down":
		if err := runDown(engine); err != nil {
			log.Fatal("Rollback failed:", err)
		}
	case "status":
		if err := showStatus(engine); err != nil {
			log.Fatal("Failed to get status:", err)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", *command)
		showHelp()
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Println(`CCDash Migration Tool

Usage:
  migrate [options]

Options:
  -cmd string
        Command to run: up, down, status (default "status")
  -db string
        Database path (default: ~/.ccdash/ccdash.db)
  -help
        Show help

Commands:
  up      Run all pending migrations
  down    Roll back the last migration
  status  Show migration status

Examples:
  # Show current migration status
  migrate -cmd status
  
  # Run all pending migrations
  migrate -cmd up
  
  # Roll back the last migration
  migrate -cmd down
  
  # Use a custom database path
  migrate -db /path/to/db.db -cmd status`)
}

func runUp(engine *migration.Engine) error {
	fmt.Println("Running pending migrations...")
	return engine.Up()
}

func runDown(engine *migration.Engine) error {
	fmt.Println("Rolling back last migration...")
	return engine.Down()
}

func showStatus(engine *migration.Engine) error {
	status, err := engine.Status()
	if err != nil {
		return err
	}
	
	fmt.Println("Migration Status:")
	fmt.Println("=================")
	
	if status.CurrentVersion == "" {
		fmt.Println("Current version: (none)")
	} else {
		fmt.Printf("Current version: %s", status.CurrentVersion)
		if status.Dirty {
			fmt.Print(" (dirty)")
		}
		fmt.Println()
	}
	
	fmt.Printf("Applied migrations: %d\n", len(status.Applied))
	fmt.Printf("Pending migrations: %d\n", len(status.Pending))
	fmt.Printf("Total available: %d\n", status.TotalAvailable)
	
	if len(status.Pending) > 0 {
		fmt.Println("\nPending Migrations:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "VERSION\tNAME\tCHECKSUM")
		fmt.Fprintln(w, "-------\t----\t--------")
		
		for _, m := range status.Pending {
			fmt.Fprintf(w, "%s\t%s\t%s\n", m.Version, migration.FormatName(m.Name), m.Checksum[:8])
		}
		w.Flush()
	}
	
	return nil
}