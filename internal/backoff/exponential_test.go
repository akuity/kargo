package backoff

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestJitteredExponential(t *testing.T) {
	testCases := []struct {
		failureCount int
		cap          time.Duration
		expectedMin  time.Duration
		expectedMax  time.Duration
	}{
		{
			failureCount: 1,
			cap:          time.Minute,
			expectedMin:  time.Second,
			expectedMax:  2 * time.Second,
		},
		{
			failureCount: 2,
			cap:          time.Minute,
			expectedMin:  2 * time.Second,
			expectedMax:  4 * time.Second,
		},
		{
			failureCount: 3,
			cap:          time.Minute,
			expectedMin:  4 * time.Second,
			expectedMax:  8 * time.Second,
		},
		{
			failureCount: 4,
			cap:          time.Minute,
			expectedMin:  8 * time.Second,
			expectedMax:  16 * time.Second,
		},
		{
			failureCount: 5,
			cap:          time.Minute,
			expectedMin:  16 * time.Second,
			expectedMax:  32 * time.Second,
		},
		{
			failureCount: 6,
			cap:          time.Minute,
			expectedMin:  30 * time.Second,
			expectedMax:  time.Minute,
		},
		{
			failureCount: 7,
			cap:          time.Minute,
			expectedMin:  30 * time.Second,
			expectedMax:  time.Minute,
		},
		{
			failureCount: 8,
			cap:          time.Minute,
			expectedMin:  30 * time.Second,
			expectedMax:  time.Minute,
		},
	}
	for _, testCase := range testCases {
		t.Run(strconv.Itoa(testCase.failureCount), func(t *testing.T) {
			delay1 := JitteredExponential(testCase.failureCount, testCase.cap)

			require.Less(t, testCase.expectedMin.Seconds(), delay1.Seconds())
			require.Less(t, delay1.Seconds(), testCase.expectedMax.Seconds())

			// Make sure the jitter works
			delay2 := JitteredExponential(testCase.failureCount, testCase.cap)
			require.NotEqual(t, delay1, delay2)
		})
	}
}
