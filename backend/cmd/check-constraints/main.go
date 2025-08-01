package main

import (
	"database/sql"
	"fmt"
	"log"

	"ccdash-backend/internal/config"

	_ "github.com/marcboeker/go-duckdb"
)

func main() {
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	db, err := sql.Open("duckdb", cfg.DatabasePath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	fmt.Println("=== Database Constraints Check ===")
	
	// Check sessions with NULL project_id
	var nullProjectCount int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE project_id IS NULL").Scan(&nullProjectCount)
	if err != nil {
		log.Printf("Error checking NULL project_id: %v", err)
	} else {
		fmt.Printf("Sessions with NULL project_id: %d\n", nullProjectCount)
	}

	// Check total sessions
	var totalSessions int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&totalSessions)
	if err != nil {
		log.Printf("Error checking total sessions: %v", err)
	} else {
		fmt.Printf("Total sessions: %d\n", totalSessions)
	}

	// Check total projects
	var totalProjects int
	err = db.QueryRow("SELECT COUNT(*) FROM projects").Scan(&totalProjects)
	if err != nil {
		log.Printf("Error checking total projects: %v", err)
	} else {
		fmt.Printf("Total projects: %d\n", totalProjects)
	}

	// Check sessions with valid project_id that reference existing projects
	var validReferences int
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM sessions s 
		INNER JOIN projects p ON s.project_id = p.id 
		WHERE s.project_id IS NOT NULL
	`).Scan(&validReferences)
	if err != nil {
		log.Printf("Error checking valid references: %v", err)
	} else {
		fmt.Printf("Sessions with valid project references: %d\n", validReferences)
	}

	// Check for orphaned sessions (project_id not NULL but no matching project)
	var orphanedSessions int
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM sessions s 
		LEFT JOIN projects p ON s.project_id = p.id 
		WHERE s.project_id IS NOT NULL AND p.id IS NULL
	`).Scan(&orphanedSessions)
	if err != nil {
		log.Printf("Error checking orphaned sessions: %v", err)
	} else {
		fmt.Printf("Orphaned sessions (invalid project_id): %d\n", orphanedSessions)
	}

	fmt.Println("\n=== Foreign Key Constraint Test ===")
	
	// Try to add foreign key constraint
	constraintQuery := `ALTER TABLE sessions ADD CONSTRAINT fk_sessions_project_id 
						FOREIGN KEY (project_id) REFERENCES projects(id)`
	
	_, err = db.Exec(constraintQuery)
	if err != nil {
		fmt.Printf("Failed to add foreign key constraint: %v\n", err)
		fmt.Println("This is expected if the constraint already exists or DuckDB doesn't support this syntax")
	} else {
		fmt.Println("Foreign key constraint added successfully!")
	}

	fmt.Println("\n=== Database Integrity Status ===")
	if nullProjectCount == 0 && orphanedSessions == 0 {
		fmt.Println("✅ Database integrity is good - all sessions have valid project references")
	} else if nullProjectCount > 0 {
		fmt.Printf("⚠️  Warning: %d sessions have NULL project_id (legacy data)\n", nullProjectCount)
	}
	if orphanedSessions > 0 {
		fmt.Printf("❌ Error: %d sessions have invalid project_id references\n", orphanedSessions)
	}
}