package governance

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-github/v76/github"
	"github.com/stretchr/testify/require"
)

func Test_applyPRPolicyRunner_run(t *testing.T) {
	testCases := []struct {
		name     string
		config   []byte
		isPR     bool
		cfg      config
		prGetErr error
		prBody   string
		assert   func(
			t *testing.T,
			getCalls int,
			labelsAdded map[string]struct{},
			err error,
		)
	}{
		{
			name:   "decode error — config is not a bool",
			config: []byte(`"yes"`),
			isPR:   true,
			assert: func(t *testing.T, getCalls int, labelsAdded map[string]struct{}, err error) {
				require.ErrorContains(t, err, "decoding applyPRPolicy config")
				require.Zero(t, getCalls)
				require.Empty(t, labelsAdded)
			},
		},
		{
			name:   "false on PR — no-op",
			config: []byte(`false`),
			isPR:   true,
			assert: func(t *testing.T, getCalls int, labelsAdded map[string]struct{}, err error) {
				require.NoError(t, err)
				require.Zero(t, getCalls)
				require.Empty(t, labelsAdded)
			},
		},
		{
			name:   "true on issue — silent no-op (PR-only action)",
			config: []byte(`true`),
			isPR:   false,
			assert: func(t *testing.T, getCalls int, labelsAdded map[string]struct{}, err error) {
				require.NoError(t, err)
				require.Zero(t, getCalls)
				require.Empty(t, labelsAdded)
			},
		},
		{
			name:     "PR fetch error propagates",
			config:   []byte(`true`),
			isPR:     true,
			prGetErr: errors.New("network"),
			assert: func(t *testing.T, getCalls int, labelsAdded map[string]struct{}, err error) {
				require.ErrorContains(t, err, "error fetching PR for policy check")
				require.Equal(t, 1, getCalls)
				require.Empty(t, labelsAdded)
			},
		},
		{
			name:   "happy path — delegates to applyPRPolicy and OnPass fires",
			config: []byte(`true`),
			isPR:   true,
			prBody: "no linked issue here.",
			cfg: config{
				PullRequests: &pullRequestsConfig{
					OnPass: &onPassConfig{
						Actions: []action{
							mustAction("addLabels", []string{"policy/passed"}),
						},
					},
				},
			},
			assert: func(t *testing.T, getCalls int, labelsAdded map[string]struct{}, err error) {
				require.NoError(t, err)
				require.Equal(t, 1, getCalls)
				require.Equal(t, map[string]struct{}{"policy/passed": {}}, labelsAdded)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var getCalls int
			labelsAdded := map[string]struct{}{}
			issuesFake := &fakeIssuesClient{
				AddLabelsToIssueFn: func(
					_ context.Context,
					_, _ string,
					_ int,
					labels []string,
				) ([]*github.Label, *github.Response, error) {
					for _, l := range labels {
						labelsAdded[l] = struct{}{}
					}
					return nil, nil, nil
				},
			}
			prsFake := &fakePullRequestsClient{
				GetFn: func(
					_ context.Context,
					_, _ string,
					_ int,
				) (*github.PullRequest, *github.Response, error) {
					getCalls++
					if testCase.prGetErr != nil {
						return nil, nil, testCase.prGetErr
					}
					return &github.PullRequest{
						Number: github.Ptr(1),
						Body:   github.Ptr(testCase.prBody),
						User:   &github.User{Login: github.Ptr("someone")},
					}, nil, nil
				},
			}
			err := applyPRPolicyRunner{}.run(
				t.Context(),
				&actionContext{
					repoContext: repoContext{
						cfg:          testCase.cfg,
						issuesClient: issuesFake,
						prsClient:    prsFake,
						owner:        "akuity",
						repo:         "kargo",
					},
					number: 1,
					isPR:   testCase.isPR,
				},
				testCase.config,
			)
			testCase.assert(t, getCalls, labelsAdded, err)
		})
	}
}
