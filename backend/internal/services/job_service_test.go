package services

import (
	"database/sql"
	"testing"
	"time"

	"ccdash-backend/internal/models"
	"github.com/google/uuid"
	_ "github.com/marcboeker/go-duckdb"
)

func setupJobTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create projects table first (foreign key dependency)
	createProjectsTableQuery := `
		CREATE TABLE projects (
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
		)`

	if _, err := db.Exec(createProjectsTableQuery); err != nil {
		t.Fatalf("Failed to create projects table: %v", err)
	}

	// Create jobs table
	createJobsTableQuery := `
		CREATE TABLE jobs (
			id VARCHAR PRIMARY KEY,
			project_id VARCHAR NOT NULL,
			command TEXT NOT NULL,
			execution_directory TEXT NOT NULL,
			yolo_mode BOOLEAN DEFAULT FALSE,
			status VARCHAR NOT NULL DEFAULT 'pending',
			priority INTEGER DEFAULT 0,
			created_at VARCHAR NOT NULL,
			started_at VARCHAR,
			completed_at VARCHAR,
			output_log TEXT,
			error_log TEXT,
			exit_code INTEGER,
			pid INTEGER,
			scheduled_at VARCHAR,
			schedule_type VARCHAR,
			FOREIGN KEY (project_id) REFERENCES projects(id)
		)`

	if _, err := db.Exec(createJobsTableQuery); err != nil {
		t.Fatalf("Failed to create jobs table: %v", err)
	}

	return db
}

