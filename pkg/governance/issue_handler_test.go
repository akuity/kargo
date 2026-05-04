package governance

import (
	"context"
	"testing"

	"github.com/google/go-github/v76/github"
	"github.com/stretchr/testify/require"
)

func Test_issueHandler_handleOpened(t *testing.T) {
	testCases := []struct {
		name                string
		cfg                 issuesConfig
		initialLabels       []string
		expectedLabelsAdded map[string]struct{}
	}{
		{
			name: "all required labels present",
			cfg: issuesConfig{
				RequiredLabelPrefixes: []string{"kind", "priority"},
			},
			initialLabels:       []string{"kind/bug", "priority/high"},
			expectedLabelsAdded: map[string]struct{}{},
		},
		{
			name: "missing kind label",
			cfg: issuesConfig{
				RequiredLabelPrefixes: []string{"kind", "priority"},
			},
			initialLabels:       []string{"priority/high"},
			expectedLabelsAdded: map[string]struct{}{"needs/kind": {}},
		},
		{
			name: "missing all required labels",
			cfg: issuesConfig{
				RequiredLabelPrefixes: []string{"kind", "priority", "area"},
			},
			initialLabels: []string{},
			expectedLabelsAdded: map[string]struct{}{
				"needs/kind":     {},
				"needs/priority": {},
				"needs/area":     {},
			},
		},
		{
			name:                "no label governance configured",
			cfg:                 issuesConfig{},
			initialLabels:       []string{},
			expectedLabelsAdded: map[string]struct{}{},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Our fake client will close over this map and update it with any labels
			// that get added.
			labelsAdded := map[string]struct{}{}

			issuesClient := &fakeIssuesClient{
				AddLabelsToIssueFn: func(
					_ context.Context,
					_ string,
					_ string,
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
				cfg:          testCase.cfg,
				owner:        "akuity",
				repo:         "kargo",
				issuesClient: issuesClient,
			}
			err := h.handleOpened(t.Context(), event)
			require.NoError(t, err)
			require.Equal(t, testCase.expectedLabelsAdded, labelsAdded)
		})
	}
}
