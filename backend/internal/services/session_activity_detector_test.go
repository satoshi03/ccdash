package services

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"claudeee-backend/internal/models"
	_ "github.com/marcboeker/go-duckdb/v2"
)

func setupTestDBForActivity(t *testing.T) (*sql.DB, *SessionActivityDetector) {
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

	detector := NewSessionActivityDetector(db)
	return db, detector
}

func TestNewSessionActivityDetector(t *testing.T) {
	db, detector := setupTestDBForActivity(t)
	defer db.Close()

	if detector == nil {
		t.Error("NewSessionActivityDetector returned nil")
	}
	if detector.db != db {
		t.Error("SessionActivityDetector db field not set correctly")
	}
}

func TestCalculateMessageScore(t *testing.T) {
	db, detector := setupTestDBForActivity(t)
	defer db.Close()

	testCases := []struct {
		name         string
		lastActivity time.Time
		expectedMin  float64
		expectedMax  float64
	}{
		{
			name:         "Very recent activity",
			lastActivity: time.Now().Add(-2 * time.Minute),
			expectedMin:  0.8,
			expectedMax:  1.0,
		},
		{
			name:         "Recent activity",
			lastActivity: time.Now().Add(-10 * time.Minute),
			expectedMin:  0.7,
			expectedMax:  0.9,
		},
		{
			name:         "Moderate activity",
			lastActivity: time.Now().Add(-25 * time.Minute),
			expectedMin:  0.4,
			expectedMax:  0.6,
		},
		{
			name:         "Old activity",
			lastActivity: time.Now().Add(-45 * time.Minute),
			expectedMin:  0.1,
			expectedMax:  0.3,
		},
		{
			name:         "Very old activity",
			lastActivity: time.Now().Add(-90 * time.Minute),
			expectedMin:  0.0,
			expectedMax:  0.1,
		},
		{
			name:         "Zero time",
			lastActivity: time.Time{},
			expectedMin:  0.0,
			expectedMax:  0.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := detector.calculateMessageScore("test-session", tc.lastActivity)
			if score < tc.expectedMin || score > tc.expectedMax {
				t.Errorf("Expected score between %f and %f, got %f", tc.expectedMin, tc.expectedMax, score)
			}
		})
	}
}

func TestAnalyzeMessagePattern(t *testing.T) {
	db, detector := setupTestDBForActivity(t)
	defer db.Close()

	sessionID := "test-session-pattern"
	now := time.Now()

	// Insert test messages
	testMessages := []struct {
		id        string
		msgType   string
		msgRole   string
		timestamp time.Time
	}{
		{"msg1", "text", "user", now.Add(-10 * time.Minute)},
		{"msg2", "text", "assistant", now.Add(-9 * time.Minute)},
		{"msg3", "tool_call", "assistant", now.Add(-8 * time.Minute)},
		{"msg4", "tool_result", "user", now.Add(-7 * time.Minute)},
		{"msg5", "text", "user", now.Add(-2 * time.Minute)},
	}

	for _, msg := range testMessages {
		_, err := db.Exec(`
			INSERT INTO messages (id, session_id, message_type, message_role, timestamp, content) 
			VALUES (?, ?, ?, ?, ?, ?)
		`, msg.id, sessionID, msg.msgType, msg.msgRole, msg.timestamp, "test content")
		if err != nil {
			t.Fatalf("Failed to insert test message: %v", err)
		}
	}

	// Analyze pattern
	pattern, err := detector.analyzeMessagePattern(sessionID)
	if err != nil {
		t.Fatalf("Failed to analyze message pattern: %v", err)
	}

	// Verify pattern analysis
	if pattern.MessageCount != 5 {
		t.Errorf("Expected message count 5, got %d", pattern.MessageCount)
	}

	if pattern.LastMessageType == nil || *pattern.LastMessageType != "text" {
		t.Errorf("Expected last message type 'text', got %v", pattern.LastMessageType)
	}

	if pattern.LastMessageRole == nil || *pattern.LastMessageRole != "user" {
		t.Errorf("Expected last message role 'user', got %v", pattern.LastMessageRole)
	}

	if pattern.HasPendingToolCall {
		t.Error("Expected no pending tool calls")
	}

	if pattern.TimeSinceLastUser < 1*time.Minute || pattern.TimeSinceLastUser > 5*time.Minute {
		t.Errorf("Expected time since last user to be around 2 minutes, got %v", pattern.TimeSinceLastUser)
	}
}

