package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/marcboeker/go-duckdb"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--help" {
		fmt.Println("Usage: database-status")
		fmt.Println("Shows current database status including counts of sessions, messages, and session windows.")
		return
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	dbPath := filepath.Join(homeDir, ".ccdash", "ccdash.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Println("Database does not exist.")
		fmt.Printf("Expected location: %s\n", dbPath)
		return
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Printf("Database: %s\n\n", dbPath)

	// Check main tables
	tables := []struct {
		name    string
		query   string
	}{
		{"Sessions", "SELECT COUNT(*) FROM sessions"},
		{"Messages", "SELECT COUNT(*) FROM messages"},
		{"Session Windows", "SELECT COUNT(*) FROM session_windows"},
		{"File Sync States", "SELECT COUNT(*) FROM file_sync_state"},
	}

	for _, table := range tables {
		var count int
		err := db.QueryRow(table.query).Scan(&count)
		if err != nil {
			fmt.Printf("%-15s: Error - %v\n", table.name, err)
		} else {
			fmt.Printf("%-15s: %d\n", table.name, count)
		}
	}

	// Check token usage
	var totalTokens int
	err = db.QueryRow(`
		SELECT COALESCE(SUM(input_tokens + output_tokens), 0) 
		FROM messages
	`).Scan(&totalTokens)
	if err != nil {
		fmt.Printf("%-15s: Error - %v\n", "Total Tokens", err)
	} else {
		fmt.Printf("%-15s: %d\n", "Total Tokens", totalTokens)
	}

	// Check recent activity
	var recentMessages int
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM messages 
		WHERE timestamp >= (CURRENT_TIMESTAMP::TIMESTAMP - INTERVAL 24 HOUR)
	`).Scan(&recentMessages)
	if err != nil {
		fmt.Printf("%-15s: Error - %v\n", "Recent (24h)", err)
	} else {
		fmt.Printf("%-15s: %d messages\n", "Recent (24h)", recentMessages)
	}
}