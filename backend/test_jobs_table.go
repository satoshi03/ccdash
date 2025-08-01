package main

import (
	"fmt"
	"log"
	
	"ccdash-backend/internal/database"
)

func main() {
	db, err := database.Initialize()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	
	// Simple test: try to select from jobs table
	_, err = db.Exec("SELECT COUNT(*) FROM jobs;")
	if err != nil {
		fmt.Printf("❌ Jobs table not accessible: %v\n", err)
		return
	}
	
	fmt.Println("✅ Jobs table exists and is accessible!")
	
	// First create a test project (required due to foreign key constraint)
	projectInsert := `INSERT INTO projects (id, name, path, created_at, updated_at) 
					  VALUES ('test-project', 'Test Project', '/test/dir', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`
	
	_, err = db.Exec(projectInsert)
	if err != nil {
		fmt.Printf("❌ Failed to create test project: %v\n", err)
		return
	}
	
	// Test insert (this will verify all required columns exist)
	testInsert := `INSERT INTO jobs (id, project_id, command, execution_directory, status, created_at) 
				   VALUES ('test-id', 'test-project', 'test command', '/test/dir', 'pending', '2025-08-01T10:00:00Z')`
	
	_, err = db.Exec(testInsert)
	if err != nil {
		fmt.Printf("❌ Jobs table structure issue: %v\n", err)
		return
	}
	
	fmt.Println("✅ Jobs table structure is correct!")
	
	// Clean up test data
	db.Exec("DELETE FROM jobs WHERE id = 'test-id'")
	db.Exec("DELETE FROM projects WHERE id = 'test-project'")
	
	fmt.Println("🎉 Database initialization successful - Phase 2 ready!")
}