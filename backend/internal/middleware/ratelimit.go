package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter represents a rate limiter
type RateLimiter struct {
	mu       sync.RWMutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// cleanup removes old entries from the rate limiter
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, times := range rl.requests {
			// Remove times outside the window
			var validTimes []time.Time
			for _, t := range times {
				if now.Sub(t) < rl.window {
					validTimes = append(validTimes, t)
				}
			}
			
			if len(validTimes) == 0 {
				delete(rl.requests, key)
			} else {
				rl.requests[key] = validTimes
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	
	// Get existing requests for this key
	times, exists := rl.requests[key]
	if !exists {
		times = []time.Time{}
	}

	// Remove old requests outside the window
	var validTimes []time.Time
	for _, t := range times {
		if now.Sub(t) < rl.window {
			validTimes = append(validTimes, t)
		}
	}

	// Check if we can allow this request
	if len(validTimes) >= rl.limit {
		return false
	}

	// Add current request
	validTimes = append(validTimes, now)
	rl.requests[key] = validTimes

	return true
}

// GetRemaining returns the number of remaining requests for a key
func (rl *RateLimiter) GetRemaining(key string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	times, exists := rl.requests[key]
	if !exists {
		return rl.limit
	}

	now := time.Now()
	var validCount int
	for _, t := range times {
		if now.Sub(t) < rl.window {
			validCount++
		}
	}

	remaining := rl.limit - validCount
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetResetTime returns when the rate limit will reset for a key
func (rl *RateLimiter) GetResetTime(key string) time.Time {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	times, exists := rl.requests[key]
	if !exists || len(times) == 0 {
		return time.Now()
	}

	// Find the oldest valid request
	now := time.Now()
	var oldestValid *time.Time
	for _, t := range times {
		if now.Sub(t) < rl.window {
			if oldestValid == nil || t.Before(*oldestValid) {
				oldestValid = &t
			}
		}
	}

	if oldestValid == nil {
		return time.Now()
	}

	return oldestValid.Add(rl.window)
}

// Rate limiting middleware configurations
var (
	// General API rate limiter: 100 requests per minute
	apiRateLimiter = NewRateLimiter(100, time.Minute)
	
	// Authentication rate limiter: 10 requests per minute (stricter for auth endpoints)
	authRateLimiter = NewRateLimiter(10, time.Minute)
	
	// Task execution rate limiter: 5 requests per minute (very strict for dangerous operations)
	taskRateLimiter = NewRateLimiter(5, time.Minute)
)

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(limiter *RateLimiter, keyGenerator func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := keyGenerator(c)
		
		if !limiter.Allow(key) {
			resetTime := limiter.GetResetTime(key)
			
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.limit))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))
			c.Header("Retry-After", fmt.Sprintf("%d", int(time.Until(resetTime).Seconds())+1))
			
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": fmt.Sprintf("Too many requests. Try again in %v", time.Until(resetTime).Round(time.Second)),
				"retry_after": int(time.Until(resetTime).Seconds()) + 1,
			})
			c.Abort()
			return
		}

		remaining := limiter.GetRemaining(key)
		resetTime := limiter.GetResetTime(key)
		
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))
		
		c.Next()
	}
}

// API rate limiting - by IP address
func APIRateLimit() gin.HandlerFunc {
	return RateLimitMiddleware(apiRateLimiter, func(c *gin.Context) string {
		return "api:" + c.ClientIP()
	})
}

// Auth rate limiting - by IP address (stricter)
func AuthRateLimit() gin.HandlerFunc {
	return RateLimitMiddleware(authRateLimiter, func(c *gin.Context) string {
		return "auth:" + c.ClientIP()
	})
}

// Task execution rate limiting - by user ID if authenticated, otherwise IP
func TaskRateLimit() gin.HandlerFunc {
	return RateLimitMiddleware(taskRateLimiter, func(c *gin.Context) string {
		if userID, exists := c.Get("user_id"); exists {
			return "task:user:" + userID.(string)
		}
		return "task:ip:" + c.ClientIP()
	})
}

// Custom rate limiting with configurable parameters
func CustomRateLimit(limit int, window time.Duration, keyPrefix string) gin.HandlerFunc {
	limiter := NewRateLimiter(limit, window)
	return RateLimitMiddleware(limiter, func(c *gin.Context) string {
		if userID, exists := c.Get("user_id"); exists {
			return keyPrefix + ":user:" + userID.(string)
		}
		return keyPrefix + ":ip:" + c.ClientIP()
	})
}