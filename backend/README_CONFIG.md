# Configuration Guide

This document describes the configuration options available for the Claudeee backend server.

## Environment Variables

The following environment variables can be used to configure the application:

### Database Configuration

- **`CLAUDEEE_DB_PATH`** (optional)
  - Full path to the DuckDB database file
  - Default: `${HOME}/.claudeee/claudeee.db`
  - Example: `/custom/path/to/claudeee.db`

### Server Configuration

- **`PORT`** (optional)
  - Port number for the HTTP server
  - Default: `8080`
  - Example: `3001`

- **`FRONTEND_URL`** (optional)
  - URL of the frontend application for CORS configuration
  - Default: `http://localhost:3000`
  - Example: `https://mydomain.com`

### Claude Integration

- **`CLAUDE_PROJECTS_DIR`** (optional)
  - Directory where Claude Code stores project logs
  - Default: `${HOME}/.claude/projects`
  - Example: `/custom/claude/projects`

## Configuration Examples

### Development Environment

```bash
# Use default settings (no environment variables needed)
go run cmd/server/main.go
```

### Custom Database Location

```bash
export CLAUDEEE_DB_PATH="/data/claudeee/database.db"
export PORT="3001"
go run cmd/server/main.go
```

### Production Environment

```bash
export CLAUDEEE_DB_PATH="/var/lib/claudeee/claudeee.db"
export PORT="8080"
export FRONTEND_URL="https://claudeee.mydomain.com"
export CLAUDE_PROJECTS_DIR="/home/user/.claude/projects"
go run cmd/server/main.go
```

### Docker Environment

```bash
docker run -e CLAUDEEE_DB_PATH="/app/data/claudeee.db" \
           -e PORT="8080" \
           -e FRONTEND_URL="http://localhost:3000" \
           -v /host/data:/app/data \
           claudeee-backend
```

## Directory Structure

By default, the application creates the following directory structure:

```
${HOME}/.claudeee/
├── claudeee.db          # Main database file
├── claudeee.db.wal      # DuckDB write-ahead log
└── claudeee.db.tmp      # Temporary files
```

## Security Considerations

- Ensure database directory has appropriate permissions (recommended: 755)
- Database files should be readable/writable only by the application user
- In production, consider using a dedicated data directory outside the home directory
- Regularly backup the database file for data safety

## Troubleshooting

### Database Permission Issues

```bash
# Fix directory permissions
chmod 755 ~/.claudeee
chmod 644 ~/.claudeee/claudeee.db*
```

### Custom Database Location

```bash
# Create custom directory
mkdir -p /custom/path/to/database
export CLAUDEEE_DB_PATH="/custom/path/to/database/claudeee.db"
```

### Port Already in Use

```bash
# Use different port
export PORT="8081"
```

## Configuration Validation

The application validates configuration on startup and will log the following information:

```
Server starting on :8080
Database path: /Users/username/.claudeee/claudeee.db
Claude projects directory: /Users/username/.claude/projects
Frontend URL: http://localhost:3000
```

Check these logs to ensure your configuration is being applied correctly.