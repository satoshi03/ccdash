package services

import (
	"database/sql"
	"testing"
	"time"

	"ccdash-backend/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/marcboeker/go-duckdb"
)

func setupAuthTestDB() (*sql.DB, error) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return nil, err
	}

	// Create users table
	_, err = db.Exec(`
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			roles TEXT NOT NULL DEFAULT '["user"]',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_login TIMESTAMP,
			is_active BOOLEAN DEFAULT TRUE,
			failed_login_attempts INTEGER DEFAULT 0,
			locked_until TIMESTAMP NULL
		)
	`)
	if err != nil {
		return nil, err
	}

	// Create refresh tokens table
	_, err = db.Exec(`
		CREATE TABLE refresh_tokens (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			token_hash TEXT NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			revoked_at TIMESTAMP NULL,
			is_revoked BOOLEAN DEFAULT FALSE
		)
	`)
	if err != nil {
		return nil, err
	}

	// Create audit logs table
	_, err = db.Exec(`
		CREATE TABLE audit_logs (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			user_email TEXT,
			action TEXT NOT NULL,
			resource TEXT NOT NULL,
			details TEXT,
			ip_address TEXT,
			user_agent TEXT,
			success BOOLEAN DEFAULT TRUE,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func TestAuthService_RegisterUser(t *testing.T) {
	db, err := setupAuthTestDB()
	require.NoError(t, err)
	defer db.Close()

	auditService := NewAuditService(db)
	authService := NewAuthService(db, "test-secret", auditService)

	t.Run("successful registration", func(t *testing.T) {
		req := models.UserRegistrationRequest{
			Email:    "test@example.com",
			Password: "password123",
			Roles:    []string{"user"},
		}

		user, err := authService.RegisterUser(req, "127.0.0.1", "test-agent")
		require.NoError(t, err)
		assert.NotEmpty(t, user.ID)
		assert.Equal(t, "test@example.com", user.Email)
		assert.True(t, user.IsActive)
		assert.Equal(t, []string{"user"}, user.Roles)
		assert.NotEmpty(t, user.PasswordHash)
	})

	t.Run("duplicate email registration", func(t *testing.T) {
		req := models.UserRegistrationRequest{
			Email:    "test@example.com", // Same email as above
			Password: "password123",
		}

		_, err := authService.RegisterUser(req, "127.0.0.1", "test-agent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("invalid role registration", func(t *testing.T) {
		req := models.UserRegistrationRequest{
			Email:    "test2@example.com",
			Password: "password123",
			Roles:    []string{"invalid-role"},
		}

		_, err := authService.RegisterUser(req, "127.0.0.1", "test-agent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid role")
	})

	t.Run("default roles when none specified", func(t *testing.T) {
		req := models.UserRegistrationRequest{
			Email:    "test3@example.com",
			Password: "password123",
		}

		user, err := authService.RegisterUser(req, "127.0.0.1", "test-agent")
		require.NoError(t, err)
		assert.Equal(t, []string{"user"}, user.Roles)
	})
}

func TestAuthService_LoginUser(t *testing.T) {
	db, err := setupAuthTestDB()
	require.NoError(t, err)
	defer db.Close()

	auditService := NewAuditService(db)
	authService := NewAuthService(db, "test-secret", auditService)

	// Register a user first
	regReq := models.UserRegistrationRequest{
		Email:    "login@example.com",
		Password: "password123",
		Roles:    []string{"user"},
	}
	user, err := authService.RegisterUser(regReq, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	t.Run("successful login", func(t *testing.T) {
		loginReq := models.UserLoginRequest{
			Email:    "login@example.com",
			Password: "password123",
		}

		response, err := authService.LoginUser(loginReq, "127.0.0.1", "test-agent")
		require.NoError(t, err)
		assert.Equal(t, user.ID, response.User.ID)
		assert.NotEmpty(t, response.AccessToken)
		assert.NotEmpty(t, response.RefreshToken)
		assert.Greater(t, response.ExpiresIn, int64(0))
	})

	t.Run("invalid credentials", func(t *testing.T) {
		loginReq := models.UserLoginRequest{
			Email:    "login@example.com",
			Password: "wrongpassword",
		}

		_, err := authService.LoginUser(loginReq, "127.0.0.1", "test-agent")
		assert.Error(t, err)
		assert.Equal(t, "invalid credentials", err.Error())
	})

	t.Run("non-existent user", func(t *testing.T) {
		loginReq := models.UserLoginRequest{
			Email:    "nonexistent@example.com",
			Password: "password123",
		}

		_, err := authService.LoginUser(loginReq, "127.0.0.1", "test-agent")
		assert.Error(t, err)
		assert.Equal(t, "invalid credentials", err.Error())
	})
}

func TestAuthService_ValidateAccessToken(t *testing.T) {
	db, err := setupAuthTestDB()
	require.NoError(t, err)
	defer db.Close()

	auditService := NewAuditService(db)
	authService := NewAuthService(db, "test-secret", auditService)

	// Register and get user
	regReq := models.UserRegistrationRequest{
		Email:    "token@example.com",
		Password: "password123",
		Roles:    []string{"admin"},
	}
	user, err := authService.RegisterUser(regReq, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	// Generate token
	token, err := authService.GenerateAccessToken(user)
	require.NoError(t, err)

	t.Run("valid token", func(t *testing.T) {
		claims, err := authService.ValidateAccessToken(token)
		require.NoError(t, err)
		assert.Equal(t, user.ID, claims.UserID)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.Roles, claims.Roles)
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := authService.ValidateAccessToken("invalid-token")
		assert.Error(t, err)
	})

	t.Run("wrong secret", func(t *testing.T) {
		wrongSecretService := NewAuthService(db, "wrong-secret", auditService)
		_, err := wrongSecretService.ValidateAccessToken(token)
		assert.Error(t, err)
	})
}

func TestAuthService_RefreshToken(t *testing.T) {
	db, err := setupAuthTestDB()
	require.NoError(t, err)
	defer db.Close()

	auditService := NewAuditService(db)
	authService := NewAuthService(db, "test-secret", auditService)

	// Register and login user
	regReq := models.UserRegistrationRequest{
		Email:    "refresh@example.com",
		Password: "password123",
		Roles:    []string{"user"},
	}
	user, err := authService.RegisterUser(regReq, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	loginReq := models.UserLoginRequest{
		Email:    "refresh@example.com",
		Password: "password123",
	}
	loginResponse, err := authService.LoginUser(loginReq, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	t.Run("valid refresh token", func(t *testing.T) {
		response, err := authService.RefreshAccessToken(loginResponse.RefreshToken, "127.0.0.1", "test-agent")
		require.NoError(t, err)
		assert.Equal(t, user.ID, response.User.ID)
		assert.NotEmpty(t, response.AccessToken)
		assert.NotEmpty(t, response.RefreshToken)
		// New refresh token should be different
		assert.NotEqual(t, loginResponse.RefreshToken, response.RefreshToken)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		_, err := authService.RefreshAccessToken("invalid-refresh", "127.0.0.1", "test-agent")
		assert.Error(t, err)
		assert.Equal(t, "invalid refresh token", err.Error())
	})
}

func TestAuthService_HasPermission(t *testing.T) {
	db, err := setupAuthTestDB()
	require.NoError(t, err)
	defer db.Close()

	auditService := NewAuditService(db)
	authService := NewAuthService(db, "test-secret", auditService)

	t.Run("admin has all permissions", func(t *testing.T) {
		adminUser := &models.User{
			ID:    "admin-1",
			Email: "admin@example.com",
			Roles: []string{"admin"},
		}

		assert.True(t, authService.HasPermission(adminUser, models.PermissionViewDashboard))
		assert.True(t, authService.HasPermission(adminUser, models.PermissionExecuteTasks))
		assert.True(t, authService.HasPermission(adminUser, models.PermissionManageSystem))
	})

	t.Run("user has limited permissions", func(t *testing.T) {
		regularUser := &models.User{
			ID:    "user-1",
			Email: "user@example.com",
			Roles: []string{"user"},
		}

		assert.True(t, authService.HasPermission(regularUser, models.PermissionViewDashboard))
		assert.True(t, authService.HasPermission(regularUser, models.PermissionSyncLogs))
		assert.False(t, authService.HasPermission(regularUser, models.PermissionExecuteTasks))
		assert.False(t, authService.HasPermission(regularUser, models.PermissionManageSystem))
	})

	t.Run("viewer has minimal permissions", func(t *testing.T) {
		viewerUser := &models.User{
			ID:    "viewer-1",
			Email: "viewer@example.com",
			Roles: []string{"viewer"},
		}

		assert.True(t, authService.HasPermission(viewerUser, models.PermissionViewDashboard))
		assert.False(t, authService.HasPermission(viewerUser, models.PermissionSyncLogs))
		assert.False(t, authService.HasPermission(viewerUser, models.PermissionExecuteTasks))
	})
}

func TestAuthService_HasAnyRole(t *testing.T) {
	db, err := setupAuthTestDB()
	require.NoError(t, err)
	defer db.Close()

	auditService := NewAuditService(db)
	authService := NewAuthService(db, "test-secret", auditService)

	user := &models.User{
		ID:    "user-1",
		Email: "user@example.com",
		Roles: []string{"user", "viewer"},
	}

	assert.True(t, authService.HasAnyRole(user, "admin", "user"))
	assert.True(t, authService.HasAnyRole(user, "viewer"))
	assert.False(t, authService.HasAnyRole(user, "admin", "superuser"))
}

func TestAuthService_LogoutUser(t *testing.T) {
	db, err := setupAuthTestDB()
	require.NoError(t, err)
	defer db.Close()

	auditService := NewAuditService(db)
	authService := NewAuthService(db, "test-secret", auditService)

	// Register and login user
	regReq := models.UserRegistrationRequest{
		Email:    "logout@example.com",
		Password: "password123",
	}
	user, err := authService.RegisterUser(regReq, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	loginReq := models.UserLoginRequest{
		Email:    "logout@example.com",
		Password: "password123",
	}
	loginResponse, err := authService.LoginUser(loginReq, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	// Logout should revoke all refresh tokens
	err = authService.LogoutUser(user.ID, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	// Try to use refresh token after logout
	_, err = authService.RefreshAccessToken(loginResponse.RefreshToken, "127.0.0.1", "test-agent")
	assert.Error(t, err)
	assert.Equal(t, "invalid refresh token", err.Error())
}

func TestFailedLoginAttempts(t *testing.T) {
	db, err := setupAuthTestDB()
	require.NoError(t, err)
	defer db.Close()

	auditService := NewAuditService(db)
	authService := NewAuthService(db, "test-secret", auditService)

	// Register user
	regReq := models.UserRegistrationRequest{
		Email:    "lockout@example.com",
		Password: "password123",
	}
	_, err = authService.RegisterUser(regReq, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	// Attempt multiple failed logins
	loginReq := models.UserLoginRequest{
		Email:    "lockout@example.com",
		Password: "wrongpassword",
	}

	// First 4 attempts should just fail
	for i := 0; i < 4; i++ {
		_, err := authService.LoginUser(loginReq, "127.0.0.1", "test-agent")
		assert.Error(t, err)
		assert.Equal(t, "invalid credentials", err.Error())
	}

	// 5th attempt should lock the account
	_, err = authService.LoginUser(loginReq, "127.0.0.1", "test-agent")
	assert.Error(t, err)

	// Check that user is locked
	user, err := authService.GetUserByEmail("lockout@example.com")
	require.NoError(t, err)
	assert.Equal(t, 5, user.FailedLoginAttempts)
	assert.NotNil(t, user.LockedUntil)
	assert.True(t, user.LockedUntil.After(time.Now()))

	// Even with correct password, login should fail due to lockout
	correctReq := models.UserLoginRequest{
		Email:    "lockout@example.com",
		Password: "password123",
	}
	_, err = authService.LoginUser(correctReq, "127.0.0.1", "test-agent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "locked until")
}