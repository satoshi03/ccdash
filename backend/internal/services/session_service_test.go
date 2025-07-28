package services

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"ccdash-backend/internal/models"
	_ "github.com/marcboeker/go-duckdb"
)

func setupTestDBForSession(t *testing.T) *sql.DB {
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

	return db
}

func TestNewSessionService(t *testing.T) {
	db := setupTestDBForSession(t)
	defer db.Close()

	service := NewSessionService(db)
	
	if service == nil {
		t.Error("NewSessionService returned nil")
	}
	if service.db != db {
		t.Error("SessionService db field not set correctly")
	}
}

func TestCreateOrUpdateSession_CreateNew(t *testing.T) {
	db := setupTestDBForSession(t)
	defer db.Close()

	service := NewSessionService(db)
	
	sessionID := "test-session-1"
	projectName := "test-project"
	projectPath := "/test/path"
	messageTime := time.Now()

	err := service.CreateOrUpdateSession(sessionID, projectName, projectPath, messageTime)
	if err != nil {
		t.Fatalf("CreateOrUpdateSession failed: %v", err)
	}

	// Verify session was created
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = ?", sessionID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query session count: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 session, got %d", count)
	}

	// Verify session details
	var actualProjectName, actualProjectPath, status string
	var startTime time.Time
	err = db.QueryRow("SELECT project_name, project_path, start_time, status FROM sessions WHERE id = ?", sessionID).Scan(
		&actualProjectName, &actualProjectPath, &startTime, &status)
	if err != nil {
		t.Fatalf("Failed to query session details: %v", err)
	}

	if actualProjectName != projectName {
		t.Errorf("Expected project name %s, got %s", projectName, actualProjectName)
	}
	if actualProjectPath != projectPath {
		t.Errorf("Expected project path %s, got %s", projectPath, actualProjectPath)
	}
	if status != "active" {
		t.Errorf("Expected status 'active', got %s", status)
	}
	if !startTime.Equal(messageTime) {
		t.Errorf("Expected start time %v, got %v", messageTime, startTime)
	}
}

func TestCreateOrUpdateSession_UpdateExisting(t *testing.T) {
	db := setupTestDBForSession(t)
	defer db.Close()

	service := NewSessionService(db)
	
	sessionID := "test-session-2"
	projectName := "test-project"
	projectPath := "/test/path"
	originalTime := time.Now()

	// Create session first
	err := service.CreateOrUpdateSession(sessionID, projectName, projectPath, originalTime)
	if err != nil {
		t.Fatalf("Initial CreateOrUpdateSession failed: %v", err)
	}

	// Update with earlier time
	earlierTime := originalTime.Add(-1 * time.Hour)
	err = service.CreateOrUpdateSession(sessionID, projectName, projectPath, earlierTime)
	if err != nil {
		t.Fatalf("Update CreateOrUpdateSession failed: %v", err)
	}

	// Verify session count is still 1
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = ?", sessionID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query session count: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 session after update, got %d", count)
	}

	// Verify start time was updated to earlier time
	var startTime time.Time
	err = db.QueryRow("SELECT start_time FROM sessions WHERE id = ?", sessionID).Scan(&startTime)
	if err != nil {
		t.Fatalf("Failed to query start time: %v", err)
	}
	if !startTime.Equal(earlierTime) {
		t.Errorf("Expected start time %v, got %v", earlierTime, startTime)
	}

	// Update with later time (should not change)
	laterTime := originalTime.Add(1 * time.Hour)
	err = service.CreateOrUpdateSession(sessionID, projectName, projectPath, laterTime)
	if err != nil {
		t.Fatalf("Later time update failed: %v", err)
	}

	// Verify start time was not updated
	err = db.QueryRow("SELECT start_time FROM sessions WHERE id = ?", sessionID).Scan(&startTime)
	if err != nil {
		t.Fatalf("Failed to query start time after later update: %v", err)
	}
	if !startTime.Equal(earlierTime) {
		t.Errorf("Expected start time to remain %v, got %v", earlierTime, startTime)
	}
}

