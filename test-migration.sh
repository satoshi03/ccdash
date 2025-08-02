#!/bin/bash

# Migration test script
set -e

echo "=== Migration Test Starting ==="

# Test database path
TEST_DB="test_migration.db"

# Clean up existing test database
if [ -f "$TEST_DB" ]; then
    echo "Removing existing test database..."
    rm "$TEST_DB"
fi

# Create test database and required tables
echo "Creating test database with initial schema..."
cat << 'EOF' | duckdb "$TEST_DB"
-- Create projects table (required for foreign key)
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- Insert test project
INSERT INTO projects (id, name, created_at, updated_at) 
VALUES ('test-project-id', 'Test Project', '2025-08-02T10:00:00Z', '2025-08-02T10:00:00Z');
EOF

echo "Initial schema created."

# Test UP migration
echo ""
echo "=== Testing UP Migration ==="
duckdb "$TEST_DB" < backend/migrations/20250802000001_add_jobs_table.up.sql

# Verify tables exist
echo "Verifying jobs table exists..."
TABLES=$(duckdb "$TEST_DB" -csv -c "SELECT table_name FROM information_schema.tables WHERE table_schema = 'main' ORDER BY table_name;")
echo "Tables in database:"
echo "$TABLES"

# Verify jobs table structure
echo ""
echo "Verifying jobs table structure..."
COLUMNS=$(duckdb "$TEST_DB" -csv -c "SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'jobs' ORDER BY ordinal_position;")
echo "Jobs table columns:"
echo "$COLUMNS"

# Verify indexes
echo ""
echo "Verifying indexes..."
INDEXES=$(duckdb "$TEST_DB" -csv -c "SELECT table_name, index_name FROM duckdb_indexes() WHERE table_name = 'jobs';")
echo "Jobs table indexes:"
echo "$INDEXES"

# Test inserting data
echo ""
echo "Testing data insertion..."
duckdb "$TEST_DB" << 'EOF'
INSERT INTO jobs (
    id, project_id, command, execution_directory, yolo_mode, 
    status, priority, created_at
) VALUES (
    'job-123', 'test-project-id', 'echo "test"', '/tmp', 
    false, 'pending', 0, '2025-08-02T10:00:00Z'
);
EOF

# Verify data
JOB_COUNT=$(duckdb "$TEST_DB" -csv -c "SELECT COUNT(*) FROM jobs;")
echo "Jobs count after insert: $JOB_COUNT"

# Test DOWN migration
echo ""
echo "=== Testing DOWN Migration ==="
duckdb "$TEST_DB" < backend/migrations/20250802000001_add_jobs_table.down.sql

# Verify table is dropped
echo "Verifying jobs table is dropped..."
TABLES_AFTER=$(duckdb "$TEST_DB" -csv -c "SELECT table_name FROM information_schema.tables WHERE table_schema = 'main' ORDER BY table_name;")
echo "Tables in database after DOWN migration:"
echo "$TABLES_AFTER"

# Test idempotency - run UP migration twice
echo ""
echo "=== Testing Idempotency ==="
echo "Running UP migration twice..."
duckdb "$TEST_DB" < backend/migrations/20250802000001_add_jobs_table.up.sql
duckdb "$TEST_DB" < backend/migrations/20250802000001_add_jobs_table.up.sql
echo "UP migration ran twice without errors"

# Run DOWN migration twice
echo "Running DOWN migration twice..."
duckdb "$TEST_DB" < backend/migrations/20250802000001_add_jobs_table.down.sql
duckdb "$TEST_DB" < backend/migrations/20250802000001_add_jobs_table.down.sql
echo "DOWN migration ran twice without errors"

# Clean up
echo ""
echo "Cleaning up test database..."
rm "$TEST_DB"

echo ""
echo "=== Migration Test Completed Successfully ==="