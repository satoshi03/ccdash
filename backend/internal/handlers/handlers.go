package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"os/exec"
	"strconv"
	"time"

	"ccdash-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	tokenService         *services.TokenService
	sessionService       *services.SessionService
	sessionWindowService *services.SessionWindowService
	p90PredictionService *services.P90PredictionService
}

func NewHandler(tokenService *services.TokenService, sessionService *services.SessionService, sessionWindowService *services.SessionWindowService, p90PredictionService *services.P90PredictionService) *Handler {
	return &Handler{
		tokenService:         tokenService,
		sessionService:       sessionService,
		sessionWindowService: sessionWindowService,
		p90PredictionService: p90PredictionService,
	}
}

func (h *Handler) GetTokenUsage(c *gin.Context) {
	usage, err := h.tokenService.GetCurrentTokenUsage()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get token usage",
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
			"error":   "Failed to get sessions",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"count":    len(sessions),
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
			"error":   "Failed to get session details",
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
				"error":   "Failed to get session messages",
				"details": err.Error(),
			})
			return
		}

		tokenUsage, err := h.tokenService.GetTokenUsageBySession(sessionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to get session token usage",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"session":     session,
			"messages":    paginatedMessages,
			"token_usage": tokenUsage,
		})
	} else {
		// Use existing non-paginated method for backward compatibility
		messages, err := h.sessionService.GetSessionMessages(sessionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to get session messages",
				"details": err.Error(),
			})
			return
		}

		tokenUsage, err := h.tokenService.GetTokenUsageBySession(sessionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to get session token usage",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"session":     session,
			"messages":    messages,
			"token_usage": tokenUsage,
		})
	}
}

func (h *Handler) SyncLogs(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)

	// Enable differential sync to fix partial log reading issues
	useDiffSync := true

	if useDiffSync {
		// Use new differential sync service
		diffSyncService := services.NewDiffSyncService(db, h.tokenService, h.sessionService)

		stats, err := diffSyncService.SyncAllLogs()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to sync logs",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Logs synced successfully (differential)",
			"stats":   stats,
		})
	} else {
		// Use legacy full sync
		parser := services.NewJSONLParser(db, h.tokenService, h.sessionService)

		if err := parser.SyncAllLogs(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to sync logs",
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
			"error":   "Failed to get session activity report",
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
			"error":   "Failed to get recent sessions",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"hours":    hours,
	})
}

func (h *Handler) GetAvailableTokens(c *gin.Context) {
	plan := c.DefaultQuery("plan", "pro")

	usage, err := h.tokenService.GetCurrentTokenUsage()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get token usage",
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
		"plan":             plan,
		"usage_limit":      usage.UsageLimit,
		"used_tokens":      usage.TotalTokens,
	})
}

func (h *Handler) GetCurrentMonthCosts(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"current_month_cost": 0.0,
		"currency":           "USD",
		"note":               "Cost tracking not implemented yet",
	})
}

func (h *Handler) GetTasks(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"tasks": []interface{}{},
		"count": 0,
		"note":  "Task scheduling not implemented yet",
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
			"error":   "Failed to get session windows",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"windows": windows,
		"count":   len(windows),
	})
}

// GetP90Predictions returns p90 limit predictions for tokens, messages, and costs
func (h *Handler) GetP90Predictions(c *gin.Context) {
	prediction, err := h.p90PredictionService.CalculateP90Limits()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to calculate p90 predictions",
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
			"error":   "Failed to calculate p90 predictions for project",
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
			"error":   "Failed to get burn rate history",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"burn_rate_history": history,
		"hours":             hours,
	})
}

// GetInitializationStatus returns the current initialization status
func (h *Handler) GetInitializationStatus(c *gin.Context) {
	initService := services.GetGlobalInitializationService()
	state := initService.GetState()
	c.JSON(http.StatusOK, state)
}

// ClaudeCommandRequest represents the request body for executing Claude Code commands
type ClaudeCommandRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	Command   string `json:"command" binding:"required"`
	Timeout   int    `json:"timeout,omitempty"` // seconds, default 300 (5 minutes)
}

// ClaudeCommandResponse represents the response for Claude Code command execution
type ClaudeCommandResponse struct {
	SessionID string `json:"session_id"`
	Command   string `json:"command"`
	Output    string `json:"output"`
	Error     string `json:"error,omitempty"`
	ExitCode  int    `json:"exit_code"`
	Duration  int64  `json:"duration_ms"`
	Success   bool   `json:"success"`
}

// ExecuteClaudeCommand executes a Claude Code command for a specific session
func (h *Handler) ExecuteClaudeCommand(c *gin.Context) {
	var req ClaudeCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Set default timeout to 5 minutes
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 300
	}

	// Validate session exists
	session, err := h.sessionService.GetSessionByID(req.SessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Session not found",
			"details": err.Error(),
		})
		return
	}

	startTime := time.Now()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// Build the claude command
	// Use --resume flag to target specific session
	claudeCmd := []string{"claude", "--resume", req.SessionID, req.Command}

	// Execute the command
	cmd := exec.CommandContext(ctx, claudeCmd[0], claudeCmd[1:]...)

	// Set working directory to the session's project path if available
	if session.ProjectPath != "" {
		cmd.Dir = session.ProjectPath
	}

	output, execErr := cmd.CombinedOutput()
	duration := time.Since(startTime).Milliseconds()

	response := ClaudeCommandResponse{
		SessionID: req.SessionID,
		Command:   req.Command,
		Output:    string(output),
		Duration:  duration,
		Success:   execErr == nil && cmd.ProcessState != nil && cmd.ProcessState.Success(),
	}

	if execErr != nil {
		response.Error = execErr.Error()
		if exitError, ok := execErr.(*exec.ExitError); ok {
			response.ExitCode = exitError.ExitCode()
		} else {
			response.ExitCode = -1
		}
	}

	// Return appropriate HTTP status
	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusBadRequest, response)
	}
}

// GetAvailableClaudeCommands returns available Claude Code commands and options
func (h *Handler) GetAvailableClaudeCommands(c *gin.Context) {
	commands := []gin.H{
		{
			"name":        "continue",
			"description": "Continue the conversation in the session",
			"example":     "npm run build",
		},
		{
			"name":        "resume",
			"description": "Resume the specific session with a new command",
			"example":     "run the tests",
		},
		{
			"name":        "code_review",
			"description": "Request code review for recent changes",
			"example":     "please review the latest changes",
		},
		{
			"name":        "debug",
			"description": "Debug specific issues",
			"example":     "help me debug this error: [error message]",
		},
		{
			"name":        "optimize",
			"description": "Request code optimization suggestions",
			"example":     "optimize this function for performance",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"commands": commands,
		"note":     "Commands will be executed using 'claude --resume <session_id> <command>'",
	})
}
