package services

import (
	"fmt"
	"os"
	"strings"
)

// CommandWhitelist defines allowed commands and their restrictions
type CommandWhitelist struct {
	// Map of allowed command prefixes to their descriptions
	allowedCommands map[string]string
	// Flag to enable/disable whitelist enforcement
	enabled bool
}

// NewCommandWhitelist creates a new command whitelist
func NewCommandWhitelist() *CommandWhitelist {
	whitelist := &CommandWhitelist{
		allowedCommands: make(map[string]string),
		enabled:         true,
	}

	// Check if whitelist is disabled via environment variable
	if os.Getenv("CCDASH_DISABLE_COMMAND_WHITELIST") == "true" {
		whitelist.enabled = false
		return whitelist
	}

	// Initialize default allowed commands
	whitelist.initializeDefaults()

	// Load additional commands from environment variable
	if customCommands := os.Getenv("CCDASH_ALLOWED_COMMANDS"); customCommands != "" {
		for _, cmd := range strings.Split(customCommands, ",") {
			cmd = strings.TrimSpace(cmd)
			if cmd != "" {
				whitelist.allowedCommands[cmd] = "Custom allowed command"
			}
		}
	}

	return whitelist
}

// initializeDefaults sets up the default allowed commands
func (cw *CommandWhitelist) initializeDefaults() {
	// Git commands (read-only)
	cw.allowedCommands["git status"] = "Check git repository status"
	cw.allowedCommands["git diff"] = "Show git differences"
	cw.allowedCommands["git log"] = "Show git commit history"
	cw.allowedCommands["git branch"] = "List git branches"
	cw.allowedCommands["git show"] = "Show git commit details"
	cw.allowedCommands["git remote"] = "Show git remotes"

	// Package management (read-only)
	cw.allowedCommands["npm list"] = "List npm packages"
	cw.allowedCommands["npm audit"] = "Check npm security vulnerabilities"
	cw.allowedCommands["yarn list"] = "List yarn packages"
	cw.allowedCommands["yarn audit"] = "Check yarn security vulnerabilities"
	cw.allowedCommands["go mod graph"] = "Show Go module dependency graph"
	cw.allowedCommands["go list"] = "List Go packages"
	cw.allowedCommands["pip list"] = "List Python packages"
	cw.allowedCommands["pip freeze"] = "Show installed Python packages"

	// Build and test commands
	cw.allowedCommands["npm test"] = "Run npm tests"
	cw.allowedCommands["npm run test"] = "Run npm test script"
	cw.allowedCommands["yarn test"] = "Run yarn tests"
	cw.allowedCommands["go test"] = "Run Go tests"
	cw.allowedCommands["pytest"] = "Run Python tests"
	cw.allowedCommands["make test"] = "Run make tests"

	// Linting and formatting
	cw.allowedCommands["npm run lint"] = "Run npm lint"
	cw.allowedCommands["yarn lint"] = "Run yarn lint"
	cw.allowedCommands["go fmt"] = "Format Go code"
	cw.allowedCommands["gofmt"] = "Format Go code"
	cw.allowedCommands["prettier"] = "Format code with Prettier"
	cw.allowedCommands["eslint"] = "Run ESLint"
	cw.allowedCommands["ruff"] = "Run Ruff Python linter"
	cw.allowedCommands["black"] = "Format Python code"

	// Code analysis
	cw.allowedCommands["npm run typecheck"] = "Run TypeScript type checking"
	cw.allowedCommands["tsc"] = "TypeScript compiler"
	cw.allowedCommands["go vet"] = "Run Go static analysis"
	cw.allowedCommands["mypy"] = "Run Python type checking"

	// Documentation
	cw.allowedCommands["npm run docs"] = "Generate npm documentation"
	cw.allowedCommands["go doc"] = "Show Go documentation"
	cw.allowedCommands["pydoc"] = "Show Python documentation"

	// File listing (limited)
	cw.allowedCommands["ls"] = "List directory contents"
	cw.allowedCommands["find . -name"] = "Find files by name"
	cw.allowedCommands["tree"] = "Show directory tree"

	// Development server commands
	cw.allowedCommands["npm run dev"] = "Start development server"
	cw.allowedCommands["yarn dev"] = "Start development server"
	cw.allowedCommands["npm start"] = "Start application"
	cw.allowedCommands["yarn start"] = "Start application"

	// Build commands
	cw.allowedCommands["npm run build"] = "Build project"
	cw.allowedCommands["yarn build"] = "Build project"
	cw.allowedCommands["go build"] = "Build Go project"
	cw.allowedCommands["make"] = "Run make"
	cw.allowedCommands["make build"] = "Run make build"
}

// IsCommandAllowed checks if a command is allowed
func (cw *CommandWhitelist) IsCommandAllowed(command string) bool {
	// If whitelist is disabled, allow all commands
	if !cw.enabled {
		return true
	}

	command = strings.TrimSpace(command)

	// Check exact match first
	if _, ok := cw.allowedCommands[command]; ok {
		return true
	}

	// Check if command starts with any allowed prefix
	for allowedCmd := range cw.allowedCommands {
		if strings.HasPrefix(command, allowedCmd+" ") || command == allowedCmd {
			return true
		}
	}

	return false
}

// ValidateCommand validates a command against the whitelist
func (cw *CommandWhitelist) ValidateCommand(command string) error {
	if !cw.IsCommandAllowed(command) {
		return fmt.Errorf("command not in whitelist: %s", command)
	}
	return nil
}

// GetAllowedCommands returns a list of all allowed commands
func (cw *CommandWhitelist) GetAllowedCommands() map[string]string {
	result := make(map[string]string)
	for cmd, desc := range cw.allowedCommands {
		result[cmd] = desc
	}
	return result
}

// IsEnabled returns whether the whitelist is enabled
func (cw *CommandWhitelist) IsEnabled() bool {
	return cw.enabled
}