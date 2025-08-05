package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/marcboeker/go-duckdb"
)

func TestAuditService_LogEvent(t *testing.T) {
	db, err := setupAuthTestDB()
	require.NoError(t, err)
	defer db.Close()

	auditService := NewAuditService(db)
	userID := "test-user-123"

	t.Run("successful event logging", func(t *testing.T) {
		err := auditService.LogEvent(&userID, "test@example.com", "user.login", "auth", 
			`{"email": "test@example.com"}`, "127.0.0.1", "test-agent", true)
		assert.NoError(t, err)
	})

	t.Run("log event without user ID", func(t *testing.T) {
		err := auditService.LogEvent(nil, "anonymous", "system.startup", "system",
			`{"version": "1.0.0"}`, "", "system", true)
		assert.NoError(t, err)
	})

	t.Run("log failed event", func(t *testing.T) {
		err := auditService.LogEvent(&userID, "test@example.com", "user.login", "auth",
			`{"reason": "invalid_password"}`, "127.0.0.1", "test-agent", false)
		assert.NoError(t, err)
	})
}

func TestAuditService_GetAuditLogs(t *testing.T) {
	db, err := setupAuthTestDB()
	require.NoError(t, err)
	defer db.Close()

	auditService := NewAuditService(db)
	userID1 := "user-1"
	userID2 := "user-2"

	// Insert test data
	auditService.LogEvent(&userID1, "user1@example.com", "user.login", "auth", `{}`, "127.0.0.1", "agent1", true)
	auditService.LogEvent(&userID2, "user2@example.com", "user.login", "auth", `{}`, "127.0.0.2", "agent2", true)
	auditService.LogEvent(&userID1, "user1@example.com", "user.logout", "auth", `{}`, "127.0.0.1", "agent1", true)
	auditService.LogEvent(nil, "system", "system.startup", "system", `{}`, "", "system", true)

	t.Run("get all logs", func(t *testing.T) {
		logs, err := auditService.GetAuditLogs(nil, nil, 10, 0)
		require.NoError(t, err)
		assert.Len(t, logs, 4)
	})

	t.Run("filter by user ID", func(t *testing.T) {
		logs, err := auditService.GetAuditLogs(&userID1, nil, 10, 0)
		require.NoError(t, err)
		assert.Len(t, logs, 2)
		for _, log := range logs {
			assert.Equal(t, userID1, *log.UserID)
		}
	})

	t.Run("filter by action", func(t *testing.T) {
		action := "user.login"
		logs, err := auditService.GetAuditLogs(nil, &action, 10, 0)
		require.NoError(t, err)
		assert.Len(t, logs, 2)
		for _, log := range logs {
			assert.Equal(t, action, log.Action)
		}
	})

	t.Run("pagination", func(t *testing.T) {
		logs, err := auditService.GetAuditLogs(nil, nil, 2, 0)
		require.NoError(t, err)
		assert.Len(t, logs, 2)

		logs, err = auditService.GetAuditLogs(nil, nil, 2, 2)
		require.NoError(t, err)
		assert.Len(t, logs, 2)
	})
}

func TestAuditService_GetAuditLogStats(t *testing.T) {
	db, err := setupAuthTestDB()
	require.NoError(t, err)
	defer db.Close()

	auditService := NewAuditService(db)
	userID := "test-user"

	// Insert test data
	auditService.LogEvent(&userID, "test@example.com", "user.login", "auth", `{}`, "127.0.0.1", "agent", true)
	auditService.LogEvent(&userID, "test@example.com", "user.login", "auth", `{}`, "127.0.0.1", "agent", false)
	auditService.LogEvent(&userID, "test@example.com", "user.logout", "auth", `{}`, "127.0.0.1", "agent", true)
	auditService.LogEvent(&userID, "test@example.com", "task.execute", "tasks", `{}`, "127.0.0.1", "agent", true)

	stats, err := auditService.GetAuditLogStats()
	require.NoError(t, err)

	assert.Equal(t, 4, stats["total_events"])
	assert.Equal(t, 1, stats["failed_events"])
	
	eventsByAction, ok := stats["events_by_action"].([]map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, eventsByAction)
	
	// Check that login events are most frequent
	mostFrequent := eventsByAction[0]
	assert.Equal(t, "user.login", mostFrequent["action"])
	assert.Equal(t, 2, mostFrequent["count"])
}