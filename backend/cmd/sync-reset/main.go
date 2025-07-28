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
		fmt.Println("Usage: sync-reset")
		fmt.Println("Resets file synchronization state to force complete resync of all JSONL files.")
		fmt.Println("Use this when you want to reprocess all log files from scratch.")
		return
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	dbPath := filepath.Join(homeDir, ".ccdash", "ccdash.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Println("Database does not exist. No sync states to reset.")
		return
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Clear all sync states
	result, err := db.Exec("DELETE FROM file_sync_state")
	if err != nil {
		fmt.Printf("Error clearing sync states: %v\n", err)
		os.Exit(1)
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Cleared %d file sync states\n", rowsAffected)
	fmt.Println("Next log sync will process all files completely from the beginning.")
}