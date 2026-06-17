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
		name         string
		config       []byte
		addLabelsErr error
		assert       func(t *testing.T, sentLabels []string, err error)
	}{
		{
			name:   "decode error — config is not a list of strings",
			config: []byte(`true`),
			assert: func(t *testing.T, sentLabels []string, err error) {
				require.ErrorContains(t, err, "decoding addLabels config")
				require.Nil(t, sentLabels)
			},
		},
		{
			name:   "empty list — no API call",
			config: []byte(`[]`),
			assert: func(t *testing.T, sentLabels []string, err error) {
				require.NoError(t, err)
				require.Nil(t, sentLabels)
			},
		},
		{
			name:   "happy path — labels passed through",
			config: []byte("- foo\n- bar\n"),
			assert: func(t *testing.T, sentLabels []string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"foo", "bar"}, sentLabels)
			},
		},
		{
			name:         "API error propagates with wrapping",
			config:       []byte("- foo\n"),
			addLabelsErr: errors.New("upstream boom"),
			assert: func(t *testing.T, sentLabels []string, err error) {
				require.ErrorContains(t, err, "error adding labels: upstream boom")
				require.Equal(t, []string{"foo"}, sentLabels)
			},
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
					repoContext: repoContext{
						issuesClient: fake,
						owner:        "akuity",
						repo:         "kargo",
					},
					number: 1,
				},
				testCase.config,
			)
			testCase.assert(t, sentLabels, err)
		})
	}
}
