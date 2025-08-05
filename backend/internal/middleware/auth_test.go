package middleware

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ccdash-backend/internal/models"
	"ccdash-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/marcboeker/go-duckdb"
)

func setupTestAuthMiddleware() (*AuthMiddleware, *services.AuthService, *sql.DB, error) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return nil, nil, nil, err
	}

	// Create required tables
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
		return nil, nil, nil, err
	}

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
		return nil, nil, nil, err
	}

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
		return nil, nil, nil, err
	}

	auditService := services.NewAuditService(db)
	authService := services.NewAuthService(db, "test-secret", auditService)
	authMiddleware := NewAuthMiddleware(authService, auditService)

	return authMiddleware, authService, db, nil
}

func TestAuthMiddleware_RequireAuth(t *testing.T) {
	authMiddleware, authService, db, err := setupTestAuthMiddleware()
	require.NoError(t, err)
	defer db.Close()

	// Register a test user
	regReq := models.UserRegistrationRequest{
		Email:    "test@example.com",
		Password: "password123",
		Roles:    []string{"user"},
	}
	user, err := authService.RegisterUser(regReq, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	// Generate valid token
	validToken, err := authService.GenerateAccessToken(user)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)

	t.Run("missing authorization header", func(t *testing.T) {
		r := gin.New()
		r.Use(authMiddleware.RequireAuth())
		r.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, "Authorization header required", response["error"])
	})

	t.Run("invalid authorization header format", func(t *testing.T) {
		r := gin.New()
		r.Use(authMiddleware.RequireAuth())
		r.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, "Invalid authorization header format", response["error"])
	})

	t.Run("invalid token", func(t *testing.T) {
		r := gin.New()
		r.Use(authMiddleware.RequireAuth())
		r.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, "Invalid or expired token", response["error"])
	})

	t.Run("valid token", func(t *testing.T) {
		r := gin.New()
		r.Use(authMiddleware.RequireAuth())
		r.GET("/protected", func(c *gin.Context) {
			userFromContext, _ := c.Get("user")
			userObj := userFromContext.(*models.User)
			c.JSON(http.StatusOK, gin.H{
				"message":  "success",
				"user_id":  userObj.ID,
				"email":    userObj.Email,
			})
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, "success", response["message"])
		assert.Equal(t, user.ID, response["user_id"])
		assert.Equal(t, user.Email, response["email"])
	})
}

