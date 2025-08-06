package auth

import (
	"testing"
)

func TestGenerateAPIKey(t *testing.T) {
	key, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("Failed to generate API key: %v", err)
	}

	if !IsValidAPIKey(key) {
		t.Errorf("Generated key is not valid: %s", key)
	}

	if len(key) != 68 { // "ftr_" (4) + 64 hex characters
		t.Errorf("Generated key has wrong length: expected 68, got %d", len(key))
	}

	// Ensure prefix is correct
	if key[:4] != "ftr_" {
		t.Errorf("Generated key has wrong prefix: expected 'ftr_', got '%s'", key[:4])
	}
}

func TestIsValidAPIKey(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		valid bool
	}{
		{"Valid key", "ftr_1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", true},
		{"Invalid prefix", "invalid_1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", false},
		{"Too short", "ftr_123", false},
		{"Too long", "ftr_1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", false},
		{"Invalid hex", "ftr_1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcgef", false},
		{"Empty string", "", false},
		{"Only prefix", "ftr_", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if IsValidAPIKey(tt.key) != tt.valid {
				t.Errorf("IsValidAPIKey(%s) = %v, expected %v", tt.key, !tt.valid, tt.valid)
			}
		})
	}
}

func TestKeyHasher(t *testing.T) {
	hasher := NewKeyHasher(nil)
	key, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("Failed to generate API key: %v", err)
	}

	// Test hashing
	hash, err := hasher.HashKey(key)
	if err != nil {
		t.Fatalf("Failed to hash key: %v", err)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	// Test comparison with correct key
	if !hasher.CompareKey(key, hash) {
		t.Error("CompareKey should return true for correct key")
	}

	// Test comparison with incorrect key
	wrongKey, _ := GenerateAPIKey()
	if hasher.CompareKey(wrongKey, hash) {
		t.Error("CompareKey should return false for incorrect key")
	}

	// Test with invalid key format
	if hasher.CompareKey("invalid", hash) {
		t.Error("CompareKey should return false for invalid key format")
	}
}

func TestExtractKeyFromAuthHeader(t *testing.T) {
	validKey := "ftr_1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	
	tests := []struct {
		name        string
		header      string
		expectedKey string
		expectError bool
	}{
		{"Valid Bearer token", "Bearer " + validKey, validKey, false},
		{"Valid ApiKey token", "ApiKey " + validKey, validKey, false},
		{"Case insensitive bearer", "bearer " + validKey, validKey, false},
		{"Case insensitive apikey", "apikey " + validKey, validKey, false},
		{"Empty header", "", "", true},
		{"No space separator", "Bearer" + validKey, "", true},
		{"Invalid prefix", "Basic " + validKey, "", true},
		{"Invalid key format", "Bearer invalid_key", "", true},
		{"Missing key", "Bearer ", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := ExtractKeyFromAuthHeader(tt.header)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if key != tt.expectedKey {
					t.Errorf("Expected key %s, got %s", tt.expectedKey, key)
				}
			}
		})
	}
}

func TestSecureCompare(t *testing.T) {
	tests := []struct {
		name     string
		a, b     string
		expected bool
	}{
		{"Equal strings", "hello", "hello", true},
		{"Different strings", "hello", "world", false},
		{"Empty strings", "", "", true},
		{"One empty", "hello", "", false},
		{"Different lengths", "hello", "hello world", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SecureCompare(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("SecureCompare(%q, %q) = %v, expected %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}