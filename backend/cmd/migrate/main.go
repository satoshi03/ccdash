package main

import (
	"ccdash-backend/internal/config"
	"ccdash-backend/internal/migration"
	"ccdash-backend/migrations"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"text/tabwriter"
	
	_ "github.com/marcboeker/go-duckdb"
)

func main() {
	var (
		configPath = flag.String("config", "", "Path to config file")
		dbPath     = flag.String("db", "", "Database path (overrides config)")
		port       = flag.Int("port", 0, "Server port (overrides config)")
		help       = flag.Bool("help", false, "Show help")
	)
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <command>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  up      Run all pending migrations\n")
		fmt.Fprintf(os.Stderr, "  down    Roll back the last migration\n")
		fmt.Fprintf(os.Stderr, "  status  Show migration status\n")
		fmt.Fprintf(os.Stderr, "  create  Create a new migration (not implemented yet)\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
	
	flag.Parse()
	
	if *help || flag.NArg() == 0 {
		flag.Usage()
		os.Exit(0)
	}
	
	command := flag.Arg(0)
	
	// Load config
	cfg, err := loadConfig(*configPath, *dbPath, *port)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	// Initialize database
	db, err := initDB(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	
	// Create migration engine
	engine, err := migration.NewEngineWithEmbed(db, migrations.FS)
	if err != nil {
		log.Fatalf("Failed to create migration engine: %v", err)
	}
	
	// Execute command
	switch command {
	case "up":
		if err := engine.Up(); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		fmt.Println("Migrations completed successfully")
		
	case "down":
		if err := engine.Down(); err != nil {
			log.Fatalf("Rollback failed: %v", err)
		}
		fmt.Println("Rollback completed successfully")
		
	case "status":
		status, err := engine.Status()
		if err != nil {
			log.Fatalf("Failed to get status: %v", err)
		}
		printStatus(status)
		
	case "create":
		fmt.Println("Create command not implemented yet")
		os.Exit(1)
		
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		flag.Usage()
		os.Exit(1)
	}
}

func loadConfig(configPath, dbPath string, port int) (*config.Config, error) {
	// For now, just use environment-based config
	// TODO: Implement config file loading if needed
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	
	// Override with command line flags
	if dbPath != "" {
		cfg.DatabasePath = dbPath
		cfg.DatabaseDir = filepath.Dir(dbPath)
	}
	if port != 0 {
		cfg.ServerPort = fmt.Sprintf("%d", port)
	}
	
	return cfg, nil
}

func initDB(cfg *config.Config) (*sql.DB, error) {
	if err := cfg.EnsureDatabaseDir(); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}
	
	db, err := sql.Open("duckdb", cfg.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	
	return db, nil
}

func printStatus(status *migration.Status) {
	fmt.Printf("Current Version: %s\n", status.CurrentVersion)
	if status.Dirty {
		fmt.Printf("WARNING: Database is in dirty state!\n")
	}
	fmt.Println()
	
	if len(status.Applied) > 0 {
		fmt.Println("Applied Migrations:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "VERSION\tNAME\tAPPLIED AT\tSTATUS")
		fmt.Fprintln(w, "-------\t----\t----------\t------")
		for _, v := range status.Applied {
			appliedAt := ""
			if v.AppliedAt != nil {
				appliedAt = v.AppliedAt.Format("2006-01-02 15:04:05")
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", v.Version, v.Name, appliedAt, v.Status)
		}
		w.Flush()
		fmt.Println()
	}
	
	if len(status.Pending) > 0 {
		fmt.Println("Pending Migrations:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "VERSION\tNAME")
		fmt.Fprintln(w, "-------\t----")
		for _, p := range status.Pending {
			fmt.Fprintf(w, "%s\t%s\n", p.Version, p.Name)
		}
		w.Flush()
	} else {
		fmt.Println("No pending migrations")
	}
}