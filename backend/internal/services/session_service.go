package services

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
	
	"claudeee-backend/internal/models"
)

type SessionService struct {
	db               *sql.DB
	activityDetector *SessionActivityDetector
}

func NewSessionService(db *sql.DB) *SessionService {
	return &SessionService{
		db:               db,
		activityDetector: NewSessionActivityDetector(db),
	}
}

func (s *SessionService) GetAllSessions() ([]models.SessionSummary, error) {
	// Simplified query without JOIN for better performance
	query := `
		SELECT 
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
		ORDER BY s.start_time DESC
	`
	
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}
	defer rows.Close()
	
	var sessions []models.SessionSummary
	
	for rows.Next() {
		var session models.SessionSummary
		var startTime sql.NullTime
		
		err := rows.Scan(
			&session.ID,
			&session.ProjectName,
			&session.ProjectPath,
			&startTime,
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
		
		// Handle NULL start_time
		if startTime.Valid {
			session.StartTime = startTime.Time
		} else {
			session.StartTime = session.CreatedAt
		}
		
		// Set default values for performance (avoid additional queries)
		session.LastActivity = session.StartTime  // Use start_time as fallback
		session.IsActive = false  // Default to inactive for list view
		
		if session.EndTime != nil {
			duration := session.EndTime.Sub(session.StartTime)
			session.Duration = &duration
		}
		
		// Skip generated code extraction for performance in GetAllSessions
		// This can be added later on-demand per session
		session.GeneratedCode = nil
		
		sessions = append(sessions, session)
	}
	
	return sessions, nil
}

