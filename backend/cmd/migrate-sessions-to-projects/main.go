package main

import (
	"fmt"
	"log"

	"ccdash-backend/internal/database"
	"ccdash-backend/internal/services"
)

func main() {
	fmt.Println("Starting session-to-project migration...")

	// Initialize database
	db, err := database.Initialize()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Create services
	sessionService := services.NewSessionService(db)

	// Get sessions without project_id
	sessions, err := sessionService.GetSessionsWithoutProjectID()
	if err != nil {
		log.Fatal("Failed to get sessions without project_id:", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions need migration.")
		return
	}

	fmt.Printf("Found %d sessions to migrate...\n", len(sessions))

	migratedCount := 0
	errorCount := 0

	for i, session := range sessions {
		fmt.Printf("Migrating session %d/%d: %s (%s)\n", 
			i+1, len(sessions), session.ID, session.ProjectName)

		err := sessionService.MigrateSessionToProject(session.ID)
		if err != nil {
			log.Printf("Failed to migrate session %s: %v", session.ID, err)
			errorCount++
			continue
		}

		migratedCount++
	}

	fmt.Printf("\nMigration completed!")
	fmt.Printf("Successfully migrated: %d sessions\n", migratedCount)
	if errorCount > 0 {
		fmt.Printf("Errors: %d sessions\n", errorCount)
	}

	// Show final statistics
	remainingSessions, err := sessionService.GetSessionsWithoutProjectID()
	if err != nil {
		log.Printf("Warning: failed to get remaining session count: %v", err)
		return
	}

	fmt.Printf("Remaining sessions without project_id: %d\n", len(remainingSessions))
}