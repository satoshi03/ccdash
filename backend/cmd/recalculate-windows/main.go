package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/marcboeker/go-duckdb"
	"claudeee-backend/internal/services"
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

	dbPath := filepath.Join(homeDir, ".claudeee", "claudeee.db")
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

	// Clear existing session windows
	_, err = db.Exec("DELETE FROM session_windows")
	if err != nil {
		fmt.Printf("Error clearing session windows: %v\n", err)
		os.Exit(1)
	}

	// Clear session_window_id from messages
	_, err = db.Exec("UPDATE messages SET session_window_id = NULL")
	if err != nil {
		fmt.Printf("Error clearing message window references: %v\n", err)
		os.Exit(1)
	}

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