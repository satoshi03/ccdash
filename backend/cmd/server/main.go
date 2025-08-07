package main

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"ccdash-backend/internal/config"
	"ccdash-backend/internal/database"
	"ccdash-backend/internal/handlers"
	"ccdash-backend/internal/middleware"
	"ccdash-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// isPrivateIP checks if an IP address is in private ranges
func isPrivateIP(ip string) bool {
	privateRanges := []string{
		"10.0.0.0/8",     // Class A private
		"172.16.0.0/12",  // Class B private
		"192.168.0.0/16", // Class C private
		"127.0.0.0/8",    // Loopback
	}
	
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	
	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(parsedIP) {
			return true
		}
	}
	
	return false
}

// isAllowedOrigin checks if an origin should be allowed for CORS
func isAllowedOrigin(origin string, allowedOrigins []string) bool {
	// Check explicit allowed origins first
	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}
	}
	
	// Parse the origin URL to check if it's from a private IP
	parsedURL, err := url.Parse(origin)
	if err != nil {
		return false
	}
	
	// Extract hostname/IP from the URL
	hostname := parsedURL.Hostname()
	if hostname == "" {
		return false
	}
	
	// Allow localhost and 127.0.0.1 always
	if hostname == "localhost" || hostname == "127.0.0.1" {
		return true
	}
	
	// Check if it's a private IP address
	if isPrivateIP(hostname) {
		// Additional security: only allow HTTP/HTTPS on standard ports for private IPs
		port := parsedURL.Port()
		scheme := parsedURL.Scheme
		
		if scheme != "http" && scheme != "https" {
			return false
		}
		
		// SECURITY: Only allow specific development ports for local development
		// No arbitrary port access allowed
		if port == "" || port == "3000" { // Only allow frontend dev server port
			return true
		}
		// Allow standard web ports only if explicitly configured in production
		if (port == "80" || port == "443") && os.Getenv("GIN_MODE") == "release" {
			return true
		}
	}
	
	return false
}

// parseInt parses a string to int with a default value
func parseInt(s string, defaultValue int) int {
	if parsed, err := strconv.Atoi(s); err == nil {
		return parsed
	}
	return defaultValue
}

