package security

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// APIKeyRecord represents an API key with metadata
type APIKeyRecord struct {
	KeyHash    string    `json:"key_hash"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	IsActive   bool      `json:"is_active"`
	LastUsed   *time.Time `json:"last_used,omitempty"`
	UsageCount int       `json:"usage_count"`
}

// KeyRotationManager manages API key rotation and validation
type KeyRotationManager struct {
	keys       map[string]*APIKeyRecord // key_hash -> record
	mu         sync.RWMutex
	maxKeys    int
	keyTTL     time.Duration
}

// NewKeyRotationManager creates a new key rotation manager
func NewKeyRotationManager(maxKeys int, keyTTL time.Duration) *KeyRotationManager {
	krm := &KeyRotationManager{
		keys:    make(map[string]*APIKeyRecord),
		maxKeys: maxKeys,
		keyTTL:  keyTTL,
	}

	// Start cleanup goroutine
	go krm.cleanup()
	return krm
}

// AddKey adds a new API key to the rotation manager
func (krm *KeyRotationManager) AddKey(key string) error {
	keyHash := krm.hashKey(key)
	now := time.Now()
	expiresAt := now.Add(krm.keyTTL)

	krm.mu.Lock()
	defer krm.mu.Unlock()

	// Check if we've reached the maximum number of keys
	if len(krm.keys) >= krm.maxKeys {
		// Remove the oldest inactive key
		krm.removeOldestInactiveKey()
	}

	krm.keys[keyHash] = &APIKeyRecord{
		KeyHash:    keyHash,
		CreatedAt:  now,
		ExpiresAt:  &expiresAt,
		IsActive:   true,
		UsageCount: 0,
	}

	return nil
}

// ValidateKey validates an API key and updates its usage statistics
func (krm *KeyRotationManager) ValidateKey(key string) bool {
	keyHash := krm.hashKey(key)

	krm.mu.Lock()
	defer krm.mu.Unlock()

	record, exists := krm.keys[keyHash]
	if !exists {
		return false
	}

	// Check if key is active
	if !record.IsActive {
		return false
	}

	// Check if key is expired
	if record.ExpiresAt != nil && time.Now().After(*record.ExpiresAt) {
		record.IsActive = false
		return false
	}

	// Update usage statistics
	now := time.Now()
	record.LastUsed = &now
	record.UsageCount++

	return true
}

// RotateKey deactivates the old key and creates a new one
func (krm *KeyRotationManager) RotateKey(oldKey string) (string, error) {
	// Generate new key
	newKey, err := GenerateRandomKey(32) // 32 bytes = 256 bits
	if err != nil {
		return "", fmt.Errorf("failed to generate new key: %w", err)
	}

	// Add new key
	if err := krm.AddKey(newKey); err != nil {
		return "", fmt.Errorf("failed to add new key: %w", err)
	}

	// Deactivate old key (keep it for a grace period)
	oldKeyHash := krm.hashKey(oldKey)
	krm.mu.Lock()
	if record, exists := krm.keys[oldKeyHash]; exists {
		record.IsActive = false
		// Set expiry to 1 hour from now for grace period
		graceExpiry := time.Now().Add(time.Hour)
		record.ExpiresAt = &graceExpiry
	}
	krm.mu.Unlock()

	return newKey, nil
}

// GetKeyStatistics returns usage statistics for all keys
func (krm *KeyRotationManager) GetKeyStatistics() []APIKeyRecord {
	krm.mu.RLock()
	defer krm.mu.RUnlock()

	stats := make([]APIKeyRecord, 0, len(krm.keys))
	for _, record := range krm.keys {
		stats = append(stats, *record)
	}

	return stats
}

// hashKey creates a SHA-256 hash of the API key
func (krm *KeyRotationManager) hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// removeOldestInactiveKey removes the oldest inactive key
func (krm *KeyRotationManager) removeOldestInactiveKey() {
	var oldestKey string
	var oldestTime time.Time

	for keyHash, record := range krm.keys {
		if !record.IsActive && (oldestKey == "" || record.CreatedAt.Before(oldestTime)) {
			oldestKey = keyHash
			oldestTime = record.CreatedAt
		}
	}

	if oldestKey != "" {
		delete(krm.keys, oldestKey)
	}
}

// cleanup removes expired keys periodically
func (krm *KeyRotationManager) cleanup() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		krm.mu.Lock()
		for keyHash, record := range krm.keys {
			if record.ExpiresAt != nil && now.After(*record.ExpiresAt) {
				delete(krm.keys, keyHash)
			}
		}
		krm.mu.Unlock()
	}
}