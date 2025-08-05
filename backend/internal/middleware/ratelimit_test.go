package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute) // 2 requests per minute

	t.Run("allows requests within limit", func(t *testing.T) {
		key := "test-key-1"
		
		assert.True(t, limiter.Allow(key))
		assert.True(t, limiter.Allow(key))
		assert.False(t, limiter.Allow(key)) // Third request should be denied
	})

	t.Run("tracks remaining requests", func(t *testing.T) {
		key := "test-key-2"
		
		assert.Equal(t, 2, limiter.GetRemaining(key))
		limiter.Allow(key)
		assert.Equal(t, 1, limiter.GetRemaining(key))
		limiter.Allow(key)
		assert.Equal(t, 0, limiter.GetRemaining(key))
	})

	t.Run("separate keys are tracked independently", func(t *testing.T) {
		key1 := "test-key-3"
		key2 := "test-key-4"
		
		assert.True(t, limiter.Allow(key1))
		assert.True(t, limiter.Allow(key1))
		assert.False(t, limiter.Allow(key1))
		
		// key2 should still have full allowance
		assert.True(t, limiter.Allow(key2))
		assert.True(t, limiter.Allow(key2))
		assert.False(t, limiter.Allow(key2))
	})

	t.Run("reset time is calculated correctly", func(t *testing.T) {
		key := "test-key-5"
		
		// Use up the allowance
		limiter.Allow(key)
		limiter.Allow(key)
		
		resetTime := limiter.GetResetTime(key)
		assert.True(t, resetTime.After(time.Now()))
		assert.True(t, resetTime.Before(time.Now().Add(time.Minute + time.Second)))
	})
}

func TestRateLimiterWithShortWindow(t *testing.T) {
	limiter := NewRateLimiter(2, 100*time.Millisecond) // 2 requests per 100ms

	key := "short-window-key"
	
	// Use up the allowance
	assert.True(t, limiter.Allow(key))
	assert.True(t, limiter.Allow(key))
	assert.False(t, limiter.Allow(key))
	
	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)
	
	// Should be able to make requests again
	assert.True(t, limiter.Allow(key))
	assert.True(t, limiter.Allow(key))
	assert.False(t, limiter.Allow(key))
}

func TestAPIRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("allows requests within limit", func(t *testing.T) {
		// Create a new rate limiter with a small limit for testing
		testLimiter := NewRateLimiter(2, time.Minute)
		
		r := gin.New()
		r.Use(RateLimitMiddleware(testLimiter, func(c *gin.Context) string {
			return "test:" + c.ClientIP()
		}))
		r.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		// First two requests should succeed
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
			assert.NotEmpty(t, w.Header().Get("X-RateLimit-Limit"))
			assert.NotEmpty(t, w.Header().Get("X-RateLimit-Remaining"))
			assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
		}

		// Third request should be rate limited
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))
		assert.NotEmpty(t, w.Header().Get("Retry-After"))
	})
}

func TestAuthRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Mock the authRateLimiter for testing
	originalLimiter := authRateLimiter
	authRateLimiter = NewRateLimiter(1, time.Minute) // Very strict for testing
	defer func() { authRateLimiter = originalLimiter }()

	r := gin.New()
	r.Use(AuthRateLimit())
	r.POST("/auth/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "login attempt"})
	})

	// First request should succeed
	req := httptest.NewRequest("POST", "/auth/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Second request should be rate limited
	req = httptest.NewRequest("POST", "/auth/login", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestTaskRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Mock the taskRateLimiter for testing
	originalLimiter := taskRateLimiter
	taskRateLimiter = NewRateLimiter(1, time.Minute) // Very strict for testing
	defer func() { taskRateLimiter = originalLimiter }()

	r := gin.New()
	r.Use(TaskRateLimit())
	r.POST("/tasks", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "task created"})
	})

	// First request should succeed
	req := httptest.NewRequest("POST", "/tasks", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Second request should be rate limited
	req = httptest.NewRequest("POST", "/tasks", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestCustomRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(CustomRateLimit(1, time.Minute, "custom"))
	r.GET("/custom", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "custom endpoint"})
	})

	// First request should succeed
	req := httptest.NewRequest("GET", "/custom", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Second request should be rate limited
	req = httptest.NewRequest("GET", "/custom", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestRateLimitWithUserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	
	// Middleware to set user context
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user-123")
		c.Next()
	})
	
	r.Use(TaskRateLimit())
	r.POST("/user-tasks", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "user task"})
	})

	// The rate limiter should use the user ID from context
	req := httptest.NewRequest("POST", "/user-tasks", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}