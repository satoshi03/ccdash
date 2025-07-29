package services

import (
	"database/sql"
	"fmt"
	"ccdash-backend/internal/models"
	"github.com/google/uuid"
)

type SessionWindowMessageService struct {
	db *sql.DB
}

func NewSessionWindowMessageService(db *sql.DB) *SessionWindowMessageService {
	return &SessionWindowMessageService{db: db}
}

// AddMessageToWindow メッセージをセッションウィンドウに関連付け
func (s *SessionWindowMessageService) AddMessageToWindow(sessionWindowID string, messageID string) error {
	relationID := uuid.New().String()
	query := `
		INSERT OR IGNORE INTO session_window_messages (id, session_window_id, message_id) 
		VALUES (?, ?, ?)
	`
	_, err := s.db.Exec(query, relationID, sessionWindowID, messageID)
	if err != nil {
		return fmt.Errorf("failed to add message to session window: %w", err)
	}
	return nil
}

// AddMessagesToWindow 複数のメッセージをセッションウィンドウに関連付け
func (s *SessionWindowMessageService) AddMessagesToWindow(sessionWindowID string, messageIDs []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO session_window_messages (id, session_window_id, message_id) 
		VALUES (?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, messageID := range messageIDs {
		relationID := uuid.New().String()
		if _, err := stmt.Exec(relationID, sessionWindowID, messageID); err != nil {
			return fmt.Errorf("failed to insert message %s: %w", messageID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// RemoveMessageFromWindow メッセージをセッションウィンドウから削除
func (s *SessionWindowMessageService) RemoveMessageFromWindow(sessionWindowID string, messageID string) error {
	query := `
		DELETE FROM session_window_messages 
		WHERE session_window_id = ? AND message_id = ?
	`
	_, err := s.db.Exec(query, sessionWindowID, messageID)
	if err != nil {
		return fmt.Errorf("failed to remove message from session window: %w", err)
	}
	return nil
}

// RemoveAllMessagesFromWindow セッションウィンドウから全メッセージを削除
func (s *SessionWindowMessageService) RemoveAllMessagesFromWindow(sessionWindowID string) error {
	query := `DELETE FROM session_window_messages WHERE session_window_id = ?`
	_, err := s.db.Exec(query, sessionWindowID)
	if err != nil {
		return fmt.Errorf("failed to remove all messages from session window: %w", err)
	}
	return nil
}

// GetMessagesByWindow セッションウィンドウ内のメッセージを取得
func (s *SessionWindowMessageService) GetMessagesByWindow(sessionWindowID string) ([]models.Message, error) {
	query := `
		SELECT m.id, m.session_id, m.parent_uuid, m.is_sidechain, m.user_type, 
		       m.message_type, m.message_role, m.model, m.content, m.input_tokens, 
		       m.cache_creation_input_tokens, m.cache_read_input_tokens, 
		       m.output_tokens, m.service_tier, m.request_id, m.timestamp, m.created_at
		FROM messages m
		INNER JOIN session_window_messages swm ON m.id = swm.message_id
		WHERE swm.session_window_id = ?
		ORDER BY m.timestamp ASC
	`
	
	rows, err := s.db.Query(query, sessionWindowID)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages by window: %w", err)
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		err := rows.Scan(
			&msg.ID, &msg.SessionID, &msg.ParentUUID, &msg.IsSidechain,
			&msg.UserType, &msg.MessageType, &msg.MessageRole, &msg.Model,
			&msg.Content, &msg.InputTokens, &msg.CacheCreationInputTokens,
			&msg.CacheReadInputTokens, &msg.OutputTokens, &msg.ServiceTier,
			&msg.RequestID, &msg.Timestamp, &msg.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	return messages, nil
}

// GetWindowsByMessage メッセージが属するセッションウィンドウを取得
func (s *SessionWindowMessageService) GetWindowsByMessage(messageID string) ([]string, error) {
	query := `
		SELECT session_window_id 
		FROM session_window_messages 
		WHERE message_id = ?
		ORDER BY created_at ASC
	`
	
	rows, err := s.db.Query(query, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to query windows by message: %w", err)
	}
	defer rows.Close()

	var windowIDs []string
	for rows.Next() {
		var windowID string
		if err := rows.Scan(&windowID); err != nil {
			return nil, fmt.Errorf("failed to scan window ID: %w", err)
		}
		windowIDs = append(windowIDs, windowID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	return windowIDs, nil
}

// GetMessageCountByWindow セッションウィンドウ内のメッセージ数を取得
func (s *SessionWindowMessageService) GetMessageCountByWindow(sessionWindowID string) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM session_window_messages 
		WHERE session_window_id = ?
	`
	
	var count int
	err := s.db.QueryRow(query, sessionWindowID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get message count: %w", err)
	}
	
	return count, nil
}

// GetAllRelations 全ての関連を取得（デバッグ用）
func (s *SessionWindowMessageService) GetAllRelations() ([]models.SessionWindowMessage, error) {
	query := `
		SELECT id, session_window_id, message_id, created_at 
		FROM session_window_messages 
		ORDER BY created_at ASC
	`
	
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all relations: %w", err)
	}
	defer rows.Close()

	var relations []models.SessionWindowMessage
	for rows.Next() {
		var rel models.SessionWindowMessage
		err := rows.Scan(&rel.ID, &rel.SessionWindowID, &rel.MessageID, &rel.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan relation: %w", err)
		}
		relations = append(relations, rel)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	return relations, nil
}

// ClearAllRelations 全ての関連を削除（リセット用）
func (s *SessionWindowMessageService) ClearAllRelations() error {
	query := `DELETE FROM session_window_messages`
	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to clear all relations: %w", err)
	}
	return nil
}