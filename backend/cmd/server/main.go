package main

import (
	"fmt"
	"log"
	"net/http"
	"runtime"

	"ccdash-backend/internal/config"
	"ccdash-backend/internal/database"
	"ccdash-backend/internal/handlers"
	"ccdash-backend/internal/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Check if database exists and perform initial sync if needed
	isNewDatabase := !cfg.DatabaseExists()
	if isNewDatabase {
		log.Println("New database detected")
	}

	db, err := database.InitializeWithConfig(cfg)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	tokenService := services.NewTokenService(db)
	sessionService := services.NewSessionService(db)
	sessionWindowService := services.NewSessionWindowService(db)
	p90PredictionService := services.NewP90PredictionService(db)

	// Perform initial log sync if this is a new database (in background)
	if isNewDatabase {
		initService := services.GetGlobalInitializationService()
		initService.StartInitialization()

		log.Println("Starting initial log sync in background...")

		// Run initialization in a separate goroutine with panic recovery
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// Capture the stack trace
					buf := make([]byte, 1024*64)
					buf = buf[:runtime.Stack(buf, false)]

					log.Printf("PANIC in initialization goroutine: %v\nStack trace:\n%s", r, buf)

					// Report panic as initialization failure
					panicErr := fmt.Errorf("initialization panic: %v", r)
					initService.FailInitialization(panicErr)
				}
			}()

			diffSyncService := services.NewDiffSyncService(db, tokenService, sessionService)
			stats, err := diffSyncService.SyncAllLogs()
			if err != nil {
				log.Printf("Warning: Initial log sync failed: %v", err)
				initService.FailInitialization(err)
			} else {
				log.Printf("Initial sync completed: %d files processed, %d new lines",
					stats.ProcessedFiles, stats.NewLines)
				initService.CompleteInitialization(stats.ProcessedFiles, stats.NewLines)
			}
		}()
	}

	handler := handlers.NewHandler(tokenService, sessionService, sessionWindowService, p90PredictionService)

	r := gin.Default()

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{cfg.FrontendURL}
	corsConfig.AllowCredentials = true
	r.Use(cors.New(corsConfig))

	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})

	api := r.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":  "healthy",
				"message": "CCDash API is running",
			})
		})

		api.GET("/initialization-status", handler.GetInitializationStatus)
		api.GET("/token-usage", handler.GetTokenUsage)
		api.GET("/sessions", handler.GetSessions)
		api.GET("/sessions/:id", handler.GetSessionDetails)
		api.GET("/sessions/:id/activity", handler.GetSessionActivityReport)
		api.GET("/claude/sessions/recent", handler.GetRecentSessions)
		api.GET("/claude/available-tokens", handler.GetAvailableTokens)
		api.GET("/costs/current-month", handler.GetCurrentMonthCosts)
		api.GET("/tasks", handler.GetTasks)
		api.GET("/session-windows", handler.GetSessionWindows)
		api.GET("/predictions/p90", handler.GetP90Predictions)
		api.GET("/predictions/p90/project/:project", handler.GetP90PredictionsByProject)
		api.GET("/predictions/burn-rate-history", handler.GetBurnRateHistory)
		api.POST("/sync-logs", handler.SyncLogs)
	}

	log.Printf("Server starting on %s:%s", cfg.ServerHost, cfg.ServerPort)
	log.Printf("Database path: %s", cfg.DatabasePath)
	log.Printf("Claude projects directory: %s", cfg.ClaudeProjectsDir)
	log.Printf("Frontend URL: %s", cfg.FrontendURL)

	if err := r.Run(cfg.ServerHost + ":" + cfg.ServerPort); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
