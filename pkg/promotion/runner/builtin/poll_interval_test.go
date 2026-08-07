package builtin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_resolvePollInterval(t *testing.T) {
	const defaultInterval = 30 * time.Second

	testCases := []struct {
		name       string
		configured string
		assertions func(*testing.T, time.Duration, error)
	}{
		{
			name:       "empty falls back to default",
			configured: "",
			assertions: func(t *testing.T, interval time.Duration, err error) {
				require.NoError(t, err)
				require.Equal(t, defaultInterval, interval)
			},
		},
		{
			name:       "explicit value takes precedence",
			configured: "45s",
			assertions: func(t *testing.T, interval time.Duration, err error) {
				require.NoError(t, err)
				require.Equal(t, 45*time.Second, interval)
			},
		},
		{
			name:       "invalid value returns an error",
			configured: "soon",
			assertions: func(t *testing.T, _ time.Duration, err error) {
				require.ErrorContains(t, err, "error parsing pollInterval")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			interval, err := resolvePollInterval(testCase.configured, defaultInterval)
			testCase.assertions(t, interval, err)
		})
	}
}
