package services

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"ccdash-backend/internal/models"
)

// JobExecutor manages the execution of jobs
type JobExecutor struct {
	jobService      *JobService
	workerCount     int
	jobQueue        chan string
	cancelMap       map[string]context.CancelFunc
	cancelMutex     sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	commandWhitelist *CommandWhitelist
}

// NewJobExecutor creates a new job executor
func NewJobExecutor(jobService *JobService, workerCount int) *JobExecutor {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &JobExecutor{
		jobService:      jobService,
		workerCount:     workerCount,
		jobQueue:        make(chan string, 100), // Buffer for pending jobs
		cancelMap:       make(map[string]context.CancelFunc),
		ctx:             ctx,
		cancel:          cancel,
		commandWhitelist: NewCommandWhitelist(),
	}
}

// Start starts the job executor workers
func (je *JobExecutor) Start() {
	log.Printf("Starting job executor with %d workers", je.workerCount)
	
	for i := 0; i < je.workerCount; i++ {
		je.wg.Add(1)
		go je.worker(i)
	}
	
	// Start job queue monitor
	je.wg.Add(1)
	go je.queueMonitor()
}

// Stop stops the job executor
func (je *JobExecutor) Stop() {
	log.Println("Stopping job executor...")
	
	// Cancel all running jobs
	je.cancelMutex.Lock()
	for jobID, cancelFunc := range je.cancelMap {
		log.Printf("Cancelling job %s", jobID)
		cancelFunc()
	}
	je.cancelMutex.Unlock()
	
	// Stop the executor
	je.cancel()
	
	// Close job queue
	close(je.jobQueue)
	
	// Wait for all workers to finish
	je.wg.Wait()
	
	log.Println("Job executor stopped")
}

// QueueJob adds a job to the execution queue
func (je *JobExecutor) QueueJob(jobID string) error {
	select {
	case je.jobQueue <- jobID:
		log.Printf("Job %s queued for execution", jobID)
		return nil
	case <-je.ctx.Done():
		return fmt.Errorf("job executor is shutting down")
	default:
		return fmt.Errorf("job queue is full")
	}
}

// CancelJob cancels a running job
func (je *JobExecutor) CancelJob(jobID string) error {
	je.cancelMutex.Lock()
	defer je.cancelMutex.Unlock()
	
	if cancelFunc, exists := je.cancelMap[jobID]; exists {
		log.Printf("Cancelling job %s", jobID)
		cancelFunc()
		delete(je.cancelMap, jobID)
		
		// Update job status
		return je.jobService.UpdateJobStatus(jobID, models.JobStatusCancelled, nil)
	}
	
	return fmt.Errorf("job %s is not running", jobID)
}

// worker is the main worker goroutine
func (je *JobExecutor) worker(workerID int) {
	defer je.wg.Done()
	
	log.Printf("Worker %d started", workerID)
	
	for {
		select {
		case jobID, ok := <-je.jobQueue:
			if !ok {
				log.Printf("Worker %d stopping: job queue closed", workerID)
				return
			}
			
			log.Printf("Worker %d processing job %s", workerID, jobID)
			je.executeJob(jobID)
			
		case <-je.ctx.Done():
			log.Printf("Worker %d stopping: context cancelled", workerID)
			return
		}
	}
}

// queueMonitor periodically checks for pending jobs
func (je *JobExecutor) queueMonitor() {
	defer je.wg.Done()
	
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			je.checkPendingJobs()
		case <-je.ctx.Done():
			return
		}
	}
}

// checkPendingJobs looks for pending immediate jobs and queues them, also checks for stale running jobs
func (je *JobExecutor) checkPendingJobs() {
	// First, check for stale running jobs
	je.checkStaleRunningJobs()
	
	// Then check for pending immediate jobs only
	pendingJobs, err := je.jobService.GetPendingImmediateJobs(10)
	if err != nil {
		log.Printf("Error getting pending immediate jobs: %v", err)
		return
	}
	
	for _, job := range pendingJobs {
		// Skip if job is already queued/running
		je.cancelMutex.RLock()
		_, isRunning := je.cancelMap[job.ID]
		je.cancelMutex.RUnlock()
		
		if isRunning {
			continue
		}
		
		// Queue the job
		select {
		case je.jobQueue <- job.ID:
			log.Printf("Queued pending immediate job %s", job.ID)
		default:
			log.Printf("Job queue full, skipping job %s", job.ID)
		}
	}
}

