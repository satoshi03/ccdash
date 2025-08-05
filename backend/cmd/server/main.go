package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"

	"ccdash-backend/internal/config"
	"ccdash-backend/internal/database"
	"ccdash-backend/internal/handlers"
	"ccdash-backend/internal/middleware"
	"ccdash-backend/internal/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
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
		
		// Allow standard ports or no port specified
		if port == "" || port == "80" || port == "443" || port == "3000" || port == "8080" {
			return true
		}
	}
	
	return false
}

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
	projectService := services.NewProjectService(db) // Phase 3: Add ProjectService
	jobService := services.NewJobService(db)         // Phase 2: Add JobService
	jobExecutor := services.NewJobExecutor(jobService, cfg.JobExecutorWorkerCount) // Phase 2: Add JobExecutor with configurable workers

	// Authentication services (Phase 4: Authentication)
	auditService := services.NewAuditService(db)
	authService := services.NewAuthService(db, cfg.JWTSecret, auditService)
	authMiddleware := middleware.NewAuthMiddleware(authService, auditService)
	authHandler := handlers.NewAuthHandler(authService, auditService)

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

	// Start job executor
	jobExecutor.Start()
	defer jobExecutor.Stop()

	// Start job scheduler
	jobScheduler := services.NewJobScheduler(db, jobService, jobExecutor, sessionWindowService, cfg.JobSchedulerPollingInterval)
	jobScheduler.Start()
	defer jobScheduler.Stop()

	handler := handlers.NewHandler(tokenService, sessionService, sessionWindowService, p90PredictionService, projectService, jobService, jobExecutor) // Phase 2: Add JobService and JobExecutor

	r := gin.Default()

	// Check for permissive CORS mode (useful for development)
	if os.Getenv("CORS_ALLOW_ALL") == "true" {
		corsConfig := cors.DefaultConfig()
		corsConfig.AllowAllOrigins = true
		corsConfig.AllowCredentials = true
		corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
		corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "DNT", "User-Agent", "If-Modified-Since", "Cache-Control", "Range"}
		r.Use(cors.New(corsConfig))
		log.Println("CORS: Allowing all origins (CORS_ALLOW_ALL=true)")
	} else {
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
					c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With, DNT, User-Agent, If-Modified-Since, Cache-Control, Range")
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
		log.Println("CORS: Also allowing all private IP addresses (10.x.x.x, 172.16-31.x.x, 192.168.x.x, localhost)")
	}

	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})

	// Apply rate limiting to all API routes
	r.Use(middleware.APIRateLimit())

	api := r.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":       "healthy",
				"message":      "CCDash API is running",
				"auth_enabled": cfg.AuthEnabled,
			})
		})

		// Authentication endpoints (always available)
		auth := api.Group("/auth")
		auth.Use(middleware.AuthRateLimit()) // Stricter rate limiting for auth
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/logout", authMiddleware.RequireAuth(), authHandler.Logout)
			auth.GET("/profile", authMiddleware.RequireAuth(), authHandler.GetProfile)
			auth.GET("/validate", authMiddleware.RequireAuth(), authHandler.ValidateToken)
		}

		// Admin-only authentication management endpoints
		authAdmin := api.Group("/auth/admin")
		if cfg.AuthEnabled {
			authAdmin.Use(authMiddleware.RequireAuth())
			authAdmin.Use(authMiddleware.RequireRole("admin"))
		}
		{
			authAdmin.GET("/users/:id", authHandler.GetUser)
			authAdmin.PUT("/users/:id/status", authHandler.UpdateUserStatus)
			authAdmin.GET("/audit-logs", authHandler.GetAuditLogs)
			authAdmin.GET("/audit-logs/stats", authHandler.GetAuditLogStats)
		}

		// Dashboard and monitoring endpoints (viewer level access when auth enabled)
		dashboard := api.Group("/")
		if cfg.AuthEnabled {
			dashboard.Use(authMiddleware.RequireAuth())
		}
		{
			dashboard.GET("/initialization-status", handler.GetInitializationStatus)
			dashboard.GET("/token-usage", handler.GetTokenUsage)
			dashboard.GET("/sessions", handler.GetSessions)
			dashboard.GET("/sessions/:id", handler.GetSessionDetails)
			dashboard.GET("/sessions/:id/activity", handler.GetSessionActivityReport)
			dashboard.GET("/claude/sessions/recent", handler.GetRecentSessions)
			dashboard.GET("/claude/available-tokens", handler.GetAvailableTokens)
			dashboard.GET("/costs/current-month", handler.GetCurrentMonthCosts)
			dashboard.GET("/tasks", handler.GetTasks)
			dashboard.GET("/session-windows", handler.GetSessionWindows)
			dashboard.GET("/predictions/p90", handler.GetP90Predictions)
			dashboard.GET("/predictions/p90/project/:project", handler.GetP90PredictionsByProject)
			dashboard.GET("/predictions/burn-rate-history", handler.GetBurnRateHistory)
		}

		// Log sync endpoints (user level access when auth enabled)
		sync := api.Group("/")
		if cfg.AuthEnabled {
			sync.Use(authMiddleware.RequireAuth())
			sync.Use(authMiddleware.RequirePermission("logs:sync"))
		}
		{
			sync.POST("/sync-logs", handler.SyncLogs)
		}
		
		// Phase 3: Projects API endpoints (user level access when auth enabled)
		projects := api.Group("/")
		if cfg.AuthEnabled {
			projects.Use(authMiddleware.RequireAuth())
		}
		{
			projects.GET("/projects", handler.GetAllProjects)
			projects.GET("/projects/:id", handler.GetProject)
			projects.GET("/projects/:id/sessions", handler.GetProjectSessions)
		}

		// Project management endpoints (admin level access when auth enabled)
		projectsAdmin := api.Group("/")
		if cfg.AuthEnabled {
			projectsAdmin.Use(authMiddleware.RequireAuth())
			projectsAdmin.Use(authMiddleware.RequirePermission("system:manage"))
		}
		{
			projectsAdmin.PUT("/projects/:id", handler.UpdateProject)
			projectsAdmin.DELETE("/projects/:id", handler.DeleteProject)
		}
		
		// Phase 2: Jobs API endpoints (task execution permission required when auth enabled)
		jobs := api.Group("/")
		if cfg.AuthEnabled {
			jobs.Use(authMiddleware.RequireAuth())
			jobs.Use(authMiddleware.RequirePermission("tasks:execute"))
		}
		jobs.Use(middleware.TaskRateLimit()) // Always apply strict rate limiting for job operations
		{
			jobs.POST("/jobs", handler.CreateJob)
			jobs.GET("/jobs", handler.GetJobs)
			jobs.GET("/jobs/:id", handler.GetJobByID)
			jobs.POST("/jobs/:id/cancel", handler.CancelJob)
			jobs.DELETE("/jobs/:id", handler.DeleteJob)
			jobs.GET("/jobs/queue/status", handler.GetJobQueueStatus)
		}
	}

	log.Printf("Server starting on %s:%s", cfg.ServerHost, cfg.ServerPort)
	log.Printf("Database path: %s", cfg.DatabasePath)
	log.Printf("Claude projects directory: %s", cfg.ClaudeProjectsDir)
	log.Printf("Frontend URL: %s", cfg.FrontendURL)
	log.Printf("Job Scheduler polling interval: %v", cfg.JobSchedulerPollingInterval)
	log.Printf("Job Executor worker count: %d", cfg.JobExecutorWorkerCount)
	log.Printf("Authentication enabled: %v", cfg.AuthEnabled)
	if cfg.AuthEnabled {
		log.Printf("JWT secret configured: %v", len(cfg.JWTSecret) > 0)
		log.Printf("Authentication endpoints available at /api/auth/*")
		log.Printf("Admin endpoints protected with role-based access control")
	} else {
		log.Printf("Running in development mode - authentication disabled")
		log.Printf("Set AUTH_ENABLED=true to enable authentication")
	}

	if err := r.Run(cfg.ServerHost + ":" + cfg.ServerPort); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
