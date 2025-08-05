package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/marcboeker/go-duckdb"
)

func main() {
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Failed to get home directory:", err)
	}

	// Open database
	dbPath := filepath.Join(homeDir, ".ccdash", "ccdash.db")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Query all jobs
	query := `
		SELECT id, status, schedule_type, scheduled_at, created_at, command
		FROM jobs 
		ORDER BY created_at DESC`

	rows, err := db.Query(query)
	if err != nil {
		log.Fatal("Failed to query jobs:", err)
	}
	defer rows.Close()

	fmt.Println("=== All Jobs in Database ===")
	fmt.Printf("%-36s %-10s %-12s %-20s %-20s %s\n", "ID", "Status", "Schedule", "Scheduled At", "Created At", "Command")
	fmt.Println(strings.Repeat("-", 120))

	count := 0
	for rows.Next() {
		var id, status, command string
		var scheduleType, scheduledAt, createdAt *string

		err := rows.Scan(&id, &status, &scheduleType, &scheduledAt, &createdAt, &command)
		if err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}

		scheduleTypeStr := "<null>"
		if scheduleType != nil {
			scheduleTypeStr = *scheduleType
		}

		scheduledAtStr := "<null>"
		if scheduledAt != nil {
			scheduledAtStr = *scheduledAt
		}

		createdAtStr := "<null>"
		if createdAt != nil {
			createdAtStr = *createdAt
		}

		if len(command) > 40 {
			command = command[:37] + "..."
		}

		fmt.Printf("%-36s %-10s %-12s %-20s %-20s %s\n", 
			id, status, scheduleTypeStr, scheduledAtStr, createdAtStr, command)
		count++
	}

	fmt.Printf("\nTotal jobs: %d\n", count)
}