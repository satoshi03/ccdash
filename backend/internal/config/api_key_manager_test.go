package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAPIKeyManager_GenerateSecureKey(t *testing.T) {
	manager := NewAPIKeyManager("/tmp/.env")
	
	key1, err := manager.generateSecureKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	
	key2, err := manager.generateSecureKey()
	if err != nil {
		t.Fatalf("Failed to generate second key: %v", err)
	}
	
	// Keys should be different
	if key1 == key2 {
		t.Error("Generated keys should be different")
	}
	
	// Keys should be 64 characters (32 bytes hex encoded)
	if len(key1) != 64 {
		t.Errorf("Key length should be 64, got %d", len(key1))
	}
	
	// Keys should be hex encoded (only contain hex characters)
	for _, char := range key1 {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			t.Errorf("Key contains non-hex character: %c", char)
		}
	}
}

func TestAPIKeyManager_TruncateKey(t *testing.T) {
	manager := NewAPIKeyManager("/tmp/.env")
	
	testCases := []struct {
		input    string
		expected string
	}{
		{"abcdefghijklmnopqrstuvwxyz1234567890", "abcdefgh...7890"},
		{"short", "short"},
		{"", ""},
		{"12345678901234567890", "12345678...7890"},
	}
	
	for _, tc := range testCases {
		result := manager.truncateKey(tc.input)
		if result != tc.expected {
			t.Errorf("truncateKey(%s) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

func TestAPIKeyManager_SaveAndLoadFromEnvFile(t *testing.T) {
	// Create temporary file
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	
	manager := NewAPIKeyManager(envPath)
	testKey := "test-api-key-12345"
	
	// Test save
	err := manager.saveToEnvFile(testKey)
	if err != nil {
		t.Fatalf("Failed to save to env file: %v", err)
	}
	
	// Test load
	loadedKey, err := manager.loadFromEnvFile()
	if err != nil {
		t.Fatalf("Failed to load from env file: %v", err)
	}
	
	if loadedKey != testKey {
		t.Errorf("Loaded key %s does not match saved key %s", loadedKey, testKey)
	}
	
	// Test that file contains the key
	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("Failed to read env file: %v", err)
	}
	
	if !strings.Contains(string(content), "CCDASH_API_KEY="+testKey) {
		t.Error("Env file does not contain the expected key")
	}
}

func TestAPIKeyManager_UpdateExistingEnvFile(t *testing.T) {
	// Create temporary file with existing content
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	
	// Create file with existing content
	initialContent := `# Existing content
APP_NAME=testapp
CCDASH_API_KEY=old-key
OTHER_VAR=value`
	
	err := os.WriteFile(envPath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}
	
	manager := NewAPIKeyManager(envPath)
	newKey := "new-api-key-67890"
	
	// Update the key
	err = manager.saveToEnvFile(newKey)
	if err != nil {
		t.Fatalf("Failed to update env file: %v", err)
	}
	
	// Verify the new key is there
	loadedKey, err := manager.loadFromEnvFile()
	if err != nil {
		t.Fatalf("Failed to load updated key: %v", err)
	}
	
	if loadedKey != newKey {
		t.Errorf("Updated key %s does not match expected key %s", loadedKey, newKey)
	}
	
	// Verify other content is preserved
	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}
	
	contentStr := string(content)
	if !strings.Contains(contentStr, "APP_NAME=testapp") {
		t.Error("Original content was not preserved")
	}
	
	if !strings.Contains(contentStr, "OTHER_VAR=value") {
		t.Error("Original content was not preserved")
	}
	
	if strings.Contains(contentStr, "old-key") {
		t.Error("Old key was not replaced")
	}
}

func TestAPIKeyManager_EnsureAPIKey_FromEnvironment(t *testing.T) {
	// Set environment variable
	testKey := "env-test-key-123"
	os.Setenv("CCDASH_API_KEY", testKey)
	defer os.Unsetenv("CCDASH_API_KEY")
	
	manager := NewAPIKeyManager("/tmp/.env")
	
	key, isNew, err := manager.EnsureAPIKey()
	if err != nil {
		t.Fatalf("EnsureAPIKey failed: %v", err)
	}
	
	if key != testKey {
		t.Errorf("Expected key %s, got %s", testKey, key)
	}
	
	if isNew {
		t.Error("Key from environment should not be marked as new")
	}
}

func TestAPIKeyManager_EnsureAPIKey_GenerateNew(t *testing.T) {
	// Clear environment variable
	os.Unsetenv("CCDASH_API_KEY")
	
	// Use temporary directory that doesn't exist
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, "subdir", ".env")
	
	manager := NewAPIKeyManager(envPath)
	
	key, isNew, err := manager.EnsureAPIKey()
	if err != nil {
		t.Fatalf("EnsureAPIKey failed: %v", err)
	}
	
	if key == "" {
		t.Error("Generated key should not be empty")
	}
	
	if !isNew {
		t.Error("Generated key should be marked as new")
	}
	
	if len(key) != 64 {
		t.Errorf("Generated key should be 64 characters, got %d", len(key))
	}
	
	// Verify key was saved to file
	loadedKey, err := manager.loadFromEnvFile()
	if err != nil {
		t.Fatalf("Failed to load saved key: %v", err)
	}
	
	if loadedKey != key {
		t.Error("Saved key does not match generated key")
	}
}

func TestAPIKeyManager_DevModeDetection(t *testing.T) {
	// Test production mode
	os.Setenv("GIN_MODE", "release")
	manager := NewAPIKeyManager("/tmp/.env")
	if manager.IsDevMode() {
		t.Error("Should not be in dev mode when GIN_MODE=release")
	}
	if manager.showFullKey {
		t.Error("Should not show full key in production mode")
	}
	
	// Test development mode
	os.Setenv("GIN_MODE", "debug")
	manager = NewAPIKeyManager("/tmp/.env")
	if !manager.IsDevMode() {
		t.Error("Should be in dev mode when GIN_MODE=debug")
	}
	if !manager.showFullKey {
		t.Error("Should show full key in development mode")
	}
	
	// Test default (no GIN_MODE set)
	os.Unsetenv("GIN_MODE")
	manager = NewAPIKeyManager("/tmp/.env")
	if !manager.IsDevMode() {
		t.Error("Should be in dev mode by default")
	}
	
	// Clean up
	os.Unsetenv("GIN_MODE")
}