package governance

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-github/v76/github"
	"github.com/stretchr/testify/require"
)

func Test_closeRunner_run(t *testing.T) {
	testCases := []struct {
		name          string
		config        []byte
		isPR          bool
		issuesEditErr error
		prsEditErr    error
		assert        func(
			t *testing.T,
			issueEditCalls, prEditCalls int,
			issueState, prState, stateReason string,
			err error,
		)
	}{
		{
			name:   "decode error — config is not a bool",
			config: []byte(`"close"`),
			assert: func(t *testing.T, issueEditCalls, prEditCalls int, _, _, _ string, err error) {
				require.ErrorContains(t, err, "decoding close config")
				require.Zero(t, issueEditCalls)
				require.Zero(t, prEditCalls)
			},
		},
		{
			name:   "false — no-op",
			config: []byte(`false`),
			assert: func(t *testing.T, issueEditCalls, prEditCalls int, _, _, _ string, err error) {
				require.NoError(t, err)
				require.Zero(t, issueEditCalls)
				require.Zero(t, prEditCalls)
			},
		},
		{
			name:   "true on issue — issuesClient.Edit called with state_reason",
			config: []byte(`true`),
			isPR:   false,
			assert: func(t *testing.T, issueEditCalls, prEditCalls int, issueState, _, stateReason string, err error) {
				require.NoError(t, err)
				require.Equal(t, 1, issueEditCalls)
				require.Zero(t, prEditCalls)
				require.Equal(t, issueStateClosed, issueState)
				require.Equal(t, stateReasonNotPlanned, stateReason)
			},
		},
		{
			name:   "true on PR — prsClient.Edit called",
			config: []byte(`true`),
			isPR:   true,
			assert: func(t *testing.T, issueEditCalls, prEditCalls int, _, prState, _ string, err error) {
				require.NoError(t, err)
				require.Zero(t, issueEditCalls)
				require.Equal(t, 1, prEditCalls)
				require.Equal(t, prStateClosed, prState)
			},
		},
		{
			name:       "PR edit error propagates",
			config:     []byte(`true`),
			isPR:       true,
			prsEditErr: errors.New("boom"),
			assert: func(t *testing.T, issueEditCalls, prEditCalls int, _, prState, _ string, err error) {
				require.ErrorContains(t, err, "error closing PR")
				require.Zero(t, issueEditCalls)
				require.Equal(t, 1, prEditCalls)
				require.Equal(t, prStateClosed, prState)
			},
		},
		{
			name:          "issue edit error propagates",
			config:        []byte(`true`),
			isPR:          false,
			issuesEditErr: errors.New("boom"),
			assert: func(t *testing.T, issueEditCalls, prEditCalls int, issueState, _, stateReason string, err error) {
				require.ErrorContains(t, err, "error closing issue")
				require.Equal(t, 1, issueEditCalls)
				require.Zero(t, prEditCalls)
				require.Equal(t, issueStateClosed, issueState)
				require.Equal(t, stateReasonNotPlanned, stateReason)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var (
				issueEditCalls int
				prEditCalls    int
				issueState     string
				prState        string
				stateReason    string
			)
			issuesFake := &fakeIssuesClient{
				EditFn: func(
					_ context.Context,
					_, _ string,
					_ int,
					req *github.IssueRequest,
				) (*github.Issue, *github.Response, error) {
					issueEditCalls++
					issueState = req.GetState()
					stateReason = req.GetStateReason()
					return nil, nil, testCase.issuesEditErr
				},
			}
			prsFake := &fakePullRequestsClient{
				EditFn: func(
					_ context.Context,
					_, _ string,
					_ int,
					pr *github.PullRequest,
				) (*github.PullRequest, *github.Response, error) {
					prEditCalls++
					prState = pr.GetState()
					return pr, nil, testCase.prsEditErr
				},
			}
			err := closeRunner{}.run(
				t.Context(),
				&actionContext{
					repoContext: repoContext{
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
			testCase.assert(
				t, issueEditCalls, prEditCalls,
				issueState, prState, stateReason, err,
			)
		})
	}
}
