package services

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SessionWindowService struct {
	db *sql.DB
}

type SessionWindow struct {
	ID                  string    `json:"id"`
	WindowStart         time.Time `json:"window_start"`
	WindowEnd           time.Time `json:"window_end"`
	ResetTime           time.Time `json:"reset_time"`
	TotalInputTokens    int       `json:"total_input_tokens"`
	TotalOutputTokens   int       `json:"total_output_tokens"`
	TotalTokens         int       `json:"total_tokens"`
	MessageCount        int       `json:"message_count"`
	SessionCount        int       `json:"session_count"`
	IsActive            bool      `json:"is_active"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func NewSessionWindowService(db *sql.DB) *SessionWindowService {
	return &SessionWindowService{db: db}
}

// GetCurrentActiveWindow returns the currently active session window
func (s *SessionWindowService) GetCurrentActiveWindow() (*SessionWindow, error) {
	query := `
		SELECT 
			id, window_start, window_end, reset_time,
			total_input_tokens, total_output_tokens, total_tokens,
			message_count, session_count, is_active,
			created_at, updated_at
		FROM session_windows 
		WHERE is_active = true
		ORDER BY window_start DESC
		LIMIT 1
	`
	
	var window SessionWindow
	err := s.db.QueryRow(query).Scan(
		&window.ID,
		&window.WindowStart,
		&window.WindowEnd,
		&window.ResetTime,
		&window.TotalInputTokens,
		&window.TotalOutputTokens,
		&window.TotalTokens,
		&window.MessageCount,
		&window.SessionCount,
		&window.IsActive,
		&window.CreatedAt,
		&window.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get current active window: %w", err)
	}
	
	return &window, nil
}

// findWindowForTime finds an existing window that contains the given time
func (s *SessionWindowService) findWindowForTime(messageTime time.Time) (*SessionWindow, error) {
	query := `
		SELECT 
			id, window_start, window_end, reset_time,
			total_input_tokens, total_output_tokens, total_tokens,
			message_count, session_count, is_active,
			created_at, updated_at
		FROM session_windows 
		WHERE ? >= window_start AND ? < window_end
		ORDER BY window_start DESC
		LIMIT 1
	`
	
	var window SessionWindow
	err := s.db.QueryRow(query, messageTime, messageTime).Scan(
		&window.ID,
		&window.WindowStart,
		&window.WindowEnd,
		&window.ResetTime,
		&window.TotalInputTokens,
		&window.TotalOutputTokens,
		&window.TotalTokens,
		&window.MessageCount,
		&window.SessionCount,
		&window.IsActive,
		&window.CreatedAt,
		&window.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil // No existing window found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find window for time: %w", err)
	}
	
	return &window, nil
}

// RecalculateAllWindows recreates all session windows based on the specification
func (s *SessionWindowService) RecalculateAllWindows() error {
	// 1. 既存のSessionWindowを全てクリア
	_, err := s.db.Exec("DELETE FROM session_windows")
	if err != nil {
		return fmt.Errorf("failed to clear existing windows: %w", err)
	}
	
	for {
		// 2. SessionWindowに含まれていない最古のメッセージを取得
		oldestMessage, err := s.getOldestUnassignedMessage()
		if err != nil {
			return fmt.Errorf("failed to get oldest unassigned message: %w", err)
		}
		
		// メッセージがなければ完了
		if oldestMessage == nil {
			break
		}
		
		// 3. そのメッセージの時刻から5時間のSessionWindowを作成（分単位切り捨て）
		windowStart := s.truncateToMinute(oldestMessage.Timestamp)
		windowEnd := windowStart.Add(WINDOW_DURATION)
		
		window := &SessionWindow{
			ID:          uuid.New().String(),
			WindowStart: windowStart,
			WindowEnd:   windowEnd,
			ResetTime:   windowEnd,
			IsActive:    true,
		}
		
		// 4. SessionWindowをデータベースに挿入
		err = s.insertWindow(window)
		if err != nil {
			return fmt.Errorf("failed to insert window: %w", err)
		}
		
		// 5. この時間範囲内のメッセージにSessionWindowを割り当て
		err = s.assignMessagesToWindow(window.ID, windowStart, windowEnd)
		if err != nil {
			return fmt.Errorf("failed to assign messages to window: %w", err)
		}
		
		// 6. ウィンドウ統計を更新
		err = s.UpdateWindowStats(window.ID)
		if err != nil {
			return fmt.Errorf("failed to update window stats: %w", err)
		}
	}
	
	return nil
}

// getOldestUnassignedMessage gets the oldest message not assigned to any session window
func (s *SessionWindowService) getOldestUnassignedMessage() (*Message, error) {
	query := `
		SELECT id, session_id, timestamp
		FROM messages 
		WHERE session_window_id IS NULL
		ORDER BY timestamp ASC
		LIMIT 1
	`
	
	var message Message
	err := s.db.QueryRow(query).Scan(
		&message.ID,
		&message.SessionID,
		&message.Timestamp,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil // No unassigned messages
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get oldest unassigned message: %w", err)
	}
	
	return &message, nil
}

// truncateToMinute truncates time to minute precision (removes seconds and nanoseconds)
func (s *SessionWindowService) truncateToMinute(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, t.Location())
}

// truncateToHour truncates time to hour precision (removes minutes, seconds and nanoseconds)
func (s *SessionWindowService) truncateToHour(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
}

// insertWindow inserts a session window into the database
func (s *SessionWindowService) insertWindow(window *SessionWindow) error {
	query := `
		INSERT INTO session_windows (
			id, window_start, window_end, reset_time, is_active
		) VALUES (?, ?, ?, ?, ?)
	`
	
	_, err := s.db.Exec(query,
		window.ID,
		window.WindowStart,
		window.WindowEnd,
		window.ResetTime,
		window.IsActive,
	)
	
	return err
}

// assignMessagesToWindow assigns all messages in the time range to the given window
func (s *SessionWindowService) assignMessagesToWindow(windowID string, windowStart, windowEnd time.Time) error {
	query := `
		UPDATE messages 
		SET session_window_id = ? 
		WHERE timestamp >= ? AND timestamp < ? AND session_window_id IS NULL
	`
	
	_, err := s.db.Exec(query, windowID, windowStart, windowEnd)
	if err != nil {
		return fmt.Errorf("failed to assign messages to window: %w", err)
	}
	
	return nil
}

// Message represents a message for window calculation
type Message struct {
	ID        string
	SessionID string
	Timestamp time.Time
}

// GetOrCreateWindowForMessage gets the appropriate window for a message, creating if necessary
func (s *SessionWindowService) GetOrCreateWindowForMessage(messageTime time.Time) (*SessionWindow, error) {
	// このメッセージの時間に適合する既存のウィンドウがあるかチェック
	existingWindow, err := s.findWindowForTime(messageTime)
	if err != nil {
		return nil, err
	}
	
	if existingWindow != nil {
		return existingWindow, nil
	}
	
	// 適合するウィンドウがない場合、このメッセージ時間を基準にウィンドウを作成
	windowStart := s.truncateToMinute(messageTime)
	tempWindowEnd := windowStart.Add(WINDOW_DURATION)
	// WindowEndも分単位を切り捨てて時間単位にする（例：10:20 -> 10:00）
	windowEnd := s.truncateToHour(tempWindowEnd)
	
	// 同じ時間範囲のウィンドウが既に存在するかチェック（競合状態回避）
	existingWindow, err = s.findWindowForTime(windowStart)
	if err != nil {
		return nil, err
	}
	if existingWindow != nil {
		return existingWindow, nil
	}
	
	// 新しいウィンドウを作成
	// ResetTimeはWindowEndと同じ（両方とも時間単位で切り捨て）
	resetTime := windowEnd
	
	window := &SessionWindow{
		ID:          uuid.New().String(),
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
		ResetTime:   resetTime,
		IsActive:    true,
	}
	
	err = s.insertWindow(window)
	if err != nil {
		return nil, fmt.Errorf("failed to insert window: %w", err)
	}
	
	return window, nil
}


// UpdateWindowStats recalculates and updates the statistics for a window using time-based calculation
func (s *SessionWindowService) UpdateWindowStats(windowID string) error {
	// First get the window time range
	var windowStart, windowEnd time.Time
	err := s.db.QueryRow(`
		SELECT window_start, window_end FROM session_windows WHERE id = ?
	`, windowID).Scan(&windowStart, &windowEnd)
	
	if err != nil {
		return fmt.Errorf("failed to get window time range: %w", err)
	}
	
	// Calculate stats directly from messages table using time range (more reliable)
	query := `
		UPDATE session_windows 
		SET 
			total_input_tokens = (
				SELECT COALESCE(SUM(input_tokens), 0) 
				FROM messages 
				WHERE timestamp >= ? AND timestamp < ?
			),
			total_output_tokens = (
				SELECT COALESCE(SUM(output_tokens), 0) 
				FROM messages 
				WHERE timestamp >= ? AND timestamp < ?
			),
			total_tokens = (
				SELECT COALESCE(SUM(input_tokens + output_tokens), 0) 
				FROM messages 
				WHERE timestamp >= ? AND timestamp < ?
			),
			message_count = (
				SELECT COUNT(*) 
				FROM messages 
				WHERE timestamp >= ? AND timestamp < ? 
				AND message_role = 'assistant'
			),
			session_count = (
				SELECT COUNT(DISTINCT session_id) 
				FROM messages 
				WHERE timestamp >= ? AND timestamp < ?
			),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	
	_, err = s.db.Exec(query, 
		windowStart, windowEnd, // total_input_tokens
		windowStart, windowEnd, // total_output_tokens  
		windowStart, windowEnd, // total_tokens
		windowStart, windowEnd, // message_count
		windowStart, windowEnd, // session_count
		windowID)
	
	if err != nil {
		return fmt.Errorf("failed to update window stats: %w", err)
	}
	
	return nil
}

