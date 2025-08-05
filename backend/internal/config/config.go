package config

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	DatabasePath     string
	DatabaseDir      string
	ServerPort       string
	ServerHost       string
	FrontendURL      string
	ClaudeProjectsDir string
	
	// Job Scheduler configuration
	JobSchedulerPollingInterval time.Duration
	JobExecutorWorkerCount      int
	
	// Authentication configuration
	JWTSecret                   string
	AuthEnabled                 bool
}

// GetConfig returns the application configuration based on environment variables
func GetConfig() (*Config, error) {
	config := &Config{}

	// Database configuration
	if dbPath := os.Getenv("CCDASH_DB_PATH"); dbPath != "" {
		config.DatabasePath = dbPath
		config.DatabaseDir = filepath.Dir(dbPath)
	} else {
		// Default database location
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		config.DatabaseDir = filepath.Join(homeDir, ".ccdash")
		config.DatabasePath = filepath.Join(config.DatabaseDir, "ccdash.db")
	}

	// Server configuration
	config.ServerPort = os.Getenv("PORT")
	if config.ServerPort == "" {
		config.ServerPort = "6060"
	}

	config.ServerHost = os.Getenv("HOST")
	if config.ServerHost == "" {
		config.ServerHost = "localhost"
	}

	// Frontend URL configuration
	config.FrontendURL = os.Getenv("FRONTEND_URL")
	if config.FrontendURL == "" {
		config.FrontendURL = "http://localhost:3000"
	}

	// Claude projects directory
	if claudeDir := os.Getenv("CLAUDE_PROJECTS_DIR"); claudeDir != "" {
		config.ClaudeProjectsDir = claudeDir
	} else {
		// Default Claude projects location
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		config.ClaudeProjectsDir = filepath.Join(homeDir, ".claude", "projects")
	}

	// Job Scheduler configuration
	// Polling interval (default: 1 minute)
	if pollingInterval := os.Getenv("JOB_SCHEDULER_POLLING_INTERVAL"); pollingInterval != "" {
		duration, err := time.ParseDuration(pollingInterval)
		if err != nil {
			return nil, err
		}
		config.JobSchedulerPollingInterval = duration
	} else {
		config.JobSchedulerPollingInterval = 1 * time.Minute
	}

	// Worker count (default: 3)
	if workerCount := os.Getenv("JOB_EXECUTOR_WORKER_COUNT"); workerCount != "" {
		count, err := strconv.Atoi(workerCount)
		if err != nil {
			return nil, err
		}
		config.JobExecutorWorkerCount = count
	} else {
		config.JobExecutorWorkerCount = 3
	}

	// Authentication configuration
	config.JWTSecret = os.Getenv("JWT_SECRET")
	if config.JWTSecret == "" {
		// Generate a random JWT secret if not provided
		secret, err := generateRandomSecret(32)
		if err != nil {
			return nil, err
		}
		config.JWTSecret = secret
	}

	// Auth enabled flag (default: false for backward compatibility)
	config.AuthEnabled = os.Getenv("AUTH_ENABLED") == "true"

	return config, nil
}

// EnsureDatabaseDir creates the database directory if it doesn't exist
func (c *Config) EnsureDatabaseDir() error {
	return os.MkdirAll(c.DatabaseDir, 0755)
}

// DatabaseExists checks if the database file exists
func (c *Config) DatabaseExists() bool {
	_, err := os.Stat(c.DatabasePath)
	return !os.IsNotExist(err)
}

// generateRandomSecret generates a random hex-encoded secret
func generateRandomSecret(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}