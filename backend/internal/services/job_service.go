package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
	
	"ccdash-backend/internal/models"
	"github.com/google/uuid"
)

type JobService struct {
	db *sql.DB
}

func NewJobService(db *sql.DB) *JobService {
	return &JobService{db: db}
}

// CreateJob creates a new job
func (js *JobService) CreateJob(req *models.CreateJobRequest) (*models.Job, error) {
	// プロジェクトの存在確認
	project, err := js.getProjectByID(req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	if project == nil {
		return nil, fmt.Errorf("project not found: %s", req.ProjectID)
	}
	
	// スケジュールパラメータの検証
	if err := js.validateScheduleParams(req.ScheduleType, req.ScheduleParams); err != nil {
		return nil, fmt.Errorf("invalid schedule parameters: %w", err)
	}
	
	job := &models.Job{
		ID:                 uuid.New().String(),
		ProjectID:          req.ProjectID,
		Command:            req.Command,
		ExecutionDirectory: project.Path,
		YoloMode:          req.YoloMode,
		Status:            models.JobStatusPending,
		Priority:          0,
		CreatedAt:         time.Now(),
		ScheduleType:      &req.ScheduleType,
	}
	
	// スケジュールタイプに応じてscheduled_atを設定
	switch req.ScheduleType {
	case models.ScheduleTypeImmediate:
		now := time.Now()
		job.ScheduledAt = &now
	case models.ScheduleTypeDelayed:
		if req.ScheduleParams != nil && req.ScheduleParams.DelayHours != nil {
			scheduledTime := time.Now().Add(time.Duration(*req.ScheduleParams.DelayHours) * time.Hour)
			job.ScheduledAt = &scheduledTime
		}
	case models.ScheduleTypeScheduled:
		if req.ScheduleParams != nil && req.ScheduleParams.ScheduledTime != nil {
			job.ScheduledAt = req.ScheduleParams.ScheduledTime
		}
	}
	
	// ScheduleParamsをJSON文字列に変換
	var scheduleParamsJSON *string
	if req.ScheduleParams != nil {
		paramsBytes, err := json.Marshal(req.ScheduleParams)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal schedule params: %w", err)
		}
		paramsStr := string(paramsBytes)
		scheduleParamsJSON = &paramsStr
		job.ScheduleParams = scheduleParamsJSON
	}
	
	query := `
		INSERT INTO jobs (
			id, project_id, command, execution_directory, yolo_mode, 
			status, priority, created_at, scheduled_at, schedule_type, schedule_params
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	_, err = js.db.Exec(query,
		job.ID, job.ProjectID, job.Command, job.ExecutionDirectory,
		job.YoloMode, job.Status, job.Priority, job.CreatedAt.Format(time.RFC3339),
		formatTimePtr(job.ScheduledAt), job.ScheduleType, scheduleParamsJSON)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}
	
	return job, nil
}

// GetJobs retrieves jobs with filters
func (js *JobService) GetJobs(filters models.JobFilters) ([]*models.Job, error) {
	query := `
		SELECT j.id, j.project_id, j.command, j.execution_directory, j.yolo_mode,
			   j.status, j.priority, j.created_at, j.started_at, j.completed_at,
			   j.output_log, j.error_log, j.exit_code, j.pid,
			   j.scheduled_at, j.schedule_type, j.schedule_params,
			   p.name as project_name, p.path as project_path
		FROM jobs j
		LEFT JOIN projects p ON j.project_id = p.id
		WHERE 1=1`
	
	args := []interface{}{}
	
	if filters.ProjectID != nil {
		query += " AND j.project_id = ?"
		args = append(args, *filters.ProjectID)
	}
	
	if filters.Status != nil {
		query += " AND j.status = ?"
		args = append(args, *filters.Status)
	}
	
	query += " ORDER BY j.priority DESC, j.created_at DESC"
	
	if filters.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filters.Limit)
	}
	
	if filters.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filters.Offset)
	}
	
	rows, err := js.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query jobs: %w", err)
	}
	defer rows.Close()
	
	var jobs []*models.Job
	for rows.Next() {
		job := &models.Job{Project: &models.Project{}}
		err := js.scanJobRow(rows, job)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job row: %w", err)
		}
		jobs = append(jobs, job)
	}
	
	return jobs, nil
}

// GetJobByID retrieves a job by ID
func (js *JobService) GetJobByID(id string) (*models.Job, error) {
	query := `
		SELECT j.id, j.project_id, j.command, j.execution_directory, j.yolo_mode,
			   j.status, j.priority, j.created_at, j.started_at, j.completed_at,
			   j.output_log, j.error_log, j.exit_code, j.pid,
			   j.scheduled_at, j.schedule_type, j.schedule_params,
			   p.name as project_name, p.path as project_path
		FROM jobs j
		LEFT JOIN projects p ON j.project_id = p.id
		WHERE j.id = ?`
	
	row := js.db.QueryRow(query, id)
	job := &models.Job{Project: &models.Project{}}
	
	err := js.scanJobRow(row, job)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get job by ID: %w", err)
	}
	
	return job, nil
}

// UpdateJobStatus updates job status and related fields
// Note: Using DELETE+INSERT workaround due to DuckDB UPDATE constraint bug
func (js *JobService) UpdateJobStatus(id string, status string, pid *int) error {
	// First, get the current job data
	job, err := js.GetJobByID(id)
	if err != nil {
		return fmt.Errorf("failed to get job for update: %w", err)
	}
	if job == nil {
		return fmt.Errorf("job not found: %s", id)
	}

	now := time.Now()

	// Delete the existing job record
	_, err = js.db.Exec("DELETE FROM jobs WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete job for update: %w", err)
	}

	// Prepare the new values
	var pidValue interface{}
	if pid != nil {
		pidValue = *pid
	}

	var startedAt interface{}
	var completedAt interface{}

	// Handle timestamps - convert *time.Time to string or nil
	if job.StartedAt != nil {
		startedAt = job.StartedAt.Format(time.RFC3339)
	} else {
		startedAt = nil
	}

	if job.CompletedAt != nil {
		completedAt = job.CompletedAt.Format(time.RFC3339)
	} else {
		completedAt = nil
	}

	// Update timestamps based on status
	if status == models.JobStatusRunning && job.StartedAt == nil {
		startedAt = now.Format(time.RFC3339)
	} else if status == models.JobStatusCompleted || status == models.JobStatusFailed || status == models.JobStatusCancelled {
		completedAt = now.Format(time.RFC3339)
		pidValue = nil // Clear PID when job completes
	}

	// Insert the updated job record
	query := `INSERT INTO jobs (
		id, project_id, command, execution_directory, yolo_mode, 
		status, priority, created_at, started_at, completed_at, 
		output_log, error_log, exit_code, pid, scheduled_at, schedule_type, schedule_params
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var scheduledAt interface{}
	if job.ScheduledAt != nil {
		scheduledAt = job.ScheduledAt.Format(time.RFC3339)
	} else {
		scheduledAt = nil
	}

	_, err = js.db.Exec(query,
		job.ID, job.ProjectID, job.Command, job.ExecutionDirectory, job.YoloMode,
		status, job.Priority, job.CreatedAt.Format(time.RFC3339), startedAt, completedAt,
		job.OutputLog, job.ErrorLog, job.ExitCode, pidValue, scheduledAt, job.ScheduleType, job.ScheduleParams,
	)
	if err != nil {
		return fmt.Errorf("failed to insert updated job: %w", err)
	}
	
	return nil
}

// UpdateJobLogs updates job output and error logs
// Note: Using DELETE+INSERT workaround due to DuckDB UPDATE constraint bug
func (js *JobService) UpdateJobLogs(id string, outputLog, errorLog *string, exitCode *int) error {
	// First, get the current job data
	job, err := js.GetJobByID(id)
	if err != nil {
		return fmt.Errorf("failed to get job for log update: %w", err)
	}
	if job == nil {
		return fmt.Errorf("job not found: %s", id)
	}

	// Delete the existing job record
	_, err = js.db.Exec("DELETE FROM jobs WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete job for log update: %w", err)
	}

	// Insert the updated job record with new logs
	query := `INSERT INTO jobs (
		id, project_id, command, execution_directory, yolo_mode, 
		status, priority, created_at, started_at, completed_at, 
		output_log, error_log, exit_code, pid, scheduled_at, schedule_type, schedule_params
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var startedAtStr interface{}
	if job.StartedAt != nil {
		startedAtStr = job.StartedAt.Format(time.RFC3339)
	} else {
		startedAtStr = nil
	}

	var completedAtStr interface{}
	if job.CompletedAt != nil {
		completedAtStr = job.CompletedAt.Format(time.RFC3339)
	} else {
		completedAtStr = nil
	}

	var scheduledAtStr interface{}
	if job.ScheduledAt != nil {
		scheduledAtStr = job.ScheduledAt.Format(time.RFC3339)
	} else {
		scheduledAtStr = nil
	}

	_, err = js.db.Exec(query,
		job.ID, job.ProjectID, job.Command, job.ExecutionDirectory, job.YoloMode,
		job.Status, job.Priority, job.CreatedAt.Format(time.RFC3339), startedAtStr, completedAtStr,
		outputLog, errorLog, exitCode, job.PID, scheduledAtStr, job.ScheduleType, job.ScheduleParams,
	)
	if err != nil {
		return fmt.Errorf("failed to insert job with updated logs: %w", err)
	}
	
	return nil
}

// DeleteJob deletes a job (only if not running)
func (js *JobService) DeleteJob(id string) error {
	// 実行中のジョブは削除不可
	job, err := js.GetJobByID(id)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if job == nil {
		return fmt.Errorf("job not found")
	}
	if job.Status == models.JobStatusRunning {
		return fmt.Errorf("cannot delete running job")
	}
	
	_, err = js.db.Exec("DELETE FROM jobs WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}
	
	return nil
}

// GetPendingJobs retrieves jobs that are ready to be executed
func (js *JobService) GetPendingJobs(limit int) ([]*models.Job, error) {
	status := models.JobStatusPending
	filters := models.JobFilters{
		Status: &status,
		Limit:  limit,
	}
	return js.GetJobs(filters)
}

// Helper methods

func (js *JobService) getProjectByID(id string) (*models.Project, error) {
	query := "SELECT id, name, path FROM projects WHERE id = ?"
	row := js.db.QueryRow(query, id)
	
	project := &models.Project{}
	err := row.Scan(&project.ID, &project.Name, &project.Path)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	return project, nil
}

func (js *JobService) scanJobRow(row interface{}, job *models.Job) error {
	var createdAt, startedAt, completedAt, scheduledAt, outputLog, errorLog sql.NullString
	var exitCode, pid sql.NullInt64
	var scheduleType, scheduleParams sql.NullString
	
	scanner, ok := row.(interface {
		Scan(dest ...interface{}) error
	})
	if !ok {
		return fmt.Errorf("invalid row type")
	}
	
	err := scanner.Scan(
		&job.ID, &job.ProjectID, &job.Command, &job.ExecutionDirectory,
		&job.YoloMode, &job.Status, &job.Priority, &createdAt,
		&startedAt, &completedAt, &outputLog, &errorLog,
		&exitCode, &pid, &scheduledAt, &scheduleType, &scheduleParams,
		&job.Project.Name, &job.Project.Path)
	
	if err != nil {
		return err
	}
	
	// NULL値の処理
	if createdAt.Valid {
		t, err := time.Parse(time.RFC3339, createdAt.String)
		if err != nil {
			return fmt.Errorf("failed to parse created_at: %w", err)
		}
		job.CreatedAt = t
	}
	if startedAt.Valid {
		t, _ := time.Parse(time.RFC3339, startedAt.String)
		job.StartedAt = &t
	}
	if completedAt.Valid {
		t, _ := time.Parse(time.RFC3339, completedAt.String)
		job.CompletedAt = &t
	}
	if scheduledAt.Valid {
		t, _ := time.Parse(time.RFC3339, scheduledAt.String)
		job.ScheduledAt = &t
	}
	if outputLog.Valid {
		job.OutputLog = &outputLog.String
	}
	if errorLog.Valid {
		job.ErrorLog = &errorLog.String
	}
	if exitCode.Valid {
		code := int(exitCode.Int64)
		job.ExitCode = &code
	}
	if pid.Valid {
		p := int(pid.Int64)
		job.PID = &p
	}
	if scheduleType.Valid {
		job.ScheduleType = &scheduleType.String
	}
	if scheduleParams.Valid {
		job.ScheduleParams = &scheduleParams.String
	}
	
	return nil
}

func formatTimePtr(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}

// validateScheduleParams validates schedule parameters based on schedule type
func (js *JobService) validateScheduleParams(scheduleType string, params *models.ScheduleParams) error {
	switch scheduleType {
	case models.ScheduleTypeImmediate:
		// immediateタイプはパラメータ不要
		return nil
	case models.ScheduleTypeAfterReset:
		// after_resetタイプはパラメータ不要
		return nil
	case models.ScheduleTypeDelayed:
		if params == nil || params.DelayHours == nil {
			return fmt.Errorf("delay_hours is required for delayed schedule type")
		}
		if *params.DelayHours < 1 || *params.DelayHours > 168 { // 最大1週間
			return fmt.Errorf("delay_hours must be between 1 and 168")
		}
		return nil
	case models.ScheduleTypeScheduled:
		if params == nil || params.ScheduledTime == nil {
			return fmt.Errorf("scheduled_time is required for scheduled type")
		}
		if params.ScheduledTime.Before(time.Now()) {
			return fmt.Errorf("scheduled_time must be in the future")
		}
		return nil
	default:
		return fmt.Errorf("invalid schedule type: %s", scheduleType)
	}
}

// GetScheduledJobs retrieves jobs that are scheduled for execution
func (js *JobService) GetScheduledJobs() ([]*models.Job, error) {
	now := time.Now()
	query := `
		SELECT j.id, j.project_id, j.command, j.execution_directory, j.yolo_mode,
			   j.status, j.priority, j.created_at, j.started_at, j.completed_at,
			   j.output_log, j.error_log, j.exit_code, j.pid,
			   j.scheduled_at, j.schedule_type, j.schedule_params,
			   p.name as project_name, p.path as project_path
		FROM jobs j
		LEFT JOIN projects p ON j.project_id = p.id
		WHERE j.status = ? 
		  AND j.scheduled_at IS NOT NULL
		  AND j.scheduled_at <= ?
		ORDER BY j.priority DESC, j.scheduled_at ASC`
	
	rows, err := js.db.Query(query, models.JobStatusPending, now.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("failed to query scheduled jobs: %w", err)
	}
	defer rows.Close()
	
	var jobs []*models.Job
	for rows.Next() {
		job := &models.Job{Project: &models.Project{}}
		err := js.scanJobRow(rows, job)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scheduled job row: %w", err)
		}
		jobs = append(jobs, job)
	}
	
	return jobs, nil
}