package governance

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_convertToDraftRunner_run(t *testing.T) {
	testCases := []struct {
		name              string
		config            []byte
		isPR              bool
		convertErr        error
		expectCalls       int
		expectErrContains string
	}{
		{
			name:              "decode error — config is not a bool",
			config:            []byte(`"yes"`),
			isPR:              true,
			expectErrContains: "decoding convertToDraft config",
		},
		{
			name:        "false on PR — no-op",
			config:      []byte(`false`),
			isPR:        true,
			expectCalls: 0,
		},
		{
			name:        "true on issue — silent no-op (PR-only action)",
			config:      []byte(`true`),
			isPR:        false,
			expectCalls: 0,
		},
		{
			name:        "true on PR — ConvertToDraft called",
			config:      []byte(`true`),
			isPR:        true,
			expectCalls: 1,
		},
		{
			name:              "ConvertToDraft error propagates",
			config:            []byte(`true`),
			isPR:              true,
			convertErr:        errors.New("boom"),
			expectCalls:       1,
			expectErrContains: "error converting PR to draft",
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
					prsClient: fake,
					owner:     "akuity",
					repo:      "kargo",
					number:    1,
					isPR:      testCase.isPR,
				},
				testCase.config,
			)
			if testCase.expectErrContains != "" {
				require.ErrorContains(t, err, testCase.expectErrContains)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, testCase.expectCalls, calls)
		})
	}
}
