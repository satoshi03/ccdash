package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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
	
	ticker        *time.Ticker
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	
	// Last known reset time to detect window changes
	lastResetTime *time.Time
	resetMutex    sync.RWMutex
}

// NewJobScheduler creates a new job scheduler
func NewJobScheduler(db *sql.DB, jobService *JobService, jobExecutor *JobExecutor, windowService *SessionWindowService) *JobScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &JobScheduler{
		jobService:    jobService,
		jobExecutor:   jobExecutor,
		windowService: windowService,
		db:            db,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start starts the scheduler
func (js *JobScheduler) Start() {
	log.Println("Starting job scheduler")
	
	// Check every minute
	js.ticker = time.NewTicker(1 * time.Minute)
	
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
	// Check for after_reset jobs
	if err := js.checkAfterResetJobs(); err != nil {
		log.Printf("Error checking after_reset jobs: %v", err)
	}
	
	// Check for delayed and scheduled jobs
	if err := js.checkScheduledJobs(); err != nil {
		log.Printf("Error checking scheduled jobs: %v", err)
	}
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
			ORDER BY priority DESC, created_at ASC`
		
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

// checkScheduledJobs checks for delayed and scheduled jobs
func (js *JobScheduler) checkScheduledJobs() error {
	now := time.Now()
	
	// Get all pending jobs with scheduled_at <= now
	query := `
		SELECT id, schedule_type, schedule_params 
		FROM jobs 
		WHERE status = ? 
		AND scheduled_at IS NOT NULL 
		AND scheduled_at <= CAST(? AS TIMESTAMP)
		AND schedule_type IN (?, ?)
		ORDER BY priority DESC, scheduled_at ASC`
	
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
	}
	
	var jobs []scheduledJob
	for rows.Next() {
		var job scheduledJob
		if err := rows.Scan(&job.ID, &job.ScheduleType, &job.ScheduleParams); err != nil {
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