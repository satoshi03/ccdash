package main

import (
	"database/sql"
	"fmt"
	"log"

	"claudeee-backend/internal/database"
	"claudeee-backend/internal/services"
)

func main() {
	// Initialize database
	db, err := database.Initialize()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	fmt.Println("Starting cost recalculation...")

	// Add total_cost column if it doesn't exist
	_, err = db.Exec(`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS total_cost DOUBLE DEFAULT 0.0`)
	if err != nil {
		log.Fatalf("Failed to add total_cost column: %v", err)
	}

	// Initialize cost calculator
	calculator := services.NewPricingCalculator()

	// Get all sessions
	rows, err := db.Query(`SELECT id FROM sessions`)
	if err != nil {
		log.Fatalf("Failed to get sessions: %v", err)
	}
	defer rows.Close()

	var sessionCount int
	for rows.Next() {
		var sessionID string
		if err := rows.Scan(&sessionID); err != nil {
			log.Printf("Error scanning session ID: %v", err)
			continue
		}

		// Calculate cost for this session
		cost, err := calculateSessionCost(db, calculator, sessionID)
		if err != nil {
			log.Printf("Error calculating cost for session %s: %v", sessionID, err)
			continue
		}

		// Update session with calculated cost
		_, err = db.Exec(`UPDATE sessions SET total_cost = ? WHERE id = ?`, cost, sessionID)
		if err != nil {
			log.Printf("Error updating cost for session %s: %v", sessionID, err)
			continue
		}

		sessionCount++
		fmt.Printf("Updated session %s with cost $%.6f\n", sessionID, cost)
	}

	fmt.Printf("Successfully updated %d sessions with calculated costs\n", sessionCount)
}

func calculateSessionCost(db *sql.DB, calculator *services.PricingCalculator, sessionID string) (float64, error) {
	// Get all assistant messages for this session
	query := `
		SELECT 
			model,
			input_tokens,
			cache_creation_input_tokens,
			cache_read_input_tokens,
			output_tokens
		FROM messages 
		WHERE session_id = ? AND message_role = 'assistant'
	`

	rows, err := db.Query(query, sessionID)
	if err != nil {
		return 0, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var totalCost float64
	for rows.Next() {
		var model *string
		var inputTokens, cacheCreationTokens, cacheReadTokens, outputTokens int

		err := rows.Scan(&model, &inputTokens, &cacheCreationTokens, &cacheReadTokens, &outputTokens)
		if err != nil {
			return 0, fmt.Errorf("failed to scan message: %w", err)
		}

		// Calculate cost for this message
		messageCost := calculator.CalculateMessageCost(
			model,
			inputTokens,
			outputTokens,
			cacheCreationTokens,
			cacheReadTokens,
		)

		totalCost += messageCost
	}

	return totalCost, nil
}