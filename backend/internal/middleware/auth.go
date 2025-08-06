package middleware

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"ccdash-backend/internal/config"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware handles API key authentication
type AuthMiddleware struct {
	apiKey string
	// Whitelist of paths that don't require authentication
	publicPaths []string
}

// NewAuthMiddleware creates a new authentication middleware instance
func NewAuthMiddleware() *AuthMiddleware {
	publicPaths := []string{
		"/api/v1/health",
		"/api/health",
	}
	
	// API Key is now always required - if not set, it will be auto-generated
	
	// Try to get existing API key or generate one
	envFilePath := filepath.Join(".", ".env")
	if homeDir, err := os.UserHomeDir(); err == nil {
		// Prefer .env in user's home directory if it exists
		homeEnvPath := filepath.Join(homeDir, ".env")
		if _, err := os.Stat(homeEnvPath); err == nil {
			envFilePath = homeEnvPath
		}
	}
	
	keyManager := config.NewAPIKeyManager(envFilePath)
	apiKey, isNewKey, err := keyManager.EnsureAPIKey()
	if err != nil {
		log.Printf("❌ Failed to ensure API key: %v", err)
		log.Printf("🚨 API key generation failed - server will not start")
		os.Exit(1)
	}
	
	if isNewKey {
		log.Printf("🎯 Copy the API key above and use it for authentication")
		log.Printf("🌐 Frontend users: Set this key in the authentication form")
	}
	
	return &AuthMiddleware{
		apiKey:      apiKey,
		publicPaths: publicPaths,
	}
}

// Authenticate returns a Gin middleware handler for API key authentication
func (a *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if the path is in the public paths list
		path := c.Request.URL.Path
		for _, publicPath := range a.publicPaths {
			if strings.HasPrefix(path, publicPath) {
				c.Next()
				return
			}
		}

		// API key is now always required (no development bypass)

		// Check for API key in header
		providedKey := c.GetHeader("X-API-Key")
		if providedKey == "" {
			// Also check Authorization header with Bearer token format
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				providedKey = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		// Validate API key
		if providedKey == "" || providedKey != a.apiKey {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized: Invalid or missing API key",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// IsAuthEnabled returns whether authentication is enabled
func (a *AuthMiddleware) IsAuthEnabled() bool {
	return a.apiKey != ""
}