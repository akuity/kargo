package credentials

import (
	"testing"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
)

func TestCalculateCacheTTL(t *testing.T) {
	const margin = 5 * time.Minute

	tests := []struct {
		name     string
		expiry   time.Time
		margin   time.Duration
		expected time.Duration
	}{
		{
			name:     "zero expiry returns default",
			expiry:   time.Time{},
			margin:   margin,
			expected: cache.DefaultExpiration,
		},
		{
			name:     "expiry far in the future returns remaining minus margin",
			expiry:   time.Now().Add(time.Hour),
			margin:   margin,
			expected: 55 * time.Minute,
		},
		{
			name:     "expiry in the past returns default",
			expiry:   time.Now().Add(-time.Hour),
			margin:   margin,
			expected: cache.DefaultExpiration,
		},
		{
			name:     "remaining equals margin returns default",
			expiry:   time.Now().Add(margin),
			margin:   margin,
			expected: cache.DefaultExpiration,
		},
		{
			name:     "remaining less than margin returns default",
			expiry:   time.Now().Add(margin - time.Second),
			margin:   margin,
			expected: cache.DefaultExpiration,
		},
		{
			name:     "zero margin returns full remaining time",
			expiry:   time.Now().Add(30 * time.Minute),
			margin:   0,
			expected: 30 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateCacheTTL(tt.expiry, tt.margin)
			// Allow 1 second of tolerance since time.Now() is called in both
			// the test setup and the function under test.
			assert.InDelta(t, tt.expected, result, float64(time.Second))
		})
	}
}
