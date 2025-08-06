package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// HashingConfig contains configuration for key hashing
type HashingConfig struct {
	BCryptCost int
}

// DefaultHashingConfig returns default hashing configuration
func DefaultHashingConfig() *HashingConfig {
	return &HashingConfig{
		BCryptCost: 12, // Good balance of security and performance
	}
}

// KeyHasher handles API key hashing operations
type KeyHasher struct {
	config *HashingConfig
}

// NewKeyHasher creates a new key hasher with the given configuration
func NewKeyHasher(config *HashingConfig) *KeyHasher {
	if config == nil {
		config = DefaultHashingConfig()
	}
	return &KeyHasher{config: config}
}

// HashKey hashes an API key using bcrypt
func (kh *KeyHasher) HashKey(key string) (string, error) {
	// Validate key format
	if !IsValidAPIKey(key) {
		return "", fmt.Errorf("invalid API key format")
	}

	// Hash the key using bcrypt
	hash, err := bcrypt.GenerateFromPassword([]byte(key), kh.config.BCryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash API key: %w", err)
	}

	return string(hash), nil
}

// CompareKey compares a plain text key with a hashed key
func (kh *KeyHasher) CompareKey(key, hash string) bool {
	// Validate key format first
	if !IsValidAPIKey(key) {
		return false
	}

	// Use constant-time comparison to prevent timing attacks
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(key))
	return err == nil
}

// IsValidAPIKey validates the format of an API key
func IsValidAPIKey(key string) bool {
	// API keys should start with "ftr_" and be followed by 64 hex characters
	if !strings.HasPrefix(key, "ftr_") {
		return false
	}

	keyBody := strings.TrimPrefix(key, "ftr_")
	if len(keyBody) != 64 {
		return false
	}

	// Check if the rest is valid hex
	_, err := hex.DecodeString(keyBody)
	return err == nil
}

// GenerateAPIKey generates a new cryptographically secure API key
func GenerateAPIKey() (string, error) {
	// Generate 32 random bytes (256 bits)
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Convert to hex and prefix with "ftr_"
	return "ftr_" + hex.EncodeToString(bytes), nil
}

// ExtractKeyFromAuthHeader extracts API key from Authorization header
func ExtractKeyFromAuthHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is empty")
	}

	// Support both "Bearer" and "ApiKey" prefixes
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid authorization header format")
	}

	prefix := strings.ToLower(parts[0])
	key := parts[1]

	if prefix != "bearer" && prefix != "apikey" {
		return "", fmt.Errorf("unsupported authorization type: %s", parts[0])
	}

	if !IsValidAPIKey(key) {
		return "", fmt.Errorf("invalid API key format")
	}

	return key, nil
}

// SecureCompare performs a constant-time comparison of two strings
// This helps prevent timing attacks when comparing sensitive data
func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}