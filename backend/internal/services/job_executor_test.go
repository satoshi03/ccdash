package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"ccdash-backend/internal/models"

	_ "github.com/marcboeker/go-duckdb"
)

// setupJobExecutorTestDB creates an in-memory database for testing
func setupJobExecutorTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create necessary tables
	queries := []string{
		`CREATE TABLE projects (
			id VARCHAR PRIMARY KEY,
			name VARCHAR NOT NULL,
			path VARCHAR NOT NULL,
			description TEXT,
			repository_url VARCHAR,
			language VARCHAR,
			framework VARCHAR,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE jobs (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			command TEXT NOT NULL,
			execution_directory TEXT NOT NULL,
			yolo_mode BOOLEAN DEFAULT FALSE,
			status TEXT NOT NULL DEFAULT 'pending',
			priority INTEGER DEFAULT 0,
			created_at TEXT NOT NULL,
			started_at TEXT,
			completed_at TEXT,
			output_log TEXT,
			error_log TEXT,
			exit_code INTEGER,
			pid INTEGER,
			scheduled_at TEXT,
			schedule_type TEXT,
			FOREIGN KEY (project_id) REFERENCES projects(id)
		)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			t.Fatalf("Failed to create test table: %v", err)
		}
	}

	// Insert test project
	_, err = db.Exec(`INSERT INTO projects (id, name, path) VALUES ('test-project', 'Test Project', '/test/path')`)
	if err != nil {
		t.Fatalf("Failed to insert test project: %v", err)
	}

	return db
}

// createTestJob creates a test job in the database
func createTestJob(t *testing.T, db *sql.DB, id, command, status string) {
	query := `INSERT INTO jobs (id, project_id, command, execution_directory, status, created_at, scheduled_at, schedule_type) 
			  VALUES (?, 'test-project', ?, '/test/dir', ?, ?, ?, 'immediate')`
	now := time.Now().Format(time.RFC3339)
	_, err := db.Exec(query, id, command, status, now, now)
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}
}

func TestNewJobExecutor(t *testing.T) {
	db := setupJobExecutorTestDB(t)
	defer db.Close()

	jobService := NewJobService(db)
	executor := NewJobExecutor(jobService, 2)

	if executor == nil {
		t.Fatal("NewJobExecutor returned nil")
	}
	if executor.workerCount != 2 {
		t.Errorf("Expected worker count 2, got %d", executor.workerCount)
	}
	if executor.jobService != jobService {
		t.Error("JobService not set correctly")
	}
	if executor.jobQueue == nil {
		t.Error("Job queue not initialized")
	}
	if executor.cancelMap == nil {
		t.Error("Cancel map not initialized")
	}
}

func TestJobExecutor_QueueJob(t *testing.T) {
	db := setupJobExecutorTestDB(t)
	defer db.Close()

	jobService := NewJobService(db)
	executor := NewJobExecutor(jobService, 1)

	// Test successful queuing
	err := executor.QueueJob("test-job-1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test queue capacity (buffer size is 100)
	for i := 0; i < 99; i++ {
		err := executor.QueueJob(fmt.Sprintf("test-job-%d", i+2))
		if err != nil {
			t.Errorf("Expected no error for job %d, got %v", i+2, err)
		}
	}

	// This should fail as queue is full
	err = executor.QueueJob("overflow-job")
	if err == nil {
		t.Error("Expected error for queue overflow, got nil")
	}
	if !strings.Contains(err.Error(), "queue is full") {
		t.Errorf("Expected 'queue is full' error, got %v", err)
	}
}

func TestJobExecutor_QueueJobAfterStop(t *testing.T) {
	db := setupJobExecutorTestDB(t)
	defer db.Close()

	jobService := NewJobService(db)
	executor := NewJobExecutor(jobService, 1)

	// Stop the executor
	executor.cancel()

	// Try to queue a job - should fail
	err := executor.QueueJob("test-job")
	if err == nil {
		t.Error("Expected error after executor stop, got nil")
	}
	if !strings.Contains(err.Error(), "shutting down") {
		t.Errorf("Expected 'shutting down' error, got %v", err)
	}
}

func TestJobExecutor_ValidateCommand(t *testing.T) {
	db := setupJobExecutorTestDB(t)
	defer db.Close()

	jobService := NewJobService(db)
	executor := NewJobExecutor(jobService, 1)

	tests := []struct {
		name    string
		command string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid command",
			command: "echo hello",
			wantErr: false,
		},
		{
			name:    "empty command",
			command: "",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "command with semicolon",
			command: "echo hello; rm -rf /",
			wantErr: true,
			errMsg:  "dangerous pattern",
		},
		{
			name:    "command with pipe",
			command: "cat file | grep secret",
			wantErr: true,
			errMsg:  "dangerous pattern",
		},
		{
			name:    "command with path traversal",
			command: "cat ../../../etc/passwd",
			wantErr: true,
			errMsg:  "path traversal",
		},
		{
			name:    "rm -rf command",
			command: "rm -rf /home/user",
			wantErr: true,
			errMsg:  "dangerous pattern",
		},
		{
			name:    "shutdown command",
			command: "shutdown -h now",
			wantErr: true,
			errMsg:  "dangerous pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.validateCommand(tt.command)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for command '%s', got nil", tt.command)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing '%s', got %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for command '%s', got %v", tt.command, err)
				}
			}
		})
	}
}

func TestJobExecutor_BuildCommand(t *testing.T) {
	db := setupJobExecutorTestDB(t)
	defer db.Close()

	jobService := NewJobService(db)
	executor := NewJobExecutor(jobService, 1)

	tests := []struct {
		name     string
		command  string
		yoloMode bool
		expected []string
	}{
		{
			name:     "normal mode",
			command:  "echo hello",
			yoloMode: false,
			expected: []string{"claude", "-p", "echo hello"},
		},
		{
			name:     "yolo mode",
			command:  "ls -la",
			yoloMode: true,
			expected: []string{"claude", "--dangerously-skip-permissions", "-p", "ls -la"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.buildCommand(tt.command, tt.yoloMode)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d args, got %d", len(tt.expected), len(result))
				return
			}
			for i, arg := range result {
				if arg != tt.expected[i] {
					t.Errorf("Expected arg[%d] = '%s', got '%s'", i, tt.expected[i], arg)
				}
			}
		})
	}
}

func TestJobExecutor_SanitizeCommand(t *testing.T) {
	db := setupJobExecutorTestDB(t)
	defer db.Close()

	jobService := NewJobService(db)
	executor := NewJobExecutor(jobService, 1)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal command",
			input:    "echo hello",
			expected: "echo hello",
		},
		{
			name:     "command with multiple spaces",
			input:    "echo    hello    world",
			expected: "echo hello world",
		},
		{
			name:     "command with leading/trailing spaces",
			input:    "  echo hello  ",
			expected: "echo hello",
		},
		{
			name:     "command with control characters",
			input:    "echo\x00hello\x1fworld",
			expected: "echohelloworld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.sanitizeCommand(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestJobExecutor_GetRunningJobs(t *testing.T) {
	db := setupJobExecutorTestDB(t)
	defer db.Close()

	jobService := NewJobService(db)
	executor := NewJobExecutor(jobService, 1)

	// Initially should be empty
	runningJobs := executor.GetRunningJobs()
	if len(runningJobs) != 0 {
		t.Errorf("Expected 0 running jobs initially, got %d", len(runningJobs))
	}

	// Add some mock running jobs
	_, cancel1 := context.WithCancel(context.Background())
	_, cancel2 := context.WithCancel(context.Background())
	
	executor.cancelMutex.Lock()
	executor.cancelMap["job-1"] = cancel1
	executor.cancelMap["job-2"] = cancel2
	executor.cancelMutex.Unlock()

	runningJobs = executor.GetRunningJobs()
	if len(runningJobs) != 2 {
		t.Errorf("Expected 2 running jobs, got %d", len(runningJobs))
	}

	// Check that the job IDs are correct
	jobIDSet := make(map[string]bool)
	for _, jobID := range runningJobs {
		jobIDSet[jobID] = true
	}
	if !jobIDSet["job-1"] || !jobIDSet["job-2"] {
		t.Errorf("Expected job-1 and job-2 in running jobs, got %v", runningJobs)
	}

	// Clean up
	cancel1()
	cancel2()
}

func TestJobExecutor_GetQueueStatus(t *testing.T) {
	db := setupJobExecutorTestDB(t)
	defer db.Close()

	jobService := NewJobService(db)
	executor := NewJobExecutor(jobService, 3)

	// Add some jobs to queue
	executor.QueueJob("job-1")
	executor.QueueJob("job-2")

	// Add mock running job
	_, cancel := context.WithCancel(context.Background())
	executor.cancelMutex.Lock()
	executor.cancelMap["running-job"] = cancel
	executor.cancelMutex.Unlock()

	status := executor.GetQueueStatus()

	if status["worker_count"] != 3 {
		t.Errorf("Expected worker_count 3, got %v", status["worker_count"])
	}
	if status["running_jobs"] != 1 {
		t.Errorf("Expected running_jobs 1, got %v", status["running_jobs"])
	}
	if status["queued_jobs"] != 2 {
		t.Errorf("Expected queued_jobs 2, got %v", status["queued_jobs"])
	}

	// Claude availability check (this will depend on whether claude is installed)
	claudeAvailable, ok := status["claude_available"].(bool)
	if !ok {
		t.Error("Expected claude_available to be a boolean")
	}
	_ = claudeAvailable // Use the variable to avoid unused warning

	// Clean up
	cancel()
}

func TestJobExecutor_CancelJob(t *testing.T) {
	db := setupJobExecutorTestDB(t)
	defer db.Close()

	jobService := NewJobService(db)
	executor := NewJobExecutor(jobService, 1)

	// Create a test job in database
	createTestJob(t, db, "test-job", "echo test", models.JobStatusRunning)

	// Test cancelling non-running job
	err := executor.CancelJob("nonexistent-job")
	if err == nil {
		t.Error("Expected error for non-running job, got nil")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Errorf("Expected 'not running' error, got %v", err)
	}

	// Add mock running job
	_, cancel := context.WithCancel(context.Background())
	executor.cancelMutex.Lock()
	executor.cancelMap["test-job"] = cancel
	executor.cancelMutex.Unlock()

	// Test successful cancellation
	err = executor.CancelJob("test-job")
	if err != nil {
		t.Errorf("Expected no error for job cancellation, got %v", err)
	}

	// Verify job is no longer in running map
	executor.cancelMutex.RLock()
	_, exists := executor.cancelMap["test-job"]
	executor.cancelMutex.RUnlock()
	if exists {
		t.Error("Expected job to be removed from running map after cancellation")
	}
}

func TestJobExecutor_StartStop(t *testing.T) {
	db := setupJobExecutorTestDB(t)
	defer db.Close()

	jobService := NewJobService(db)
	executor := NewJobExecutor(jobService, 2)

	// Start the executor
	executor.Start()

	// Give it a moment to start workers
	time.Sleep(100 * time.Millisecond)

	// Check that context is not cancelled
	select {
	case <-executor.ctx.Done():
		t.Error("Executor context should not be cancelled after start")
	default:
		// Expected
	}

	// Stop the executor
	executor.Stop()

	// Check that context is cancelled
	select {
	case <-executor.ctx.Done():
		// Expected
	case <-time.After(1 * time.Second):
		t.Error("Executor context should be cancelled after stop")
	}
}

// Integration test for job execution flow
func TestJobExecutor_JobExecutionFlow(t *testing.T) {
	db := setupJobExecutorTestDB(t)
	defer db.Close()

	jobService := NewJobService(db)
	executor := NewJobExecutor(jobService, 1)

	// Create a test job that should be safe to execute
	createTestJob(t, db, "test-job", "echo test", models.JobStatusPending)

	// Start executor
	executor.Start()
	defer executor.Stop()

	// Queue the job
	err := executor.QueueJob("test-job")
	if err != nil {
		t.Fatalf("Failed to queue job: %v", err)
	}

	// Wait for job processing (this is a simple test, in real scenario job would be processed)
	time.Sleep(200 * time.Millisecond)

	// Note: Full execution test would require mocking the command execution
	// since we don't want to actually run claude commands in tests
}

// Benchmark tests
func BenchmarkJobExecutor_QueueJob(b *testing.B) {
	db := setupJobExecutorTestDB(&testing.T{})
	defer db.Close()

	jobService := NewJobService(db)
	executor := NewJobExecutor(jobService, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := executor.QueueJob(fmt.Sprintf("job-%d", i))
		if err != nil && !strings.Contains(err.Error(), "queue is full") {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkJobExecutor_ValidateCommand(b *testing.B) {
	db := setupJobExecutorTestDB(&testing.T{})
	defer db.Close()

	jobService := NewJobService(db)
	executor := NewJobExecutor(jobService, 1)

	commands := []string{
		"echo hello",
		"ls -la",
		"cat file.txt",
		"grep pattern file.txt",
		"find . -name '*.go'",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		command := commands[i%len(commands)]
		executor.validateCommand(command)
	}
}