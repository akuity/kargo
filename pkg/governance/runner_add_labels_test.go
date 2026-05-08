package governance

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-github/v76/github"
	"github.com/stretchr/testify/require"
)

func Test_addLabelsRunner_run(t *testing.T) {
	testCases := []struct {
		name              string
		config            []byte
		addLabelsErr      error
		expectedAdded     []string
		expectErrContains string
	}{
		{
			name:              "decode error — config is not a list of strings",
			config:            []byte(`true`),
			expectErrContains: "decoding addLabels config",
		},
		{
			name:          "empty list — no API call",
			config:        []byte(`[]`),
			expectedAdded: nil,
		},
		{
			name:          "happy path — labels passed through",
			config:        []byte("- foo\n- bar\n"),
			expectedAdded: []string{"foo", "bar"},
		},
		{
			name:              "API error propagates with wrapping",
			config:            []byte("- foo\n"),
			addLabelsErr:      errors.New("upstream boom"),
			expectedAdded:     []string{"foo"},
			expectErrContains: "error adding labels: upstream boom",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var sentLabels []string
			fake := &fakeIssuesClient{
				AddLabelsToIssueFn: func(
					_ context.Context,
					_, _ string,
					_ int,
					labels []string,
				) ([]*github.Label, *github.Response, error) {
					sentLabels = labels
					return nil, nil, testCase.addLabelsErr
				},
			}
			err := addLabelsRunner{}.run(
				t.Context(),
				&actionContext{
					issuesClient: fake,
					owner:        "akuity",
					repo:         "kargo",
					number:       1,
				},
				testCase.config,
			)
			if testCase.expectErrContains != "" {
				require.ErrorContains(t, err, testCase.expectErrContains)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, testCase.expectedAdded, sentLabels)
		})
	}
}
