package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"
	
	"github.com/gin-gonic/gin"
	"ccdash-backend/internal/models"
	"ccdash-backend/internal/services"
)

type Handler struct {
	tokenService        *services.TokenService
	sessionService      *services.SessionService
	sessionWindowService *services.SessionWindowService
	p90PredictionService *services.P90PredictionService
	projectService      *services.ProjectService // Phase 3: Add ProjectService
	jobService          *services.JobService     // Phase 2: Add JobService
	jobExecutor         *services.JobExecutor    // Phase 2: Add JobExecutor
}

func NewHandler(tokenService *services.TokenService, sessionService *services.SessionService, sessionWindowService *services.SessionWindowService, p90PredictionService *services.P90PredictionService, projectService *services.ProjectService, jobService *services.JobService, jobExecutor *services.JobExecutor) *Handler {
	return &Handler{
		tokenService:        tokenService,
		sessionService:      sessionService,
		sessionWindowService: sessionWindowService,
		p90PredictionService: p90PredictionService,
		projectService:      projectService, // Phase 3: Initialize ProjectService
		jobService:          jobService,     // Phase 2: Initialize JobService
		jobExecutor:         jobExecutor,    // Phase 2: Initialize JobExecutor
	}
}

func (h *Handler) GetTokenUsage(c *gin.Context) {
	usage, err := h.tokenService.GetCurrentTokenUsage()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get token usage",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, usage)
}

func (h *Handler) GetSessions(c *gin.Context) {
	sessions, err := h.sessionService.GetAllSessions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get sessions",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"count": len(sessions),
	})
}

func (h *Handler) GetSessionDetails(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Session ID is required",
		})
		return
	}
	
	session, err := h.sessionService.GetSessionByID(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get session details",
			"details": err.Error(),
		})
		return
	}
	
	// Check if pagination is requested
	pageStr := c.Query("page")
	pageSizeStr := c.Query("page_size")
	
	if pageStr != "" || pageSizeStr != "" {
		// Use pagination
		page := 1
		pageSize := 20
		
		if pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}
		
		if pageSizeStr != "" {
			if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
				pageSize = ps
			}
		}
		
		paginatedMessages, err := h.sessionService.GetSessionMessagesPaginated(sessionID, page, pageSize)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get session messages",
				"details": err.Error(),
			})
			return
		}
		
		tokenUsage, err := h.tokenService.GetTokenUsageBySession(sessionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get session token usage",
				"details": err.Error(),
			})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{
			"session": session,
			"messages": paginatedMessages,
			"token_usage": tokenUsage,
		})
	} else {
		// Use existing non-paginated method for backward compatibility
		messages, err := h.sessionService.GetSessionMessages(sessionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get session messages",
				"details": err.Error(),
			})
			return
		}
		
		tokenUsage, err := h.tokenService.GetTokenUsageBySession(sessionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get session token usage",
				"details": err.Error(),
			})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{
			"session": session,
			"messages": messages,
			"token_usage": tokenUsage,
		})
	}
}

func (h *Handler) SyncLogs(c *gin.Context) {
	// Initialize中はsync-logsを受け付けない
	initService := services.GetGlobalInitializationService()
	if initService.IsInitializing() {
		c.JSON(http.StatusConflict, gin.H{
			"error": "System is currently initializing",
			"message": "Please wait for initialization to complete before syncing logs",
			"status": initService.GetState().Status,
		})
		return
	}
	
	db := c.MustGet("db").(*sql.DB)
	
	// Enable differential sync to fix partial log reading issues
	useDiffSync := true
	
	if useDiffSync {
		// Use new differential sync service
		diffSyncService := services.NewDiffSyncService(db, h.tokenService, h.sessionService)
		
		stats, err := diffSyncService.SyncAllLogs()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to sync logs",
				"details": err.Error(),
			})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{
			"message": "Logs synced successfully (differential)",
			"stats": stats,
		})
	} else {
		// Use legacy full sync
		parser := services.NewJSONLParser(db, h.tokenService, h.sessionService)
		
		if err := parser.SyncAllLogs(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to sync logs",
				"details": err.Error(),
			})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{
			"message": "Logs synced successfully (full)",
		})
	}
}

// GetSessionActivityReport returns detailed activity analysis for a session
func (h *Handler) GetSessionActivityReport(c *gin.Context) {
	sessionID := c.Param("id")
	
	report, err := h.sessionService.GetSessionActivityReport(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get session activity report",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, report)
}

func (h *Handler) GetRecentSessions(c *gin.Context) {
	hours := c.DefaultQuery("hours", "720")
	
	sessions, err := h.sessionService.GetAllSessions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get recent sessions",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"hours": hours,
	})
}

func (h *Handler) GetAvailableTokens(c *gin.Context) {
	plan := c.DefaultQuery("plan", "pro")
	
	usage, err := h.tokenService.GetCurrentTokenUsage()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get token usage",
			"details": err.Error(),
		})
		return
	}
	
	availableTokens := usage.UsageLimit - usage.TotalTokens
	if availableTokens < 0 {
		availableTokens = 0
	}
	
	c.JSON(http.StatusOK, gin.H{
		"available_tokens": availableTokens,
		"plan": plan,
		"usage_limit": usage.UsageLimit,
		"used_tokens": usage.TotalTokens,
	})
}

