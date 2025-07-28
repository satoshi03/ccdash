package main

import (
	"fmt"
	"log"

	"ccdash-backend/internal/database"
	"ccdash-backend/internal/services"
)

func main() {
	fmt.Println("Starting session start time fix...")

	db, err := database.Initialize()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Fix all session start times based on first message timestamp using individual updates
	sessionQuery := `SELECT id FROM sessions WHERE EXISTS (SELECT 1 FROM messages WHERE session_id = sessions.id)`
	sessionRows, err := db.Query(sessionQuery)
	if err != nil {
		log.Fatalf("Failed to get sessions: %v", err)
	}
	defer sessionRows.Close()

	var sessionIDs []string
	for sessionRows.Next() {
		var sessionID string
		if err := sessionRows.Scan(&sessionID); err != nil {
			log.Printf("Error scanning session ID: %v", err)
			continue
		}
		sessionIDs = append(sessionIDs, sessionID)
	}

	fmt.Printf("Updating start times for %d sessions...\n", len(sessionIDs))

	updateQuery := `
		UPDATE sessions 
		SET start_time = (
			SELECT MIN(timestamp) 
			FROM messages 
			WHERE messages.session_id = ?
		)
		WHERE id = ?
	`

	updatedCount := 0
	for _, sessionID := range sessionIDs {
		result, err := db.Exec(updateQuery, sessionID, sessionID)
		if err != nil {
			log.Printf("Error updating session %s: %v", sessionID, err)
			continue
		}
		if rowsAffected, _ := result.RowsAffected(); rowsAffected > 0 {
			updatedCount++
		}
	}

	fmt.Printf("✅ Fixed start times for %d sessions\n", updatedCount)

	// Update all session statistics using already retrieved session IDs
	tokenService := services.NewTokenService(db)
	
	fmt.Printf("Updating statistics for %d sessions...\n", len(sessionIDs))

	statsUpdatedCount := 0
	for _, sessionID := range sessionIDs {
		if err := tokenService.UpdateSessionTokens(sessionID); err != nil {
			log.Printf("Error updating session %s: %v", sessionID, err)
			continue
		}
		statsUpdatedCount++
	}

	fmt.Printf("✅ Updated statistics for %d sessions\n", statsUpdatedCount)
	fmt.Println("Session time fix completed successfully!")
}