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

	dbPath := filepath.Join(homeDir, ".claudeee", "claudeee.db")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		return
	}
	defer db.Close()

	sessionID := "af5361c0-b11d-425c-9d48-00db63722694"
	
	// Check session exists
	var sessionExists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM sessions WHERE id = ?)", sessionID).Scan(&sessionExists)
	if err != nil {
		fmt.Printf("Error checking session: %v\n", err)
		return
	}
	fmt.Printf("Session %s exists: %v\n", sessionID, sessionExists)

	// Check messages count for this session
	var messageCount int
	err = db.QueryRow("SELECT COUNT(*) FROM messages WHERE session_id = ?", sessionID).Scan(&messageCount)
	if err != nil {
		fmt.Printf("Error counting messages: %v\n", err)
		return
	}
	fmt.Printf("Messages for session %s: %d\n", sessionID, messageCount)

	// Get some sample messages
	rows, err := db.Query("SELECT id, message_type, timestamp FROM messages WHERE session_id = ? ORDER BY timestamp DESC LIMIT 5", sessionID)
	if err != nil {
		fmt.Printf("Error getting messages: %v\n", err)
		return
	}
	defer rows.Close()

	fmt.Println("Sample messages:")
	for rows.Next() {
		var id, messageType, timestamp string
		err := rows.Scan(&id, &messageType, &timestamp)
		if err != nil {
			fmt.Printf("Error scanning row: %v\n", err)
			continue
		}
		fmt.Printf("  %s | %s | %s\n", id, messageType, timestamp)
	}

	// Check session details
	var projectName, projectPath, startTime, endTime sql.NullString
	var totalTokens int
	var messageCountSession int
	err = db.QueryRow(`
		SELECT project_name, project_path, start_time, end_time, 
		       total_input_tokens + total_output_tokens as total_tokens, message_count 
		FROM sessions WHERE id = ?`, sessionID).Scan(&projectName, &projectPath, &startTime, &endTime, &totalTokens, &messageCountSession)
	if err != nil {
		fmt.Printf("Error getting session details: %v\n", err)
		return
	}
	
	fmt.Printf("Session details:\n")
	fmt.Printf("  Project: %s (%s)\n", projectName.String, projectPath.String)
	fmt.Printf("  Start: %s\n", startTime.String)
	fmt.Printf("  End: %s\n", endTime.String)
	fmt.Printf("  Total tokens: %d\n", totalTokens)
	fmt.Printf("  Message count: %d\n", messageCountSession)
}