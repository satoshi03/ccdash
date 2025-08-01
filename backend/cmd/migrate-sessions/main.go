package main

import (
	"fmt"
	"log"

	"ccdash-backend/internal/config"
	"ccdash-backend/internal/database"
	"ccdash-backend/internal/services"
)

func main() {
	fmt.Println("=== Session Migration Tool ===")
	
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	db, err := database.InitializeWithConfig(cfg)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	sessionService := services.NewSessionService(db)

	// Get sessions without project_id
	sessions, err := sessionService.GetSessionsWithoutProjectID()
	if err != nil {
		log.Fatal("Failed to get sessions for migration:", err)
	}

	if len(sessions) == 0 {
		fmt.Println("✅ No sessions need migration - all sessions already have project_id")
		return
	}

	fmt.Printf("Found %d sessions that need migration\n", len(sessions))
	
	migratedCount := 0
	errorCount := 0

	for _, session := range sessions {
		fmt.Printf("Migrating session %s (project: %s)...\n", session.ID, session.ProjectName)
		
		err := sessionService.MigrateSessionToProject(session.ID)
		if err != nil {
			fmt.Printf("  ❌ Error: %v\n", err)
			errorCount++
			continue
		}
		
		fmt.Printf("  ✅ Success\n")
		migratedCount++
	}

	fmt.Printf("\n=== Migration Complete ===\n")
	fmt.Printf("✅ Successfully migrated: %d sessions\n", migratedCount)
	if errorCount > 0 {
		fmt.Printf("❌ Failed to migrate: %d sessions\n", errorCount)
	}
	fmt.Printf("Total processed: %d sessions\n", len(sessions))
}