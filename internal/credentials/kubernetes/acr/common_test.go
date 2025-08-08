package acr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenCacheKey(t *testing.T) {
	testCases := []struct {
		name     string
		parts    []string
		expected string
	}{
		{
			name:     "single part",
			parts:    []string{"test"},
			expected: "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", // sha256 of "test"
		},
		{
			name:     "multiple parts",
			parts:    []string{"registry", "project"},
			expected: "1ce59597b20d4eaf682ca8ea2f9e542eacb3b008cd0eedaddee686f5565d2c04", // sha256 of "registry:project"
		},
		{
			name:     "empty parts",
			parts:    []string{},
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", // sha256 of empty string
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenCacheKey(tt.parts...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTokenCacheKeyConsistency(t *testing.T) {
	// Test that the same inputs always produce the same output
	parts := []string{"myregistry", "project1"}
	result1 := tokenCacheKey(parts...)
	result2 := tokenCacheKey(parts...)
	assert.Equal(t, result1, result2, "Cache key should be deterministic")
}

func TestTokenCacheKeyUniqueness(t *testing.T) {
	// Test that different inputs produce different outputs
	key1 := tokenCacheKey("registry1", "project1")
	key2 := tokenCacheKey("registry2", "project1")
	key3 := tokenCacheKey("registry1", "project2")

	assert.NotEqual(t, key1, key2, "Different registries should produce different cache keys")
	assert.NotEqual(t, key1, key3, "Different projects should produce different cache keys")
	assert.NotEqual(t, key2, key3, "Different combinations should produce different cache keys")
}
