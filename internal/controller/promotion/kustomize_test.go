package promotion

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
)

func TestNewKustomizeMechanism(t *testing.T) {
	pm := newKustomizeMechanism(&credentials.FakeDB{})
	kpm, ok := pm.(*gitMechanism)
	require.True(t, ok)
	require.NotNil(t, kpm.selectUpdatesFn)
	require.NotNil(t, kpm.applyConfigManagementFn)
}

func TestSelectKustomizeUpdates(t *testing.T) {
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
			name: "no kustomize updates",
			updates: []api.GitRepoUpdate{
				{
					RepoURL: "fake-url",
				},
			},
			assertions: func(selectedUpdates []api.GitRepoUpdate) {
				require.Empty(t, selectedUpdates)
			},
		},
		{
			name: "some kustomize updates",
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
			testCase.assertions(selectKustomizeUpdates(testCase.updates))
		})
	}
}

func TestKustomizerApply(t *testing.T) {
	const (
		testImage = "fake-image"
		testTag   = "fake-tag"
	)
	testCases := []struct {
		name       string
		setImageFn func(dir, image, tag string) error
		assertions func(changes []string, err error)
	}{
		{
			name: "error running kustomize edit set image",
			setImageFn: func(string, string, string) error {
				return errors.New("something went wrong")
			},
			assertions: func(_ []string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error updating image")
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "success",
			setImageFn: func(string, string, string) error {
				return nil
			},
			assertions: func(changes []string, err error) {
				require.NoError(t, err)
				require.Len(t, changes, 1)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				(&kustomizer{
					setImageFn: testCase.setImageFn,
				}).apply(
					api.GitRepoUpdate{
						Kustomize: &api.KustomizePromotionMechanism{
							Images: []api.KustomizeImageUpdate{
								{
									Image: testImage,
									Path:  "fake-path",
								},
							},
						},
					},
					api.Freight{
						Images: []api.Image{
							{
								RepoURL: testImage,
								Tag:     testTag,
							},
						},
					},
					"",
					"",
				),
			)
		})
	}
}