// checkStaleRunningJobs checks for jobs marked as running but not tracked by executor
func (je *JobExecutor) checkStaleRunningJobs() {
	// Get running jobs from database
	status := models.JobStatusRunning
	filters := models.JobFilters{
		Status: &status,
		Limit:  50,
	}
	
	runningJobs, err := je.jobService.GetJobs(filters)
	if err != nil {
		log.Printf("Error getting running jobs for stale check: %v", err)
		return
	}
	
	for _, job := range runningJobs {
		// Check if job is tracked by executor
		je.cancelMutex.RLock()
		_, isTracked := je.cancelMap[job.ID]
		je.cancelMutex.RUnlock()
		
		if !isTracked {
			// Job is marked as running but not tracked by executor
			log.Printf("Found stale running job %s, checking process status", job.ID)
			
			if job.PID != nil {
				// Check if process actually exists
				if !je.isProcessRunning(*job.PID) {
					log.Printf("Process %d for job %s is not running, marking as failed", *job.PID, job.ID)
					je.jobService.UpdateJobStatus(job.ID, models.JobStatusFailed, nil)
					errorMsg := "Process not found (likely crashed or killed)"
					je.jobService.UpdateJobLogs(job.ID, nil, &errorMsg, nil)
					continue
				}
			}
			
			// Check if job has been running too long (30 minutes timeout)
			if job.StartedAt != nil {
				runningTime := time.Since(*job.StartedAt)
				if runningTime > 30*time.Minute {
					log.Printf("Job %s running too long (%v), marking as failed", job.ID, runningTime)
					
					// Try to kill the process if PID exists
					if job.PID != nil {
						je.killProcess(*job.PID)
					}
					
					je.jobService.UpdateJobStatus(job.ID, models.JobStatusFailed, nil)
					errorMsg := fmt.Sprintf("Job timeout after %v", runningTime)
					exitCode := -1
					je.jobService.UpdateJobLogs(job.ID, nil, &errorMsg, &exitCode)
				}
			}
		}
	}
}

// isProcessRunning checks if a process with given PID is still running
func (je *JobExecutor) isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	
	// Send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// killProcess attempts to gracefully kill a process, then force kill if necessary
func (je *JobExecutor) killProcess(pid int) {
	process, err := os.FindProcess(pid)
	if err != nil {
		log.Printf("Process %d not found", pid)
		return
	}
	
	// Try graceful shutdown first
	log.Printf("Sending SIGTERM to process %d", pid)
	err = process.Signal(syscall.SIGTERM)
	if err != nil {
		log.Printf("Failed to send SIGTERM to process %d: %v", pid, err)
		return
	}
	
	// Wait for graceful shutdown
	time.Sleep(5 * time.Second)
	
	// Check if still running
	if je.isProcessRunning(pid) {
		log.Printf("Process %d still running, sending SIGKILL", pid)
		process.Signal(syscall.SIGKILL)
	}
}

