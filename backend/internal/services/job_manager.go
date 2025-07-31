package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

// JobStatus represents the status of an asynchronous job
type JobStatus string

const (
	JobPending   JobStatus = "pending"
	JobRunning   JobStatus = "running"
	JobCompleted JobStatus = "completed"
	JobFailed    JobStatus = "failed"
	JobCancelled JobStatus = "cancelled"
)

// Job represents an asynchronous Claude Code execution job
type Job struct {
	ID          string     `json:"id"`
	SessionID   string     `json:"session_id"`
	Command     string     `json:"command"`
	Status      JobStatus  `json:"status"`
	Output      string     `json:"output"`
	Error       string     `json:"error,omitempty"`
	Progress    int        `json:"progress"` // 0-100
	StartTime   time.Time  `json:"start_time"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Duration    int64      `json:"duration_ms"`
	ProjectPath string     `json:"project_path,omitempty"`
	YoloMode    bool       `json:"yolo_mode,omitempty"`
	
	// Internal fields
	ctx    context.Context
	cancel context.CancelFunc
}

// JobManager manages asynchronous Claude Code execution jobs
type JobManager struct {
	jobs         map[string]*Job
	mutex        sync.RWMutex
	sessionService *SessionService
	maxJobs      int
	jobTTL       time.Duration
}

// NewJobManager creates a new JobManager instance
func NewJobManager(sessionService *SessionService) *JobManager {
	jm := &JobManager{
		jobs:           make(map[string]*Job),
		sessionService: sessionService,
		maxJobs:        100, // Maximum number of jobs to keep in memory
		jobTTL:         24 * time.Hour, // Jobs expire after 24 hours
	}
	
	// Start cleanup goroutine
	go jm.cleanupExpiredJobs()
	
	return jm
}

// CreateJob creates a new asynchronous job
func (jm *JobManager) CreateJob(sessionID, command string, timeout int, yoloMode bool) (*Job, error) {
	// Validate session exists
	session, err := jm.sessionService.GetSessionByID(sessionID)
	if err != nil {
		return nil, err
	}

	// Create job context with timeout
	if timeout <= 0 {
		timeout = 300 // Default 5 minutes
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)

	job := &Job{
		ID:          uuid.New().String(),
		SessionID:   sessionID,
		Command:     command,
		Status:      JobPending,
		Progress:    0,
		StartTime:   time.Now(),
		ProjectPath: session.ProjectPath,
		YoloMode:    yoloMode,
		ctx:         ctx,
		cancel:      cancel,
	}

	jm.mutex.Lock()
	defer jm.mutex.Unlock()

	// Check max jobs limit
	if len(jm.jobs) >= jm.maxJobs {
		cancel()
		return nil, fmt.Errorf("maximum number of jobs reached")
	}

	jm.jobs[job.ID] = job
	return job, nil
}

// StartJob starts executing a job in the background
func (jm *JobManager) StartJob(jobID string) error {
	jm.mutex.Lock()
	job, exists := jm.jobs[jobID]
	jm.mutex.Unlock()
	
	if !exists {
		return fmt.Errorf("job not found")
	}

	if job.Status != JobPending {
		return fmt.Errorf("job is not in pending status")
	}

	// Update status to running
	jm.mutex.Lock()
	job.Status = JobRunning
	jm.mutex.Unlock()

	// Start execution in goroutine
	go jm.executeJob(job)
	
	return nil
}

// executeJob executes the job in background
func (jm *JobManager) executeJob(job *Job) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Job %s panicked: %v", job.ID, r)
			jm.mutex.Lock()
			job.Status = JobFailed
			job.Error = fmt.Sprintf("Job panicked: %v", r)
			endTime := time.Now()
			job.EndTime = &endTime
			job.Duration = endTime.Sub(job.StartTime).Milliseconds()
			job.cancel()
			jm.mutex.Unlock()
		}
	}()

	startTime := time.Now()
	
	// Build Claude command
	claudeCmd := []string{"claude", "-p"}
	
	// Add yolo mode flag if requested
	if job.YoloMode {
		claudeCmd = append(claudeCmd, "--dangerously-skip-permissions")
	}
	
	// Add resume and command arguments
	claudeCmd = append(claudeCmd, "--resume", job.SessionID, job.Command)

	// Create command with context
	cmd := exec.CommandContext(job.ctx, claudeCmd[0], claudeCmd[1:]...)

	// Set working directory
	if job.ProjectPath != "" {
		cmd.Dir = job.ProjectPath
	}

	// Inherit environment
	cmd.Env = os.Environ()
	cmd.Stdin = nil

	// Set process group
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	log.Printf("Starting async job %s: %v", job.ID, claudeCmd)
	log.Printf("Working directory: %s", cmd.Dir)

	// Execute command
	output, execErr := cmd.CombinedOutput()
	endTime := time.Now()
	duration := endTime.Sub(startTime).Milliseconds()

	// Update job status
	jm.mutex.Lock()
	defer jm.mutex.Unlock()

	job.Output = string(output)
	job.Duration = duration
	job.EndTime = &endTime
	job.Progress = 100

	if execErr != nil {
		job.Status = JobFailed
		job.Error = execErr.Error()
		log.Printf("Job %s failed: %v", job.ID, execErr)
	} else if job.ctx.Err() != nil {
		job.Status = JobCancelled
		job.Error = "Job was cancelled or timed out"
		log.Printf("Job %s cancelled: %v", job.ID, job.ctx.Err())
	} else {
		job.Status = JobCompleted
		log.Printf("Job %s completed successfully in %dms", job.ID, duration)
	}

	job.cancel()
}

// GetJob retrieves a job by ID
func (jm *JobManager) GetJob(jobID string) (*Job, error) {
	jm.mutex.RLock()
	defer jm.mutex.RUnlock()

	job, exists := jm.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found")
	}

	// Return a copy to avoid race conditions
	jobCopy := *job
	return &jobCopy, nil
}

// GetAllJobs returns all jobs (for admin purposes)
func (jm *JobManager) GetAllJobs() []*Job {
	jm.mutex.RLock()
	defer jm.mutex.RUnlock()

	jobs := make([]*Job, 0, len(jm.jobs))
	for _, job := range jm.jobs {
		jobCopy := *job
		jobs = append(jobs, &jobCopy)
	}

	return jobs
}

// GetJobsBySession returns jobs for a specific session
func (jm *JobManager) GetJobsBySession(sessionID string) []*Job {
	jm.mutex.RLock()
	defer jm.mutex.RUnlock()

	var jobs []*Job
	for _, job := range jm.jobs {
		if job.SessionID == sessionID {
			jobCopy := *job
			jobs = append(jobs, &jobCopy)
		}
	}

	return jobs
}

// CancelJob cancels a running job
func (jm *JobManager) CancelJob(jobID string) error {
	jm.mutex.Lock()
	defer jm.mutex.Unlock()

	job, exists := jm.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found")
	}

	if job.Status == JobRunning {
		job.cancel()
		job.Status = JobCancelled
		job.Error = "Job cancelled by user"
		endTime := time.Now()
		job.EndTime = &endTime
		job.Duration = endTime.Sub(job.StartTime).Milliseconds()
	}

	return nil
}

// DeleteJob removes a job from memory
func (jm *JobManager) DeleteJob(jobID string) error {
	jm.mutex.Lock()
	defer jm.mutex.Unlock()

	job, exists := jm.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found")
	}

	// Cancel if still running
	if job.Status == JobRunning {
		job.cancel()
	}

	delete(jm.jobs, jobID)
	return nil
}

// cleanupExpiredJobs removes old jobs to prevent memory leaks
func (jm *JobManager) cleanupExpiredJobs() {
	ticker := time.NewTicker(1 * time.Hour) // Run cleanup every hour
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			jm.mutex.Lock()
			now := time.Now()
			for jobID, job := range jm.jobs {
				// Remove jobs older than TTL
				if now.Sub(job.StartTime) > jm.jobTTL {
					if job.Status == JobRunning {
						job.cancel()
					}
					delete(jm.jobs, jobID)
					log.Printf("Cleaned up expired job: %s", jobID)
				}
			}
			jm.mutex.Unlock()
		}
	}
}