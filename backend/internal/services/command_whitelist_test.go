package services

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandWhitelist(t *testing.T) {
	tests := []struct {
		name            string
		command         string
		envWhitelist    string
		envDisable      string
		expectedAllowed bool
	}{
		// Allowed commands
		{
			name:            "Git status allowed",
			command:         "git status",
			expectedAllowed: true,
		},
		{
			name:            "Git diff allowed",
			command:         "git diff --cached",
			expectedAllowed: true,
		},
		{
			name:            "npm test allowed",
			command:         "npm test",
			expectedAllowed: true,
		},
		{
			name:            "npm run test allowed",
			command:         "npm run test",
			expectedAllowed: true,
		},
		{
			name:            "go test with args allowed",
			command:         "go test ./... -v",
			expectedAllowed: true,
		},
		{
			name:            "make test allowed",
			command:         "make test",
			expectedAllowed: true,
		},
		{
			name:            "ls allowed",
			command:         "ls -la",
			expectedAllowed: true,
		},
		// Disallowed commands
		{
			name:            "rm command not allowed",
			command:         "rm -rf node_modules",
			expectedAllowed: false,
		},
		{
			name:            "curl not allowed by default",
			command:         "curl https://example.com",
			expectedAllowed: false,
		},
		{
			name:            "arbitrary command not allowed",
			command:         "echo 'hello world'",
			expectedAllowed: false,
		},
		{
			name:            "git push not allowed",
			command:         "git push origin main",
			expectedAllowed: false,
		},
		// Custom allowed commands via env
		{
			name:            "Custom command allowed via env",
			command:         "custom-tool",
			envWhitelist:    "custom-tool,another-tool",
			expectedAllowed: true,
		},
		{
			name:            "Custom command with args allowed",
			command:         "custom-tool --help",
			envWhitelist:    "custom-tool",
			expectedAllowed: true,
		},
		// Whitelist disabled
		{
			name:            "Any command allowed when whitelist disabled",
			command:         "rm -rf /",
			envDisable:      "true",
			expectedAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore env vars
			oldWhitelist := os.Getenv("CCDASH_ALLOWED_COMMANDS")
			oldDisable := os.Getenv("CCDASH_DISABLE_COMMAND_WHITELIST")
			defer func() {
				os.Setenv("CCDASH_ALLOWED_COMMANDS", oldWhitelist)
				os.Setenv("CCDASH_DISABLE_COMMAND_WHITELIST", oldDisable)
			}()

			// Set test env vars
			if tt.envWhitelist != "" {
				os.Setenv("CCDASH_ALLOWED_COMMANDS", tt.envWhitelist)
			} else {
				os.Unsetenv("CCDASH_ALLOWED_COMMANDS")
			}
			if tt.envDisable != "" {
				os.Setenv("CCDASH_DISABLE_COMMAND_WHITELIST", tt.envDisable)
			} else {
				os.Unsetenv("CCDASH_DISABLE_COMMAND_WHITELIST")
			}

			// Create whitelist and test
			whitelist := NewCommandWhitelist()
			allowed := whitelist.IsCommandAllowed(tt.command)
			assert.Equal(t, tt.expectedAllowed, allowed)

			// Test ValidateCommand method
			err := whitelist.ValidateCommand(tt.command)
			if tt.expectedAllowed {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not in whitelist")
			}
		})
	}
}

func TestCommandWhitelistDefaults(t *testing.T) {
	// Clear env vars
	oldWhitelist := os.Getenv("CCDASH_ALLOWED_COMMANDS")
	oldDisable := os.Getenv("CCDASH_DISABLE_COMMAND_WHITELIST")
	defer func() {
		os.Setenv("CCDASH_ALLOWED_COMMANDS", oldWhitelist)
		os.Setenv("CCDASH_DISABLE_COMMAND_WHITELIST", oldDisable)
	}()
	os.Unsetenv("CCDASH_ALLOWED_COMMANDS")
	os.Unsetenv("CCDASH_DISABLE_COMMAND_WHITELIST")

	whitelist := NewCommandWhitelist()

	// Test that some default commands are present
	expectedDefaults := []string{
		"git status",
		"git diff",
		"npm test",
		"go test",
		"ls",
	}

	for _, cmd := range expectedDefaults {
		assert.True(t, whitelist.IsCommandAllowed(cmd), "Command %s should be allowed", cmd)
	}

	// Test GetAllowedCommands
	allowedCommands := whitelist.GetAllowedCommands()
	assert.NotEmpty(t, allowedCommands)
	assert.Contains(t, allowedCommands, "git status")
	assert.Contains(t, allowedCommands, "npm test")
}

func TestCommandWhitelistEnabled(t *testing.T) {
	tests := []struct {
		name        string
		envDisable  string
		expected    bool
	}{
		{
			name:       "Enabled by default",
			envDisable: "",
			expected:   true,
		},
		{
			name:       "Disabled via env",
			envDisable: "true",
			expected:   false,
		},
		{
			name:       "Not disabled with false value",
			envDisable: "false",
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldDisable := os.Getenv("CCDASH_DISABLE_COMMAND_WHITELIST")
			defer os.Setenv("CCDASH_DISABLE_COMMAND_WHITELIST", oldDisable)

			if tt.envDisable != "" {
				os.Setenv("CCDASH_DISABLE_COMMAND_WHITELIST", tt.envDisable)
			} else {
				os.Unsetenv("CCDASH_DISABLE_COMMAND_WHITELIST")
			}

			whitelist := NewCommandWhitelist()
			assert.Equal(t, tt.expected, whitelist.IsEnabled())
		})
	}
}