// executeJob executes a single job
func (je *JobExecutor) executeJob(jobID string) {
	// Get job details
	job, err := je.jobService.GetJobByID(jobID)
	if err != nil {
		log.Printf("Error getting job %s: %v", jobID, err)
		return
	}
	
	if job == nil {
		log.Printf("Job %s not found", jobID)
		return
	}
	
	if job.Status != models.JobStatusPending {
		log.Printf("Job %s is not pending (status: %s)", jobID, job.Status)
		return
	}
	
	// Validate command
	if err := je.validateCommand(job.Command); err != nil {
		log.Printf("Invalid command for job %s: %v", jobID, err)
		je.jobService.UpdateJobStatus(jobID, models.JobStatusFailed, nil)
		errMsg := err.Error()
		je.jobService.UpdateJobLogs(jobID, nil, &errMsg, nil)
		return
	}
	
	// Create job context with timeout
	jobCtx, cancel := context.WithTimeout(je.ctx, 30*time.Minute)
	defer cancel()
	
	// Store cancel function
	je.cancelMutex.Lock()
	je.cancelMap[jobID] = cancel
	je.cancelMutex.Unlock()
	
	// Clean up cancel function when done
	defer func() {
		je.cancelMutex.Lock()
		delete(je.cancelMap, jobID)
		je.cancelMutex.Unlock()
	}()
	
	// Build Claude Code command
	cmdArgs := je.buildCommand(job.Command, job.YoloMode)
	
	log.Printf("Executing job %s: %v in directory %s", jobID, cmdArgs, job.ExecutionDirectory)
	
	// Prepare command
	cmd := exec.CommandContext(jobCtx, cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = job.ExecutionDirectory
	
	// Inherit environment variables from backend process
	cmd.Env = os.Environ()
	
	// Ensure Claude Code environment variables are set
	// These are required for claude command to work properly
	claudeCodeSet := false
	entrypointSet := false
	for i, env := range cmd.Env {
		if strings.HasPrefix(env, "CLAUDECODE=") {
			cmd.Env[i] = "CLAUDECODE=1"
			claudeCodeSet = true
		} else if strings.HasPrefix(env, "CLAUDE_CODE_ENTRYPOINT=") {
			cmd.Env[i] = "CLAUDE_CODE_ENTRYPOINT=cli"
			entrypointSet = true
		}
	}
	if !claudeCodeSet {
		cmd.Env = append(cmd.Env, "CLAUDECODE=1")
	}
	if !entrypointSet {
		cmd.Env = append(cmd.Env, "CLAUDE_CODE_ENTRYPOINT=cli")
	}
	
	// Set process attributes to prevent TTY conflicts
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Inherit the same user/group permissions as the backend process
		Credential: nil, // nil means inherit current process credentials
		// Prevent the process from being stopped by TTY signals
		Setsid: true, // Create a new session to detach from controlling terminal
	}
	
	// Set stdin to /dev/null to prevent hanging on input
	devNull, err := os.OpenFile(os.DevNull, os.O_RDONLY, 0)
	if err != nil {
		log.Printf("Error opening /dev/null for job %s: %v", jobID, err)
		je.jobService.UpdateJobStatus(jobID, models.JobStatusFailed, nil)
		errorMsg := fmt.Sprintf("Failed to open /dev/null: %v", err)
		je.jobService.UpdateJobLogs(jobID, nil, &errorMsg, nil)
		return
	}
	defer devNull.Close()
	cmd.Stdin = devNull
	
	// Capture output pipes BEFORE starting command
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Error creating stdout pipe for job %s: %v", jobID, err)
		je.jobService.UpdateJobStatus(jobID, models.JobStatusFailed, nil)
		errorMsg := fmt.Sprintf("Failed to create stdout pipe: %v", err)
		je.jobService.UpdateJobLogs(jobID, nil, &errorMsg, nil)
		return
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("Error creating stderr pipe for job %s: %v", jobID, err)
		je.jobService.UpdateJobStatus(jobID, models.JobStatusFailed, nil)
		errorMsg := fmt.Sprintf("Failed to create stderr pipe: %v", err)
		je.jobService.UpdateJobLogs(jobID, nil, &errorMsg, nil)
		return
	}
	
	// Update job status to running
	err = je.jobService.UpdateJobStatus(jobID, models.JobStatusRunning, nil)
	if err != nil {
		log.Printf("Error updating job %s status to running: %v", jobID, err)
		return
	}
	
	// Start the command
	if err := cmd.Start(); err != nil {
		log.Printf("Error starting command for job %s: %v", jobID, err)
		errorMsg := fmt.Sprintf("Failed to start command: %v", err)
		je.jobService.UpdateJobStatus(jobID, models.JobStatusFailed, nil)
		je.jobService.UpdateJobLogs(jobID, nil, &errorMsg, nil)
		return
	}
	
	// Update with actual PID
	pid := cmd.Process.Pid
	err = je.jobService.UpdateJobStatus(jobID, models.JobStatusRunning, &pid)
	if err != nil {
		log.Printf("Error updating job %s PID: %v", jobID, err)
	}
	
	// Stream output
	var outputBuffer, errorBuffer strings.Builder
	
	// Start output goroutines
	var outputWg sync.WaitGroup
	outputWg.Add(2)
	
	go func() {
		defer outputWg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			outputBuffer.WriteString(line + "\n")
			log.Printf("Job %s stdout: %s", jobID, line)
		}
	}()
	
	go func() {
		defer outputWg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			errorBuffer.WriteString(line + "\n")
			log.Printf("Job %s stderr: %s", jobID, line)
		}
	}()
	
	// Wait for command to complete with timeout handling
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	
	select {
	case err = <-done:
		// Command completed normally
	case <-jobCtx.Done():
		// Context cancelled (timeout or manual cancellation)
		log.Printf("Job %s timed out or was cancelled, killing process", jobID)
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		err = jobCtx.Err()
	}
	
	// Wait for output goroutines to finish
	outputWg.Wait()
	
	// Get output and error logs
	outputLog := outputBuffer.String()
	errorLog := errorBuffer.String()
	
	// Determine exit status
	var exitCode int
	var status string
	
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
			status = models.JobStatusFailed
		} else {
			exitCode = -1
			status = models.JobStatusFailed
			if errorLog == "" {
				errorLog = err.Error()
			}
		}
	} else {
		exitCode = 0
		status = models.JobStatusCompleted
	}
	
	// Check if job was cancelled
	if jobCtx.Err() == context.Canceled {
		status = models.JobStatusCancelled
		if errorLog == "" {
			errorLog = "Job was cancelled"
		}
	}
	
	log.Printf("Job %s completed with status %s, exit code %d", jobID, status, exitCode)
	
	// Update job status and logs
	err = je.jobService.UpdateJobStatus(jobID, status, nil)
	if err != nil {
		log.Printf("Error updating job %s final status: %v", jobID, err)
	}
	
	err = je.jobService.UpdateJobLogs(jobID, &outputLog, &errorLog, &exitCode)
	if err != nil {
		log.Printf("Error updating job %s logs: %v", jobID, err)
	}
}

