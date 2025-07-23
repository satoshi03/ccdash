package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"claudeee-backend/internal/models"
	_ "github.com/marcboeker/go-duckdb/v2"
)

func setupTestDBForJSONL(t *testing.T) (*sql.DB, *TokenService, *SessionService) {
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
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			is_active BOOLEAN DEFAULT TRUE,
			generated_code TEXT
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
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (session_id) REFERENCES sessions(id)
		);
	`

	_, err = db.Exec(createTables)
	if err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	tokenService := NewTokenService(db)
	sessionService := NewSessionService(db)

	return db, tokenService, sessionService
}

func TestNewJSONLParser(t *testing.T) {
	db, tokenService, sessionService := setupTestDBForJSONL(t)
	defer db.Close()

	parser := NewJSONLParser(db, tokenService, sessionService)
	
	if parser == nil {
		t.Error("NewJSONLParser returned nil")
	}
	if parser.db != db {
		t.Error("JSONLParser db field not set correctly")
	}
	if parser.tokenService != tokenService {
		t.Error("JSONLParser tokenService field not set correctly")
	}
	if parser.sessionService != sessionService {
		t.Error("JSONLParser sessionService field not set correctly")
	}
}

func TestExtractProjectNameFromCwd(t *testing.T) {
	db, tokenService, sessionService := setupTestDBForJSONL(t)
	defer db.Close()

	parser := NewJSONLParser(db, tokenService, sessionService)

	testCases := []struct {
		cwd      string
		expected string
	}{
		{"/Users/satoshi/git/manavi", "manavi"},
		{"/Users/satoshi/git/manavi/backend", "manavi"},
		{"/Users/satoshi/git/manavi/frontend", "manavi"},
		{"/Users/satoshi/git/manavi/src", "manavi"},
		{"/Users/satoshi/git/manavi/lib", "manavi"},
		{"/Users/satoshi/git/claude-pilot", "claude-pilot"},
		{"/Users/satoshi/git/claude-pilot/backend/src", "claude-pilot"},
		{"/home/user/projects/myproject", "myproject"},
		{"/home/user/projects/myproject/frontend/src", "myproject"},
		{"", "unknown"},
		{"/", "unknown"},
		{".", "unknown"},
		{"/Users/satoshi/git/project-name", "project-name"},
		{"/Users/satoshi/git/project_name", "project_name"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("cwd=%s", tc.cwd), func(t *testing.T) {
			result := parser.extractProjectNameFromCwd(tc.cwd)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestConvertProjectNameToPath(t *testing.T) {
	db, tokenService, sessionService := setupTestDBForJSONL(t)
	defer db.Close()

	parser := NewJSONLParser(db, tokenService, sessionService)

	testCases := []struct {
		projectName string
		expected    string
	}{
		{"-Users-satoshi-git-manavi", "/Users/satoshi/git/manavi"},
		{"-home-user-projects-myproject", "/home/user/projects/myproject"},
		{"regular-project", "regular-project"},
		{"simple", "simple"},
		{"-a-b-c", "/a/b/c"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("projectName=%s", tc.projectName), func(t *testing.T) {
			result := parser.convertProjectNameToPath(tc.projectName)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestConvertContentToString(t *testing.T) {
	db, tokenService, sessionService := setupTestDBForJSONL(t)
	defer db.Close()

	parser := NewJSONLParser(db, tokenService, sessionService)

	testCases := []struct {
		content  interface{}
		expected string
	}{
		{"simple string", "simple string"},
		{123, "123"},
		{true, "true"},
		{map[string]interface{}{"key": "value"}, `{"key":"value"}`},
		{[]interface{}{"item1", "item2"}, `["item1","item2"]`},
		{map[string]interface{}{"text": "hello", "type": "text"}, `{"text":"hello","type":"text"}`},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			result := parser.convertContentToString(tc.content)
			if tc.content != "simple string" && tc.content != 123 && tc.content != true {
				// For JSON marshaled content, we need to check if it's valid JSON
				var jsonCheck interface{}
				if err := json.Unmarshal([]byte(result), &jsonCheck); err != nil {
					t.Errorf("Result is not valid JSON: %s", result)
				}
			} else {
				if result != tc.expected {
					t.Errorf("Expected %s, got %s", tc.expected, result)
				}
			}
		})
	}
}

func TestInsertMessage(t *testing.T) {
	db, tokenService, sessionService := setupTestDBForJSONL(t)
	defer db.Close()

	parser := NewJSONLParser(db, tokenService, sessionService)

	// Create a test session first
	sessionID := "test-session-1"
	_, err := db.Exec(`
		INSERT INTO sessions (id, project_name, project_path, start_time) 
		VALUES (?, ?, ?, ?)
	`, sessionID, "test-project", "/test/path", time.Now())
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Test inserting a new message
	parentUUID := "parent-1"
	messageType := "text"
	model := "claude-3-sonnet"
	requestID := "req-1"
	
	message := &models.Message{
		ID:          "msg-1",
		SessionID:   sessionID,
		ParentUUID:  &parentUUID,
		IsSidechain: false,
		MessageType: &messageType,
		Model:       &model,
		Timestamp:   time.Now(),
		InputTokens: 100,
		OutputTokens: 200,
		RequestID:   &requestID,
	}

	content := "Test message content"
	message.Content = &content
	role := "user"
	message.MessageRole = &role

	err = parser.insertMessage(message)
	if err != nil {
		t.Fatalf("Failed to insert message: %v", err)
	}

	// Verify message was inserted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM messages WHERE id = ?", message.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query message count: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 message, got %d", count)
	}

	// Test updating the same message
	message.InputTokens = 150
	message.OutputTokens = 250

	err = parser.insertMessage(message)
	if err != nil {
		t.Fatalf("Failed to update message: %v", err)
	}

	// Verify message was updated
	var inputTokens, outputTokens int
	err = db.QueryRow("SELECT input_tokens, output_tokens FROM messages WHERE id = ?", message.ID).Scan(&inputTokens, &outputTokens)
	if err != nil {
		t.Fatalf("Failed to query updated message: %v", err)
	}
	if inputTokens != 150 || outputTokens != 250 {
		t.Errorf("Expected tokens 150/250, got %d/%d", inputTokens, outputTokens)
	}

	// Verify still only one message
	err = db.QueryRow("SELECT COUNT(*) FROM messages WHERE id = ?", message.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query message count after update: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 message after update, got %d", count)
	}
}

func TestProcessLogEntry(t *testing.T) {
	db, tokenService, sessionService := setupTestDBForJSONL(t)
	defer db.Close()

	parser := NewJSONLParser(db, tokenService, sessionService)

	// Create a test log entry
	parentUUID := "parent-uuid-1"
	requestID := "req-1"
	msgType := "text"
	model := "claude-3-sonnet"
	
	entry := &models.LogEntry{
		UUID:        "test-uuid-1",
		SessionID:   "test-session-1",
		ParentUUID:  &parentUUID,
		IsSidechain: false,
		UserType:    "human",
		Timestamp:   time.Now(),
		RequestID:   &requestID,
		Cwd:         "/Users/satoshi/git/testproject",
		Message: models.LogMessage{
			Type:  &msgType,
			Role:  "user",
			Model: &model,
			Content: "Test message content",
			Usage: &models.Usage{
				InputTokens:  100,
				OutputTokens: 200,
				ServiceTier:  "free",
			},
		},
	}

	err := parser.processLogEntry(entry, "fallback-project")
	if err != nil {
		t.Fatalf("Failed to process log entry: %v", err)
	}

	// Verify session was created
	var sessionCount int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = ?", entry.SessionID).Scan(&sessionCount)
	if err != nil {
		t.Fatalf("Failed to query session count: %v", err)
	}
	if sessionCount != 1 {
		t.Errorf("Expected 1 session, got %d", sessionCount)
	}

	// Verify message was inserted
	var messageCount int
	err = db.QueryRow("SELECT COUNT(*) FROM messages WHERE id = ?", entry.UUID).Scan(&messageCount)
	if err != nil {
		t.Fatalf("Failed to query message count: %v", err)
	}
	if messageCount != 1 {
		t.Errorf("Expected 1 message, got %d", messageCount)
	}

	// Verify project name was extracted from cwd
	var projectName string
	err = db.QueryRow("SELECT project_name FROM sessions WHERE id = ?", entry.SessionID).Scan(&projectName)
	if err != nil {
		t.Fatalf("Failed to query project name: %v", err)
	}
	if projectName != "testproject" {
		t.Errorf("Expected project name 'testproject', got '%s'", projectName)
	}
}

func TestProcessLogEntry_WithoutCwd(t *testing.T) {
	db, tokenService, sessionService := setupTestDBForJSONL(t)
	defer db.Close()

	parser := NewJSONLParser(db, tokenService, sessionService)

	// Create a test log entry without cwd
	parentUUID2 := "parent-uuid-2"
	requestID2 := "req-2"
	msgType2 := "text"
	model2 := "claude-3-sonnet"
	
	entry := &models.LogEntry{
		UUID:        "test-uuid-2",
		SessionID:   "test-session-2",
		ParentUUID:  &parentUUID2,
		IsSidechain: false,
		UserType:    "human",
		Timestamp:   time.Now(),
		RequestID:   &requestID2,
		Cwd:         "", // No cwd
		Message: models.LogMessage{
			Type:    &msgType2,
			Role:    "user",
			Model:   &model2,
			Content: "Test message without cwd",
			Usage: &models.Usage{
				InputTokens:  50,
				OutputTokens: 100,
				ServiceTier:  "free",
			},
		},
	}

	err := parser.processLogEntry(entry, "-Users-satoshi-git-fallback")
	if err != nil {
		t.Fatalf("Failed to process log entry: %v", err)
	}

	// Verify session was created with converted project name
	var projectName, projectPath string
	err = db.QueryRow("SELECT project_name, project_path FROM sessions WHERE id = ?", entry.SessionID).Scan(&projectName, &projectPath)
	if err != nil {
		t.Fatalf("Failed to query session info: %v", err)
	}
	if projectName != "-Users-satoshi-git-fallback" {
		t.Errorf("Expected project name '-Users-satoshi-git-fallback', got '%s'", projectName)
	}
	if projectPath != "/Users/satoshi/git/fallback" {
		t.Errorf("Expected project path '/Users/satoshi/git/fallback', got '%s'", projectPath)
	}
}

func TestParseJSONLFile(t *testing.T) {
	db, tokenService, sessionService := setupTestDBForJSONL(t)
	defer db.Close()

	parser := NewJSONLParser(db, tokenService, sessionService)

	// Create a temporary JSONL file
	tmpFile, err := os.CreateTemp("", "test-*.jsonl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write test data
	parent1 := "parent-1"
	req1 := "req-1"
	uuid1 := "uuid-1"
	textType := "text"
	model := "claude-3-sonnet"
	
	testData := []models.LogEntry{
		{
			UUID:        "uuid-1",
			SessionID:   "session-1",
			ParentUUID:  &parent1,
			IsSidechain: false,
			UserType:    "human",
			Timestamp:   time.Now(),
			RequestID:   &req1,
			Cwd:         "/Users/test/project1",
			Message: models.LogMessage{
				Type:    &textType,
				Role:    "user",
				Model:   &model,
				Content: "First message",
				Usage: &models.Usage{
					InputTokens:  100,
					OutputTokens: 0,
					ServiceTier:  "free",
				},
			},
		},
		{
			UUID:        "uuid-2",
			SessionID:   "session-1",
			ParentUUID:  &uuid1,
			IsSidechain: false,
			UserType:    "claude",
			Timestamp:   time.Now().Add(1 * time.Minute),
			RequestID:   &req1,
			Cwd:         "/Users/test/project1",
			Message: models.LogMessage{
				Type:    &textType,
				Role:    "assistant",
				Model:   &model,
				Content: "Response message",
				Usage: &models.Usage{
					InputTokens:  0,
					OutputTokens: 200,
					ServiceTier:  "free",
				},
			},
		},
	}

	for _, entry := range testData {
		jsonData, err := json.Marshal(entry)
		if err != nil {
			t.Fatalf("Failed to marshal test data: %v", err)
		}
		tmpFile.WriteString(string(jsonData) + "\n")
	}

	// Add an empty line to test line skipping
	tmpFile.WriteString("\n")

	// Add an invalid JSON line to test error handling
	tmpFile.WriteString("invalid json line\n")

	tmpFile.Close()

	// Parse the file
	err = parser.parseJSONLFile(tmpFile.Name(), "test-project")
	if err != nil {
		t.Fatalf("Failed to parse JSONL file: %v", err)
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

	// Verify messages were inserted
	var messageCount int
	err = db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&messageCount)
	if err != nil {
		t.Fatalf("Failed to query message count: %v", err)
	}
	if messageCount != 2 {
		t.Errorf("Expected 2 messages, got %d", messageCount)
	}

	// Verify project name was extracted correctly
	var projectName string
	err = db.QueryRow("SELECT project_name FROM sessions WHERE id = ?", "session-1").Scan(&projectName)
	if err != nil {
		t.Fatalf("Failed to query project name: %v", err)
	}
	if projectName != "project1" {
		t.Errorf("Expected project name 'project1', got '%s'", projectName)
	}
}

func TestSyncProjectLogs(t *testing.T) {
	db, tokenService, sessionService := setupTestDBForJSONL(t)
	defer db.Close()

	parser := NewJSONLParser(db, tokenService, sessionService)

	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "test-sync-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test JSONL files
	testFiles := []string{"session1.jsonl", "session2.jsonl", "other.txt"}
	
	for i, fileName := range testFiles {
		if strings.HasSuffix(fileName, ".jsonl") {
			filePath := filepath.Join(tmpDir, fileName)
			file, err := os.Create(filePath)
			if err != nil {
				t.Fatalf("Failed to create test file %s: %v", fileName, err)
			}

			// Write a simple test entry
			reqID := fmt.Sprintf("req-%d", i)
			msgType := "text"
			model := "claude-3-sonnet"
			
			entry := models.LogEntry{
				UUID:        fmt.Sprintf("uuid-%d", i),
				SessionID:   fmt.Sprintf("session-%d", i),
				UserType:    "human",
				Timestamp:   time.Now(),
				RequestID:   &reqID,
				Cwd:         fmt.Sprintf("/test/project%d", i),
				Message: models.LogMessage{
					Type:    &msgType,
					Role:    "user",
					Model:   &model,
					Content: fmt.Sprintf("Test message %d", i),
					Usage: &models.Usage{
						InputTokens:  100,
						OutputTokens: 0,
						ServiceTier:  "free",
					},
				},
			}

			jsonData, err := json.Marshal(entry)
			if err != nil {
				t.Fatalf("Failed to marshal test data: %v", err)
			}
			file.WriteString(string(jsonData) + "\n")
			file.Close()
		} else {
			// Create a non-JSONL file
			filePath := filepath.Join(tmpDir, fileName)
			file, err := os.Create(filePath)
			if err != nil {
				t.Fatalf("Failed to create test file %s: %v", fileName, err)
			}
			file.WriteString("This is not a JSONL file\n")
			file.Close()
		}
	}

	// Sync the project logs
	err = parser.syncProjectLogs(tmpDir, "test-project")
	if err != nil {
		t.Fatalf("Failed to sync project logs: %v", err)
	}

	// Verify only JSONL files were processed
	var sessionCount int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&sessionCount)
	if err != nil {
		t.Fatalf("Failed to query session count: %v", err)
	}
	if sessionCount != 2 {
		t.Errorf("Expected 2 sessions, got %d", sessionCount)
	}

	var messageCount int
	err = db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&messageCount)
	if err != nil {
		t.Fatalf("Failed to query message count: %v", err)
	}
	if messageCount != 2 {
		t.Errorf("Expected 2 messages, got %d", messageCount)
	}
}