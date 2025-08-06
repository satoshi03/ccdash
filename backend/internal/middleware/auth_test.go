package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		apiKey         string
		requestHeader  string
		requestValue   string
		path           string
		expectedStatus int
		ginMode        string
	}{
		{
			name:           "Valid API key in X-API-Key header",
			apiKey:         "test-api-key-123",
			requestHeader:  "X-API-Key",
			requestValue:   "test-api-key-123",
			path:           "/api/test",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid API key in Authorization header",
			apiKey:         "test-api-key-123",
			requestHeader:  "Authorization",
			requestValue:   "Bearer test-api-key-123",
			path:           "/api/test",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid API key",
			apiKey:         "test-api-key-123",
			requestHeader:  "X-API-Key",
			requestValue:   "wrong-key",
			path:           "/api/test",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Missing API key",
			apiKey:         "test-api-key-123",
			requestHeader:  "",
			requestValue:   "",
			path:           "/api/test",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Public path without API key",
			apiKey:         "test-api-key-123",
			requestHeader:  "",
			requestValue:   "",
			path:           "/api/v1/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Development mode without API key set",
			apiKey:         "",
			requestHeader:  "",
			requestValue:   "",
			path:           "/api/test",
			expectedStatus: http.StatusOK,
			ginMode:        "debug",
		},
		{
			name:           "Production mode without API key set should block",
			apiKey:         "",
			requestHeader:  "",
			requestValue:   "",
			path:           "/api/test",
			expectedStatus: http.StatusUnauthorized,
			ginMode:        "release",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore environment variables
			oldAPIKey := os.Getenv("CCDASH_API_KEY")
			oldGinMode := os.Getenv("GIN_MODE")
			defer func() {
				os.Setenv("CCDASH_API_KEY", oldAPIKey)
				os.Setenv("GIN_MODE", oldGinMode)
			}()

			// Set test environment
			if tt.apiKey != "" {
				os.Setenv("CCDASH_API_KEY", tt.apiKey)
			} else {
				os.Unsetenv("CCDASH_API_KEY")
			}
			if tt.ginMode != "" {
				os.Setenv("GIN_MODE", tt.ginMode)
			}

			// Create middleware
			auth := NewAuthMiddleware()

			// Create test router
			router := gin.New()
			router.Use(auth.Authenticate())
			router.GET("/api/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})
			router.GET("/api/v1/health", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			// Create test request
			req, _ := http.NewRequest("GET", tt.path, nil)
			if tt.requestHeader != "" {
				req.Header.Set(tt.requestHeader, tt.requestValue)
			}

			// Perform request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check response
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestIsAuthEnabled(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected bool
	}{
		{
			name:     "With API key",
			apiKey:   "test-key",
			expected: true,
		},
		{
			name:     "Without API key",
			apiKey:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldAPIKey := os.Getenv("CCDASH_API_KEY")
			defer os.Setenv("CCDASH_API_KEY", oldAPIKey)

			if tt.apiKey != "" {
				os.Setenv("CCDASH_API_KEY", tt.apiKey)
			} else {
				os.Unsetenv("CCDASH_API_KEY")
			}

			auth := NewAuthMiddleware()
			assert.Equal(t, tt.expected, auth.IsAuthEnabled())
		})
	}
}