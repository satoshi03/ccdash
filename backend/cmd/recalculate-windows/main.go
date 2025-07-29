package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"ccdash-backend/internal/services"

	_ "github.com/marcboeker/go-duckdb"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--help" {
		fmt.Println("Usage: recalculate-windows")
		fmt.Println("Recalculates all session windows based on existing messages.")
		fmt.Println("This clears existing session windows and recreates them using the proper algorithm.")
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
		return
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Check if there are messages to process
	var messageCount int
	err = db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&messageCount)
	if err != nil {
		fmt.Printf("Error counting messages: %v\n", err)
		os.Exit(1)
	}

	if messageCount == 0 {
		fmt.Println("No messages found. Nothing to recalculate.")
		return
	}

	fmt.Printf("Found %d messages. Recalculating session windows...\n", messageCount)

	// Use the service to recalculate windows
	windowService := services.NewSessionWindowService(db)
	err = windowService.RecalculateAllWindows()
	if err != nil {
		fmt.Printf("Error recalculating windows: %v\n", err)
		os.Exit(1)
	}

	// Check results
	var windowCount int
	err = db.QueryRow("SELECT COUNT(*) FROM session_windows").Scan(&windowCount)
	if err != nil {
		fmt.Printf("Error counting windows: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully created %d session windows\n", windowCount)
}
