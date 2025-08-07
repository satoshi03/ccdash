package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Session represents a user session
type Session struct {
	ID        string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
	LastUsed  time.Time
}

// SessionManager manages user sessions
type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	timeout  time.Duration
}

// NewSessionManager creates a new session manager
func NewSessionManager(timeout time.Duration) *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
		timeout:  timeout,
	}

	// Start cleanup goroutine
	go sm.cleanupSessions()
	return sm
}

// CreateSession creates a new session for the given user ID
func (sm *SessionManager) CreateSession(userID string) (*Session, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(sm.timeout),
		LastUsed:  now,
	}

	sm.mu.Lock()
	sm.sessions[sessionID] = session
	sm.mu.Unlock()

	return session, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*Session, bool) {
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return nil, false
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		sm.DeleteSession(sessionID)
		return nil, false
	}

	// Update last used time
	sm.mu.Lock()
	session.LastUsed = time.Now()
	// Extend expiry
	session.ExpiresAt = time.Now().Add(sm.timeout)
	sm.mu.Unlock()

	return session, true
}

// DeleteSession removes a session
func (sm *SessionManager) DeleteSession(sessionID string) {
	sm.mu.Lock()
	delete(sm.sessions, sessionID)
	sm.mu.Unlock()
}

// cleanupSessions removes expired sessions
func (sm *SessionManager) cleanupSessions() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		sm.mu.Lock()
		for id, session := range sm.sessions {
			if now.After(session.ExpiresAt) {
				delete(sm.sessions, id)
			}
		}
		sm.mu.Unlock()
	}
}

// generateSessionID generates a cryptographically secure session ID
func generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// SessionMiddleware returns a Gin middleware for session management
func SessionMiddleware(sessionManager *SessionManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for session cookie or header
		sessionID := c.GetHeader("X-Session-ID")
		if sessionID == "" {
			// Try to get from cookie
			if cookie, err := c.Request.Cookie("session_id"); err == nil {
				sessionID = cookie.Value
			}
		}

		if sessionID != "" {
			if session, exists := sessionManager.GetSession(sessionID); exists {
				c.Set("session", session)
				c.Set("user_id", session.UserID)
			}
		}

		c.Next()
	}
}

// RequireSessionMiddleware ensures a valid session exists
func RequireSessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, exists := c.Get("session"); !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Valid session required",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}