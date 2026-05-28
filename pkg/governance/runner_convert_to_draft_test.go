package governance

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_convertToDraftRunner_run(t *testing.T) {
	testCases := []struct {
		name       string
		config     []byte
		isPR       bool
		convertErr error
		assert     func(t *testing.T, calls int, err error)
	}{
		{
			name:   "decode error — config is not a bool",
			config: []byte(`"yes"`),
			isPR:   true,
			assert: func(t *testing.T, calls int, err error) {
				require.ErrorContains(t, err, "decoding convertToDraft config")
				require.Zero(t, calls)
			},
		},
		{
			name:   "false on PR — no-op",
			config: []byte(`false`),
			isPR:   true,
			assert: func(t *testing.T, calls int, err error) {
				require.NoError(t, err)
				require.Zero(t, calls)
			},
		},
		{
			name:   "true on issue — silent no-op (PR-only action)",
			config: []byte(`true`),
			isPR:   false,
			assert: func(t *testing.T, calls int, err error) {
				require.NoError(t, err)
				require.Zero(t, calls)
			},
		},
		{
			name:   "true on PR — ConvertToDraft called",
			config: []byte(`true`),
			isPR:   true,
			assert: func(t *testing.T, calls int, err error) {
				require.NoError(t, err)
				require.Equal(t, 1, calls)
			},
		},
		{
			name:       "ConvertToDraft error propagates",
			config:     []byte(`true`),
			isPR:       true,
			convertErr: errors.New("boom"),
			assert: func(t *testing.T, calls int, err error) {
				require.ErrorContains(t, err, "error converting PR to draft")
				require.Equal(t, 1, calls)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var calls int
			fake := &fakePullRequestsClient{
				ConvertToDraftFn: func(
					_ context.Context,
					_, _ string,
					_ int,
				) error {
					calls++
					return testCase.convertErr
				},
			}
			err := convertToDraftRunner{}.run(
				t.Context(),
				&actionContext{
					repoContext: repoContext{
						prsClient: fake,
						owner:     "akuity",
						repo:      "kargo",
					},
					number: 1,
					isPR:   testCase.isPR,
				},
				testCase.config,
			)
			testCase.assert(t, calls, err)
		})
	}
}
