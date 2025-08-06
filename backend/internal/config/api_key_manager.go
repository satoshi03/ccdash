package config

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// APIKeyManager manages API key generation and storage
type APIKeyManager struct {
	envFilePath  string
	showFullKey  bool // Show full key in console (development mode only)
	isDevMode    bool
}

// NewAPIKeyManager creates a new API key manager
func NewAPIKeyManager(envFilePath string) *APIKeyManager {
	// Determine if we're in development mode
	isDevMode := os.Getenv("GIN_MODE") != "release"
	
	return &APIKeyManager{
		envFilePath: envFilePath,
		showFullKey: isDevMode, // Only show full key in development
		isDevMode:   isDevMode,
	}
}

// EnsureAPIKey ensures an API key exists, generating one if necessary
func (m *APIKeyManager) EnsureAPIKey() (string, bool, error) {
	// 1. Try to get from environment variable
	if key := os.Getenv("CCDASH_API_KEY"); key != "" {
		log.Printf("🔑 Using API key from environment variable")
		return key, false, nil
	}
	
	// 2. Try to load from .env file
	if key, err := m.loadFromEnvFile(); err == nil && key != "" {
		log.Printf("🔑 Using existing API key from .env file")
		// Set the environment variable for this session
		os.Setenv("CCDASH_API_KEY", key)
		return key, false, nil
	}
	
	// 3. Generate new key
	key, err := m.generateSecureKey()
	if err != nil {
		return "", false, fmt.Errorf("failed to generate API key: %v", err)
	}
	
	// 4. Save to .env file
	if err := m.saveToEnvFile(key); err != nil {
		log.Printf("⚠️  Warning: Failed to save API key to .env file: %v", err)
		log.Printf("🔧 Please manually add: CCDASH_API_KEY=%s", key)
	} else {
		log.Printf("💾 API key saved to %s", m.envFilePath)
	}
	
	// 5. Display key information
	m.displayKeyInfo(key, true)
	
	// Set the environment variable for this session
	os.Setenv("CCDASH_API_KEY", key)
	
	return key, true, nil
}

// generateSecureKey generates a cryptographically secure random API key
func (m *APIKeyManager) generateSecureKey() (string, error) {
	// Generate 32 bytes (256 bits) of random data
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	
	// Convert to hex string (64 characters)
	return hex.EncodeToString(bytes), nil
}

// loadFromEnvFile loads API key from .env file
func (m *APIKeyManager) loadFromEnvFile() (string, error) {
	if _, err := os.Stat(m.envFilePath); os.IsNotExist(err) {
		return "", err
	}
	
	file, err := os.Open(m.envFilePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "CCDASH_API_KEY=") {
			key := strings.TrimPrefix(line, "CCDASH_API_KEY=")
			// Remove quotes if present
			key = strings.Trim(key, `"'`)
			if key != "" {
				return key, nil
			}
		}
	}
	
	return "", fmt.Errorf("CCDASH_API_KEY not found in .env file")
}

// saveToEnvFile saves or updates API key in .env file
func (m *APIKeyManager) saveToEnvFile(apiKey string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(m.envFilePath), 0755); err != nil {
		return err
	}
	
	var lines []string
	keyExists := false
	
	// Read existing file if it exists
	if file, err := os.Open(m.envFilePath); err == nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(strings.TrimSpace(line), "CCDASH_API_KEY=") {
				// Update existing key
				lines = append(lines, fmt.Sprintf("CCDASH_API_KEY=%s", apiKey))
				keyExists = true
			} else {
				lines = append(lines, line)
			}
		}
		file.Close()
	}
	
	// Add key if it doesn't exist
	if !keyExists {
		if len(lines) > 0 {
			// Add a blank line before the API key section if file has content
			lines = append(lines, "")
			lines = append(lines, "# CCDash API Authentication")
		}
		lines = append(lines, fmt.Sprintf("CCDASH_API_KEY=%s", apiKey))
	}
	
	// Write back to file
	file, err := os.Create(m.envFilePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	for _, line := range lines {
		if _, err := fmt.Fprintln(file, line); err != nil {
			return err
		}
	}
	
	return nil
}

// displayKeyInfo displays API key information to the console
func (m *APIKeyManager) displayKeyInfo(key string, isNewKey bool) {
	if isNewKey {
		log.Printf("")
		log.Printf("🔐 New API key generated!")
		log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	}
	
	if m.showFullKey {
		// Development mode: show full key
		log.Printf("🔑 API Key: %s", key)
		if isNewKey {
			log.Printf("⚠️  Development mode: Full key displayed above")
		}
	} else {
		// Production mode: show truncated key
		truncated := m.truncateKey(key)
		log.Printf("🔑 API Key: %s (full key in .env file)", truncated)
		if isNewKey {
			log.Printf("🔒 Production mode: Key truncated for security")
		}
	}
	
	if isNewKey {
		log.Printf("💾 Key saved to: %s", m.envFilePath)
		log.Printf("🔧 Use this key for API authentication")
		log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		log.Printf("")
	}
}

// truncateKey returns a truncated version of the API key for secure display
func (m *APIKeyManager) truncateKey(key string) string {
	if len(key) <= 12 {
		return key
	}
	return fmt.Sprintf("%s...%s", key[:8], key[len(key)-4:])
}

// GetEnvFilePath returns the .env file path
func (m *APIKeyManager) GetEnvFilePath() string {
	return m.envFilePath
}

// IsDevMode returns whether we're in development mode
func (m *APIKeyManager) IsDevMode() bool {
	return m.isDevMode
}