func (s *SessionService) GetSessionByID(sessionID string) (*models.SessionSummary, error) {
	query := `
		SELECT 
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
			s.created_at,
			MAX(m.timestamp) as last_activity
		FROM sessions s
		LEFT JOIN messages m ON s.id = m.session_id
		WHERE s.id = ?
		GROUP BY s.id, s.project_name, s.project_path, s.start_time, s.end_time, 
				 s.total_input_tokens, s.total_output_tokens, s.total_tokens, 
				 s.message_count, s.status, s.created_at
	`
	
	var session models.SessionSummary
	var lastActivity sql.NullTime
	var startTime sql.NullTime
	
	err := s.db.QueryRow(query, sessionID).Scan(
		&session.ID,
		&session.ProjectName,
		&session.ProjectPath,
		&startTime,
		&session.EndTime,
		&session.TotalInputTokens,
		&session.TotalOutputTokens,
		&session.TotalTokens,
		&session.MessageCount,
		&session.Status,
		&session.CreatedAt,
		&lastActivity,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	
	if lastActivity.Valid {
		session.LastActivity = lastActivity.Time
	}
	
	// Handle NULL start_time
	if startTime.Valid {
		session.StartTime = startTime.Time
	} else {
		// Use the first message timestamp if start_time is NULL
		var firstMessageTime sql.NullTime
		err = s.db.QueryRow("SELECT MIN(timestamp) FROM messages WHERE session_id = ?", sessionID).Scan(&firstMessageTime)
		if err == nil && firstMessageTime.Valid {
			session.StartTime = firstMessageTime.Time
		} else {
			// Fallback to created_at if no messages found
			session.StartTime = session.CreatedAt
		}
	}
	
	session.IsActive = s.isSessionActive(session.Session, lastActivity.Time)
	
	if session.EndTime != nil {
		duration := session.EndTime.Sub(session.StartTime)
		session.Duration = &duration
	} else if lastActivity.Valid {
		duration := lastActivity.Time.Sub(session.StartTime)
		session.Duration = &duration
	}
	
	generatedCode, err := s.extractGeneratedCode(session.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to extract generated code: %w", err)
	}
	session.GeneratedCode = generatedCode
	
	return &session, nil
}

func (s *SessionService) GetSessionMessages(sessionID string) ([]models.Message, error) {
	query := `
		SELECT 
			id, session_id, parent_uuid, is_sidechain, user_type, message_type,
			message_role, model, content, input_tokens, cache_creation_input_tokens,
			cache_read_input_tokens, output_tokens, service_tier, request_id,
			timestamp, created_at
		FROM messages 
		WHERE session_id = ?
		ORDER BY timestamp ASC
	`
	
	rows, err := s.db.Query(query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session messages: %w", err)
	}
	defer rows.Close()
	
	var messages []models.Message
	
	for rows.Next() {
		var message models.Message
		err := rows.Scan(
			&message.ID,
			&message.SessionID,
			&message.ParentUUID,
			&message.IsSidechain,
			&message.UserType,
			&message.MessageType,
			&message.MessageRole,
			&message.Model,
			&message.Content,
			&message.InputTokens,
			&message.CacheCreationInputTokens,
			&message.CacheReadInputTokens,
			&message.OutputTokens,
			&message.ServiceTier,
			&message.RequestID,
			&message.Timestamp,
			&message.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		
		messages = append(messages, message)
	}
	
	return messages, nil
}

// PaginatedMessagesResult represents paginated message results
type PaginatedMessagesResult struct {
	Messages    []models.Message `json:"messages"`
	Total       int              `json:"total"`
	Page        int              `json:"page"`
	PageSize    int              `json:"page_size"`
	TotalPages  int              `json:"total_pages"`
	HasNext     bool             `json:"has_next"`
	HasPrevious bool             `json:"has_previous"`
}

func (s *SessionService) GetSessionMessagesPaginated(sessionID string, page, pageSize int) (*PaginatedMessagesResult, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20 // Default page size
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM messages WHERE session_id = ?`
	var total int
	err := s.db.QueryRow(countQuery, sessionID).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to get message count: %w", err)
	}

	// Calculate pagination info
	totalPages := (total + pageSize - 1) / pageSize
	offset := (page - 1) * pageSize

	// Get paginated messages
	query := `
		SELECT 
			id, session_id, parent_uuid, is_sidechain, user_type, message_type,
			message_role, model, content, input_tokens, cache_creation_input_tokens,
			cache_read_input_tokens, output_tokens, service_tier, request_id,
			timestamp, created_at
		FROM messages 
		WHERE session_id = ?
		ORDER BY timestamp ASC
		LIMIT ? OFFSET ?
	`
	
	rows, err := s.db.Query(query, sessionID, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get session messages: %w", err)
	}
	defer rows.Close()
	
	var messages []models.Message
	
	for rows.Next() {
		var message models.Message
		err := rows.Scan(
			&message.ID,
			&message.SessionID,
			&message.ParentUUID,
			&message.IsSidechain,
			&message.UserType,
			&message.MessageType,
			&message.MessageRole,
			&message.Model,
			&message.Content,
			&message.InputTokens,
			&message.CacheCreationInputTokens,
			&message.CacheReadInputTokens,
			&message.OutputTokens,
			&message.ServiceTier,
			&message.RequestID,
			&message.Timestamp,
			&message.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		
		messages = append(messages, message)
	}

	return &PaginatedMessagesResult{
		Messages:    messages,
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
	}, nil
}

func (s *SessionService) CreateOrUpdateSession(sessionID, projectName, projectPath string, messageTime ...time.Time) error {
	// Check if session exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM sessions WHERE id = ?)`
	err := s.db.QueryRow(checkQuery, sessionID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check session existence: %w", err)
	}
	
	if exists {
		// Session exists, update start time if a message time is provided and it's earlier
		if len(messageTime) > 0 {
			updateQuery := `
				UPDATE sessions 
				SET start_time = ? 
				WHERE id = ? AND start_time > ?
			`
			_, err = s.db.Exec(updateQuery, messageTime[0], sessionID, messageTime[0])
			if err != nil {
				return fmt.Errorf("failed to update session start time: %w", err)
			}
		}
		return nil
	} else {
		// Use provided message time if available, otherwise get from database
		var startTime time.Time
		if len(messageTime) > 0 {
			startTime = messageTime[0]
		} else {
			// Get the earliest message timestamp for this session to use as start time
			var firstMessageTime sql.NullTime
			timeQuery := `SELECT MIN(timestamp) FROM messages WHERE session_id = ?`
			err = s.db.QueryRow(timeQuery, sessionID).Scan(&firstMessageTime)
			
			// If no messages found yet, use current time as fallback
			startTime = time.Now()
			if err == nil && firstMessageTime.Valid {
				startTime = firstMessageTime.Time
			}
		}
		
		// Insert new session
		insertQuery := `
			INSERT INTO sessions (id, project_name, project_path, start_time, status)
			VALUES (?, ?, ?, ?, 'active')
		`
		_, err = s.db.Exec(insertQuery, sessionID, projectName, projectPath, startTime)
		if err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}
	}
	
	return nil
}

func (s *SessionService) isSessionActive(session models.Session, lastActivity time.Time) bool {
	// Use the new advanced activity detector
	return s.activityDetector.IsSessionActive(session.ID, session, lastActivity)
}

func (s *SessionService) extractGeneratedCode(sessionID string) ([]string, error) {
	query := `
		SELECT content 
		FROM messages 
		WHERE session_id = ? 
		AND message_role = 'assistant'
		AND content IS NOT NULL
		ORDER BY timestamp ASC
	`
	
	rows, err := s.db.Query(query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages for code extraction: %w", err)
	}
	defer rows.Close()
	
	var codeBlocks []string
	
	for rows.Next() {
		var content sql.NullString
		err := rows.Scan(&content)
		if err != nil {
			continue
		}
		
		if content.Valid {
			extractedCode := extractCodeFromContent(content.String)
			if len(extractedCode) > 0 {
				codeBlocks = append(codeBlocks, extractedCode...)
			}
		}
	}
	
	return codeBlocks, nil
}

// GetSessionActivityReport returns detailed activity analysis for a session
func (s *SessionService) GetSessionActivityReport(sessionID string) (map[string]interface{}, error) {
	// Get session details
	session, err := s.GetSessionByID(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Get last activity
	lastActivity := session.LastActivity
	if session.EndTime != nil {
		lastActivity = *session.EndTime
	}

	// Generate detailed report
	report := s.activityDetector.GetSessionActivityReport(sessionID, session.Session, lastActivity)
	return report, nil
}

func extractCodeFromContent(content string) []string {
	codeBlockRegex := regexp.MustCompile("```[\\s\\S]*?```")
	matches := codeBlockRegex.FindAllString(content, -1)
	
	var codeBlocks []string
	for _, match := range matches {
		// Remove the ```language and ``` wrappers
		code := regexp.MustCompile("```\\w*\\n?|\\n?```").ReplaceAllString(match, "")
		code = strings.TrimSpace(code)
		if code != "" {
			codeBlocks = append(codeBlocks, code)
		}
	}
	
	return codeBlocks
}