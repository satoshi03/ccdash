package services

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb/v2"
)

func setupTestDB(t *testing.T) *sql.DB {
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
			message_role TEXT,
			content TEXT,
			timestamp TIMESTAMP,
			input_tokens INTEGER DEFAULT 0,
			output_tokens INTEGER DEFAULT 0,
			FOREIGN KEY (session_id) REFERENCES sessions(id)
		);
	`

	_, err = db.Exec(createTables)
	if err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	return db
}

func TestNewTokenService(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewTokenService(db)
	if service == nil {
		t.Error("NewTokenService returned nil")
	}
	if service.db != db {
		t.Error("TokenService db field not set correctly")
	}
}

func TestGetUsageLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewTokenService(db)
	
	limit := service.getUsageLimit()
	if limit != CLAUDE_PRO_LIMIT {
		t.Errorf("Expected usage limit %d, got %d", CLAUDE_PRO_LIMIT, limit)
	}
}

func TestGetCurrentTokenUsage_NoMessages(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewTokenService(db)
	
	usage, err := service.GetCurrentTokenUsage()
	if err != nil {
		t.Fatalf("GetCurrentTokenUsage failed: %v", err)
	}

	if usage.TotalTokens != 0 {
		t.Errorf("Expected total tokens 0, got %d", usage.TotalTokens)
	}
	if usage.InputTokens != 0 {
		t.Errorf("Expected input tokens 0, got %d", usage.InputTokens)
	}
	if usage.OutputTokens != 0 {
		t.Errorf("Expected output tokens 0, got %d", usage.OutputTokens)
	}
	if usage.UsageLimit != CLAUDE_PRO_LIMIT {
		t.Errorf("Expected usage limit %d, got %d", CLAUDE_PRO_LIMIT, usage.UsageLimit)
	}
	if usage.UsageRate != 0.0 {
		t.Errorf("Expected usage rate 0.0, got %f", usage.UsageRate)
	}
	if usage.ActiveSessions != 0 {
		t.Errorf("Expected active sessions 0, got %d", usage.ActiveSessions)
	}
}

func TestGetCurrentTokenUsage_WithMessages(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewTokenService(db)
	
	// Insert test session
	sessionID := "test-session-1"
	_, err := db.Exec(`
		INSERT INTO sessions (id, project_name, project_path, start_time) 
		VALUES (?, ?, ?, ?)
	`, sessionID, "test-project", "/path/to/test", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert test session: %v", err)
	}

	// Insert test messages within the current window
	now := time.Now()
	testMessages := []struct {
		id           string
		role         string
		inputTokens  int
		outputTokens int
		timestamp    time.Time
	}{
		{"msg1", "user", 100, 0, now.Add(-2 * time.Hour)},
		{"msg2", "assistant", 0, 200, now.Add(-2 * time.Hour)},
		{"msg3", "user", 150, 0, now.Add(-1 * time.Hour)},
		{"msg4", "assistant", 0, 300, now.Add(-1 * time.Hour)},
	}

	for _, msg := range testMessages {
		_, err := db.Exec(`
			INSERT INTO messages (id, session_id, message_role, content, timestamp, input_tokens, output_tokens) 
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, msg.id, sessionID, msg.role, "test content", msg.timestamp, msg.inputTokens, msg.outputTokens)
		if err != nil {
			t.Fatalf("Failed to insert test message: %v", err)
		}
	}

	usage, err := service.GetCurrentTokenUsage()
	if err != nil {
		t.Fatalf("GetCurrentTokenUsage failed: %v", err)
	}

	expectedInputTokens := 250  // 100 + 150
	expectedOutputTokens := 500 // 200 + 300
	expectedTotalTokens := 750  // 250 + 500

	if usage.InputTokens != expectedInputTokens {
		t.Errorf("Expected input tokens %d, got %d", expectedInputTokens, usage.InputTokens)
	}
	if usage.OutputTokens != expectedOutputTokens {
		t.Errorf("Expected output tokens %d, got %d", expectedOutputTokens, usage.OutputTokens)
	}
	if usage.TotalTokens != expectedTotalTokens {
		t.Errorf("Expected total tokens %d, got %d", expectedTotalTokens, usage.TotalTokens)
	}
	if usage.ActiveSessions != 1 {
		t.Errorf("Expected active sessions 1, got %d", usage.ActiveSessions)
	}

	expectedUsageRate := float64(expectedTotalTokens) / float64(CLAUDE_PRO_LIMIT)
	if usage.UsageRate != expectedUsageRate {
		t.Errorf("Expected usage rate %f, got %f", expectedUsageRate, usage.UsageRate)
	}
}

