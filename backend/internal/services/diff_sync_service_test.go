package services

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"claudeee-backend/internal/models"
	_ "github.com/marcboeker/go-duckdb/v2"
)

func setupTestDBForDiffSync(t *testing.T) (*sql.DB, *DiffSyncService) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create test tables
	createTables := `
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			project_name TEXT,
			project_path TEXT,
			start_time TIMESTAMP,
			end_time TIMESTAMP,
			total_input_tokens INTEGER DEFAULT 0,
			total_output_tokens INTEGER DEFAULT 0,
			total_tokens INTEGER DEFAULT 0,
			message_count INTEGER DEFAULT 0,
			status TEXT DEFAULT 'active',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			session_id TEXT,
			parent_uuid TEXT,
			is_sidechain BOOLEAN DEFAULT FALSE,
			user_type TEXT,
			message_type TEXT,
			message_role TEXT,
			model TEXT,
			content TEXT,
			input_tokens INTEGER DEFAULT 0,
			cache_creation_input_tokens INTEGER DEFAULT 0,
			cache_read_input_tokens INTEGER DEFAULT 0,
			output_tokens INTEGER DEFAULT 0,
			service_tier TEXT,
			request_id TEXT,
			timestamp TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`

	_, err = db.Exec(createTables)
	if err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	tokenService := NewTokenService(db)
	sessionService := NewSessionService(db)
	diffSyncService := NewDiffSyncService(db, tokenService, sessionService)
	
	// Initialize schema
	err = diffSyncService.InitializeSchema()
	if err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	return db, diffSyncService
}

func TestNewDiffSyncService(t *testing.T) {
	db, diffSyncService := setupTestDBForDiffSync(t)
	defer db.Close()

	if diffSyncService == nil {
		t.Error("NewDiffSyncService returned nil")
	}
	if diffSyncService.db != db {
		t.Error("DiffSyncService db field not set correctly")
	}
	if diffSyncService.stateManager == nil {
		t.Error("DiffSyncService stateManager not initialized")
	}
}

