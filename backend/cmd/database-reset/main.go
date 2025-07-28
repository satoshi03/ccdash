package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	dbDir := filepath.Join(homeDir, ".ccdash")
	
	// Remove all database files
	err = os.RemoveAll(dbDir)
	if err != nil {
		fmt.Printf("Error removing database directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Database reset completed. All database files have been removed.")
	fmt.Printf("Database directory: %s\n", dbDir)
	fmt.Println("The database will be recreated when the server next starts.")
}