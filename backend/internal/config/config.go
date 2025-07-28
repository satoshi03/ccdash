package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	DatabasePath     string
	DatabaseDir      string
	ServerPort       string
	FrontendURL      string
	ClaudeProjectsDir string
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
		config.ServerPort = "8080"
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