func TestGetCurrentTokenUsage_OutsideWindow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewTokenService(db)
	
	// Insert test session
	sessionID := "test-session-old"
	_, err := db.Exec(`
		INSERT INTO sessions (id, project_name, project_path, start_time) 
		VALUES (?, ?, ?, ?)
	`, sessionID, "test-project", "/path/to/test", time.Now().Add(-6*time.Hour))
	if err != nil {
		t.Fatalf("Failed to insert test session: %v", err)
	}

	// Insert message outside the 5-hour window
	oldTime := time.Now().Add(-6 * time.Hour)
	_, err = db.Exec(`
		INSERT INTO messages (id, session_id, message_role, content, timestamp, input_tokens, output_tokens) 
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "old-msg", sessionID, "assistant", "old content", oldTime, 1000, 2000)
	if err != nil {
		t.Fatalf("Failed to insert old message: %v", err)
	}

	usage, err := service.GetCurrentTokenUsage()
	if err != nil {
		t.Fatalf("GetCurrentTokenUsage failed: %v", err)
	}

	// Should not count tokens from messages outside the window
	if usage.TotalTokens != 0 {
		t.Errorf("Expected total tokens 0 for messages outside window, got %d", usage.TotalTokens)
	}
}

func TestGetTokenUsageBySession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewTokenService(db)
	
	// Insert test session
	sessionID := "test-session-specific"
	startTime := time.Now().Add(-2 * time.Hour)
	_, err := db.Exec(`
		INSERT INTO sessions (id, project_name, project_path, start_time) 
		VALUES (?, ?, ?, ?)
	`, sessionID, "test-project", "/path/to/test", startTime)
	if err != nil {
		t.Fatalf("Failed to insert test session: %v", err)
	}

	// Insert test messages for this session
	testMessages := []struct {
		id           string
		role         string
		inputTokens  int
		outputTokens int
		timestamp    time.Time
	}{
		{"msg1", "user", 100, 0, startTime},
		{"msg2", "assistant", 0, 200, startTime.Add(5 * time.Minute)},
		{"msg3", "user", 150, 0, startTime.Add(10 * time.Minute)},
		{"msg4", "assistant", 0, 300, startTime.Add(15 * time.Minute)},
	}

	for _, msg := range testMessages {
		_, err := db.Exec(`
			INSERT INTO messages (id, session_id, message_role, content, timestamp, input_tokens, output_tokens) 
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, msg.id, sessionID, msg.role, "test content", msg.timestamp, msg.inputTokens, msg.outputTokens)
		if err != nil {
			t.Fatalf("Failed to insert test message: %v", err)
		}
	}

	usage, err := service.GetTokenUsageBySession(sessionID)
	if err != nil {
		t.Fatalf("GetTokenUsageBySession failed: %v", err)
	}

	// Only assistant messages should be counted for session token usage
	expectedInputTokens := 0   // User messages are not counted
	expectedOutputTokens := 500 // 200 + 300 from assistant messages
	expectedTotalTokens := 500

	if usage.InputTokens != expectedInputTokens {
		t.Errorf("Expected input tokens %d, got %d", expectedInputTokens, usage.InputTokens)
	}
	if usage.OutputTokens != expectedOutputTokens {
		t.Errorf("Expected output tokens %d, got %d", expectedOutputTokens, usage.OutputTokens)
	}
	if usage.TotalTokens != expectedTotalTokens {
		t.Errorf("Expected total tokens %d, got %d", expectedTotalTokens, usage.TotalTokens)
	}
	if usage.ActiveSessions != 1 {
		t.Errorf("Expected active sessions 1, got %d", usage.ActiveSessions)
	}
}

func TestGetTokenUsageBySession_NonExistentSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewTokenService(db)
	
	usage, err := service.GetTokenUsageBySession("non-existent-session")
	if err == nil {
		t.Error("Expected error for non-existent session, got nil")
		return
	}

	if usage != nil {
		t.Errorf("Expected nil usage for non-existent session, got %v", usage)
	}
}