func TestAnalyzeMessagePatternWithPendingToolCall(t *testing.T) {
	db, detector := setupTestDBForActivity(t)
	defer db.Close()

	sessionID := "test-session-pending-tool"
	now := time.Now()

	// Insert test messages with pending tool call
	testMessages := []struct {
		id        string
		msgType   string
		msgRole   string
		timestamp time.Time
	}{
		{"msg1", "text", "user", now.Add(-10 * time.Minute)},
		{"msg2", "text", "assistant", now.Add(-9 * time.Minute)},
		{"msg3", "tool_call", "assistant", now.Add(-8 * time.Minute)},
		{"msg4", "tool_call", "assistant", now.Add(-7 * time.Minute)},
		{"msg5", "tool_result", "user", now.Add(-6 * time.Minute)},
		// Missing tool_result for the second tool_call
	}

	for _, msg := range testMessages {
		_, err := db.Exec(`
			INSERT INTO messages (id, session_id, message_type, message_role, timestamp, content) 
			VALUES (?, ?, ?, ?, ?, ?)
		`, msg.id, sessionID, msg.msgType, msg.msgRole, msg.timestamp, "test content")
		if err != nil {
			t.Fatalf("Failed to insert test message: %v", err)
		}
	}

	// Analyze pattern
	pattern, err := detector.analyzeMessagePattern(sessionID)
	if err != nil {
		t.Fatalf("Failed to analyze message pattern: %v", err)
	}

	// Should detect pending tool call
	if !pattern.HasPendingToolCall {
		t.Error("Expected pending tool call to be detected")
	}
}

func TestCalculateRecommendedTimeout(t *testing.T) {
	db, detector := setupTestDBForActivity(t)
	defer db.Close()

	sessionID := "test-session-timeout"
	now := time.Now()

	// Insert messages with varying intervals
	testMessages := []struct {
		id        string
		timestamp time.Time
	}{
		{"msg1", now.Add(-50 * time.Minute)},
		{"msg2", now.Add(-40 * time.Minute)}, // 10 min interval
		{"msg3", now.Add(-30 * time.Minute)}, // 10 min interval
		{"msg4", now.Add(-20 * time.Minute)}, // 10 min interval
		{"msg5", now.Add(-10 * time.Minute)}, // 10 min interval
	}

	for _, msg := range testMessages {
		_, err := db.Exec(`
			INSERT INTO messages (id, session_id, message_type, message_role, timestamp, content) 
			VALUES (?, ?, ?, ?, ?, ?)
		`, msg.id, sessionID, "text", "user", msg.timestamp, "test content")
		if err != nil {
			t.Fatalf("Failed to insert test message: %v", err)
		}
	}

	session := models.Session{
		ID:          sessionID,
		ProjectName: "test-project",
		ProjectPath: "/test/path",
	}

	timeout := detector.calculateRecommendedTimeout(sessionID, session)

	// With 10-minute average intervals, expected timeout should be around 30 minutes
	expectedMin := 25 * time.Minute
	expectedMax := 35 * time.Minute

	if timeout < expectedMin || timeout > expectedMax {
		t.Errorf("Expected timeout between %v and %v, got %v", expectedMin, expectedMax, timeout)
	}
}

func TestGetAverageMessageInterval(t *testing.T) {
	db, detector := setupTestDBForActivity(t)
	defer db.Close()

	sessionID := "test-session-interval"
	now := time.Now()

	// Insert messages with regular 5-minute intervals
	for i := 0; i < 5; i++ {
		_, err := db.Exec(`
			INSERT INTO messages (id, session_id, message_type, message_role, timestamp, content) 
			VALUES (?, ?, ?, ?, ?, ?)
		`, 
		fmt.Sprintf("msg%d", i), 
		sessionID, 
		"text", 
		"user", 
		now.Add(-time.Duration(i*5)*time.Minute),
		"test content")
		if err != nil {
			t.Fatalf("Failed to insert test message: %v", err)
		}
	}

	avgInterval := detector.getAverageMessageInterval(sessionID)

	// Expected average interval should be around 5 minutes
	expectedMin := 4 * time.Minute
	expectedMax := 6 * time.Minute

	if avgInterval < expectedMin || avgInterval > expectedMax {
		t.Errorf("Expected average interval between %v and %v, got %v", expectedMin, expectedMax, avgInterval)
	}
}

