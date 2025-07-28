package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/marcboeker/go-duckdb"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		return
	}

	dbPath := filepath.Join(homeDir, ".ccdash", "ccdash.db")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		return
	}
	defer db.Close()

	sessionID := "b6605769-96ab-444a-8763-42c8bd204239"
	
	// Check messages count for this session
	var messageCount int
	err = db.QueryRow("SELECT COUNT(*) FROM messages WHERE session_id = ?", sessionID).Scan(&messageCount)
	if err != nil {
		fmt.Printf("Error counting messages: %v\n", err)
		return
	}
	fmt.Printf("Messages for session %s: %d\n", sessionID, messageCount)

	// Check some sample messages
	rows, err := db.Query("SELECT id, message_type, message_role, timestamp FROM messages WHERE session_id = ? ORDER BY timestamp LIMIT 5", sessionID)
	if err != nil {
		fmt.Printf("Error getting messages: %v\n", err)
		return
	}
	defer rows.Close()

	fmt.Println("Sample messages:")
	for rows.Next() {
		var id, messageType, messageRole, timestamp sql.NullString
		err := rows.Scan(&id, &messageType, &messageRole, &timestamp)
		if err != nil {
			fmt.Printf("Error scanning row: %v\n", err)
			continue
		}
		fmt.Printf("  %s | %s | %s | %s\n", id.String, messageType.String, messageRole.String, timestamp.String)
	}

	// Check file sync state
	var fileState sql.NullString
	var lastLine int
	err = db.QueryRow("SELECT sync_status, last_processed_line FROM file_sync_state WHERE file_path LIKE ?", "%"+sessionID+"%").Scan(&fileState, &lastLine)
	if err != nil && err != sql.ErrNoRows {
		fmt.Printf("Error getting file state: %v\n", err)
		return
	}
	if err == sql.ErrNoRows {
		fmt.Printf("No sync state found for session %s\n", sessionID)
	} else {
		fmt.Printf("File sync state: %s, last processed line: %d\n", fileState.String, lastLine)
	}
}