func main() {
	// Load .env file if it exists (must be done before checking GIN_MODE)
	if err := godotenv.Load(); err != nil {
		// .env file is optional, so just log if not found
		log.Printf("No .env file found or error loading: %v", err)
	} else {
		log.Println("Loaded .env file")
	}

	// Set Gin mode based on environment (after loading .env)
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

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
	projectService := services.NewProjectService(db) // Phase 3: Add ProjectService
	jobService := services.NewJobService(db)         // Phase 2: Add JobService
	jobExecutor := services.NewJobExecutor(jobService, cfg.JobExecutorWorkerCount) // Phase 2: Add JobExecutor with configurable workers

	// Perform initial log sync if this is a new database (in background)
	if isNewDatabase {
		initService := services.GetGlobalInitializationService()
		initService.StartInitialization()

		log.Println("Starting initial log sync in background...")

		// Run initialization using safe goroutine with panic recovery
		middleware.SafeGoRoutineWithErrorCallback("initialization", func() error {
			diffSyncService := services.NewDiffSyncService(db, tokenService, sessionService)
			stats, err := diffSyncService.SyncAllLogs()
			if err != nil {
				log.Printf("Warning: Initial log sync failed: %v", err)
				return err
			}
			
			log.Printf("Initial sync completed: %d files processed, %d new lines",
				stats.ProcessedFiles, stats.NewLines)
			initService.CompleteInitialization(stats.ProcessedFiles, stats.NewLines)
			return nil
		}, func(err error) {
			initService.FailInitialization(err)
		})
	}

	// Start job executor
	jobExecutor.Start()
	defer jobExecutor.Stop()

	// Start job scheduler
	jobScheduler := services.NewJobScheduler(db, jobService, jobExecutor, sessionWindowService, cfg.JobSchedulerPollingInterval)
	jobScheduler.Start()
	defer jobScheduler.Stop()

	handler := handlers.NewHandler(tokenService, sessionService, sessionWindowService, p90PredictionService, projectService, jobService, jobExecutor) // Phase 2: Add JobService and JobExecutor

	// Initialize authentication middleware
	authMiddleware := middleware.NewAuthMiddleware()

	// Initialize rate limiting (60 requests per minute by default)
	rateLimitRequests := 60
	if customLimit := os.Getenv("RATE_LIMIT_REQUESTS_PER_MINUTE"); customLimit != "" {
		if parsed := parseInt(customLimit, 60); parsed > 0 {
			rateLimitRequests = parsed
		}
	}

	r := gin.Default()

	// Apply global panic recovery middleware
	r.Use(middleware.RecoveryMiddleware())

	// Apply rate limiting globally (except for OPTIONS requests)
	r.Use(func(c *gin.Context) {
		if c.Request.Method != "OPTIONS" {
			middleware.RateLimitMiddleware(rateLimitRequests)(c)
		} else {
			c.Next()
		}
	})

	// SECURITY: Removed CORS_ALLOW_ALL functionality to prevent wildcard origin attacks
	// Always use strict CORS policy with explicit origin checking
	{
		// Use custom CORS logic that allows private IP addresses
		explicitlyAllowedOrigins := []string{
			cfg.FrontendURL, // Default: http://localhost:3000
		}
		
		// Add custom origins from environment variable
		if customOrigins := os.Getenv("CORS_ALLOWED_ORIGINS"); customOrigins != "" {
			for _, origin := range strings.Split(customOrigins, ",") {
				origin = strings.TrimSpace(origin)
				if origin != "" {
					explicitlyAllowedOrigins = append(explicitlyAllowedOrigins, origin)
				}
			}
		}
		
		// Custom CORS middleware that allows private IP addresses
		r.Use(func(c *gin.Context) {
			origin := c.Request.Header.Get("Origin")
			
			// Handle preflight requests
			if c.Request.Method == "OPTIONS" {
				if origin != "" && isAllowedOrigin(origin, explicitlyAllowedOrigins) {
					c.Header("Access-Control-Allow-Origin", origin)
					c.Header("Access-Control-Allow-Credentials", "true")
					c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
					c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With, DNT, User-Agent, If-Modified-Since, Cache-Control, Range, X-API-Key")
					c.Header("Access-Control-Max-Age", "86400")
					c.AbortWithStatus(204)
					return
				}
			}
			
			// Handle actual requests
			if origin != "" && isAllowedOrigin(origin, explicitlyAllowedOrigins) {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Range")
			}
			
			c.Next()
		})
		
		log.Printf("CORS: Allowing explicit origins: %v", explicitlyAllowedOrigins)
		log.Println("CORS: Also allowing private IP addresses (10.x.x.x, 172.16-31.x.x, 192.168.x.x, localhost) with strict port validation")
	}

	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})

	api := r.Group("/api")
	// Apply authentication middleware to all API routes
	api.Use(authMiddleware.Authenticate())
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
		
		// Phase 3: Projects API endpoints
		api.GET("/projects", handler.GetAllProjects)
		api.GET("/projects/:id", handler.GetProject)
		api.PUT("/projects/:id", handler.UpdateProject)
		api.DELETE("/projects/:id", handler.DeleteProject)
		api.GET("/projects/:id/sessions", handler.GetProjectSessions)
		// Note: migrate-sessions endpoint removed - migration is handled automatically by DiffSyncService
		
		// Phase 2: Jobs API endpoints
		api.POST("/jobs", handler.CreateJob)
		api.GET("/jobs", handler.GetJobs)
		api.GET("/jobs/:id", handler.GetJobByID)
		api.POST("/jobs/:id/cancel", handler.CancelJob)
		api.DELETE("/jobs/:id", handler.DeleteJob)
		api.GET("/jobs/queue/status", handler.GetJobQueueStatus)
	}

	log.Printf("Server starting on %s:%s", cfg.ServerHost, cfg.ServerPort)
	log.Printf("Database path: %s", cfg.DatabasePath)
	log.Printf("Claude projects directory: %s", cfg.ClaudeProjectsDir)
	log.Printf("Frontend URL: %s", cfg.FrontendURL)
	log.Printf("Job Scheduler polling interval: %v", cfg.JobSchedulerPollingInterval)
	log.Printf("Job Executor worker count: %d", cfg.JobExecutorWorkerCount)
	
	// Log authentication status
	if authMiddleware.IsAuthEnabled() {
		log.Println("API Key authentication: ENABLED")
	} else {
		log.Println("API Key authentication: DISABLED (development mode)")
	}

	if err := r.Run(cfg.ServerHost + ":" + cfg.ServerPort); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
