package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"claudeee-backend/internal/database"
	"claudeee-backend/internal/handlers"
	"claudeee-backend/internal/services"
)

func main() {
	db, err := database.Initialize()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	tokenService := services.NewTokenService(db)
	sessionService := services.NewSessionService(db)
	
	handler := handlers.NewHandler(tokenService, sessionService)

	r := gin.Default()
	
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{frontendURL}
	config.AllowCredentials = true
	r.Use(cors.New(config))
	
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status": "healthy",
				"message": "Claudeee API is running",
			})
		})
		
		v1.GET("/token-usage", handler.GetTokenUsage)
		v1.GET("/sessions", handler.GetSessions)
		v1.GET("/sessions/:id", handler.GetSessionDetails)
		v1.GET("/sessions/:id/activity", handler.GetSessionActivityReport)
		v1.POST("/sync-logs", handler.SyncLogs)
	}
	
	api := r.Group("/api")
	{
		api.GET("/token-usage", handler.GetTokenUsage)
		api.GET("/claude/sessions/recent", handler.GetRecentSessions)
		api.GET("/claude/available-tokens", handler.GetAvailableTokens)
		api.GET("/costs/current-month", handler.GetCurrentMonthCosts)
		api.GET("/tasks", handler.GetTasks)
		api.POST("/sync-logs", handler.SyncLogs)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("Server starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}