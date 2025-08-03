package services

import (
	"encoding/json"
	"testing"
	"time"

	"ccdash-backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobScheduler_AfterResetJobs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create services
	jobService := NewJobService(db)
	jobExecutor := NewJobExecutor(jobService, 1)
	jobExecutor.Start()
	defer jobExecutor.Stop()

	windowService := &SessionWindowService{db: db}
	scheduler := NewJobScheduler(db, jobService, jobExecutor, windowService)

	// Create a project
	projectID := "test-project-1"
	_, err := db.Exec(`
		INSERT INTO projects (id, name, path, created_at, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		projectID, "Test Project", "/test/path")
	require.NoError(t, err)

	// Create an after_reset job
	req := &models.CreateJobRequest{
		ProjectID:    projectID,
		Command:      "echo 'after reset test'",
		ScheduleType: models.ScheduleTypeAfterReset,
		YoloMode:     false,
	}
	job, err := jobService.CreateJob(req)
	require.NoError(t, err)
	assert.Equal(t, models.ScheduleTypeAfterReset, *job.ScheduleType)
	assert.Equal(t, models.JobStatusPending, job.Status)

	// Create a session window
	windowID := "window-1"
	resetTime := time.Now().Add(5 * time.Hour)
	_, err = db.Exec(`
		INSERT INTO session_windows (
			id, window_start, window_end, reset_time, is_active,
			total_input_tokens, total_output_tokens, total_tokens,
			message_count, session_count, total_cost,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, 0, 0, 0, 0, 0, 0.0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		windowID,
		time.Now().Format(time.RFC3339),
		resetTime.Format(time.RFC3339),
		resetTime.Format(time.RFC3339),
		true)
	require.NoError(t, err)

	// Trigger scheduler check
	err = scheduler.checkAfterResetJobs()
	require.NoError(t, err)

	// Wait a bit for job to be queued
	time.Sleep(100 * time.Millisecond)

	// Verify job is still pending (should be in queue now)
	updatedJob, err := jobService.GetJob(job.ID)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusPending, updatedJob.Status)

	// Simulate window reset by updating reset time
	newResetTime := time.Now().Add(10 * time.Hour)
	_, err = db.Exec(`
		UPDATE session_windows 
		SET reset_time = ?, window_end = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		newResetTime.Format(time.RFC3339),
		newResetTime.Format(time.RFC3339),
		windowID)
	require.NoError(t, err)

	// Trigger scheduler check again
	err = scheduler.checkAfterResetJobs()
	require.NoError(t, err)

	// Wait for job execution
	time.Sleep(200 * time.Millisecond)
}

func TestJobScheduler_DelayedJobs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create services
	jobService := NewJobService(db)
	jobExecutor := NewJobExecutor(jobService, 1)
	jobExecutor.Start()
	defer jobExecutor.Stop()

	windowService := &SessionWindowService{db: db}
	scheduler := NewJobScheduler(db, jobService, jobExecutor, windowService)

	// Create a project
	projectID := "test-project-2"
	_, err := db.Exec(`
		INSERT INTO projects (id, name, path, created_at, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		projectID, "Test Project 2", "/test/path2")
	require.NoError(t, err)

	// Create a delayed job (1 hour delay)
	delayHours := 1
	req := &models.CreateJobRequest{
		ProjectID:    projectID,
		Command:      "echo 'delayed test'",
		ScheduleType: models.ScheduleTypeDelayed,
		ScheduleParams: &models.ScheduleParams{
			DelayHours: &delayHours,
		},
		YoloMode: false,
	}
	job, err := jobService.CreateJob(req)
	require.NoError(t, err)
	assert.Equal(t, models.ScheduleTypeDelayed, *job.ScheduleType)
	assert.NotNil(t, job.ScheduledAt)
	assert.True(t, job.ScheduledAt.After(time.Now()))

	// Verify job is not executed yet
	err = scheduler.checkScheduledJobs()
	require.NoError(t, err)

	updatedJob, err := jobService.GetJob(job.ID)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusPending, updatedJob.Status)

	// Update scheduled_at to be in the past
	pastTime := time.Now().Add(-1 * time.Minute)
	_, err = db.Exec(`
		UPDATE jobs SET scheduled_at = ? WHERE id = ?`,
		pastTime.Format(time.RFC3339), job.ID)
	require.NoError(t, err)

	// Trigger scheduler check
	err = scheduler.checkScheduledJobs()
	require.NoError(t, err)

	// Wait for job execution
	time.Sleep(200 * time.Millisecond)
}

func TestJobScheduler_ScheduledJobs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create services
	jobService := NewJobService(db)
	jobExecutor := NewJobExecutor(jobService, 1)
	jobExecutor.Start()
	defer jobExecutor.Stop()

	windowService := &SessionWindowService{db: db}
	scheduler := NewJobScheduler(db, jobService, jobExecutor, windowService)

	// Create a project
	projectID := "test-project-3"
	_, err := db.Exec(`
		INSERT INTO projects (id, name, path, created_at, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		projectID, "Test Project 3", "/test/path3")
	require.NoError(t, err)

	// Create a scheduled job for specific time
	scheduledTime := time.Now().Add(2 * time.Hour)
	req := &models.CreateJobRequest{
		ProjectID:    projectID,
		Command:      "echo 'scheduled test'",
		ScheduleType: models.ScheduleTypeScheduled,
		ScheduleParams: &models.ScheduleParams{
			ScheduledTime: &scheduledTime,
		},
		YoloMode: false,
	}
	job, err := jobService.CreateJob(req)
	require.NoError(t, err)
	assert.Equal(t, models.ScheduleTypeScheduled, *job.ScheduleType)
	assert.NotNil(t, job.ScheduledAt)
	assert.Equal(t, scheduledTime.Format(time.RFC3339), job.ScheduledAt.Format(time.RFC3339))

	// Verify schedule params were saved
	assert.NotNil(t, job.ScheduleParams)
	var params models.ScheduleParams
	err = json.Unmarshal([]byte(*job.ScheduleParams), &params)
	require.NoError(t, err)
	assert.NotNil(t, params.ScheduledTime)

	// Verify job is not executed yet
	err = scheduler.checkScheduledJobs()
	require.NoError(t, err)

	updatedJob, err := jobService.GetJob(job.ID)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusPending, updatedJob.Status)

	// Update scheduled_at to be in the past
	pastTime := time.Now().Add(-1 * time.Minute)
	_, err = db.Exec(`
		UPDATE jobs SET scheduled_at = ? WHERE id = ?`,
		pastTime.Format(time.RFC3339), job.ID)
	require.NoError(t, err)

	// Trigger scheduler check
	err = scheduler.checkScheduledJobs()
	require.NoError(t, err)

	// Wait for job execution
	time.Sleep(200 * time.Millisecond)
}

func TestJobScheduler_GetSchedulerStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create services
	jobService := NewJobService(db)
	jobExecutor := NewJobExecutor(jobService, 1)
	windowService := &SessionWindowService{db: db}
	scheduler := NewJobScheduler(db, jobService, jobExecutor, windowService)

	// Get status before starting
	status := scheduler.GetSchedulerStatus()
	assert.False(t, status["running"].(bool))
	assert.NotEmpty(t, status["last_check"])
	assert.Nil(t, status["last_reset_time"])

	// Start scheduler
	scheduler.Start()
	defer scheduler.Stop()

	// Get status after starting
	status = scheduler.GetSchedulerStatus()
	assert.True(t, status["running"].(bool))
	assert.NotEmpty(t, status["last_check"])
}