func TestUpdateSessionTokens(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewTokenService(db)
	
	// Insert test session
	sessionID := "test-session-update"
	_, err := db.Exec(`
		INSERT INTO sessions (id, project_name, project_path, start_time) 
		VALUES (?, ?, ?, ?)
	`, sessionID, "test-project", "/path/to/test", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert test session: %v", err)
	}

	// Insert test messages
	now := time.Now()
	testMessages := []struct {
		id           string
		role         string
		inputTokens  int
		outputTokens int
		timestamp    time.Time
	}{
		{"msg1", "user", 100, 0, now},
		{"msg2", "assistant", 0, 200, now.Add(1 * time.Minute)},
		{"msg3", "user", 150, 0, now.Add(2 * time.Minute)},
		{"msg4", "assistant", 0, 300, now.Add(3 * time.Minute)},
	}

	for _, msg := range testMessages {
		_, err := db.Exec(`
			INSERT INTO messages (id, session_id, message_role, content, timestamp, input_tokens, output_tokens) 
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, msg.id, sessionID, msg.role, "test content", msg.timestamp, msg.inputTokens, msg.outputTokens)
		if err != nil {
			t.Fatalf("Failed to insert test message: %v", err)
		}
	}

	// Update session tokens
	err = service.UpdateSessionTokens(sessionID)
	if err != nil {
		t.Fatalf("UpdateSessionTokens failed: %v", err)
	}

	// Verify the session was updated correctly
	var totalInputTokens, totalOutputTokens, totalTokens, messageCount int
	var endTime time.Time
	err = db.QueryRow(`
		SELECT total_input_tokens, total_output_tokens, total_tokens, message_count, end_time 
		FROM sessions WHERE id = ?
	`, sessionID).Scan(&totalInputTokens, &totalOutputTokens, &totalTokens, &messageCount, &endTime)
	if err != nil {
		t.Fatalf("Failed to query updated session: %v", err)
	}

	// Only assistant messages should be counted for token totals
	expectedInputTokens := 0   // User messages are not counted
	expectedOutputTokens := 500 // 200 + 300 from assistant messages
	expectedTotalTokens := 500
	expectedMessageCount := 4   // All messages are counted

	if totalInputTokens != expectedInputTokens {
		t.Errorf("Expected total input tokens %d, got %d", expectedInputTokens, totalInputTokens)
	}
	if totalOutputTokens != expectedOutputTokens {
		t.Errorf("Expected total output tokens %d, got %d", expectedOutputTokens, totalOutputTokens)
	}
	if totalTokens != expectedTotalTokens {
		t.Errorf("Expected total tokens %d, got %d", expectedTotalTokens, totalTokens)
	}
	if messageCount != expectedMessageCount {
		t.Errorf("Expected message count %d, got %d", expectedMessageCount, messageCount)
	}

	// Verify end time is set to the last message timestamp
	lastMessageTime := now.Add(3 * time.Minute)
	if !endTime.Equal(lastMessageTime) {
		t.Errorf("Expected end time %v, got %v", lastMessageTime, endTime)
	}
}

func TestConstants(t *testing.T) {
	if CLAUDE_PRO_LIMIT != 7000 {
		t.Errorf("Expected CLAUDE_PRO_LIMIT 7000, got %d", CLAUDE_PRO_LIMIT)
	}
	if CLAUDE_MAX5_LIMIT != 35000 {
		t.Errorf("Expected CLAUDE_MAX5_LIMIT 35000, got %d", CLAUDE_MAX5_LIMIT)
	}
	if CLAUDE_MAX20_LIMIT != 140000 {
		t.Errorf("Expected CLAUDE_MAX20_LIMIT 140000, got %d", CLAUDE_MAX20_LIMIT)
	}
	if WINDOW_DURATION != 5*time.Hour {
		t.Errorf("Expected WINDOW_DURATION 5h, got %v", WINDOW_DURATION)
	}
}

func TestRoundToNextHour(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	service := NewTokenService(db)
	
	testCases := []struct {
		name     string
		input    time.Time
		expected time.Time
	}{
		{
			name:     "8:30 -> 8:00 (round down)",
			input:    time.Date(2024, 1, 1, 8, 30, 0, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC),
		},
		{
			name:     "13:45 -> 13:00 (round down)",
			input:    time.Date(2024, 1, 1, 13, 45, 30, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
		},
		{
			name:     "10:00 -> 10:00 (already at hour)",
			input:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		},
		{
			name:     "23:59 -> 23:00 (round down)",
			input:    time.Date(2024, 1, 1, 23, 59, 59, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 23, 0, 0, 0, time.UTC),
		},
		{
			name:     "0:15 -> 0:00 (round down)",
			input:    time.Date(2024, 1, 1, 0, 15, 0, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := service.roundToNextHour(tc.input)
			if !result.Equal(tc.expected) {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestTokenResetTimeLogic(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	service := NewTokenService(db)
	
	// roundToNextHourの動作を直接テストするケース
	testCases := []struct {
		name                string
		messageTime         time.Time
		expectedResetTime   time.Time
	}{
		{
			name:                "8:30 message + 5h should reset at 13:00",
			messageTime:         time.Date(2024, 1, 1, 8, 30, 0, 0, time.UTC),
			expectedResetTime:   time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC), // 8:30 + 5h = 13:30 -> 13:00
		},
		{
			name:                "10:15 message + 5h should reset at 15:00",
			messageTime:         time.Date(2024, 1, 1, 10, 15, 0, 0, time.UTC),
			expectedResetTime:   time.Date(2024, 1, 1, 15, 0, 0, 0, time.UTC), // 15:15 -> 15:00
		},
		{
			name:                "14:45 message + 5h should reset at 19:00",
			messageTime:         time.Date(2024, 1, 1, 14, 45, 0, 0, time.UTC),
			expectedResetTime:   time.Date(2024, 1, 1, 19, 0, 0, 0, time.UTC), // 19:45 -> 19:00
		},
		{
			name:                "23:30 message + 5h should reset at 04:00 next day",
			messageTime:         time.Date(2024, 1, 1, 23, 30, 0, 0, time.UTC),
			expectedResetTime:   time.Date(2024, 1, 2, 4, 0, 0, 0, time.UTC), // 04:30 -> 04:00
		},
		{
			name:                "9:00 message + 5h should reset at 14:00",
			messageTime:         time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
			expectedResetTime:   time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC), // 14:00 -> 14:00
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// roundToNextHourの動作を直接テスト
			messageTimePlus5Hours := tc.messageTime.Add(WINDOW_DURATION)
			actualResetTime := service.roundToNextHour(messageTimePlus5Hours)
			
			if !actualResetTime.Equal(tc.expectedResetTime) {
				t.Errorf("Expected reset time %v, got %v", tc.expectedResetTime, actualResetTime)
			}
			
			// リセット時間が正時（0分）であることを確認
			if actualResetTime.Minute() != 0 {
				t.Errorf("Expected reset time to be at 0 minutes, got %d", actualResetTime.Minute())
			}
			
			t.Logf("Message at %v + 5h = %v -> Reset at %v", 
				tc.messageTime.Format("15:04"), 
				messageTimePlus5Hours.Format("15:04"), 
				actualResetTime.Format("15:04"))
		})
	}
}

func TestTokenResetTimeIntegration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	service := NewTokenService(db)
	
	// 現在時刻より1時間前のメッセージを作成（まだウィンドウ内）
	messageTime := time.Now().Add(-1 * time.Hour)
	
	// セッションを作成
	_, err := db.Exec(`
		INSERT INTO sessions (id, project_name, project_path, start_time) 
		VALUES (?, ?, ?, ?)
	`, "test-session", "test-project", "/test/path", messageTime)
	if err != nil {
		t.Fatalf("Failed to insert test session: %v", err)
	}
	
	// メッセージを挿入
	_, err = db.Exec(`
		INSERT INTO messages (id, session_id, message_role, content, timestamp, input_tokens, output_tokens) 
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "test-msg", "test-session", "assistant", "test", messageTime, 100, 200)
	if err != nil {
		t.Fatalf("Failed to insert test message: %v", err)
	}
	
	// GetCurrentTokenUsageを呼び出してリセット時間を確認
	usage, err := service.GetCurrentTokenUsage()
	if err != nil {
		t.Fatalf("Failed to get token usage: %v", err)
	}
	
	// WindowEndがリセット時間として設定されているかチェック
	if usage.WindowEnd.Before(usage.WindowStart) {
		t.Errorf("WindowEnd should be after WindowStart")
	}
	
	// WindowEndが正時（0分）であることを確認
	if usage.WindowEnd.Minute() != 0 {
		t.Errorf("Expected WindowEnd to be at 0 minutes, got %d", usage.WindowEnd.Minute())
	}
	
	// WindowEndが秒・ナノ秒も0であることを確認
	if usage.WindowEnd.Second() != 0 || usage.WindowEnd.Nanosecond() != 0 {
		t.Errorf("Expected WindowEnd to be exactly at hour boundary, got %v", usage.WindowEnd)
	}
	
	t.Logf("Message at %v -> Reset at %v", 
		messageTime.Format("15:04"), usage.WindowEnd.Format("15:04"))
}