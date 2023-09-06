package promotion

import (
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
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
		updates    []kargoapi.GitRepoUpdate
		assertions func(selectedUpdates []kargoapi.GitRepoUpdate)
	}{
		{
			name: "no updates",
			assertions: func(selectedUpdates []kargoapi.GitRepoUpdate) {
				require.Empty(t, selectedUpdates)
			},
		},
		{
			name: "no generic git updates",
			updates: []kargoapi.GitRepoUpdate{
				{
					RepoURL:   "fake-url",
					Kustomize: &kargoapi.KustomizePromotionMechanism{},
				},
			},
			assertions: func(selectedUpdates []kargoapi.GitRepoUpdate) {
				require.Empty(t, selectedUpdates)
			},
		},
		{
			name: "some generic git updates",
			updates: []kargoapi.GitRepoUpdate{
				{
					RepoURL:   "fake-url",
					Kustomize: &kargoapi.KustomizePromotionMechanism{},
				},
				{
					RepoURL: "fake-url",
					Helm:    &kargoapi.HelmPromotionMechanism{},
				},
				{
					RepoURL: "fake-url",
				},
			},
			assertions: func(selectedUpdates []kargoapi.GitRepoUpdate) {
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
