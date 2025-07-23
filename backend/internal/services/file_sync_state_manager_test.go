package services

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"claudeee-backend/internal/models"
	_ "github.com/marcboeker/go-duckdb"
)

func setupTestDBForStateManager(t *testing.T) (*sql.DB, *FileSyncStateManager) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	stateManager := NewFileSyncStateManager(db)
	err = stateManager.InitializeSchema()
	if err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	return db, stateManager
}

func TestNewFileSyncStateManager(t *testing.T) {
	db, stateManager := setupTestDBForStateManager(t)
	defer db.Close()

	if stateManager == nil {
		t.Error("NewFileSyncStateManager returned nil")
	}
	if stateManager.db != db {
		t.Error("FileSyncStateManager db field not set correctly")
	}
}

func TestInitializeSchema(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	stateManager := NewFileSyncStateManager(db)
	err = stateManager.InitializeSchema()
	if err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	// Verify table exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM file_sync_state").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query file_sync_state table: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 records in new table, got %d", count)
	}
}

func TestUpdateFileState_Insert(t *testing.T) {
	db, stateManager := setupTestDBForStateManager(t)
	defer db.Close()

	state := &models.FileProcessingState{
		FilePath:          "/test/file.jsonl",
		LastModified:      time.Now(),
		FileSize:          1024,
		LastProcessedLine: 10,
		SyncStatus:        "completed",
	}

	err := stateManager.UpdateFileState(state)
	if err != nil {
		t.Fatalf("Failed to update file state: %v", err)
	}

	// Verify record was inserted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM file_sync_state WHERE file_path = ?", state.FilePath).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query inserted record: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 record, got %d", count)
	}
}

func TestUpdateFileState_Update(t *testing.T) {
	db, stateManager := setupTestDBForStateManager(t)
	defer db.Close()

	filePath := "/test/update.jsonl"
	
	// Insert initial state
	initialState := &models.FileProcessingState{
		FilePath:          filePath,
		LastModified:      time.Now(),
		FileSize:          1024,
		LastProcessedLine: 5,
		SyncStatus:        "pending",
	}

	err := stateManager.UpdateFileState(initialState)
	if err != nil {
		t.Fatalf("Failed to insert initial state: %v", err)
	}

	// Update state
	updatedState := &models.FileProcessingState{
		FilePath:          filePath,
		LastModified:      time.Now(),
		FileSize:          2048,
		LastProcessedLine: 15,
		SyncStatus:        "completed",
	}

	err = stateManager.UpdateFileState(updatedState)
	if err != nil {
		t.Fatalf("Failed to update state: %v", err)
	}

	// Verify update
	retrievedState, err := stateManager.GetFileState(filePath)
	if err != nil {
		t.Fatalf("Failed to get updated state: %v", err)
	}
	if retrievedState == nil {
		t.Fatal("Updated state not found")
	}
	if retrievedState.FileSize != 2048 {
		t.Errorf("Expected file size 2048, got %d", retrievedState.FileSize)
	}
	if retrievedState.LastProcessedLine != 15 {
		t.Errorf("Expected last processed line 15, got %d", retrievedState.LastProcessedLine)
	}
	if retrievedState.SyncStatus != "completed" {
		t.Errorf("Expected sync status 'completed', got '%s'", retrievedState.SyncStatus)
	}

	// Verify still only one record
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM file_sync_state WHERE file_path = ?", filePath).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query record count: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 record after update, got %d", count)
	}
}

func TestGetFileState(t *testing.T) {
	db, stateManager := setupTestDBForStateManager(t)
	defer db.Close()

	filePath := "/test/get.jsonl"
	
	// Test getting non-existent state
	state, err := stateManager.GetFileState(filePath)
	if err != nil {
		t.Fatalf("Failed to get non-existent state: %v", err)
	}
	if state != nil {
		t.Error("Expected nil for non-existent state")
	}

	// Insert state
	testState := &models.FileProcessingState{
		FilePath:          filePath,
		LastModified:      time.Now(),
		FileSize:          512,
		LastProcessedLine: 3,
		SyncStatus:        "processing",
	}

	err = stateManager.UpdateFileState(testState)
	if err != nil {
		t.Fatalf("Failed to insert test state: %v", err)
	}

	// Get state
	retrievedState, err := stateManager.GetFileState(filePath)
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}
	if retrievedState == nil {
		t.Fatal("State not found")
	}
	if retrievedState.FilePath != filePath {
		t.Errorf("Expected file path %s, got %s", filePath, retrievedState.FilePath)
	}
	if retrievedState.FileSize != 512 {
		t.Errorf("Expected file size 512, got %d", retrievedState.FileSize)
	}
	if retrievedState.LastProcessedLine != 3 {
		t.Errorf("Expected last processed line 3, got %d", retrievedState.LastProcessedLine)
	}
	if retrievedState.SyncStatus != "processing" {
		t.Errorf("Expected sync status 'processing', got '%s'", retrievedState.SyncStatus)
	}
}

