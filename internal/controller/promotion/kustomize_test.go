package promotion

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
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
			name: "no kustomize updates",
			updates: []kargoapi.GitRepoUpdate{
				{
					RepoURL: "fake-url",
				},
			},
			assertions: func(selectedUpdates []kargoapi.GitRepoUpdate) {
				require.Empty(t, selectedUpdates)
			},
		},
		{
			name: "some kustomize updates",
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
					kargoapi.GitRepoUpdate{
						Kustomize: &kargoapi.KustomizePromotionMechanism{
							Images: []kargoapi.KustomizeImageUpdate{
								{
									Image: testImage,
									Path:  "fake-path",
								},
							},
						},
					},
					kargoapi.Freight{
						Images: []kargoapi.Image{
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
