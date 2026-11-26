package governance

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-github/v76/github"
	"github.com/stretchr/testify/require"
)

func Test_issueHandler_handleOpened(t *testing.T) {
	// The handler's only job is to extract the issue's existing labels
	// from the event and call enforceRequiredLabels with the configured
	// prefixes. Branch-level coverage of enforceRequiredLabels itself
	// (all-present, partial, all-missing, errors) lives in
	// Test_repoContext_enforceRequiredLabels.
	testCases := []struct {
		name              string
		cfg               issuesConfig
		initialLabels     []string
		expectLabelsAdded map[string]struct{}
	}{
		{
			// Happy path: handler reads existing labels from the issue,
			// computes which required prefixes are missing, and asks
			// enforceRequiredLabels to add the corresponding needs/* labels.
			name: "extracts existing labels and enforces missing prefixes",
			cfg: issuesConfig{
				RequiredLabelPrefixes: []string{"kind", "priority", "area"},
			},
			initialLabels: []string{"kind/bug"},
			expectLabelsAdded: map[string]struct{}{
				"needs/priority": {},
				"needs/area":     {},
			},
		},
		{
			// No-op short-circuit at the handler level: when the config
			// has no required prefixes, the handler returns early without
			// invoking enforceRequiredLabels at all.
			name:              "no required prefixes configured: no-op",
			cfg:               issuesConfig{},
			initialLabels:     []string{},
			expectLabelsAdded: map[string]struct{}{},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			labelsAdded := map[string]struct{}{}

			issuesClient := &fakeIssuesClient{
				AddLabelsToIssueFn: func(
					_ context.Context,
					_, _ string,
					_ int,
					labels []string,
				) ([]*github.Label, *github.Response, error) {
					for _, label := range labels {
						labelsAdded[label] = struct{}{}
					}
					return nil, nil, nil
				},
			}

			initialLabels := make([]*github.Label, len(testCase.initialLabels))
			for i, l := range testCase.initialLabels {
				initialLabels[i] = &github.Label{Name: github.Ptr(l)}
			}
			event := &github.IssuesEvent{
				Action: github.Ptr("opened"),
				Issue: &github.Issue{
					Number: github.Ptr(42),
					Labels: initialLabels,
				},
				Repo: &github.Repository{
					Name:  github.Ptr("kargo"),
					Owner: &github.User{Login: github.Ptr("akuity")},
				},
				Installation: &github.Installation{ID: github.Ptr(int64(1))},
			}

			h := &issueHandler{
				repoContext: repoContext{
					cfg:          config{Issues: &testCase.cfg},
					owner:        "akuity",
					repo:         "kargo",
					issuesClient: issuesClient,
				},
			}
			err := h.handleOpened(t.Context(), event)
			require.NoError(t, err)
			require.Equal(t, testCase.expectLabelsAdded, labelsAdded)
		})
	}
}

func Test_issueHandler_handleOpened_onOpened(t *testing.T) {
	onOpened := &onOpenedConfig{
		Actions: []action{mustAction("comment", "policy")},
	}
	testCases := []struct {
		name         string
		cfg          issuesConfig
		addLabelsErr error
		assert       func(
			t *testing.T,
			labels []string,
			comments []string,
			err error,
		)
	}{
		{
			name: "runs onOpened actions after enforcing required labels",
			cfg: issuesConfig{
				RequiredLabelPrefixes: []string{"kind"},
				OnOpened:              onOpened,
			},
			assert: func(
				t *testing.T,
				labels []string,
				comments []string,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, []string{"needs/kind"}, labels)
				require.Equal(t, []string{"policy"}, comments)
			},
		},
		{
			// Without required prefixes, the handler must not return
			// before reaching the onOpened actions.
			name: "runs onOpened actions with no required prefixes",
			cfg:  issuesConfig{OnOpened: onOpened},
			assert: func(
				t *testing.T,
				labels []string,
				comments []string,
				err error,
			) {
				require.NoError(t, err)
				require.Empty(t, labels)
				require.Equal(t, []string{"policy"}, comments)
			},
		},
		{
			name: "required-label failure does not skip onOpened actions",
			cfg: issuesConfig{
				RequiredLabelPrefixes: []string{"kind"},
				OnOpened:              onOpened,
			},
			addLabelsErr: errors.New("labels boom"),
			assert: func(
				t *testing.T,
				_ []string,
				comments []string,
				err error,
			) {
				require.ErrorContains(t, err, "labels boom")
				require.Equal(t, []string{"policy"}, comments)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var labels, comments []string
			issuesClient := &fakeIssuesClient{
				AddLabelsToIssueFn: func(
					_ context.Context,
					_, _ string,
					_ int,
					added []string,
				) ([]*github.Label, *github.Response, error) {
					labels = append(labels, added...)
					return nil, nil, testCase.addLabelsErr
				},
				CreateCommentFn: func(
					_ context.Context,
					_, _ string,
					_ int,
					comment *github.IssueComment,
				) (*github.IssueComment, *github.Response, error) {
					comments = append(comments, comment.GetBody())
					return comment, nil, nil
				},
			}
			event := &github.IssuesEvent{
				Action: github.Ptr(issueActionOpened),
				Issue:  &github.Issue{Number: github.Ptr(42)},
			}
			h := &issueHandler{
				repoContext: repoContext{
					cfg:          config{Issues: &testCase.cfg},
					owner:        "akuity",
					repo:         "kargo",
					issuesClient: issuesClient,
				},
			}
			err := h.handleOpened(t.Context(), event)
			testCase.assert(t, labels, comments, err)
		})
	}
}
