-- Add jobs table for task execution
CREATE TABLE IF NOT EXISTS jobs (
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
    schedule_params TEXT,
    FOREIGN KEY (project_id) REFERENCES projects(id)
);

-- Create index for faster queries
CREATE INDEX IF NOT EXISTS idx_jobs_project_id ON jobs(project_id);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_scheduled_at ON jobs(scheduled_at);
CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at);