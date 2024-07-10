package promotion

import (
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
)

func TestNewGenericGitMechanism(t *testing.T) {
	pm := newGenericGitMechanism(
		fake.NewFakeClient(),
		&credentials.FakeDB{},
	)
	ggpm, ok := pm.(*gitMechanism)
	require.True(t, ok)
	require.Equal(t, "generic Git promotion mechanism", ggpm.name)
	require.NotNil(t, ggpm.client)
	require.NotNil(t, ggpm.selectUpdatesFn)
	require.Nil(t, ggpm.applyConfigManagementFn)
}

func TestSelectGenericGitUpdates(t *testing.T) {
	testCases := []struct {
		name       string
		updates    []kargoapi.GitRepoUpdate
		assertions func(*testing.T, []kargoapi.GitRepoUpdate)
	}{
		{
			name: "no updates",
			assertions: func(t *testing.T, selectedUpdates []kargoapi.GitRepoUpdate) {
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
			assertions: func(t *testing.T, selectedUpdates []kargoapi.GitRepoUpdate) {
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
			assertions: func(t *testing.T, selectedUpdates []kargoapi.GitRepoUpdate) {
				require.Len(t, selectedUpdates, 1)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(t, selectGenericGitUpdates(testCase.updates))
		})
	}
}
