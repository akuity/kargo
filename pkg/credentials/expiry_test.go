package credentials

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateCacheTTL(t *testing.T) {
	t.Parallel()

	const margin = 5 * time.Minute

	testCases := []struct {
		name   string
		expiry time.Time
		margin time.Duration
		// expected is the duration the credential may be cached for, or nil where
		// it is not to be cached at all.
		expected *time.Duration
	}{
		{
			name:   "unknown expiry defers to the cache's default",
			expiry: time.Time{},
			margin: margin,
			// A zero duration is what defers to that default, and is distinct from
			// declining to cache.
			expected: new(time.Duration),
		},
		{
			name:     "expiry far in the future returns remaining minus margin",
			expiry:   time.Now().Add(time.Hour),
			margin:   margin,
			expected: new(55 * time.Minute),
		},
		{
			name:     "zero margin returns the whole remaining time",
			expiry:   time.Now().Add(30 * time.Minute),
			margin:   0,
			expected: new(30 * time.Minute),
		},
		{
			name:   "expiry already past is not cached",
			expiry: time.Now().Add(-time.Hour),
			margin: margin,
		},
		{
			name:   "expiry exactly at the margin is not cached",
			expiry: time.Now().Add(margin),
			margin: margin,
		},
		{
			name:   "expiry inside the margin is not cached",
			expiry: time.Now().Add(margin - time.Second),
			margin: margin,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			ttl := CalculateCacheTTL(testCase.expiry, testCase.margin)
			if testCase.expected == nil {
				require.Nil(t, ttl)
				return
			}
			require.NotNil(t, ttl)
			// Allow a second of tolerance, time.Now() being called both in this
			// test's setup and in the function under test.
			assert.InDelta(t, *testCase.expected, *ttl, float64(time.Second))
		})
	}
}