func TestCalculateActivityScore(t *testing.T) {
	db, detector := setupTestDBForActivity(t)
	defer db.Close()

	sessionID := "test-session-score"
	now := time.Now()

	// Create a session
	session := models.Session{
		ID:          sessionID,
		ProjectName: "test-project",
		ProjectPath: "/test/path",
		StartTime:   now.Add(-1 * time.Hour),
		Status:      "active",
	}

	// Insert recent messages
	_, err := db.Exec(`
		INSERT INTO messages (id, session_id, message_type, message_role, timestamp, content) 
		VALUES (?, ?, ?, ?, ?, ?)
	`, "msg1", sessionID, "text", "user", now.Add(-2*time.Minute), "test content")
	if err != nil {
		t.Fatalf("Failed to insert test message: %v", err)
	}

	lastActivity := now.Add(-2 * time.Minute)

	// Calculate activity score
	score := detector.CalculateActivityScore(sessionID, session, lastActivity)

	// Verify score structure
	if score.TotalScore < 0 || score.TotalScore > 1 {
		t.Errorf("Expected total score between 0 and 1, got %f", score.TotalScore)
	}

	if score.MessageScore <= 0 {
		t.Error("Expected message score to be positive for recent activity")
	}

	// The session should likely be considered active with recent messages
	// Note: This depends on process and file scores, so we don't assert IsActive directly
}

func TestGetSessionActivityReport(t *testing.T) {
	db, detector := setupTestDBForActivity(t)
	defer db.Close()

	sessionID := "test-session-report"
	now := time.Now()

	// Create a session
	session := models.Session{
		ID:          sessionID,
		ProjectName: "test-project",
		ProjectPath: "/test/path",
		StartTime:   now.Add(-1 * time.Hour),
		Status:      "active",
	}

	// Insert recent messages
	_, err := db.Exec(`
		INSERT INTO messages (id, session_id, message_type, message_role, timestamp, content) 
		VALUES (?, ?, ?, ?, ?, ?)
	`, "msg1", sessionID, "text", "user", now.Add(-5*time.Minute), "test content")
	if err != nil {
		t.Fatalf("Failed to insert test message: %v", err)
	}

	lastActivity := now.Add(-5 * time.Minute)

	// Get activity report
	report := detector.GetSessionActivityReport(sessionID, session, lastActivity)

	// Verify report structure
	if report["session_id"] != sessionID {
		t.Errorf("Expected session_id %s, got %v", sessionID, report["session_id"])
	}

	if _, exists := report["is_active"]; !exists {
		t.Error("Expected is_active field in report")
	}

	if _, exists := report["total_score"]; !exists {
		t.Error("Expected total_score field in report")
	}

	if scores, exists := report["scores"]; !exists {
		t.Error("Expected scores field in report")
	} else {
		scoresMap := scores.(map[string]float64)
		expectedScores := []string{"process", "file", "message", "pattern"}
		for _, scoreType := range expectedScores {
			if _, exists := scoresMap[scoreType]; !exists {
				t.Errorf("Expected %s score in report", scoreType)
			}
		}
	}

	if _, exists := report["recommended_timeout"]; !exists {
		t.Error("Expected recommended_timeout field in report")
	}

	if _, exists := report["pattern"]; !exists {
		t.Error("Expected pattern field in report")
	}
}

func TestDetermineInactiveReason(t *testing.T) {
	db, detector := setupTestDBForActivity(t)
	defer db.Close()

	testCases := []struct {
		name           string
		score          SessionActivityScore
		expectedReason string
	}{
		{
			name: "No process running",
			score: SessionActivityScore{
				ProcessScore: 0,
				FileScore:    0.5,
				MessageScore: 0.5,
				PatternScore: 0.5,
			},
			expectedReason: "No Claude process running",
		},
		{
			name: "No file activity",
			score: SessionActivityScore{
				ProcessScore: 0.5,
				FileScore:    0,
				MessageScore: 0.5,
				PatternScore: 0.5,
			},
			expectedReason: "No recent file activity",
		},
		{
			name: "No recent messages",
			score: SessionActivityScore{
				ProcessScore: 0.5,
				FileScore:    0.5,
				MessageScore: 0,
				PatternScore: 0.5,
			},
			expectedReason: "No recent messages",
		},
		{
			name: "Pattern suggests completion",
			score: SessionActivityScore{
				ProcessScore: 0.5,
				FileScore:    0.5,
				MessageScore: 0.5,
				PatternScore: 0,
			},
			expectedReason: "Message pattern suggests completion",
		},
		{
			name: "Low overall score",
			score: SessionActivityScore{
				ProcessScore: 0.1,
				FileScore:    0.1,
				MessageScore: 0.1,
				PatternScore: 0.1,
			},
			expectedReason: "Overall activity score too low",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reason := detector.determineInactiveReason(tc.score)
			if reason != tc.expectedReason {
				t.Errorf("Expected reason '%s', got '%s'", tc.expectedReason, reason)
			}
		})
	}
}