func TestIsSessionActive(t *testing.T) {
	db := setupTestDBForSession(t)
	defer db.Close()

	service := NewSessionService(db)
	
	now := time.Now()
	
	testCases := []struct {
		name         string
		session      models.Session
		lastActivity time.Time
		expected     bool
	}{
		{
			name: "Active session with recent activity",
			session: models.Session{
				Status: "active",
			},
			lastActivity: now.Add(-10 * time.Minute),
			expected:     true,
		},
		{
			name: "Active session with old activity",
			session: models.Session{
				Status: "active",
			},
			lastActivity: now.Add(-45 * time.Minute),
			expected:     false,
		},
		{
			name: "Completed session",
			session: models.Session{
				Status: "completed",
			},
			lastActivity: now.Add(-10 * time.Minute),
			expected:     false,
		},
		{
			name: "Failed session",
			session: models.Session{
				Status: "failed",
			},
			lastActivity: now.Add(-10 * time.Minute),
			expected:     false,
		},
		{
			name: "Session with end time",
			session: models.Session{
				Status:  "active",
				EndTime: &now,
			},
			lastActivity: now.Add(-10 * time.Minute),
			expected:     false,
		},
		{
			name: "Session with zero last activity",
			session: models.Session{
				Status: "active",
			},
			lastActivity: time.Time{},
			expected:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := service.isSessionActive(tc.session, tc.lastActivity)
			if result != tc.expected {
				t.Errorf("Expected %t, got %t", tc.expected, result)
			}
		})
	}
}