func TestAuthMiddleware_RequirePermission(t *testing.T) {
	authMiddleware, authService, db, err := setupTestAuthMiddleware()
	require.NoError(t, err)
	defer db.Close()

	// Register admin user
	adminReq := models.UserRegistrationRequest{
		Email:    "admin@example.com",
		Password: "password123",
		Roles:    []string{"admin"},
	}
	adminUser, err := authService.RegisterUser(adminReq, "127.0.0.1", "test-agent")
	require.NoError(t, err)
	adminToken, err := authService.GenerateAccessToken(adminUser)
	require.NoError(t, err)

	// Register regular user
	userReq := models.UserRegistrationRequest{
		Email:    "user@example.com",
		Password: "password123",
		Roles:    []string{"user"},
	}
	regularUser, err := authService.RegisterUser(userReq, "127.0.0.1", "test-agent")
	require.NoError(t, err)
	userToken, err := authService.GenerateAccessToken(regularUser)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)

	t.Run("admin can execute tasks", func(t *testing.T) {
		r := gin.New()
		r.Use(authMiddleware.RequireAuth())
		r.Use(authMiddleware.RequirePermission(models.PermissionExecuteTasks))
		r.POST("/tasks", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "task created"})
		})

		req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer([]byte(`{}`)))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("regular user cannot execute tasks", func(t *testing.T) {
		r := gin.New()
		r.Use(authMiddleware.RequireAuth())
		r.Use(authMiddleware.RequirePermission(models.PermissionExecuteTasks))
		r.POST("/tasks", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "task created"})
		})

		req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer([]byte(`{}`)))
		req.Header.Set("Authorization", "Bearer "+userToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, "Insufficient permissions", response["error"])
		assert.Equal(t, string(models.PermissionExecuteTasks), response["required_permission"])
	})

	t.Run("regular user can view dashboard", func(t *testing.T) {
		r := gin.New()
		r.Use(authMiddleware.RequireAuth())
		r.Use(authMiddleware.RequirePermission(models.PermissionViewDashboard))
		r.GET("/dashboard", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "dashboard data"})
		})

		req := httptest.NewRequest("GET", "/dashboard", nil)
		req.Header.Set("Authorization", "Bearer "+userToken)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAuthMiddleware_RequireRole(t *testing.T) {
	authMiddleware, authService, db, err := setupTestAuthMiddleware()
	require.NoError(t, err)
	defer db.Close()

	// Register admin user
	adminReq := models.UserRegistrationRequest{
		Email:    "admin@example.com",
		Password: "password123",
		Roles:    []string{"admin"},
	}
	adminUser, err := authService.RegisterUser(adminReq, "127.0.0.1", "test-agent")
	require.NoError(t, err)
	adminToken, err := authService.GenerateAccessToken(adminUser)
	require.NoError(t, err)

	// Register regular user
	userReq := models.UserRegistrationRequest{
		Email:    "user@example.com",
		Password: "password123",
		Roles:    []string{"user"},
	}
	regularUser, err := authService.RegisterUser(userReq, "127.0.0.1", "test-agent")
	require.NoError(t, err)
	userToken, err := authService.GenerateAccessToken(regularUser)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)

	t.Run("admin can access admin endpoint", func(t *testing.T) {
		r := gin.New()
		r.Use(authMiddleware.RequireAuth())
		r.Use(authMiddleware.RequireRole("admin"))
		r.GET("/admin", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "admin panel"})
		})

		req := httptest.NewRequest("GET", "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("regular user cannot access admin endpoint", func(t *testing.T) {
		r := gin.New()
		r.Use(authMiddleware.RequireAuth())
		r.Use(authMiddleware.RequireRole("admin"))
		r.GET("/admin", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "admin panel"})
		})

		req := httptest.NewRequest("GET", "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+userToken)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, "Insufficient role", response["error"])
	})

	t.Run("user can access user or admin endpoint", func(t *testing.T) {
		r := gin.New()
		r.Use(authMiddleware.RequireAuth())
		r.Use(authMiddleware.RequireRole("user", "admin"))
		r.GET("/user-or-admin", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "accessible"})
		})

		req := httptest.NewRequest("GET", "/user-or-admin", nil)
		req.Header.Set("Authorization", "Bearer "+userToken)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAuthMiddleware_OptionalAuth(t *testing.T) {
	authMiddleware, authService, db, err := setupTestAuthMiddleware()
	require.NoError(t, err)
	defer db.Close()

	// Register a test user
	regReq := models.UserRegistrationRequest{
		Email:    "test@example.com",
		Password: "password123",
		Roles:    []string{"user"},
	}
	user, err := authService.RegisterUser(regReq, "127.0.0.1", "test-agent")
	require.NoError(t, err)
	validToken, err := authService.GenerateAccessToken(user)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)

	t.Run("works without token", func(t *testing.T) {
		r := gin.New()
		r.Use(authMiddleware.OptionalAuth())
		r.GET("/optional", func(c *gin.Context) {
			userFromContext, exists := c.Get("user")
			c.JSON(http.StatusOK, gin.H{
				"has_user": exists,
				"user":     userFromContext,
			})
		})

		req := httptest.NewRequest("GET", "/optional", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.False(t, response["has_user"].(bool))
		assert.Nil(t, response["user"])
	})

	t.Run("works with valid token", func(t *testing.T) {
		r := gin.New()
		r.Use(authMiddleware.OptionalAuth())
		r.GET("/optional", func(c *gin.Context) {
			userFromContext, exists := c.Get("user")
			var userData map[string]interface{}
			if exists {
				userObj := userFromContext.(*models.User)
				userData = map[string]interface{}{
					"id":    userObj.ID,
					"email": userObj.Email,
				}
			}
			c.JSON(http.StatusOK, gin.H{
				"has_user": exists,
				"user":     userData,
			})
		})

		req := httptest.NewRequest("GET", "/optional", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.True(t, response["has_user"].(bool))
		userData := response["user"].(map[string]interface{})
		assert.Equal(t, user.ID, userData["id"])
		assert.Equal(t, user.Email, userData["email"])
	})

	t.Run("ignores invalid token", func(t *testing.T) {
		r := gin.New()
		r.Use(authMiddleware.OptionalAuth())
		r.GET("/optional", func(c *gin.Context) {
			userFromContext, exists := c.Get("user")
			c.JSON(http.StatusOK, gin.H{
				"has_user": exists,
				"user":     userFromContext,
			})
		})

		req := httptest.NewRequest("GET", "/optional", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.False(t, response["has_user"].(bool))
		assert.Nil(t, response["user"])
	})
}