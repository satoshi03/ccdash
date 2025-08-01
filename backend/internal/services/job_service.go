package services

import (
	"database/sql"
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
	
	// immediate実行の場合はscheduled_atも設定
	if req.ScheduleType == models.ScheduleTypeImmediate {
		now := time.Now()
		job.ScheduledAt = &now
	}
	
	query := `
		INSERT INTO jobs (
			id, project_id, command, execution_directory, yolo_mode, 
			status, priority, created_at, scheduled_at, schedule_type
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	_, err = js.db.Exec(query,
		job.ID, job.ProjectID, job.Command, job.ExecutionDirectory,
		job.YoloMode, job.Status, job.Priority, job.CreatedAt.Format(time.RFC3339),
		formatTimePtr(job.ScheduledAt), job.ScheduleType)
	
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
			   j.scheduled_at, j.schedule_type,
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
			   j.scheduled_at, j.schedule_type,
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
func (js *JobService) UpdateJobStatus(id string, status string, pid *int) error {
	now := time.Now()
	
	if status == models.JobStatusRunning {
		query := "UPDATE jobs SET status = ?, pid = ?, started_at = ? WHERE id = ?"
		args := []interface{}{status, pid, now.Format(time.RFC3339), id}
		_, err := js.db.Exec(query, args...)
		if err != nil {
			return fmt.Errorf("failed to update job status: %w", err)
		}
	} else if status == models.JobStatusCompleted || status == models.JobStatusFailed || status == models.JobStatusCancelled {
		query := "UPDATE jobs SET status = ?, pid = NULL, completed_at = ? WHERE id = ?"
		args := []interface{}{status, now.Format(time.RFC3339), id}
		_, err := js.db.Exec(query, args...)
		if err != nil {
			return fmt.Errorf("failed to update job status: %w", err)
		}
	} else {
		query := "UPDATE jobs SET status = ?, pid = ? WHERE id = ?"
		args := []interface{}{status, pid, id}
		_, err := js.db.Exec(query, args...)
		if err != nil {
			return fmt.Errorf("failed to update job status: %w", err)
		}
	}
	
	return nil
}

// UpdateJobLogs updates job output and error logs
func (js *JobService) UpdateJobLogs(id string, outputLog, errorLog *string, exitCode *int) error {
	query := "UPDATE jobs SET output_log = ?, error_log = ?, exit_code = ? WHERE id = ?"
	_, err := js.db.Exec(query, outputLog, errorLog, exitCode, id)
	if err != nil {
		return fmt.Errorf("failed to update job logs: %w", err)
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
	var scheduleType sql.NullString
	
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
		&exitCode, &pid, &scheduledAt, &scheduleType,
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
	
	return nil
}

func formatTimePtr(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}