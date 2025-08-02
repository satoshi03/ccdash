-- Drop jobs table and its indexes
DROP INDEX IF EXISTS idx_jobs_created_at;
DROP INDEX IF EXISTS idx_jobs_scheduled_at;
DROP INDEX IF EXISTS idx_jobs_status;
DROP INDEX IF EXISTS idx_jobs_project_id;
DROP TABLE IF EXISTS jobs;