func createTestProject(t *testing.T, db *sql.DB) *models.Project {
	project := &models.Project{
		ID:        uuid.New().String(),
		Name:      "Test Project",
		Path:      "/test/path",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	query := `
		INSERT INTO projects (id, name, path, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`

	_, err := db.Exec(query, project.ID, project.Name, project.Path,
		project.IsActive, project.CreatedAt.Format(time.RFC3339),
		project.UpdatedAt.Format(time.RFC3339))

	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	return project
}

func TestJobService_CreateJob(t *testing.T) {
	db := setupJobTestDB(t)
	defer db.Close()

	// Create test project
	project := createTestProject(t, db)

	jobService := NewJobService(db)

	req := &models.CreateJobRequest{
		ProjectID:    project.ID,
		Command:      "implement new feature",
		YoloMode:     true,
		ScheduleType: models.ScheduleTypeImmediate,
	}

	job, err := jobService.CreateJob(req)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	// 検証
	if job.ID == "" {
		t.Error("Job ID should not be empty")
	}
	if job.Status != models.JobStatusPending {
		t.Errorf("Expected status %s, got %s", models.JobStatusPending, job.Status)
	}
	if job.Command != req.Command {
		t.Errorf("Expected command %s, got %s", req.Command, job.Command)
	}
	if job.YoloMode != req.YoloMode {
		t.Errorf("Expected yolo_mode %t, got %t", req.YoloMode, job.YoloMode)
	}
	if job.ProjectID != req.ProjectID {
		t.Errorf("Expected project_id %s, got %s", req.ProjectID, job.ProjectID)
	}
	if job.ExecutionDirectory != project.Path {
		t.Errorf("Expected execution_directory %s, got %s", project.Path, job.ExecutionDirectory)
	}
	if job.ScheduledAt == nil {
		t.Error("ScheduledAt should be set for immediate jobs")
	}
	if job.ScheduleType == nil || *job.ScheduleType != req.ScheduleType {
		t.Errorf("Expected schedule_type %s, got %v", req.ScheduleType, job.ScheduleType)
	}
}

func TestJobService_CreateJob_ProjectNotFound(t *testing.T) {
	db := setupJobTestDB(t)
	defer db.Close()

	jobService := NewJobService(db)

	req := &models.CreateJobRequest{
		ProjectID:    "non-existent-project",
		Command:      "test command",
		YoloMode:     false,
		ScheduleType: models.ScheduleTypeImmediate,
	}

	_, err := jobService.CreateJob(req)
	if err == nil {
		t.Error("Expected error for non-existent project")
	}
}

func TestJobService_GetJobByID(t *testing.T) {
	db := setupJobTestDB(t)
	defer db.Close()

	// Create test project and job
	project := createTestProject(t, db)
	jobService := NewJobService(db)

	req := &models.CreateJobRequest{
		ProjectID:    project.ID,
		Command:      "test command",
		YoloMode:     false,
		ScheduleType: models.ScheduleTypeImmediate,
	}

	createdJob, err := jobService.CreateJob(req)
	if err != nil {
		t.Fatalf("Failed to create job for test: %v", err)
	}

	// Test GetJobByID
	retrievedJob, err := jobService.GetJobByID(createdJob.ID)
	if err != nil {
		t.Fatalf("GetJobByID failed: %v", err)
	}

	if retrievedJob == nil {
		t.Fatal("Retrieved job should not be nil")
	}

	if retrievedJob.ID != createdJob.ID {
		t.Errorf("Expected job ID %s, got %s", createdJob.ID, retrievedJob.ID)
	}
	if retrievedJob.Command != createdJob.Command {
		t.Errorf("Expected command %s, got %s", createdJob.Command, retrievedJob.Command)
	}
	if retrievedJob.Project == nil {
		t.Error("Project should be populated in retrieved job")
	} else {
		if retrievedJob.Project.Name != project.Name {
			t.Errorf("Expected project name %s, got %s", project.Name, retrievedJob.Project.Name)
		}
	}
}

func TestJobService_GetJobs(t *testing.T) {
	db := setupJobTestDB(t)
	defer db.Close()

	// Create test project and multiple jobs
	project := createTestProject(t, db)
	jobService := NewJobService(db)

	// Create 3 test jobs
	for i := 0; i < 3; i++ {
		req := &models.CreateJobRequest{
			ProjectID:    project.ID,
			Command:      "test command " + string(rune(i+'1')),
			YoloMode:     i%2 == 0,
			ScheduleType: models.ScheduleTypeImmediate,
		}
		_, err := jobService.CreateJob(req)
		if err != nil {
			t.Fatalf("Failed to create test job %d: %v", i, err)
		}
	}

	// Test GetJobs with no filters
	filters := models.JobFilters{Limit: 10}
	jobs, err := jobService.GetJobs(filters)
	if err != nil {
		t.Fatalf("GetJobs failed: %v", err)
	}

	if len(jobs) != 3 {
		t.Errorf("Expected 3 jobs, got %d", len(jobs))
	}

	// Test GetJobs with project filter
	filters = models.JobFilters{
		ProjectID: &project.ID,
		Limit:     10,
	}
	jobs, err = jobService.GetJobs(filters)
	if err != nil {
		t.Fatalf("GetJobs with project filter failed: %v", err)
	}

	if len(jobs) != 3 {
		t.Errorf("Expected 3 jobs with project filter, got %d", len(jobs))
	}

	// Test GetJobs with status filter
	pendingStatus := models.JobStatusPending
	filters = models.JobFilters{
		Status: &pendingStatus,
		Limit:  10,
	}
	jobs, err = jobService.GetJobs(filters)
	if err != nil {
		t.Fatalf("GetJobs with status filter failed: %v", err)
	}

	if len(jobs) != 3 {
		t.Errorf("Expected 3 pending jobs, got %d", len(jobs))
	}
}

func TestJobService_UpdateJobStatus(t *testing.T) {
	db := setupJobTestDB(t)
	defer db.Close()

	// Create test project and job
	project := createTestProject(t, db)
	jobService := NewJobService(db)

	req := &models.CreateJobRequest{
		ProjectID:    project.ID,
		Command:      "test command",
		YoloMode:     false,
		ScheduleType: models.ScheduleTypeImmediate,
	}

	job, err := jobService.CreateJob(req)
	if err != nil {
		t.Fatalf("Failed to create job for test: %v", err)
	}

	// Test updating to running status
	pid := 12345
	err = jobService.UpdateJobStatus(job.ID, models.JobStatusRunning, &pid)
	if err != nil {
		t.Fatalf("UpdateJobStatus to running failed: %v", err)
	}

	// Verify the update
	updatedJob, err := jobService.GetJobByID(job.ID)
	if err != nil {
		t.Fatalf("Failed to get updated job: %v", err)
	}

	if updatedJob.Status != models.JobStatusRunning {
		t.Errorf("Expected status %s, got %s", models.JobStatusRunning, updatedJob.Status)
	}
	if updatedJob.PID == nil || *updatedJob.PID != pid {
		t.Errorf("Expected PID %d, got %v", pid, updatedJob.PID)
	}
	if updatedJob.StartedAt == nil {
		t.Error("StartedAt should be set when status is running")
	}

	// Test updating to completed status
	err = jobService.UpdateJobStatus(job.ID, models.JobStatusCompleted, nil)
	if err != nil {
		t.Fatalf("UpdateJobStatus to completed failed: %v", err)
	}

	// Verify the completion update
	completedJob, err := jobService.GetJobByID(job.ID)
	if err != nil {
		t.Fatalf("Failed to get completed job: %v", err)
	}

	if completedJob.Status != models.JobStatusCompleted {
		t.Errorf("Expected status %s, got %s", models.JobStatusCompleted, completedJob.Status)
	}
	if completedJob.CompletedAt == nil {
		t.Error("CompletedAt should be set when job is completed")
	}
	if completedJob.PID != nil {
		t.Error("PID should be cleared when job is completed")
	}
}

func TestJobService_DeleteJob(t *testing.T) {
	db := setupJobTestDB(t)
	defer db.Close()

	// Create test project and job
	project := createTestProject(t, db)
	jobService := NewJobService(db)

	req := &models.CreateJobRequest{
		ProjectID:    project.ID,
		Command:      "test command",
		YoloMode:     false,
		ScheduleType: models.ScheduleTypeImmediate,
	}

	job, err := jobService.CreateJob(req)
	if err != nil {
		t.Fatalf("Failed to create job for test: %v", err)
	}

	// Test deleting pending job (should work)
	err = jobService.DeleteJob(job.ID)
	if err != nil {
		t.Fatalf("DeleteJob failed: %v", err)
	}

	// Verify job is deleted
	deletedJob, err := jobService.GetJobByID(job.ID)
	if err != nil {
		t.Fatalf("Error checking deleted job: %v", err)
	}
	if deletedJob != nil {
		t.Error("Job should be deleted")
	}
}

func TestJobService_DeleteJob_RunningJob(t *testing.T) {
	db := setupJobTestDB(t)
	defer db.Close()

	// Create test project and job
	project := createTestProject(t, db)
	jobService := NewJobService(db)

	req := &models.CreateJobRequest{
		ProjectID:    project.ID,
		Command:      "test command",
		YoloMode:     false,
		ScheduleType: models.ScheduleTypeImmediate,
	}

	job, err := jobService.CreateJob(req)
	if err != nil {
		t.Fatalf("Failed to create job for test: %v", err)
	}

	// Update job to running status
	pid := 12345
	err = jobService.UpdateJobStatus(job.ID, models.JobStatusRunning, &pid)
	if err != nil {
		t.Fatalf("Failed to update job to running: %v", err)
	}

	// Test deleting running job (should fail)
	err = jobService.DeleteJob(job.ID)
	if err == nil {
		t.Error("Expected error when deleting running job")
	}
}