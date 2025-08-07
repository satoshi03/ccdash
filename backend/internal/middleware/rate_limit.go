package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter tracks request rates per IP address
type RateLimiter struct {
	visitors map[string]*Visitor
	mu       sync.RWMutex
	rate     time.Duration // Rate limit interval
	capacity int           // Number of requests allowed per interval
}

// Visitor represents a client's request history
type Visitor struct {
	limiter  chan struct{}
	lastSeen time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*Visitor),
		rate:     time.Minute,
		capacity: requestsPerMinute,
	}

	// Start cleanup goroutine to remove old visitors
	go rl.cleanupVisitors()
	return rl
}

// getVisitor returns the rate limiter for the given IP
func (rl *RateLimiter) getVisitor(ip string) *Visitor {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		// Create a new visitor with a buffered channel as a token bucket
		limiter := make(chan struct{}, rl.capacity)
		
		// Fill the bucket initially
		for i := 0; i < rl.capacity; i++ {
			limiter <- struct{}{}
		}

		v = &Visitor{
			limiter:  limiter,
			lastSeen: time.Now(),
		}
		rl.visitors[ip] = v

		// Start token refill goroutine for this visitor
		go rl.refillTokens(v)
	}

	v.lastSeen = time.Now()
	return v
}

// refillTokens refills the token bucket at the specified rate
func (rl *RateLimiter) refillTokens(v *Visitor) {
	ticker := time.NewTicker(rl.rate / time.Duration(rl.capacity))
	defer ticker.Stop()

	for range ticker.C {
		select {
		case v.limiter <- struct{}{}:
			// Token added successfully
		default:
			// Bucket is full, skip
		}

		// Check if visitor is still active
		rl.mu.RLock()
		if time.Since(v.lastSeen) > 5*time.Minute {
			rl.mu.RUnlock()
			return // Stop refilling for inactive visitors
		}
		rl.mu.RUnlock()
	}
}

// cleanupVisitors removes visitors that haven't been seen for a while
func (rl *RateLimiter) cleanupVisitors() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > 5*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(ip string) bool {
	visitor := rl.getVisitor(ip)

	select {
	case <-visitor.limiter:
		return true
	default:
		return false
	}
}

// RateLimitMiddleware returns a Gin middleware for rate limiting
func RateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	limiter := NewRateLimiter(requestsPerMinute)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		
		if !limiter.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}