package services

import (
	"database/sql"
	"fmt"
	"time"

	"ccdash-backend/internal/models"

	"github.com/google/uuid"
)

type AuditService struct {
	db *sql.DB
}

func NewAuditService(db *sql.DB) *AuditService {
	return &AuditService{
		db: db,
	}
}

// LogEvent logs an audit event
func (s *AuditService) LogEvent(userID *string, userEmail, action, resource, details, ipAddress, userAgent string, success bool) error {
	auditLog := models.AuditLog{
		ID:        uuid.New().String(),
		UserID:    userID,
		UserEmail: &userEmail,
		Action:    action,
		Resource:  resource,
		Details:   &details,
		IPAddress: &ipAddress,
		UserAgent: &userAgent,
		Success:   success,
		Timestamp: time.Now(),
	}

	query := `
		INSERT INTO audit_logs (id, user_id, user_email, action, resource, details, ip_address, user_agent, success, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query,
		auditLog.ID, auditLog.UserID, auditLog.UserEmail, auditLog.Action, auditLog.Resource,
		auditLog.Details, auditLog.IPAddress, auditLog.UserAgent, auditLog.Success, auditLog.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to log audit event: %w", err)
	}

	return nil
}

// GetAuditLogs retrieves audit logs with optional filtering
func (s *AuditService) GetAuditLogs(userID *string, action *string, limit, offset int) ([]models.AuditLog, error) {
	var logs []models.AuditLog
	var args []interface{}
	
	query := `
		SELECT id, user_id, user_email, action, resource, details, ip_address, user_agent, success, timestamp
		FROM audit_logs
		WHERE 1=1
	`

	if userID != nil {
		query += ` AND user_id = ?`
		args = append(args, *userID)
	}

	if action != nil {
		query += ` AND action = ?`
		args = append(args, *action)
	}

	query += ` ORDER BY timestamp DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var log models.AuditLog
		err := rows.Scan(
			&log.ID, &log.UserID, &log.UserEmail, &log.Action, &log.Resource,
			&log.Details, &log.IPAddress, &log.UserAgent, &log.Success, &log.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over audit logs: %w", err)
	}

	return logs, nil
}

// GetAuditLogStats returns statistics about audit logs
func (s *AuditService) GetAuditLogStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total events
	var totalEvents int
	err := s.db.QueryRow("SELECT COUNT(*) FROM audit_logs").Scan(&totalEvents)
	if err != nil {
		return nil, fmt.Errorf("failed to get total events: %w", err)
	}
	stats["total_events"] = totalEvents

	// Failed events
	var failedEvents int
	err = s.db.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE success = FALSE").Scan(&failedEvents)
	if err != nil {
		return nil, fmt.Errorf("failed to get failed events: %w", err)
	}
	stats["failed_events"] = failedEvents

	// Events by action
	actionQuery := `
		SELECT action, COUNT(*) as count
		FROM audit_logs
		GROUP BY action
		ORDER BY count DESC
		LIMIT 10
	`
	rows, err := s.db.Query(actionQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get events by action: %w", err)
	}
	defer rows.Close()

	var actionStats []map[string]interface{}
	for rows.Next() {
		var action string
		var count int
		if err := rows.Scan(&action, &count); err != nil {
			return nil, fmt.Errorf("failed to scan action stats: %w", err)
		}
		actionStats = append(actionStats, map[string]interface{}{
			"action": action,
			"count":  count,
		})
	}
	stats["events_by_action"] = actionStats

	// Recent failed login attempts
	var failedLogins int
	cutoffTime := time.Now().Add(-time.Hour)
	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM audit_logs 
		WHERE action = 'user.login' AND success = FALSE AND timestamp > ?
	`, cutoffTime).Scan(&failedLogins)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent failed logins: %w", err)
	}
	stats["recent_failed_logins"] = failedLogins

	return stats, nil
}

// CleanupOldLogs removes audit logs older than the specified duration
func (s *AuditService) CleanupOldLogs(olderThan time.Duration) error {
	cutoffTime := time.Now().Add(-olderThan)
	
	query := `DELETE FROM audit_logs WHERE timestamp < ?`
	result, err := s.db.Exec(query, cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to cleanup old logs: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		// Log the cleanup operation
		s.LogEvent(nil, "system", "audit.cleanup", "audit_logs",
			fmt.Sprintf(`{"rows_deleted": %d, "cutoff_time": "%s"}`, rowsAffected, cutoffTime.Format(time.RFC3339)),
			"", "system", true)
	}

	return nil
}