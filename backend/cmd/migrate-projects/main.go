package main

import (
	"fmt"
	"log"

	"ccdash-backend/internal/database"
	"ccdash-backend/internal/services"
)

func main() {
	fmt.Println("Starting project migration...")

	// Initialize database
	db, err := database.Initialize()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Create project service
	projectService := services.NewProjectService(db)

	// Run migration
	err = projectService.MigrateExistingSessionsToProjects()
	if err != nil {
		log.Fatal("Migration failed:", err)
	}

	fmt.Println("Project migration completed successfully!")

	// Show statistics
	projects, err := projectService.GetAllProjects()
	if err != nil {
		log.Printf("Warning: failed to get project statistics: %v", err)
		return
	}

	fmt.Printf("Total projects in database: %d\n", len(projects))
	
	fmt.Println("\nProjects:")
	for _, project := range projects {
		fmt.Printf("- ID: %s, Name: %s, Path: %s\n", project.ID, project.Name, project.Path)
	}
}