// AssignMessageToWindow assigns a message to a specific session window
func (s *SessionWindowService) AssignMessageToWindow(messageTimestamp time.Time, sessionID string, windowID string) error {
	query := `
		UPDATE messages 
		SET session_window_id = ? 
		WHERE timestamp = ? AND session_id = ?
	`
	
	_, err := s.db.Exec(query, windowID, messageTimestamp, sessionID)
	if err != nil {
		return fmt.Errorf("failed to assign message to window: %w", err)
	}
	
	return nil
}

// GetRecentWindows returns recent session windows
func (s *SessionWindowService) GetRecentWindows(limit int) ([]*SessionWindow, error) {
	query := `
		SELECT 
			id, window_start, window_end, reset_time,
			total_input_tokens, total_output_tokens, total_tokens,
			message_count, session_count, is_active,
			created_at, updated_at
		FROM session_windows 
		ORDER BY window_start DESC
		LIMIT ?
	`
	
	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent windows: %w", err)
	}
	defer rows.Close()
	
	var windows []*SessionWindow
	
	for rows.Next() {
		var window SessionWindow
		err := rows.Scan(
			&window.ID,
			&window.WindowStart,
			&window.WindowEnd,
			&window.ResetTime,
			&window.TotalInputTokens,
			&window.TotalOutputTokens,
			&window.TotalTokens,
			&window.MessageCount,
			&window.SessionCount,
			&window.IsActive,
			&window.CreatedAt,
			&window.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan window: %w", err)
		}
		
		windows = append(windows, &window)
	}
	
	return windows, nil
}

// deactivateWindow marks a window as inactive
func (s *SessionWindowService) deactivateWindow(windowID string) error {
	query := `
		UPDATE session_windows 
		SET is_active = false, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	
	_, err := s.db.Exec(query, windowID)
	if err != nil {
		return fmt.Errorf("failed to deactivate window: %w", err)
	}
	
	return nil
}


// roundToNextHour truncates time to the nearest hour (same as existing logic)
func (s *SessionWindowService) roundToNextHour(t time.Time) time.Time {
	return t.Truncate(time.Hour)
}