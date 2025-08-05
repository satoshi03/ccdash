package middleware

import (
	"net/http"
	"os"
	"strings"

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
	apiKey := os.Getenv("CCDASH_API_KEY")
	if apiKey == "" {
		// In development mode, if no API key is set, we'll allow access
		// In production, this should be a fatal error
		if os.Getenv("GIN_MODE") != "release" {
			return &AuthMiddleware{
				apiKey: "",
				publicPaths: []string{
					"/api/v1/health",
					"/api/health",
				},
			}
		}
	}

	return &AuthMiddleware{
		apiKey: apiKey,
		publicPaths: []string{
			"/api/v1/health",
			"/api/health",
		},
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

		// In development mode with no API key set, allow all requests
		if a.apiKey == "" && os.Getenv("GIN_MODE") != "release" {
			c.Next()
			return
		}

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