func (h *Handler) GetCurrentMonthCosts(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"current_month_cost": 0.0,
		"currency": "USD",
		"note": "Cost tracking not implemented yet",
	})
}

func (h *Handler) GetTasks(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"tasks": []interface{}{},
		"count": 0,
		"note": "Task scheduling not implemented yet",
	})
}

func (h *Handler) GetSessionWindows(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	
	windows, err := h.sessionWindowService.GetRecentWindows(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get session windows",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"windows": windows,
		"count": len(windows),
	})
}

// GetP90Predictions returns p90 limit predictions for tokens, messages, and costs
func (h *Handler) GetP90Predictions(c *gin.Context) {
	prediction, err := h.p90PredictionService.CalculateP90Limits()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to calculate p90 predictions",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, prediction)
}

// GetP90PredictionsByProject returns p90 predictions for a specific project
func (h *Handler) GetP90PredictionsByProject(c *gin.Context) {
	projectName := c.Param("project")
	if projectName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Project name is required",
		})
		return
	}
	
	prediction, err := h.p90PredictionService.GetP90LimitsByProject(projectName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to calculate p90 predictions for project",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, prediction)
}

// GetBurnRateHistory returns historical burn rate data
func (h *Handler) GetBurnRateHistory(c *gin.Context) {
	hoursStr := c.DefaultQuery("hours", "24")
	
	hours, err := strconv.Atoi(hoursStr)
	if err != nil || hours <= 0 {
		hours = 24
	}
	if hours > 168 { // Max 1 week
		hours = 168
	}
	
	history, err := h.p90PredictionService.GetBurnRateHistory(hours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get burn rate history",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"burn_rate_history": history,
		"hours": hours,
	})
}

// GetInitializationStatus returns the current initialization status
func (h *Handler) GetInitializationStatus(c *gin.Context) {
	initService := services.GetGlobalInitializationService()
	state := initService.GetState()
	c.JSON(http.StatusOK, state)
}

// Phase 3: Projects API Handlers

// GetAllProjects returns all active projects
func (h *Handler) GetAllProjects(c *gin.Context) {
	projects, err := h.projectService.GetAllProjects()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get projects",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"projects": projects,
		"count": len(projects),
	})
}

// GetProject returns a specific project by ID
func (h *Handler) GetProject(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Project ID is required",
		})
		return
	}
	
	project, err := h.projectService.GetProjectByID(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get project",
			"details": err.Error(),
		})
		return
	}
	
	if project == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Project not found",
		})
		return
	}
	
	// Get sessions for this project
	sessions, err := h.sessionService.GetSessionsByProject(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get project sessions",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"project": project,
		"sessions": sessions,
		"session_count": len(sessions),
	})
}

// UpdateProject updates an existing project
func (h *Handler) UpdateProject(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Project ID is required",
		})
		return
	}
	
	// Get existing project
	project, err := h.projectService.GetProjectByID(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get project",
			"details": err.Error(),
		})
		return
	}
	
	if project == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Project not found",
		})
		return
	}
	
	// Parse request body
	var updateRequest struct {
		Description   *string `json:"description"`
		RepositoryURL *string `json:"repository_url"`
		Language      *string `json:"language"`
		Framework     *string `json:"framework"`
		IsActive      *bool   `json:"is_active"`
	}
	
	if err := c.ShouldBindJSON(&updateRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
			"details": err.Error(),
		})
		return
	}
	
	// Update fields if provided
	if updateRequest.Description != nil {
		project.Description = updateRequest.Description
	}
	if updateRequest.RepositoryURL != nil {
		project.RepositoryURL = updateRequest.RepositoryURL
	}
	if updateRequest.Language != nil {
		project.Language = updateRequest.Language
	}
	if updateRequest.Framework != nil {
		project.Framework = updateRequest.Framework
	}
	if updateRequest.IsActive != nil {
		project.IsActive = *updateRequest.IsActive
	}
	
	// Update project
	err = h.projectService.UpdateProject(project)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update project",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"project": project,
		"message": "Project updated successfully",
	})
}

// DeleteProject soft deletes a project
func (h *Handler) DeleteProject(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Project ID is required",
		})
		return
	}
	
	err := h.projectService.DeleteProject(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete project",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Project deleted successfully",
	})
}

// GetProjectSessions returns all sessions for a specific project
func (h *Handler) GetProjectSessions(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Project ID is required",
		})
		return
	}
	
	sessions, err := h.sessionService.GetSessionsByProject(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get project sessions",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"count": len(sessions),
		"project_id": projectID,
	})
}

// MigrateSessionsToProjects migrates sessions without project_id to use projects
func (h *Handler) MigrateSessionsToProjects(c *gin.Context) {
	// Get sessions without project_id
	sessions, err := h.sessionService.GetSessionsWithoutProjectID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get sessions for migration",
			"details": err.Error(),
		})
		return
	}
	
	if len(sessions) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "No sessions need migration",
			"migrated_count": 0,
		})
		return
	}
	
	migratedCount := 0
	errorCount := 0
	
	for _, session := range sessions {
		err := h.sessionService.MigrateSessionToProject(session.ID)
		if err != nil {
			errorCount++
			continue
		}
		migratedCount++
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Migration completed",
		"migrated_count": migratedCount,
		"error_count": errorCount,
		"total_sessions": len(sessions),
	})
}

