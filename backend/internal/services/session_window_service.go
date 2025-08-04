package services

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SessionWindowService struct {
	db              *sql.DB
	relationService *SessionWindowMessageService
}

type SessionWindow struct {
	ID                string    `json:"id"`
	WindowStart       time.Time `json:"window_start"`
	WindowEnd         time.Time `json:"window_end"`
	ResetTime         time.Time `json:"reset_time"`
	TotalInputTokens  int       `json:"total_input_tokens"`
	TotalOutputTokens int       `json:"total_output_tokens"`
	TotalTokens       int       `json:"total_tokens"`
	MessageCount      int       `json:"message_count"`
	SessionCount      int       `json:"session_count"`
	TotalCost         float64   `json:"total_cost"`
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func NewSessionWindowService(db *sql.DB) *SessionWindowService {
	return &SessionWindowService{
		db:              db,
		relationService: NewSessionWindowMessageService(db),
	}
}

// GetCurrentActiveWindow returns the currently active session window
func (s *SessionWindowService) GetCurrentActiveWindow() (*SessionWindow, error) {
	query := `
		SELECT 
			id, window_start, window_end, reset_time,
			total_input_tokens, total_output_tokens, total_tokens,
			message_count, session_count, COALESCE(total_cost, 0) as total_cost, is_active,
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
		&window.TotalCost,
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
			message_count, session_count, COALESCE(total_cost, 0) as total_cost, is_active,
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
		&window.TotalCost,
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
	// 1. 既存のSessionWindowとリレーションを全てクリア
	err := s.relationService.ClearAllRelations()
	if err != nil {
		return fmt.Errorf("failed to clear existing relations: %w", err)
	}

	_, err = s.db.Exec("DELETE FROM session_windows")
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
		tempWindowEnd := windowStart.Add(WINDOW_DURATION)
		// WindowEndも時間単位で切り捨て（例：10:28 -> 10:00）
		windowEnd := s.truncateToHour(tempWindowEnd)
		// ResetTimeはWindowEndと同じ（時間単位切り捨て）
		resetTime := windowEnd

		window := &SessionWindow{
			ID:          uuid.New().String(),
			WindowStart: windowStart,
			WindowEnd:   windowEnd,
			ResetTime:   resetTime,
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
		SELECT m.id, m.session_id, m.timestamp
		FROM messages m
		LEFT JOIN session_window_messages swm ON m.id = swm.message_id
		WHERE swm.message_id IS NULL
		ORDER BY m.timestamp ASC
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
		SELECT m.id
		FROM messages m
		LEFT JOIN session_window_messages swm ON m.id = swm.message_id
		WHERE m.timestamp >= ? AND m.timestamp < ? AND swm.message_id IS NULL
	`

	rows, err := s.db.Query(query, windowStart, windowEnd)
	if err != nil {
		return fmt.Errorf("failed to get messages for assignment: %w", err)
	}
	defer rows.Close()

	var messageIDs []string
	for rows.Next() {
		var messageID string
		if err := rows.Scan(&messageID); err != nil {
			return fmt.Errorf("failed to scan message ID: %w", err)
		}
		messageIDs = append(messageIDs, messageID)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error during row iteration: %w", err)
	}

	// Use relation service to add messages to window
	if len(messageIDs) > 0 {
		err = s.relationService.AddMessagesToWindow(windowID, messageIDs)
		if err != nil {
			return fmt.Errorf("failed to add messages to window via relation service: %w", err)
		}
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

	// Calculate total_cost for this window
	totalCost, err := s.calculateWindowCostByID(windowID)
	if err != nil {
		// Continue without cost calculation if it fails
		totalCost = 0.0
	}

	// Calculate stats using relation table
	query := `
		UPDATE session_windows 
		SET 
			total_input_tokens = (
				SELECT COALESCE(SUM(m.input_tokens), 0) 
				FROM messages m
				INNER JOIN session_window_messages swm ON m.id = swm.message_id
				WHERE swm.session_window_id = ?
			),
			total_output_tokens = (
				SELECT COALESCE(SUM(m.output_tokens), 0) 
				FROM messages m
				INNER JOIN session_window_messages swm ON m.id = swm.message_id
				WHERE swm.session_window_id = ?
			),
			total_tokens = (
				SELECT COALESCE(SUM(m.input_tokens + m.output_tokens), 0) 
				FROM messages m
				INNER JOIN session_window_messages swm ON m.id = swm.message_id
				WHERE swm.session_window_id = ?
			),
			message_count = (
				SELECT COUNT(*) 
				FROM messages m
				INNER JOIN session_window_messages swm ON m.id = swm.message_id
				WHERE swm.session_window_id = ? AND m.message_role = 'assistant'
			),
			session_count = (
				SELECT COUNT(DISTINCT m.session_id) 
				FROM messages m
				INNER JOIN session_window_messages swm ON m.id = swm.message_id
				WHERE swm.session_window_id = ?
			),
			total_cost = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err = s.db.Exec(query,
		windowID,  // total_input_tokens
		windowID,  // total_output_tokens
		windowID,  // total_tokens
		windowID,  // message_count
		windowID,  // session_count
		totalCost, // total_cost
		windowID)

	if err != nil {
		return fmt.Errorf("failed to update window stats: %w", err)
	}

	return nil
}

// calculateWindowCost calculates the total cost for messages in a specific window
func (s *SessionWindowService) calculateWindowCost(windowStart, windowEnd time.Time) (float64, error) {
	// Get window ID for the given time range
	var windowID string
	err := s.db.QueryRow(`
		SELECT id FROM session_windows 
		WHERE window_start = ? AND window_end = ?
	`, windowStart, windowEnd).Scan(&windowID)

	if err != nil {
		return 0.0, fmt.Errorf("failed to find window for cost calculation: %w", err)
	}

	return s.calculateWindowCostByID(windowID)
}

// calculateWindowCostByID calculates the total cost for messages in a specific window by ID
func (s *SessionWindowService) calculateWindowCostByID(windowID string) (float64, error) {
	pricingCalculator := NewPricingCalculator()

	query := `
		SELECT 
			m.model,
			COALESCE(SUM(m.input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(m.output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(m.cache_creation_input_tokens), 0) as total_cache_creation_tokens,
			COALESCE(SUM(m.cache_read_input_tokens), 0) as total_cache_read_tokens
		FROM messages m
		INNER JOIN session_window_messages swm ON m.id = swm.message_id
		WHERE swm.session_window_id = ?
		AND m.message_role = 'assistant'
		AND m.model IS NOT NULL
		GROUP BY m.model
	`

	rows, err := s.db.Query(query, windowID)
	if err != nil {
		return 0.0, fmt.Errorf("failed to query messages for cost calculation: %w", err)
	}
	defer rows.Close()

	var totalCost float64

	for rows.Next() {
		var model string
		var inputTokens, outputTokens, cacheCreationTokens, cacheReadTokens int

		err := rows.Scan(&model, &inputTokens, &outputTokens, &cacheCreationTokens, &cacheReadTokens)
		if err != nil {
			return 0.0, fmt.Errorf("failed to scan message data for cost calculation: %w", err)
		}

		cost := pricingCalculator.CalculateCost(
			model,
			inputTokens,
			outputTokens,
			cacheCreationTokens,
			cacheReadTokens,
		)

		totalCost += cost
	}

	if err := rows.Err(); err != nil {
		return 0.0, fmt.Errorf("error iterating over messages for cost calculation: %w", err)
	}

	return totalCost, nil
}

// AssignMessageToWindow assigns a message to a specific session window
func (s *SessionWindowService) AssignMessageToWindow(messageTimestamp time.Time, sessionID string, windowID string) error {
	query := `
		SELECT id FROM messages 
		WHERE timestamp = ? AND session_id = ?
	`

	var messageID string
	err := s.db.QueryRow(query, messageTimestamp, sessionID).Scan(&messageID)
	if err != nil {
		return fmt.Errorf("failed to find message: %w", err)
	}

	err = s.relationService.AddMessageToWindow(windowID, messageID)
	if err != nil {
		return fmt.Errorf("failed to assign message to window via relation service: %w", err)
	}

	return nil
}

// GetRecentWindows returns recent session windows
func (s *SessionWindowService) GetRecentWindows(limit int) ([]*SessionWindow, error) {
	query := `
		SELECT 
			id, window_start, window_end, reset_time,
			total_input_tokens, total_output_tokens, total_tokens,
			message_count, session_count, COALESCE(total_cost, 0) as total_cost, is_active,
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
			&window.TotalCost,
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

// GetActiveWindow returns the currently active session window
func (s *SessionWindowService) GetActiveWindow() (*SessionWindow, error) {
	query := `
		SELECT id, window_start, window_end, reset_time, 
		       total_input_tokens, total_output_tokens, total_tokens,
		       message_count, session_count, total_cost, is_active,
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
		&window.TotalCost,
		&window.IsActive,
		&window.CreatedAt,
		&window.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active window: %w", err)
	}
	
	return &window, nil
}
