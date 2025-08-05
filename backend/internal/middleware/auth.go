package middleware

import (
	"net/http"
	"strings"

	"ccdash-backend/internal/models"
	"ccdash-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware handles JWT authentication
type AuthMiddleware struct {
	authService  *services.AuthService
	auditService *services.AuditService
}

func NewAuthMiddleware(authService *services.AuthService, auditService *services.AuditService) *AuthMiddleware {
	return &AuthMiddleware{
		authService:  authService,
		auditService: auditService,
	}
}

// RequireAuth middleware that requires valid JWT authentication
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			m.auditService.LogEvent(nil, "", "auth.missing_token", "auth",
				`{"reason": "missing_authorization_header"}`,
				c.ClientIP(), c.GetHeader("User-Agent"), false)
			
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>" format
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			m.auditService.LogEvent(nil, "", "auth.invalid_token_format", "auth",
				`{"reason": "invalid_bearer_format"}`,
				c.ClientIP(), c.GetHeader("User-Agent"), false)
			
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		token := tokenParts[1]
		claims, err := m.authService.ValidateAccessToken(token)
		if err != nil {
			m.auditService.LogEvent(nil, "", "auth.invalid_token", "auth",
				`{"reason": "token_validation_failed"}`,
				c.ClientIP(), c.GetHeader("User-Agent"), false)
			
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Get full user information
		user, err := m.authService.GetUserByID(claims.UserID)
		if err != nil {
			m.auditService.LogEvent(&claims.UserID, claims.Email, "auth.user_not_found", "auth",
				`{"reason": "user_lookup_failed"}`,
				c.ClientIP(), c.GetHeader("User-Agent"), false)
			
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not found",
			})
			c.Abort()
			return
		}

		// Check if user is still active
		if !user.IsActive {
			m.auditService.LogEvent(&user.ID, user.Email, "auth.inactive_user", "auth",
				`{"reason": "user_inactive"}`,
				c.ClientIP(), c.GetHeader("User-Agent"), false)
			
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Account is inactive",
			})
			c.Abort()
			return
		}

		// Store user and claims in context
		c.Set("user", user)
		c.Set("claims", claims)
		c.Set("user_id", user.ID)
		c.Set("user_email", user.Email)
		c.Set("user_roles", user.Roles)

		c.Next()
	}
}

// RequirePermission middleware that requires specific permission
func (m *AuthMiddleware) RequirePermission(permission models.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			c.Abort()
			return
		}

		userObj := user.(*models.User)
		if !m.authService.HasPermission(userObj, permission) {
			m.auditService.LogEvent(&userObj.ID, userObj.Email, "auth.permission_denied", "auth",
				`{"required_permission": "`+string(permission)+`", "user_roles": "`+strings.Join(userObj.Roles, ",")+`"}`,
				c.ClientIP(), c.GetHeader("User-Agent"), false)
			
			c.JSON(http.StatusForbidden, gin.H{
				"error":              "Insufficient permissions",
				"required_permission": string(permission),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireRole middleware that requires specific role(s)
func (m *AuthMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			c.Abort()
			return
		}

		userObj := user.(*models.User)
		if !m.authService.HasAnyRole(userObj, roles...) {
			m.auditService.LogEvent(&userObj.ID, userObj.Email, "auth.role_denied", "auth",
				`{"required_roles": "`+strings.Join(roles, ",")+`", "user_roles": "`+strings.Join(userObj.Roles, ",")+`"}`,
				c.ClientIP(), c.GetHeader("User-Agent"), false)
			
			c.JSON(http.StatusForbidden, gin.H{
				"error":         "Insufficient role",
				"required_roles": roles,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// OptionalAuth middleware that adds user info to context if valid token is present
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.Next()
			return
		}

		token := tokenParts[1]
		claims, err := m.authService.ValidateAccessToken(token)
		if err != nil {
			c.Next()
			return
		}

		user, err := m.authService.GetUserByID(claims.UserID)
		if err != nil || !user.IsActive {
			c.Next()
			return
		}

		// Store user and claims in context
		c.Set("user", user)
		c.Set("claims", claims)
		c.Set("user_id", user.ID)
		c.Set("user_email", user.Email)
		c.Set("user_roles", user.Roles)

		c.Next()
	}
}