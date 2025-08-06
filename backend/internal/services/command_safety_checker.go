package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// CommandSafetyChecker uses Claude Code to analyze command safety
type CommandSafetyChecker struct {
	claudeCodePath string
	workingDir     string
	enabled        bool
	maxCheckTime   time.Duration
}

// NewCommandSafetyChecker creates a new command safety checker
func NewCommandSafetyChecker(workingDir string) *CommandSafetyChecker {
	checker := &CommandSafetyChecker{
		claudeCodePath: "claude", // Default to claude command
		workingDir:     workingDir,
		enabled:        true,
		maxCheckTime:   30 * time.Second, // Max time for safety check
	}

	// Check if safety checking is disabled
	if os.Getenv("CCDASH_DISABLE_SAFETY_CHECK") == "true" {
		checker.enabled = false
	}

	// Allow custom Claude Code path
	if customPath := os.Getenv("CCDASH_CLAUDE_CODE_PATH"); customPath != "" {
		checker.claudeCodePath = customPath
	}

	return checker
}

// CheckCommandSafety uses Claude Code to analyze if a command is safe to execute
func (c *CommandSafetyChecker) CheckCommandSafety(command string) error {
	// If safety checking is disabled, allow all commands
	if !c.enabled {
		return nil
	}

	// Skip safety check for obviously safe commands
	if c.isObviouslySafe(command) {
		return nil
	}

	// Create safety check prompt for Claude Code
	safetyPrompt := c.createSafetyPrompt(command)

	// Execute Claude Code safety check
	result, err := c.runClaudeCodeSafetyCheck(safetyPrompt)
	if err != nil {
		return fmt.Errorf("safety check failed: %v", err)
	}

	// Parse Claude Code response
	if c.isSafeResponse(result) {
		return nil
	}

	return fmt.Errorf("command rejected for safety reasons: %s", c.extractReason(result))
}

// isObviouslySafe checks for commands that are clearly safe and don't need AI analysis
func (c *CommandSafetyChecker) isObviouslySafe(command string) bool {
	safeCommands := []string{
		"git status",
		"git diff",
		"git log",
		"ls",
		"pwd",
		"whoami",
		"date",
		"echo",
		"cat package.json",
		"npm list",
		"go version",
		"node --version",
		"python --version",
	}

	command = strings.TrimSpace(strings.ToLower(command))
	
	for _, safe := range safeCommands {
		if command == safe || strings.HasPrefix(command, safe+" ") {
			return true
		}
	}

	return false
}

// createSafetyPrompt creates a prompt for Claude Code to analyze command safety
func (c *CommandSafetyChecker) createSafetyPrompt(command string) string {
	return fmt.Sprintf(`Please analyze the safety of the following command and respond with either "SAFE" or "UNSAFE" followed by a brief reason.

Command to analyze: %s
Working directory: %s

Consider the command UNSAFE if it:
1. Deletes or modifies system files (like /etc, /usr, /var, etc.)
2. Uses sudo or attempts privilege escalation
3. Downloads and executes untrusted code
4. Accesses or exposes sensitive information
5. Could cause resource exhaustion (fork bombs, infinite loops)
6. Modifies critical configuration files
7. Performs destructive operations without clear purpose

Consider the command SAFE if it:
1. Only reads files or displays information
2. Performs standard development tasks (build, test, lint)
3. Manages project dependencies in a standard way
4. Creates or modifies files only within the project directory

Respond in this exact format:
SAFE: [brief reason] 
OR
UNSAFE: [brief reason]`, command, c.workingDir)
}

// runClaudeCodeSafetyCheck executes Claude Code with the safety check prompt
func (c *CommandSafetyChecker) runClaudeCodeSafetyCheck(prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.maxCheckTime)
	defer cancel()

	// Execute Claude Code with --print flag and the prompt
	cmd := exec.CommandContext(ctx, c.claudeCodePath, "--print", prompt)
	cmd.Dir = c.workingDir

	output, err := cmd.Output()
	if err != nil {
		// If Claude Code is not available, fail safely by rejecting the command
		return "", fmt.Errorf("Claude Code not available for safety check: %v", err)
	}

	return string(output), nil
}

// isSafeResponse checks if Claude Code response indicates the command is safe
func (c *CommandSafetyChecker) isSafeResponse(response string) bool {
	response = strings.TrimSpace(strings.ToUpper(response))
	return strings.HasPrefix(response, "SAFE:")
}

// extractReason extracts the reason from Claude Code response
func (c *CommandSafetyChecker) extractReason(response string) string {
	lines := strings.Split(strings.TrimSpace(response), "\n")
	if len(lines) > 0 && lines[0] != "" {
		// Return first line which should contain the SAFE/UNSAFE response
		return lines[0]
	}
	return "No reason provided"
}

// IsEnabled returns whether safety checking is enabled
func (c *CommandSafetyChecker) IsEnabled() bool {
	return c.enabled
}

// ValidateCommand is an alias for CheckCommandSafety for compatibility
func (c *CommandSafetyChecker) ValidateCommand(command string) error {
	return c.CheckCommandSafety(command)
}