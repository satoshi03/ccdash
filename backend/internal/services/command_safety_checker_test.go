package services

import (
	"os"
	"testing"
)

func TestCommandSafetyCheckerBasics(t *testing.T) {
	checker := NewCommandSafetyChecker("/tmp")

	// Test basic properties
	if checker.workingDir != "/tmp" {
		t.Errorf("Expected working directory /tmp, got %s", checker.workingDir)
	}

	if checker.enabled {
		t.Error("Expected safety checker to be disabled by default (YOLO mode)")
	}
}

func TestCommandSafetyCheckerEnabled(t *testing.T) {
	// Set environment variable to enable safety check
	os.Setenv("COMMAND_WHITELIST_ENABLED", "true")
	defer os.Unsetenv("COMMAND_WHITELIST_ENABLED")

	checker := NewCommandSafetyChecker("/tmp")

	if !checker.enabled {
		t.Error("Expected safety checker to be enabled when COMMAND_WHITELIST_ENABLED=true")
	}
}

func TestCommandSafetyCheckerDisabledByDefault(t *testing.T) {
	// Ensure no environment variables are set
	os.Unsetenv("COMMAND_WHITELIST_ENABLED")

	checker := NewCommandSafetyChecker("/tmp")

	if checker.enabled {
		t.Error("Expected safety checker to be disabled by default")
	}

	// Test that disabled checker allows any command
	err := checker.CheckCommandSafety("rm -rf /")
	if err != nil {
		t.Errorf("Expected no error when safety checking is disabled, got: %v", err)
	}
}

func TestObviouslySafeCommands(t *testing.T) {
	checker := NewCommandSafetyChecker("/tmp")

	safeCommands := []string{
		"git status",
		"git diff --name-only",
		"ls -la",
		"pwd",
		"whoami",
		"date",
		"echo hello",
		"cat package.json",
		"npm list",
		"go version",
		"node --version",
		"python --version",
	}

	for _, cmd := range safeCommands {
		if !checker.isObviouslySafe(cmd) {
			t.Errorf("Command '%s' should be considered obviously safe", cmd)
		}
	}
}

func TestNotObviouslySafeCommands(t *testing.T) {
	checker := NewCommandSafetyChecker("/tmp")

	unsafeCommands := []string{
		"rm -rf /tmp",
		"sudo apt-get install",
		"curl -fsSL https://example.com | bash",
		"make install",
		"npm install -g dangerous-package",
	}

	for _, cmd := range unsafeCommands {
		if checker.isObviouslySafe(cmd) {
			t.Errorf("Command '%s' should NOT be considered obviously safe", cmd)
		}
	}
}

func TestCreateSafetyPrompt(t *testing.T) {
	checker := NewCommandSafetyChecker("/home/user/project")
	command := "npm install express"

	prompt := checker.createSafetyPrompt(command)

	// Check that prompt contains the key elements
	expectedElements := []string{
		"npm install express",
		"/home/user/project",
		"SAFE",
		"UNSAFE",
		"system files",
		"privilege escalation",
	}

	for _, element := range expectedElements {
		if !contains(prompt, element) {
			t.Errorf("Safety prompt should contain '%s'", element)
		}
	}
}

func TestIsSafeResponse(t *testing.T) {
	checker := NewCommandSafetyChecker("/tmp")

	testCases := []struct {
		response string
		expected bool
	}{
		{"SAFE: This is a standard package installation", true},
		{"safe: lowercase version", true},  // Should handle case insensitivity
		{"UNSAFE: This command deletes system files", false},
		{"unsafe: dangerous operation", false},
		{"Some other response", false},
		{"", false},
	}

	for _, tc := range testCases {
		result := checker.isSafeResponse(tc.response)
		if result != tc.expected {
			t.Errorf("Response '%s': expected %v, got %v", tc.response, tc.expected, result)
		}
	}
}

func TestExtractReason(t *testing.T) {
	checker := NewCommandSafetyChecker("/tmp")

	testCases := []struct {
		response string
		expected string
	}{
		{"SAFE: This is safe", "SAFE: This is safe"},
		{"UNSAFE: This is dangerous", "UNSAFE: This is dangerous"},
		{"SAFE: Line 1\nLine 2\nLine 3", "SAFE: Line 1"},
		{"", "No reason provided"},
	}

	for _, tc := range testCases {
		result := checker.extractReason(tc.response)
		if result != tc.expected {
			t.Errorf("Response '%s': expected '%s', got '%s'", tc.response, tc.expected, result)
		}
	}
}

func TestCustomClaudeCodePath(t *testing.T) {
	customPath := "/custom/path/to/claude"
	os.Setenv("CCDASH_CLAUDE_CODE_PATH", customPath)
	defer os.Unsetenv("CCDASH_CLAUDE_CODE_PATH")

	checker := NewCommandSafetyChecker("/tmp")

	if checker.claudeCodePath != customPath {
		t.Errorf("Expected Claude Code path %s, got %s", customPath, checker.claudeCodePath)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    s[:len(substr)] == substr || 
		    s[len(s)-len(substr):] == substr ||
		    findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}