// validateCommand validates that the command is safe to execute
func (je *JobExecutor) validateCommand(command string) error {
	// Basic command validation
	if command == "" {
		return fmt.Errorf("command cannot be empty")
	}
	
	// First check whitelist
	if err := je.commandWhitelist.ValidateCommand(command); err != nil {
		return err
	}
	
	// Additional safety checks for extremely dangerous patterns
	dangerousPatterns := []string{
		`rm -rf /`, `del /`, `format c:`, `shutdown -h`, `reboot`, 
		`mkfs`, `fdisk`, `parted`, `sudo rm -rf`, `chmod 777 /`,
	}
	
	commandLower := strings.ToLower(command)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(commandLower, pattern) {
			return fmt.Errorf("command contains potentially dangerous pattern: %s", pattern)
		}
	}
	
	// Check for path traversal
	if strings.Contains(command, "..") {
		return fmt.Errorf("command contains path traversal pattern")
	}
	
	return nil
}

// buildCommand builds the full command arguments
func (je *JobExecutor) buildCommand(command string, yoloMode bool) []string {
	args := []string{"claude"}
	
	if yoloMode {
		args = append(args, "--dangerously-skip-permissions")
	}
	
	// Use --print flag for non-interactive mode
	args = append(args, "--print", command)
	
	return args
}

// sanitizeCommand removes dangerous characters from command
func (je *JobExecutor) sanitizeCommand(command string) string {
	// Remove control characters
	re := regexp.MustCompile(`[\x00-\x1f\x7f]`)
	command = re.ReplaceAllString(command, "")
	
	// Remove multiple spaces
	re = regexp.MustCompile(`\s+`)
	command = re.ReplaceAllString(command, " ")
	
	return strings.TrimSpace(command)
}

// isClaudeCodeAvailable checks if Claude Code CLI is available
func (je *JobExecutor) isClaudeCodeAvailable() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

// GetRunningJobs returns a list of currently running job IDs
func (je *JobExecutor) GetRunningJobs() []string {
	je.cancelMutex.RLock()
	defer je.cancelMutex.RUnlock()
	
	runningJobs := make([]string, 0, len(je.cancelMap))
	for jobID := range je.cancelMap {
		runningJobs = append(runningJobs, jobID)
	}
	
	return runningJobs
}

// GetQueueStatus returns the current queue status
func (je *JobExecutor) GetQueueStatus() map[string]interface{} {
	je.cancelMutex.RLock()
	runningCount := len(je.cancelMap)
	je.cancelMutex.RUnlock()
	
	return map[string]interface{}{
		"running_jobs":      runningCount,
		"queued_jobs":       len(je.jobQueue),
		"worker_count":      je.workerCount,
		"claude_available":  je.isClaudeCodeAvailable(),
		"whitelist_enabled": je.commandWhitelist.IsEnabled(),
	}
}