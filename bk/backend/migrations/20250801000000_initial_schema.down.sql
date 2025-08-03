-- Drop all tables and indexes

DROP INDEX IF EXISTS idx_jobs_created_at;
DROP INDEX IF EXISTS idx_jobs_status;
DROP INDEX IF EXISTS idx_session_windows_times;
DROP INDEX IF EXISTS idx_messages_session_window_id;
DROP INDEX IF EXISTS idx_messages_timestamp;
DROP INDEX IF EXISTS idx_messages_session_id;
DROP INDEX IF EXISTS idx_sessions_window_times;
DROP INDEX IF EXISTS idx_sessions_start_time;
DROP INDEX IF EXISTS idx_sessions_project_name;

DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS file_sync_state;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS session_windows;
DROP TABLE IF EXISTS sessions;