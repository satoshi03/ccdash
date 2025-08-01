package main

import (
	"fmt"
	"log"
	
	"ccdash-backend/internal/database"
)

func main() {
	log.Println("Testing database initialization...")
	
	db, err := database.Initialize()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	
	// Check if jobs table exists
	var tableCount int
	err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('jobs')").Scan(&tableCount)
	if err != nil {
		log.Fatalf("Failed to check jobs table: %v", err)
	}
	
	if tableCount > 0 {
		fmt.Println("✅ Jobs table created successfully!")
		
		// Check table structure
		rows, err := db.Query("PRAGMA table_info('jobs')")
		if err != nil {
			log.Fatalf("Failed to get table info: %v", err)
		}
		defer rows.Close()
		
		fmt.Println("Jobs table columns:")
		for rows.Next() {
			var cid int
			var name, dataType string
			var notNull int
			var defaultValue interface{}
			var pk int
			
			err = rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
			if err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}
			
			fmt.Printf("  - %s (%s)\n", name, dataType)
		}
	} else {
		fmt.Println("❌ Jobs table was not created")
	}
}