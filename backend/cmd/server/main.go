package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"claudeee-backend/internal/database"
	"claudeee-backend/internal/handlers"
	"claudeee-backend/internal/services"
)

func main() {
	// Check if database exists and perform initial sync if needed
	isNewDatabase, err := checkAndInitializeDatabase()
	if err != nil {
		log.Fatal("Failed to check/initialize database:", err)
	}

	db, err := database.Initialize()
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
		
		log.Println("New database detected, starting initial log sync in background...")
		
		// Run initialization in a separate goroutine so server can start immediately
		go func() {
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

	api := r.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status": "healthy",
				"message": "Claudeee API is running",
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("Server starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// checkAndInitializeDatabase checks if the database exists and returns true if it's a new database
func checkAndInitializeDatabase() (bool, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}

	dbPath := filepath.Join(homeDir, ".claudeee", "claudeee.db")
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return false, err
	}

	// Check if database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		log.Println("Database file does not exist, will create and perform initial sync")
		return true, nil
	} else if err != nil {
		return false, err
	}

	log.Println("Existing database found")
	return false, nil
}