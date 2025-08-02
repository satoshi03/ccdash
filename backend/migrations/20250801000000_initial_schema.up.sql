-- Initial schema migration
-- This represents the current state of the database

-- Create tables in correct order to handle foreign key dependencies

CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR PRIMARY KEY,
    project_name VARCHAR NOT NULL,
    project_path VARCHAR NOT NULL,
    start_time TIMESTAMP NOT NULL,
    last_activity TIMESTAMP NOT NULL,
    total_duration_ms BIGINT DEFAULT 0,
    input_tokens BIGINT DEFAULT 0,
    output_tokens BIGINT DEFAULT 0,
    total_tokens BIGINT DEFAULT 0,
    cache_read_tokens BIGINT DEFAULT 0,
    cache_write_tokens BIGINT DEFAULT 0,
    tool_invocations INTEGER DEFAULT 0,
    messages_count INTEGER DEFAULT 0,
    window_start TIMESTAMP,
    window_end TIMESTAMP,
    cost_cents DOUBLE DEFAULT 0,
    status VARCHAR DEFAULT 'active',
    cwd VARCHAR
);

CREATE TABLE IF NOT EXISTS session_windows (
    id VARCHAR PRIMARY KEY,
    window_start TIMESTAMP NOT NULL,
    window_end TIMESTAMP NOT NULL,
    reset_time TIMESTAMP NOT NULL,
    total_input_tokens BIGINT DEFAULT 0,
    total_output_tokens BIGINT DEFAULT 0,
    total_cache_read_tokens BIGINT DEFAULT 0,
    total_cache_write_tokens BIGINT DEFAULT 0,
    total_cost_cents DOUBLE DEFAULT 0,
    sessions_count INTEGER DEFAULT 0,
    messages_count INTEGER DEFAULT 0,
    UNIQUE(window_start, window_end)
);

CREATE TABLE IF NOT EXISTS messages (
    id VARCHAR PRIMARY KEY,
    session_id VARCHAR NOT NULL,
    role VARCHAR NOT NULL,
    type VARCHAR,
    cache_read_input_tokens BIGINT DEFAULT 0,
    cache_write_input_tokens BIGINT DEFAULT 0,
    input_tokens BIGINT DEFAULT 0,
    output_tokens BIGINT DEFAULT 0,
    model VARCHAR,
    timestamp TIMESTAMP NOT NULL,
    content TEXT,
    tool_name VARCHAR,
    tool_input TEXT,
    is_error BOOLEAN DEFAULT false,
    error TEXT,
    session_window_id VARCHAR,
    FOREIGN KEY (session_id) REFERENCES sessions(id),
    FOREIGN KEY (session_window_id) REFERENCES session_windows(id)
);

CREATE TABLE IF NOT EXISTS file_sync_state (
    file_path VARCHAR PRIMARY KEY,
    last_sync_position BIGINT NOT NULL,
    last_sync_time TIMESTAMP NOT NULL,
    file_size BIGINT NOT NULL,
    file_modified_time TIMESTAMP NOT NULL,
    checksum VARCHAR
);

CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    path TEXT,
    first_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    total_sessions INTEGER DEFAULT 0,
    total_tokens BIGINT DEFAULT 0,
    total_cost_cents DOUBLE DEFAULT 0,
    metadata JSON
);

CREATE TABLE IF NOT EXISTS jobs (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    status TEXT NOT NULL,
    command TEXT NOT NULL,
    working_directory TEXT,
    environment JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    timeout_seconds INTEGER,
    result JSON,
    error TEXT,
    metadata JSON
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_sessions_project_name ON sessions(project_name);
CREATE INDEX IF NOT EXISTS idx_sessions_start_time ON sessions(start_time);
CREATE INDEX IF NOT EXISTS idx_sessions_window_times ON sessions(window_start, window_end);
CREATE INDEX IF NOT EXISTS idx_messages_session_id ON messages(session_id);
CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);
CREATE INDEX IF NOT EXISTS idx_messages_session_window_id ON messages(session_window_id);
CREATE INDEX IF NOT EXISTS idx_session_windows_times ON session_windows(window_start, window_end);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at);