func TestNeedsProcessing(t *testing.T) {
	db, stateManager := setupTestDBForStateManager(t)
	defer db.Close()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test-needs-*.jsonl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("test content\n")
	tmpFile.Close()

	// Test with no previous state (should need processing)
	needsProcessing, lastState, err := stateManager.NeedsProcessing(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to check needs processing: %v", err)
	}
	if !needsProcessing {
		t.Error("Expected needs processing to be true for new file")
	}
	if lastState != nil {
		t.Error("Expected last state to be nil for new file")
	}

	// Create file state
	fileInfo, err := os.Stat(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	state := &models.FileProcessingState{
		FilePath:          tmpFile.Name(),
		LastModified:      fileInfo.ModTime(),
		FileSize:          fileInfo.Size(),
		LastProcessedLine: 1,
		SyncStatus:        "completed",
	}

	err = stateManager.UpdateFileState(state)
	if err != nil {
		t.Fatalf("Failed to update file state: %v", err)
	}

	// Test with matching state (should not need processing)
	needsProcessing, lastState, err = stateManager.NeedsProcessing(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to check needs processing: %v", err)
	}
	if needsProcessing {
		t.Error("Expected needs processing to be false for unchanged file")
	}
	if lastState == nil {
		t.Error("Expected last state to be returned")
	}

	// Modify file
	file, err := os.OpenFile(tmpFile.Name(), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for modification: %v", err)
	}
	file.WriteString("more content\n")
	file.Close()

	// Test with modified file (should need processing)
	needsProcessing, lastState, err = stateManager.NeedsProcessing(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to check needs processing after modification: %v", err)
	}
	if !needsProcessing {
		t.Error("Expected needs processing to be true for modified file")
	}
	if lastState == nil {
		t.Error("Expected last state to be returned for modified file")
	}
}

func TestCalculateFileChecksum(t *testing.T) {
	db, stateManager := setupTestDBForStateManager(t)
	defer db.Close()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test-checksum-*.jsonl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := "test content for checksum"
	tmpFile.WriteString(content)
	tmpFile.Close()

	// Calculate checksum
	checksum1, err := stateManager.CalculateFileChecksum(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to calculate checksum: %v", err)
	}
	if checksum1 == "" {
		t.Error("Expected non-empty checksum")
	}

	// Calculate checksum again (should be same)
	checksum2, err := stateManager.CalculateFileChecksum(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to calculate checksum again: %v", err)
	}
	if checksum1 != checksum2 {
		t.Errorf("Expected same checksum, got %s and %s", checksum1, checksum2)
	}

	// Modify file and calculate checksum (should be different)
	file, err := os.OpenFile(tmpFile.Name(), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for modification: %v", err)
	}
	file.WriteString(" modified")
	file.Close()

	checksum3, err := stateManager.CalculateFileChecksum(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to calculate checksum after modification: %v", err)
	}
	if checksum1 == checksum3 {
		t.Error("Expected different checksum after modification")
	}
}

func TestGetAllFileStates(t *testing.T) {
	db, stateManager := setupTestDBForStateManager(t)
	defer db.Close()

	// Initially no states
	states, err := stateManager.GetAllFileStates()
	if err != nil {
		t.Fatalf("Failed to get all file states: %v", err)
	}
	if len(states) != 0 {
		t.Errorf("Expected 0 states, got %d", len(states))
	}

	// Add multiple states
	testStates := []*models.FileProcessingState{
		{
			FilePath:     "/test/file1.jsonl",
			LastModified: time.Now(),
			FileSize:     100,
			SyncStatus:   "completed",
		},
		{
			FilePath:     "/test/file2.jsonl",
			LastModified: time.Now(),
			FileSize:     200,
			SyncStatus:   "pending",
		},
	}

	for _, state := range testStates {
		err := stateManager.UpdateFileState(state)
		if err != nil {
			t.Fatalf("Failed to update file state: %v", err)
		}
	}

	// Get all states
	states, err = stateManager.GetAllFileStates()
	if err != nil {
		t.Fatalf("Failed to get all file states: %v", err)
	}
	if len(states) != 2 {
		t.Errorf("Expected 2 states, got %d", len(states))
	}

	// Verify states
	foundPaths := make(map[string]bool)
	for _, state := range states {
		foundPaths[state.FilePath] = true
	}
	if !foundPaths["/test/file1.jsonl"] || !foundPaths["/test/file2.jsonl"] {
		t.Error("Expected both file paths to be found")
	}
}

func TestResetFileState(t *testing.T) {
	db, stateManager := setupTestDBForStateManager(t)
	defer db.Close()

	filePath := "/test/reset.jsonl"
	
	// Add a state
	state := &models.FileProcessingState{
		FilePath:     filePath,
		LastModified: time.Now(),
		FileSize:     100,
		SyncStatus:   "completed",
	}

	err := stateManager.UpdateFileState(state)
	if err != nil {
		t.Fatalf("Failed to update file state: %v", err)
	}

	// Verify state exists
	retrievedState, err := stateManager.GetFileState(filePath)
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}
	if retrievedState == nil {
		t.Fatal("State not found before reset")
	}

	// Reset state
	err = stateManager.ResetFileState(filePath)
	if err != nil {
		t.Fatalf("Failed to reset file state: %v", err)
	}

	// Verify state is gone
	retrievedState, err = stateManager.GetFileState(filePath)
	if err != nil {
		t.Fatalf("Failed to get state after reset: %v", err)
	}
	if retrievedState != nil {
		t.Error("Expected state to be nil after reset")
	}
}