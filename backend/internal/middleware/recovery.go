package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// RecoveryMiddleware provides panic recovery with detailed logging
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// Capture stack trace
				stack := debug.Stack()
				
				// Log the panic with stack trace
				log.Printf("PANIC RECOVERED: %v\nRequest: %s %s\nStack trace:\n%s", 
					r, c.Request.Method, c.Request.URL.Path, stack)

				// Return 500 Internal Server Error
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
					"message": "The server encountered an unexpected condition that prevented it from fulfilling the request.",
				})
				c.Abort()
			}
		}()

		c.Next()
	}
}

// SafeGoRoutine runs a function in a goroutine with panic recovery
func SafeGoRoutine(name string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Capture the stack trace
				buf := make([]byte, 1024*64)
				buf = buf[:runtime.Stack(buf, false)]

				log.Printf("PANIC in goroutine '%s': %v\nStack trace:\n%s", name, r, buf)
			}
		}()

		fn()
	}()
}

// SafeGoRoutineWithErrorCallback runs a function in a goroutine with panic recovery and error callback
func SafeGoRoutineWithErrorCallback(name string, fn func() error, onError func(error)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Capture the stack trace
				buf := make([]byte, 1024*64)
				buf = buf[:runtime.Stack(buf, false)]

				log.Printf("PANIC in goroutine '%s': %v\nStack trace:\n%s", name, r, buf)
				
				// Call error callback with panic error
				panicErr := fmt.Errorf("goroutine panic: %v", r)
				if onError != nil {
					onError(panicErr)
				}
			}
		}()

		if err := fn(); err != nil && onError != nil {
			onError(err)
		}
	}()
}