func TestExtractCodeFromContent(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name: "Single code block",
			content: "Here's some code:\n```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```",
			expected: []string{"func main() {\n    fmt.Println(\"Hello\")\n}"},
		},
		{
			name: "Multiple code blocks",
			content: "First block:\n```python\nprint(\"Hello\")\n```\nSecond block:\n```js\nconsole.log(\"World\")\n```",
			expected: []string{"print(\"Hello\")", "console.log(\"World\")"},
		},
		{
			name: "Code block with language",
			content: "```typescript\ninterface User {\n  name: string;\n}\n```",
			expected: []string{"interface User {\n  name: string;\n}"},
		},
		{
			name: "Code block without language",
			content: "```\nsome code here\n```",
			expected: []string{"some code here"},
		},
		{
			name:     "No code blocks",
			content:  "This is just regular text without any code blocks.",
			expected: []string{},
		},
		{
			name: "Empty code block",
			content: "```\n\n```",
			expected: []string{},
		},
		{
			name: "Code block with only whitespace",
			content: "```\n   \n\n  \n```",
			expected: []string{},
		},
		{
			name: "Mixed content",
			content: "Some text ```go\npackage main\n``` more text ```python\nprint(\"hi\")\n``` end",
			expected: []string{"package main", "print(\"hi\")"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractCodeFromContent(tc.content)
			if len(result) != len(tc.expected) {
				t.Errorf("Expected %d code blocks, got %d", len(tc.expected), len(result))
				return
			}
			for i, expected := range tc.expected {
				if result[i] != expected {
					t.Errorf("Expected code block %d to be %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}

func TestGetSessionMessages(t *testing.T) {
	db := setupTestDBForSession(t)
	defer db.Close()

	service := NewSessionService(db)
	
	sessionID := "test-session-messages"
	
	// Create test session
	_, err := db.Exec(`
		INSERT INTO sessions (id, project_name, project_path, start_time) 
		VALUES (?, ?, ?, ?)
	`, sessionID, "test-project", "/test/path", time.Now())
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Create test messages
	testMessages := []struct {
		id        string
		role      string
		content   string
		timestamp time.Time
		tokens    int
	}{
		{"msg1", "user", "First message", time.Now(), 100},
		{"msg2", "assistant", "Response", time.Now().Add(1 * time.Minute), 200},
		{"msg3", "user", "Second message", time.Now().Add(2 * time.Minute), 150},
	}

	for _, msg := range testMessages {
		_, err := db.Exec(`
			INSERT INTO messages (id, session_id, message_role, content, timestamp, input_tokens) 
			VALUES (?, ?, ?, ?, ?, ?)
		`, msg.id, sessionID, msg.role, msg.content, msg.timestamp, msg.tokens)
		if err != nil {
			t.Fatalf("Failed to insert test message: %v", err)
		}
	}

	// Get messages
	messages, err := service.GetSessionMessages(sessionID)
	if err != nil {
		t.Fatalf("GetSessionMessages failed: %v", err)
	}

	if len(messages) != len(testMessages) {
		t.Errorf("Expected %d messages, got %d", len(testMessages), len(messages))
	}

	// Verify messages are in timestamp order
	for i, msg := range messages {
		if msg.ID != testMessages[i].id {
			t.Errorf("Expected message ID %s, got %s", testMessages[i].id, msg.ID)
		}
		if msg.MessageRole != nil && *msg.MessageRole != testMessages[i].role {
			t.Errorf("Expected role %s, got %s", testMessages[i].role, *msg.MessageRole)
		}
		if msg.Content != nil && *msg.Content != testMessages[i].content {
			t.Errorf("Expected content %s, got %s", testMessages[i].content, *msg.Content)
		}
	}
}

func TestGetSessionMessagesPaginated(t *testing.T) {
	db := setupTestDBForSession(t)
	defer db.Close()

	service := NewSessionService(db)
	
	sessionID := "test-session-paginated"
	
	// Create test session
	_, err := db.Exec(`
		INSERT INTO sessions (id, project_name, project_path, start_time) 
		VALUES (?, ?, ?, ?)
	`, sessionID, "test-project", "/test/path", time.Now())
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Create 25 test messages
	baseTime := time.Now()
	for i := 0; i < 25; i++ {
		_, err := db.Exec(`
			INSERT INTO messages (id, session_id, message_role, content, timestamp, input_tokens) 
			VALUES (?, ?, ?, ?, ?, ?)
		`, 
		fmt.Sprintf("msg%d", i), 
		sessionID, 
		"user", 
		fmt.Sprintf("Message %d", i), 
		baseTime.Add(time.Duration(i) * time.Minute), 
		100)
		if err != nil {
			t.Fatalf("Failed to insert test message %d: %v", i, err)
		}
	}

	// Test first page
	result, err := service.GetSessionMessagesPaginated(sessionID, 1, 10)
	if err != nil {
		t.Fatalf("GetSessionMessagesPaginated failed: %v", err)
	}

	if result.Total != 25 {
		t.Errorf("Expected total 25, got %d", result.Total)
	}
	if result.Page != 1 {
		t.Errorf("Expected page 1, got %d", result.Page)
	}
	if result.PageSize != 10 {
		t.Errorf("Expected page size 10, got %d", result.PageSize)
	}
	if result.TotalPages != 3 {
		t.Errorf("Expected total pages 3, got %d", result.TotalPages)
	}
	if !result.HasNext {
		t.Error("Expected HasNext to be true")
	}
	if result.HasPrevious {
		t.Error("Expected HasPrevious to be false")
	}
	if len(result.Messages) != 10 {
		t.Errorf("Expected 10 messages, got %d", len(result.Messages))
	}

	// Test second page
	result, err = service.GetSessionMessagesPaginated(sessionID, 2, 10)
	if err != nil {
		t.Fatalf("GetSessionMessagesPaginated page 2 failed: %v", err)
	}

	if result.Page != 2 {
		t.Errorf("Expected page 2, got %d", result.Page)
	}
	if !result.HasNext {
		t.Error("Expected HasNext to be true")
	}
	if !result.HasPrevious {
		t.Error("Expected HasPrevious to be true")
	}
	if len(result.Messages) != 10 {
		t.Errorf("Expected 10 messages, got %d", len(result.Messages))
	}

	// Test last page
	result, err = service.GetSessionMessagesPaginated(sessionID, 3, 10)
	if err != nil {
		t.Fatalf("GetSessionMessagesPaginated page 3 failed: %v", err)
	}

	if result.Page != 3 {
		t.Errorf("Expected page 3, got %d", result.Page)
	}
	if result.HasNext {
		t.Error("Expected HasNext to be false")
	}
	if !result.HasPrevious {
		t.Error("Expected HasPrevious to be true")
	}
	if len(result.Messages) != 5 {
		t.Errorf("Expected 5 messages, got %d", len(result.Messages))
	}
}

func TestGetSessionMessagesPaginated_EdgeCases(t *testing.T) {
	db := setupTestDBForSession(t)
	defer db.Close()

	service := NewSessionService(db)
	
	sessionID := "test-session-edge"
	
	// Create test session
	_, err := db.Exec(`
		INSERT INTO sessions (id, project_name, project_path, start_time) 
		VALUES (?, ?, ?, ?)
	`, sessionID, "test-project", "/test/path", time.Now())
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Test with page < 1
	result, err := service.GetSessionMessagesPaginated(sessionID, 0, 10)
	if err != nil {
		t.Fatalf("GetSessionMessagesPaginated with page 0 failed: %v", err)
	}
	if result.Page != 1 {
		t.Errorf("Expected page to be corrected to 1, got %d", result.Page)
	}

	// Test with pageSize > 100
	result, err = service.GetSessionMessagesPaginated(sessionID, 1, 150)
	if err != nil {
		t.Fatalf("GetSessionMessagesPaginated with large page size failed: %v", err)
	}
	if result.PageSize != 20 {
		t.Errorf("Expected page size to be corrected to 20, got %d", result.PageSize)
	}

	// Test with pageSize < 1
	result, err = service.GetSessionMessagesPaginated(sessionID, 1, 0)
	if err != nil {
		t.Fatalf("GetSessionMessagesPaginated with page size 0 failed: %v", err)
	}
	if result.PageSize != 20 {
		t.Errorf("Expected page size to be corrected to 20, got %d", result.PageSize)
	}
}

func TestExtractGeneratedCode(t *testing.T) {
	db := setupTestDBForSession(t)
	defer db.Close()

	service := NewSessionService(db)
	
	sessionID := "test-session-code"
	
	// Create test session
	_, err := db.Exec(`
		INSERT INTO sessions (id, project_name, project_path, start_time) 
		VALUES (?, ?, ?, ?)
	`, sessionID, "test-project", "/test/path", time.Now())
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Insert messages with code blocks
	testMessages := []struct {
		id      string
		role    string
		content string
	}{
		{"msg1", "user", "Please write a function"},
		{"msg2", "assistant", "Here's a function:\n```go\nfunc hello() {\n    fmt.Println(\"Hello\")\n}\n```"},
		{"msg3", "user", "Can you add another one?"},
		{"msg4", "assistant", "Sure:\n```python\ndef greet():\n    print(\"Hi\")\n```\nAnd also:\n```js\nfunction say() {\n    console.log(\"Hey\");\n}\n```"},
		{"msg5", "assistant", "This has no code blocks"},
	}

	for i, msg := range testMessages {
		_, err := db.Exec(`
			INSERT INTO messages (id, session_id, message_role, content, timestamp) 
			VALUES (?, ?, ?, ?, ?)
		`, msg.id, sessionID, msg.role, msg.content, time.Now().Add(time.Duration(i)*time.Minute))
		if err != nil {
			t.Fatalf("Failed to insert test message: %v", err)
		}
	}

	// Extract generated code
	codeBlocks, err := service.extractGeneratedCode(sessionID)
	if err != nil {
		t.Fatalf("extractGeneratedCode failed: %v", err)
	}

	expectedCodeBlocks := []string{
		"func hello() {\n    fmt.Println(\"Hello\")\n}",
		"def greet():\n    print(\"Hi\")",
		"function say() {\n    console.log(\"Hey\");\n}",
	}

	if len(codeBlocks) != len(expectedCodeBlocks) {
		t.Errorf("Expected %d code blocks, got %d", len(expectedCodeBlocks), len(codeBlocks))
	}

	for i, expected := range expectedCodeBlocks {
		if i < len(codeBlocks) && codeBlocks[i] != expected {
			t.Errorf("Expected code block %d to be %q, got %q", i, expected, codeBlocks[i])
		}
	}
}

func TestGetSessionByID(t *testing.T) {
	db := setupTestDBForSession(t)
	defer db.Close()

	service := NewSessionService(db)
	
	sessionID := "test-session-byid"
	startTime := time.Now()
	
	// Create test session
	_, err := db.Exec(`
		INSERT INTO sessions (id, project_name, project_path, start_time, total_tokens) 
		VALUES (?, ?, ?, ?, ?)
	`, sessionID, "test-project", "/test/path", startTime, 500)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Insert a message for last activity
	_, err = db.Exec(`
		INSERT INTO messages (id, session_id, message_role, content, timestamp) 
		VALUES (?, ?, ?, ?, ?)
	`, "msg1", sessionID, "assistant", "Test message", startTime.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("Failed to insert test message: %v", err)
	}

	// Get session by ID
	session, err := service.GetSessionByID(sessionID)
	if err != nil {
		t.Fatalf("GetSessionByID failed: %v", err)
	}

	if session.ID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, session.ID)
	}
	if session.ProjectName != "test-project" {
		t.Errorf("Expected project name 'test-project', got %s", session.ProjectName)
	}
	if session.TotalTokens != 500 {
		t.Errorf("Expected total tokens 500, got %d", session.TotalTokens)
	}
	if session.Duration == nil {
		t.Error("Expected duration to be set")
	}
	if len(session.GeneratedCode) != 0 {
		t.Errorf("Expected generated code to be empty, got %d items", len(session.GeneratedCode))
	}
}

func TestGetSessionByID_NotFound(t *testing.T) {
	db := setupTestDBForSession(t)
	defer db.Close()

	service := NewSessionService(db)
	
	_, err := service.GetSessionByID("non-existent-session")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
	if err == nil {
		t.Error("Expected error for non-existent session, got nil")
	}
}