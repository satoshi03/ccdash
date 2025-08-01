-- Rollback initial schema

-- Drop indexes
DROP INDEX IF EXISTS idx_jobs_priority_created;
DROP INDEX IF EXISTS idx_jobs_created_at;
DROP INDEX IF EXISTS idx_jobs_status;
DROP INDEX IF EXISTS idx_jobs_project_id;

DROP INDEX IF EXISTS idx_projects_path;
DROP INDEX IF EXISTS idx_projects_active;
DROP INDEX IF EXISTS idx_projects_name;

DROP INDEX IF EXISTS idx_file_sync_state_modified;
DROP INDEX IF EXISTS idx_file_sync_state_status;
DROP INDEX IF EXISTS idx_file_sync_state_path;

DROP INDEX IF EXISTS idx_session_window_messages_message_id;
DROP INDEX IF EXISTS idx_session_window_messages_window_id;

DROP INDEX IF EXISTS idx_session_windows_reset_time;
DROP INDEX IF EXISTS idx_session_windows_active;
DROP INDEX IF EXISTS idx_session_windows_times;

DROP INDEX IF EXISTS idx_messages_message_role;
DROP INDEX IF EXISTS idx_messages_timestamp;
DROP INDEX IF EXISTS idx_messages_session_id;

DROP INDEX IF EXISTS idx_sessions_status;
DROP INDEX IF EXISTS idx_sessions_start_time;
DROP INDEX IF EXISTS idx_sessions_project_id;
DROP INDEX IF EXISTS idx_sessions_project_name;

-- Drop tables in reverse order of dependencies
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS session_window_messages;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS session_windows;
DROP TABLE IF EXISTS file_sync_state;
DROP TABLE IF EXISTS projects;