// Phase 2: Jobs API Handlers

// CreateJob creates a new job
func (h *Handler) CreateJob(c *gin.Context) {
	var req models.CreateJobRequest
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
			"details": err.Error(),
		})
		return
	}
	
	// Validate schedule type
	if req.ScheduleType == "" {
		req.ScheduleType = models.ScheduleTypeImmediate
	}
	
	validScheduleTypes := map[string]bool{
		models.ScheduleTypeImmediate:  true,
		models.ScheduleTypeAfterReset: true,
		models.ScheduleTypeCustom:     true,
	}
	
	if !validScheduleTypes[req.ScheduleType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid schedule type",
			"valid_types": []string{
				models.ScheduleTypeImmediate,
				models.ScheduleTypeAfterReset,
				models.ScheduleTypeCustom,
			},
		})
		return
	}
	
	job, err := h.jobService.CreateJob(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create job",
			"details": err.Error(),
		})
		return
	}
	
	// Queue job for immediate execution
	if req.ScheduleType == models.ScheduleTypeImmediate {
		if err := h.jobExecutor.QueueJob(job.ID); err != nil {
			// Job was created but couldn't be queued - log warning but don't fail
			log.Printf("Warning: Job %s created but couldn't be queued: %v", job.ID, err)
		}
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"job": job,
		"message": "Job created successfully",
	})
}

// GetJobs retrieves jobs with optional filtering
func (h *Handler) GetJobs(c *gin.Context) {
	// Parse query parameters
	var filters models.JobFilters
	
	if projectID := c.Query("project_id"); projectID != "" {
		filters.ProjectID = &projectID
	}
	
	if status := c.Query("status"); status != "" {
		filters.Status = &status
	}
	
	// Parse limit with default
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}
	filters.Limit = limit
	
	// Parse offset
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			filters.Offset = parsedOffset
		}
	}
	
	jobs, err := h.jobService.GetJobs(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get jobs",
			"details": err.Error(),
		})
		return
	}
	
	// Add queue status for context
	queueStatus := h.jobExecutor.GetQueueStatus()
	
	c.JSON(http.StatusOK, gin.H{
		"jobs": jobs,
		"count": len(jobs),
		"filters": filters,
		"queue_status": queueStatus,
	})
}

// GetJobByID retrieves a specific job by ID
func (h *Handler) GetJobByID(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Job ID is required",
		})
		return
	}
	
	job, err := h.jobService.GetJobByID(jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get job",
			"details": err.Error(),
		})
		return
	}
	
	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Job not found",
		})
		return
	}
	
	// Add additional context
	isRunning := false
	runningJobs := h.jobExecutor.GetRunningJobs()
	for _, runningJobID := range runningJobs {
		if runningJobID == jobID {
			isRunning = true
			break
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"job": job,
		"is_running": isRunning,
	})
}

// CancelJob cancels a running job
func (h *Handler) CancelJob(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Job ID is required",
		})
		return
	}
	
	// Check if job exists and is cancellable
	job, err := h.jobService.GetJobByID(jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get job",
			"details": err.Error(),
		})
		return
	}
	
	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Job not found",
		})
		return
	}
	
	// Check if job can be cancelled
	if job.Status != models.JobStatusRunning && job.Status != models.JobStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Job cannot be cancelled",
			"current_status": job.Status,
			"message": "Only running or pending jobs can be cancelled",
		})
		return
	}
	
	// Cancel the job
	err = h.jobExecutor.CancelJob(jobID)
	if err != nil {
		// If not running in executor, just update status
		if job.Status == models.JobStatusPending {
			err = h.jobService.UpdateJobStatus(jobID, models.JobStatusCancelled, nil)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to cancel job",
					"details": err.Error(),
				})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to cancel job",
				"details": err.Error(),
			})
			return
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Job cancelled successfully",
		"job_id": jobID,
	})
}

// DeleteJob deletes a job
func (h *Handler) DeleteJob(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Job ID is required",
		})
		return
	}
	
	err := h.jobService.DeleteJob(jobID)
	if err != nil {
		// Check specific error types
		if strings.Contains(err.Error(), "cannot delete running job") {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Cannot delete running job",
				"message": "Please cancel the job before deleting",
			})
			return
		}
		
		if strings.Contains(err.Error(), "job not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Job not found",
			})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete job",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Job deleted successfully",
		"job_id": jobID,
	})
}

// GetJobQueueStatus returns the current job executor status
func (h *Handler) GetJobQueueStatus(c *gin.Context) {
	status := h.jobExecutor.GetQueueStatus()
	runningJobs := h.jobExecutor.GetRunningJobs()
	
	c.JSON(http.StatusOK, gin.H{
		"queue_status": status,
		"running_jobs": runningJobs,
	})
}

