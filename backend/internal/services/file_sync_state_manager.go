package services

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"os"
	"time"

	"claudeee-backend/internal/models"
)

type FileSyncStateManager struct {
	db *sql.DB
}

func NewFileSyncStateManager(db *sql.DB) *FileSyncStateManager {
	return &FileSyncStateManager{db: db}
}

// InitializeSchema creates the file_sync_state table if it doesn't exist
func (f *FileSyncStateManager) InitializeSchema() error {
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS file_sync_state (
			file_path VARCHAR PRIMARY KEY,
			last_modified TIMESTAMP NOT NULL,
			file_size BIGINT NOT NULL,
			last_processed_line INTEGER DEFAULT 0,
			processed_until TIMESTAMP,
			checksum VARCHAR(64),
			sync_status VARCHAR DEFAULT 'pending',
			last_sync_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			error_message TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`
	
	_, err := f.db.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create file_sync_state table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_file_sync_state_path ON file_sync_state (file_path);",
		"CREATE INDEX IF NOT EXISTS idx_file_sync_state_status ON file_sync_state (sync_status);",
		"CREATE INDEX IF NOT EXISTS idx_file_sync_state_modified ON file_sync_state (last_modified);",
	}

	for _, indexQuery := range indexes {
		_, err := f.db.Exec(indexQuery)
		if err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// GetFileState retrieves the processing state of a file
func (f *FileSyncStateManager) GetFileState(filePath string) (*models.FileProcessingState, error) {
	query := `
		SELECT file_path, last_modified, file_size, last_processed_line, 
			   processed_until, checksum, sync_status, last_sync_time, 
			   error_message, created_at, updated_at
		FROM file_sync_state 
		WHERE file_path = ?
	`
	
	var state models.FileProcessingState
	err := f.db.QueryRow(query, filePath).Scan(
		&state.FilePath,
		&state.LastModified,
		&state.FileSize,
		&state.LastProcessedLine,
		&state.ProcessedUntil,
		&state.Checksum,
		&state.SyncStatus,
		&state.LastSyncTime,
		&state.ErrorMessage,
		&state.CreatedAt,
		&state.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // File not found in state table
		}
		return nil, fmt.Errorf("failed to get file state: %w", err)
	}
	
	return &state, nil
}

// UpdateFileState updates or inserts the processing state of a file using UPSERT
func (f *FileSyncStateManager) UpdateFileState(state *models.FileProcessingState) error {
	now := time.Now()
	state.UpdatedAt = now
	state.LastSyncTime = now
	
	// Use INSERT OR REPLACE to handle both insert and update atomically
	query := `
		INSERT OR REPLACE INTO file_sync_state (
			file_path, last_modified, file_size, last_processed_line,
			processed_until, checksum, sync_status, last_sync_time,
			error_message, created_at, updated_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, 
			COALESCE((SELECT created_at FROM file_sync_state WHERE file_path = ?), ?),
			?
		)
	`
	
	// Set created_at to now if it's a new record
	if state.CreatedAt.IsZero() {
		state.CreatedAt = now
	}
	
	_, err := f.db.Exec(query,
		state.FilePath,
		state.LastModified,
		state.FileSize,
		state.LastProcessedLine,
		state.ProcessedUntil,
		state.Checksum,
		state.SyncStatus,
		state.LastSyncTime,
		state.ErrorMessage,
		state.FilePath, // for COALESCE subquery
		state.CreatedAt,
		state.UpdatedAt,
	)
	
	if err != nil {
		return fmt.Errorf("failed to upsert file state: %w", err)
	}
	
	return nil
}

// NeedsProcessing checks if a file needs to be processed based on its current state
func (f *FileSyncStateManager) NeedsProcessing(filePath string) (bool, *models.FileProcessingState, error) {
	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false, nil, fmt.Errorf("failed to stat file: %w", err)
	}
	
	// Get previous processing state
	lastState, err := f.GetFileState(filePath)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get file state: %w", err)
	}
	
	// If no previous state, needs processing
	if lastState == nil {
		return true, nil, nil
	}
	
	// Check if file has been modified
	if fileInfo.ModTime().After(lastState.LastModified) {
		return true, lastState, nil
	}
	
	// Check if file size has changed
	if fileInfo.Size() != lastState.FileSize {
		return true, lastState, nil
	}
	
	// Check if last processing failed
	if lastState.SyncStatus == "error" || lastState.SyncStatus == "processing" {
		return true, lastState, nil
	}
	
	return false, lastState, nil
}

// CalculateFileChecksum calculates the SHA256 checksum of a file
func (f *FileSyncStateManager) CalculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file for checksum: %w", err)
	}
	defer file.Close()
	
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}
	
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// GetAllFileStates retrieves all file processing states
func (f *FileSyncStateManager) GetAllFileStates() ([]models.FileProcessingState, error) {
	query := `
		SELECT file_path, last_modified, file_size, last_processed_line,
			   processed_until, checksum, sync_status, last_sync_time,
			   error_message, created_at, updated_at
		FROM file_sync_state
		ORDER BY last_sync_time DESC
	`
	
	rows, err := f.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query file states: %w", err)
	}
	defer rows.Close()
	
	var states []models.FileProcessingState
	for rows.Next() {
		var state models.FileProcessingState
		err := rows.Scan(
			&state.FilePath,
			&state.LastModified,
			&state.FileSize,
			&state.LastProcessedLine,
			&state.ProcessedUntil,
			&state.Checksum,
			&state.SyncStatus,
			&state.LastSyncTime,
			&state.ErrorMessage,
			&state.CreatedAt,
			&state.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file state: %w", err)
		}
		states = append(states, state)
	}
	
	return states, nil
}

// CleanupOldStates removes state records for files that no longer exist and resets stuck processing states
func (f *FileSyncStateManager) CleanupOldStates() error {
	// First, reset any stuck "processing" states that are older than 5 minutes
	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
	_, err := f.db.Exec(`
		UPDATE file_sync_state 
		SET sync_status = 'pending', error_message = 'Reset from stuck processing state' 
		WHERE sync_status = 'processing' AND last_sync_time < ?
	`, fiveMinutesAgo)
	if err != nil {
		fmt.Printf("Warning: failed to reset stuck processing states: %v\n", err)
	}

	states, err := f.GetAllFileStates()
	if err != nil {
		return fmt.Errorf("failed to get file states for cleanup: %w", err)
	}
	
	for _, state := range states {
		if _, err := os.Stat(state.FilePath); os.IsNotExist(err) {
			// File no longer exists, remove from state table
			_, err := f.db.Exec("DELETE FROM file_sync_state WHERE file_path = ?", state.FilePath)
			if err != nil {
				fmt.Printf("Warning: failed to remove old state for %s: %v\n", state.FilePath, err)
			} else {
				fmt.Printf("Removed old state for deleted file: %s\n", state.FilePath)
			}
		}
	}
	
	return nil
}

// ResetFileState resets the processing state of a file (forces reprocessing)
func (f *FileSyncStateManager) ResetFileState(filePath string) error {
	_, err := f.db.Exec("DELETE FROM file_sync_state WHERE file_path = ?", filePath)
	if err != nil {
		return fmt.Errorf("failed to reset file state: %w", err)
	}
	return nil
}