package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	
	"github.com/gin-gonic/gin"
	"claudeee-backend/internal/services"
)

type Handler struct {
	tokenService        *services.TokenService
	sessionService      *services.SessionService
	sessionWindowService *services.SessionWindowService
}

func NewHandler(tokenService *services.TokenService, sessionService *services.SessionService, sessionWindowService *services.SessionWindowService) *Handler {
	return &Handler{
		tokenService:        tokenService,
		sessionService:      sessionService,
		sessionWindowService: sessionWindowService,
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

