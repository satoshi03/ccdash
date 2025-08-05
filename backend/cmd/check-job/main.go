package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/marcboeker/go-duckdb"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: check-job <job-id>")
		os.Exit(1)
	}

	jobID := os.Args[1]

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

	// Query job details
	query := `
		SELECT 
			id, project_id, command, execution_directory, yolo_mode,
			status, priority, created_at, started_at, completed_at,
			scheduled_at, schedule_type, schedule_params
		FROM jobs 
		WHERE id = ?`

	var (
		id, projectID, command, execDir, status    string
		yoloMode                                   bool
		priority                                   int
		createdAt                                  string
		startedAt, completedAt, scheduledAt        *string
		scheduleType, scheduleParams               *string
	)

	err = db.QueryRow(query, jobID).Scan(
		&id, &projectID, &command, &execDir, &yoloMode,
		&status, &priority, &createdAt, &startedAt, &completedAt,
		&scheduledAt, &scheduleType, &scheduleParams,
	)

	if err == sql.ErrNoRows {
		fmt.Printf("Job not found: %s\n", jobID)
		os.Exit(1)
	} else if err != nil {
		log.Fatal("Failed to query job:", err)
	}

	// Print job details
	fmt.Println("=== Job Details ===")
	fmt.Printf("ID: %s\n", id)
	fmt.Printf("Project ID: %s\n", projectID)
	fmt.Printf("Command: %s\n", command)
	fmt.Printf("Execution Directory: %s\n", execDir)
	fmt.Printf("YOLO Mode: %v\n", yoloMode)
	fmt.Printf("Status: %s\n", status)
	fmt.Printf("Priority: %d\n", priority)
	fmt.Printf("Created At: %s\n", createdAt)
	
	if startedAt != nil {
		fmt.Printf("Started At: %s\n", *startedAt)
	} else {
		fmt.Println("Started At: <null>")
	}
	
	if completedAt != nil {
		fmt.Printf("Completed At: %s\n", *completedAt)
	} else {
		fmt.Println("Completed At: <null>")
	}
	
	if scheduledAt != nil {
		fmt.Printf("Scheduled At: %s\n", *scheduledAt)
		// Parse and show in local time
		if t, err := time.Parse(time.RFC3339, *scheduledAt); err == nil {
			fmt.Printf("Scheduled At (Local): %s\n", t.Local().Format("2006-01-02 15:04:05 MST"))
		}
	} else {
		fmt.Println("Scheduled At: <null>")
	}
	
	if scheduleType != nil {
		fmt.Printf("Schedule Type: %s\n", *scheduleType)
	} else {
		fmt.Println("Schedule Type: <null>")
	}
	
	if scheduleParams != nil {
		fmt.Printf("Schedule Params (raw): %s\n", *scheduleParams)
		// Try to parse JSON
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(*scheduleParams), &params); err == nil {
			fmt.Println("Schedule Params (parsed):")
			for k, v := range params {
				fmt.Printf("  %s: %v\n", k, v)
			}
		}
	} else {
		fmt.Println("Schedule Params: <null>")
	}

	// Check current time
	fmt.Println("\n=== Time Check ===")
	fmt.Printf("Current Time: %s\n", time.Now().Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("Current Time (UTC): %s\n", time.Now().UTC().Format("2006-01-02 15:04:05 MST"))
}