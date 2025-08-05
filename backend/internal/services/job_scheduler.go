package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"ccdash-backend/internal/models"
)

// JobScheduler manages scheduled job execution
type JobScheduler struct {
	jobService    *JobService
	jobExecutor   *JobExecutor
	windowService *SessionWindowService
	db            *sql.DB
	
	ticker          *time.Ticker
	pollingInterval time.Duration
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	
	// Last known reset time to detect window changes
	lastResetTime *time.Time
	resetMutex    sync.RWMutex
}

// NewJobScheduler creates a new job scheduler
func NewJobScheduler(db *sql.DB, jobService *JobService, jobExecutor *JobExecutor, windowService *SessionWindowService, pollingInterval time.Duration) *JobScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &JobScheduler{
		jobService:      jobService,
		jobExecutor:     jobExecutor,
		windowService:   windowService,
		db:              db,
		pollingInterval: pollingInterval,
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Start starts the scheduler
func (js *JobScheduler) Start() {
	log.Printf("Starting job scheduler with polling interval: %v", js.pollingInterval)
	
	// Use configured polling interval
	js.ticker = time.NewTicker(js.pollingInterval)
	
	js.wg.Add(1)
	go js.schedulerLoop()
}

// Stop stops the scheduler
func (js *JobScheduler) Stop() {
	log.Println("Stopping job scheduler")
	
	js.cancel()
	if js.ticker != nil {
		js.ticker.Stop()
	}
	js.wg.Wait()
	
	log.Println("Job scheduler stopped")
}

// schedulerLoop is the main scheduler loop
func (js *JobScheduler) schedulerLoop() {
	defer js.wg.Done()
	
	// Initial check
	js.checkAndExecuteJobs()
	
	for {
		select {
		case <-js.ctx.Done():
			return
		case <-js.ticker.C:
			js.checkAndExecuteJobs()
		}
	}
}

// checkAndExecuteJobs checks for jobs that need to be executed
func (js *JobScheduler) checkAndExecuteJobs() {
	// Check for after_reset jobs with retry
	if err := js.checkAfterResetJobsWithRetry(); err != nil {
		log.Printf("Error checking after_reset jobs: %v", err)
	}
	
	// Check for delayed and scheduled jobs with retry
	if err := js.checkScheduledJobsWithRetry(); err != nil {
		log.Printf("Error checking scheduled jobs: %v", err)
	}
}

// checkAfterResetJobsWithRetry checks for after_reset jobs with retry logic
func (js *JobScheduler) checkAfterResetJobsWithRetry() error {
	const maxRetries = 3
	const retryDelay = 5 * time.Second
	
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		err := js.checkAfterResetJobs()
		if err == nil {
			return nil
		}
		
		lastErr = err
		log.Printf("Error checking after_reset jobs (attempt %d/%d): %v", i+1, maxRetries, err)
		
		// Check if it's a database connection error
		if isDBConnectionError(err) && i < maxRetries-1 {
			log.Printf("Database connection error detected, retrying in %v...", retryDelay)
			time.Sleep(retryDelay)
			continue
		}
		
		// For non-connection errors, return immediately
		return err
	}
	
	return fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// checkAfterResetJobs checks for jobs scheduled to run after reset
func (js *JobScheduler) checkAfterResetJobs() error {
	// Get current active window
	activeWindow, err := js.windowService.GetActiveWindow()
	if err != nil {
		return fmt.Errorf("failed to get active window: %w", err)
	}
	
	if activeWindow == nil {
		return nil // No active window
	}
	
	// Check if reset time has changed
	js.resetMutex.RLock()
	lastReset := js.lastResetTime
	js.resetMutex.RUnlock()
	
	if lastReset == nil || !activeWindow.ResetTime.Equal(*lastReset) {
		// Reset time has changed, execute after_reset jobs
		log.Printf("SessionWindow reset detected. New reset time: %v", activeWindow.ResetTime)
		
		// Update last reset time
		js.resetMutex.Lock()
		js.lastResetTime = &activeWindow.ResetTime
		js.resetMutex.Unlock()
		
		// Get all pending after_reset jobs
		query := `
			SELECT id FROM jobs 
			WHERE status = ? AND schedule_type = ?
			ORDER BY priority DESC, CAST(created_at AS TIMESTAMP) ASC`
		
		rows, err := js.db.Query(query, models.JobStatusPending, models.ScheduleTypeAfterReset)
		if err != nil {
			return fmt.Errorf("failed to query after_reset jobs: %w", err)
		}
		defer rows.Close()
		
		var jobIDs []string
		for rows.Next() {
			var jobID string
			if err := rows.Scan(&jobID); err != nil {
				return fmt.Errorf("failed to scan job ID: %w", err)
			}
			jobIDs = append(jobIDs, jobID)
		}
		
		// Queue jobs for execution
		for _, jobID := range jobIDs {
			if err := js.jobExecutor.QueueJob(jobID); err != nil {
				log.Printf("Failed to queue after_reset job %s: %v", jobID, err)
			} else {
				log.Printf("Queued after_reset job %s for execution", jobID)
			}
		}
	}
	
	return nil
}

// checkScheduledJobsWithRetry checks for scheduled jobs with retry logic
func (js *JobScheduler) checkScheduledJobsWithRetry() error {
	const maxRetries = 3
	const retryDelay = 5 * time.Second
	
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		err := js.checkScheduledJobs()
		if err == nil {
			return nil
		}
		
		lastErr = err
		log.Printf("Error checking scheduled jobs (attempt %d/%d): %v", i+1, maxRetries, err)
		
		// Check if it's a database connection error
		if isDBConnectionError(err) && i < maxRetries-1 {
			log.Printf("Database connection error detected, retrying in %v...", retryDelay)
			time.Sleep(retryDelay)
			continue
		}
		
		// For non-connection errors, return immediately
		return err
	}
	
	return fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// checkScheduledJobs checks for delayed and scheduled jobs
func (js *JobScheduler) checkScheduledJobs() error {
	now := time.Now().UTC()
	
	
	// Get all pending jobs with scheduled_at <= now
	query := `
		SELECT id, schedule_type, schedule_params, scheduled_at 
		FROM jobs 
		WHERE status = ? 
		AND scheduled_at IS NOT NULL 
		AND scheduled_at <= ?
		AND schedule_type IN (?, ?)
		ORDER BY priority DESC, CAST(scheduled_at AS TIMESTAMP) ASC`
	
	rows, err := js.db.Query(query, 
		models.JobStatusPending, 
		now.Format(time.RFC3339),
		models.ScheduleTypeDelayed,
		models.ScheduleTypeScheduled)
	if err != nil {
		return fmt.Errorf("failed to query scheduled jobs: %w", err)
	}
	defer rows.Close()
	
	type scheduledJob struct {
		ID             string
		ScheduleType   string
		ScheduleParams *string
		ScheduledAt    string
	}
	
	var jobs []scheduledJob
	for rows.Next() {
		var job scheduledJob
		if err := rows.Scan(&job.ID, &job.ScheduleType, &job.ScheduleParams, &job.ScheduledAt); err != nil {
			return fmt.Errorf("failed to scan scheduled job: %w", err)
		}
		jobs = append(jobs, job)
	}
	
	
	// Queue jobs for execution
	for _, job := range jobs {
		if err := js.jobExecutor.QueueJob(job.ID); err != nil {
			log.Printf("Failed to queue scheduled job %s: %v", job.ID, err)
		} else {
			log.Printf("Queued %s job %s for execution", job.ScheduleType, job.ID)
		}
	}
	
	return nil
}

// GetSchedulerStatus returns the current scheduler status
func (js *JobScheduler) GetSchedulerStatus() map[string]interface{} {
	js.resetMutex.RLock()
	lastReset := js.lastResetTime
	js.resetMutex.RUnlock()
	
	status := map[string]interface{}{
		"running": js.ticker != nil,
		"last_check": time.Now().Format(time.RFC3339),
	}
	
	if lastReset != nil {
		status["last_reset_time"] = lastReset.Format(time.RFC3339)
	}
	
	return status
}

// isDBConnectionError checks if an error is a database connection error
func isDBConnectionError(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	connectionErrors := []string{
		"database is locked",
		"connection refused",
		"no such host",
		"connection reset",
		"broken pipe",
		"bad connection",
		"driver: bad connection",
		"sql: database is closed",
	}
	
	for _, connErr := range connectionErrors {
		if strings.Contains(strings.ToLower(errStr), connErr) {
			return true
		}
	}
	
	return false
}