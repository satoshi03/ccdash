package services

import (
	"database/sql"
	"fmt"
	"time"
	
	"claudeee-backend/internal/models"
)

type TokenService struct {
	db *sql.DB
}

func NewTokenService(db *sql.DB) *TokenService {
	return &TokenService{db: db}
}

const (
	CLAUDE_PRO_LIMIT  = 7000
	CLAUDE_MAX5_LIMIT = 35000
	CLAUDE_MAX20_LIMIT = 140000
	WINDOW_DURATION = 5 * time.Hour
)

func (s *TokenService) GetCurrentTokenUsage() (*models.TokenUsage, error) {
	// 現在時刻から5時間のウィンドウを計算
	now := time.Now()
	potentialWindowStart := now.Add(-WINDOW_DURATION)
	
	// 5時間のウィンドウ内で最初のメッセージがある時刻を取得
	windowStartQuery := `
		SELECT MIN(timestamp) 
		FROM messages 
		WHERE timestamp >= ?
	`
	
	var actualWindowStart sql.NullTime
	err := s.db.QueryRow(windowStartQuery, potentialWindowStart).Scan(&actualWindowStart)
	if err != nil {
		return nil, fmt.Errorf("failed to get window start: %w", err)
	}
	
	// ウィンドウの開始時刻と終了時刻を決定
	var windowStart, windowEnd time.Time
	
	if actualWindowStart.Valid {
		// メッセージが存在する場合、そのメッセージの時刻からウィンドウを計算
		windowStart = actualWindowStart.Time
		// リセット時間を毎時0分に調整（8:30→13:00のように）
		windowEnd = s.roundToNextHour(windowStart.Add(WINDOW_DURATION))
		
		// ウィンドウ終了時刻が現在時刻より前の場合は、現在時刻から5時間のウィンドウを使用
		if windowEnd.Before(now) {
			windowStart = potentialWindowStart
			windowEnd = s.roundToNextHour(now.Add(WINDOW_DURATION))
		}
	} else {
		// メッセージがない場合は、現在時刻から5時間後をリセット時間とする
		windowStart = now
		windowEnd = s.roundToNextHour(now.Add(WINDOW_DURATION))
	}
	
	query := `
		SELECT 
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(input_tokens + output_tokens), 0) as total_tokens,
			COUNT(DISTINCT session_id) as active_sessions
		FROM messages 
		WHERE timestamp >= ? AND timestamp <= ?
	`
	
	var totalInputTokens, totalOutputTokens, totalTokens, activeSessions int
	
	err = s.db.QueryRow(query, windowStart, windowEnd).Scan(
		&totalInputTokens,
		&totalOutputTokens, 
		&totalTokens,
		&activeSessions,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get token usage: %w", err)
	}
	
	usageLimit := s.getUsageLimit()
	usageRate := float64(totalTokens) / float64(usageLimit)
	
	return &models.TokenUsage{
		TotalTokens:    totalTokens,
		InputTokens:    totalInputTokens,
		OutputTokens:   totalOutputTokens,
		UsageLimit:     usageLimit,
		UsageRate:      usageRate,
		WindowStart:    windowStart,
		WindowEnd:      windowEnd,
		ActiveSessions: activeSessions,
	}, nil
}

func (s *TokenService) getUsageLimit() int {
	return CLAUDE_PRO_LIMIT
}

// roundToNextHour は時刻を次の正時（0分）に切り上げます
// ただし、トークンリセット時間は切り下げるため、メッセージ時刻+5時間の時刻を切り下げます
// 例: 8:30 + 5h = 13:30 -> 13:00, 10:15 + 5h = 15:15 -> 15:00
func (s *TokenService) roundToNextHour(t time.Time) time.Time {
	// 時刻を正時に切り下げ（トークンリセット時間のため）
	return t.Truncate(time.Hour)
}

func (s *TokenService) GetTokenUsageBySession(sessionID string) (*models.TokenUsage, error) {
	query := `
		SELECT 
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(input_tokens + output_tokens), 0) as total_tokens,
			MIN(timestamp) as start_time,
			MAX(timestamp) as end_time
		FROM messages 
		WHERE session_id = ?
		AND message_role = 'assistant'
	`
	
	var totalInputTokens, totalOutputTokens, totalTokens int
	var startTime, endTime time.Time
	
	err := s.db.QueryRow(query, sessionID).Scan(
		&totalInputTokens,
		&totalOutputTokens,
		&totalTokens,
		&startTime,
		&endTime,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get session token usage: %w", err)
	}
	
	usageLimit := s.getUsageLimit()
	usageRate := float64(totalTokens) / float64(usageLimit)
	
	return &models.TokenUsage{
		TotalTokens:    totalTokens,
		InputTokens:    totalInputTokens,
		OutputTokens:   totalOutputTokens,
		UsageLimit:     usageLimit,
		UsageRate:      usageRate,
		WindowStart:    startTime,
		WindowEnd:      endTime,
		ActiveSessions: 1,
	}, nil
}

func (s *TokenService) GetActiveSessionsInWindow() ([]models.Session, error) {
	now := time.Now()
	windowStart := now.Add(-WINDOW_DURATION)
	
	query := `
		SELECT DISTINCT
			s.id,
			s.project_name,
			s.project_path,
			s.start_time,
			s.end_time,
			s.total_input_tokens,
			s.total_output_tokens,
			s.total_tokens,
			s.message_count,
			s.status,
			s.created_at
		FROM sessions s
		INNER JOIN messages m ON s.id = m.session_id
		WHERE m.timestamp >= ?
		ORDER BY s.start_time DESC
	`
	
	rows, err := s.db.Query(query, windowStart)
	if err != nil {
		return nil, fmt.Errorf("failed to get active sessions: %w", err)
	}
	defer rows.Close()
	
	var sessions []models.Session
	
	for rows.Next() {
		var session models.Session
		err := rows.Scan(
			&session.ID,
			&session.ProjectName,
			&session.ProjectPath,
			&session.StartTime,
			&session.EndTime,
			&session.TotalInputTokens,
			&session.TotalOutputTokens,
			&session.TotalTokens,
			&session.MessageCount,
			&session.Status,
			&session.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		
		sessions = append(sessions, session)
	}
	
	return sessions, nil
}

func (s *TokenService) UpdateSessionTokens(sessionID string) error {
	query := `
		UPDATE sessions 
		SET 
			total_input_tokens = (
				SELECT COALESCE(SUM(input_tokens), 0) 
				FROM messages 
				WHERE session_id = ? AND message_role = 'assistant'
			),
			total_output_tokens = (
				SELECT COALESCE(SUM(output_tokens), 0) 
				FROM messages 
				WHERE session_id = ? AND message_role = 'assistant'
			),
			total_tokens = (
				SELECT COALESCE(SUM(input_tokens + output_tokens), 0) 
				FROM messages 
				WHERE session_id = ? AND message_role = 'assistant'
			),
			message_count = (
				SELECT COUNT(*) 
				FROM messages 
				WHERE session_id = ?
			),
			end_time = (
				SELECT MAX(timestamp) FROM messages WHERE session_id = ?
			)
		WHERE id = ?
	`
	
	_, err := s.db.Exec(query, sessionID, sessionID, sessionID, sessionID, sessionID, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update session tokens: %w", err)
	}
	
	return nil
}