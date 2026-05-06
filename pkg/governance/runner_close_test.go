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
		name                  string
		config                []byte
		isPR                  bool
		issuesEditErr         error
		prsEditErr            error
		expectIssuesEditCalls int
		expectPRsEditCalls    int
		expectIssueState      string
		expectPRState         string
		expectStateReason     string
		expectErrContains     string
	}{
		{
			name:              "decode error — config is not a bool",
			config:            []byte(`"close"`),
			expectErrContains: "decoding close config",
		},
		{
			name:   "false — no-op",
			config: []byte(`false`),
		},
		{
			name:                  "true on issue — issuesClient.Edit called with state_reason",
			config:                []byte(`true`),
			isPR:                  false,
			expectIssuesEditCalls: 1,
			expectIssueState:      issueStateClosed,
			expectStateReason:     stateReasonNotPlanned,
		},
		{
			name:               "true on PR — prsClient.Edit called",
			config:             []byte(`true`),
			isPR:               true,
			expectPRsEditCalls: 1,
			expectPRState:      prStateClosed,
		},
		{
			name:               "PR edit error propagates",
			config:             []byte(`true`),
			isPR:               true,
			prsEditErr:         errors.New("boom"),
			expectPRsEditCalls: 1,
			expectPRState:      prStateClosed,
			expectErrContains:  "error closing PR",
		},
		{
			name:                  "issue edit error propagates",
			config:                []byte(`true`),
			isPR:                  false,
			issuesEditErr:         errors.New("boom"),
			expectIssuesEditCalls: 1,
			expectIssueState:      issueStateClosed,
			expectStateReason:     stateReasonNotPlanned,
			expectErrContains:     "error closing issue",
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
					issuesClient: issuesFake,
					prsClient:    prsFake,
					owner:        "akuity",
					repo:         "kargo",
					number:       1,
					isPR:         testCase.isPR,
				},
				testCase.config,
			)
			if testCase.expectErrContains != "" {
				require.ErrorContains(t, err, testCase.expectErrContains)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, testCase.expectIssuesEditCalls, issueEditCalls)
			require.Equal(t, testCase.expectPRsEditCalls, prEditCalls)
			require.Equal(t, testCase.expectIssueState, issueState)
			require.Equal(t, testCase.expectPRState, prState)
			require.Equal(t, testCase.expectStateReason, stateReason)
		})
	}
}
