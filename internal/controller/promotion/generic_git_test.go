package promotion

import (
	"testing"

	"github.com/stretchr/testify/require"

	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
)

func TestNewGenericGitMechanism(t *testing.T) {
	pm := newGenericGitMechanism(&credentials.FakeDB{})
	ggpm, ok := pm.(*gitMechanism)
	require.True(t, ok)
	require.NotNil(t, ggpm.selectUpdatesFn)
	require.Nil(t, ggpm.applyConfigManagementFn)
}

func TestSelectGenericGitUpdates(t *testing.T) {
	testCases := []struct {
		name       string
		updates    []api.GitRepoUpdate
		assertions func(selectedUpdates []api.GitRepoUpdate)
	}{
		{
			name: "no updates",
			assertions: func(selectedUpdates []api.GitRepoUpdate) {
				require.Empty(t, selectedUpdates)
			},
		},
		{
			name: "no generic git updates",
			updates: []api.GitRepoUpdate{
				{
					RepoURL:   "fake-url",
					Kustomize: &api.KustomizePromotionMechanism{},
				},
			},
			assertions: func(selectedUpdates []api.GitRepoUpdate) {
				require.Empty(t, selectedUpdates)
			},
		},
		{
			name: "some generic git updates",
			updates: []api.GitRepoUpdate{
				{
					RepoURL:   "fake-url",
					Kustomize: &api.KustomizePromotionMechanism{},
				},
				{
					RepoURL: "fake-url",
					Helm:    &api.HelmPromotionMechanism{},
				},
				{
					RepoURL: "fake-url",
				},
			},
			assertions: func(selectedUpdates []api.GitRepoUpdate) {
				require.Len(t, selectedUpdates, 1)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(selectGenericGitUpdates(testCase.updates))
		})
	}
}