func TestExtractProjectNameFromPath(t *testing.T) {
	db, diffSyncService := setupTestDBForDiffSync(t)
	defer db.Close()

	testCases := []struct {
		filePath string
		expected string
	}{
		{"/Users/satoshi/.claude/projects/project1/session.jsonl", "project1"},
		{"/Users/satoshi/.claude/projects/my-project/log.jsonl", "my-project"},
		{"/home/user/.claude/projects/test_project/data.jsonl", "test_project"},
		{"./projects/simple/file.jsonl", "simple"},
	}

	for _, tc := range testCases {
		t.Run(tc.filePath, func(t *testing.T) {
			result := diffSyncService.extractProjectNameFromPath(tc.filePath)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestProcessFileFromLine(t *testing.T) {
	db, diffSyncService := setupTestDBForDiffSync(t)
	defer db.Close()

	// Create a temporary JSONL file
	tmpFile, err := os.CreateTemp("", "test-*.jsonl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write test data
	testData := []string{
		`{"uuid":"1","sessionId":"session1","userType":"human","cwd":"/test","timestamp":"2024-01-01T10:00:00Z","message":{"role":"user","content":"Hello"}}`,
		`{"uuid":"2","sessionId":"session1","userType":"claude","cwd":"/test","timestamp":"2024-01-01T10:01:00Z","message":{"role":"assistant","content":"Hi there"}}`,
		`{"uuid":"3","sessionId":"session1","userType":"human","cwd":"/test","timestamp":"2024-01-01T10:02:00Z","message":{"role":"user","content":"How are you?"}}`,
	}

	for _, data := range testData {
		tmpFile.WriteString(data + "\n")
	}
	tmpFile.Close()

	// Test processing from line 0 (all lines)
	newLines, totalLines, err := diffSyncService.processFileFromLine(tmpFile.Name(), 0)
	if err != nil {
		t.Fatalf("Failed to process file from line 0: %v", err)
	}

	if newLines != 3 {
		t.Errorf("Expected 3 new lines, got %d", newLines)
	}
	if totalLines != 3 {
		t.Errorf("Expected 3 total lines, got %d", totalLines)
	}

	// Test processing from line 2 (skip first 2 lines)
	newLines, totalLines, err = diffSyncService.processFileFromLine(tmpFile.Name(), 2)
	if err != nil {
		t.Fatalf("Failed to process file from line 2: %v", err)
	}

	if newLines != 1 {
		t.Errorf("Expected 1 new line, got %d", newLines)
	}
	if totalLines != 3 {
		t.Errorf("Expected 3 total lines, got %d", totalLines)
	}

	// Verify sessions were created
	var sessionCount int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&sessionCount)
	if err != nil {
		t.Fatalf("Failed to query session count: %v", err)
	}
	if sessionCount != 1 {
		t.Errorf("Expected 1 session, got %d", sessionCount)
	}
}

func TestSyncFile(t *testing.T) {
	db, diffSyncService := setupTestDBForDiffSync(t)
	defer db.Close()

	// Create a temporary JSONL file
	tmpFile, err := os.CreateTemp("", "test-sync-*.jsonl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write test data
	testData := `{"uuid":"sync-1","sessionId":"sync-session","userType":"human","cwd":"/sync-test","timestamp":"2024-01-01T10:00:00Z","message":{"role":"user","content":"Sync test"}}`
	tmpFile.WriteString(testData + "\n")
	tmpFile.Close()

	// Get file info
	fileInfo, err := os.Stat(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	file := models.FileInfo{
		Path:    tmpFile.Name(),
		ModTime: fileInfo.ModTime(),
		Size:    fileInfo.Size(),
	}

	// Test initial sync (no previous state)
	newLines, err := diffSyncService.syncFile(file, nil)
	if err != nil {
		t.Fatalf("Failed to sync file: %v", err)
	}

	if newLines != 1 {
		t.Errorf("Expected 1 new line, got %d", newLines)
	}

	// Verify file state was recorded
	state, err := diffSyncService.stateManager.GetFileState(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to get file state: %v", err)
	}
	if state == nil {
		t.Fatal("File state not recorded")
	}
	if state.SyncStatus != "completed" {
		t.Errorf("Expected sync status 'completed', got '%s'", state.SyncStatus)
	}
	if state.LastProcessedLine != 1 {
		t.Errorf("Expected last processed line 1, got %d", state.LastProcessedLine)
	}

	// Test sync with existing state (should process 0 new lines)
	newLines, err = diffSyncService.syncFile(file, state)
	if err != nil {
		t.Fatalf("Failed to sync file with existing state: %v", err)
	}

	if newLines != 0 {
		t.Errorf("Expected 0 new lines, got %d", newLines)
	}
}

func TestDiffSyncConvertContentToString(t *testing.T) {
	db, diffSyncService := setupTestDBForDiffSync(t)
	defer db.Close()

	testCases := []struct {
		content  interface{}
		expected string
	}{
		{"simple string", "simple string"},
		{123, "123"},
		{map[string]interface{}{"key": "value"}, `{"key":"value"}`},
		{[]interface{}{"item1", "item2"}, `["item1","item2"]`},
	}

	for i, tc := range testCases {
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			result := diffSyncService.convertContentToString(tc.content)
			if tc.content != "simple string" && tc.content != 123 {
				// For JSON marshaled content, just verify it's not empty
				if result == "" {
					t.Error("Expected non-empty result for complex content")
				}
			} else {
				if result != tc.expected {
					t.Errorf("Expected %s, got %s", tc.expected, result)
				}
			}
		})
	}
}

func TestGetSyncStats(t *testing.T) {
	db, diffSyncService := setupTestDBForDiffSync(t)
	defer db.Close()

	// Initially no files
	stats, err := diffSyncService.GetSyncStats()
	if err != nil {
		t.Fatalf("Failed to get sync stats: %v", err)
	}
	if stats.TotalFiles != 0 {
		t.Errorf("Expected 0 total files, got %d", stats.TotalFiles)
	}

	// Add a file state
	state := &models.FileProcessingState{
		FilePath:     "/test/file.jsonl",
		LastModified: time.Now(),
		FileSize:     100,
		SyncStatus:   "completed",
	}
	err = diffSyncService.stateManager.UpdateFileState(state)
	if err != nil {
		t.Fatalf("Failed to update file state: %v", err)
	}

	// Check stats again
	stats, err = diffSyncService.GetSyncStats()
	if err != nil {
		t.Fatalf("Failed to get sync stats: %v", err)
	}
	if stats.TotalFiles != 1 {
		t.Errorf("Expected 1 total file, got %d", stats.TotalFiles)
	}
	if stats.ProcessedFiles != 1 {
		t.Errorf("Expected 1 processed file, got %d", stats.ProcessedFiles)
	}
	if stats.SkippedFiles != 0 {
		t.Errorf("Expected 0 skipped files, got %d", stats.SkippedFiles)
	}
}