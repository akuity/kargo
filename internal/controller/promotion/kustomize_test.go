package promotion

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
)

func TestNewKustomizeMechanism(t *testing.T) {
	pm := newKustomizeMechanism(
		fake.NewFakeClient(),
		&credentials.FakeDB{},
	)
	kpm, ok := pm.(*gitMechanism)
	require.True(t, ok)
	require.Equal(t, "Kustomize promotion mechanism", kpm.name)
	require.NotNil(t, kpm.client)
	require.NotNil(t, kpm.selectUpdatesFn)
	require.NotNil(t, kpm.applyConfigManagementFn)
}

func TestSelectKustomizeUpdates(t *testing.T) {
	testCases := []struct {
		name       string
		updates    []kargoapi.GitRepoUpdate
		assertions func(*testing.T, []*kargoapi.GitRepoUpdate)
	}{
		{
			name: "no updates",
			assertions: func(t *testing.T, selectedUpdates []*kargoapi.GitRepoUpdate) {
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
			assertions: func(t *testing.T, selectedUpdates []*kargoapi.GitRepoUpdate) {
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
			assertions: func(t *testing.T, selectedUpdates []*kargoapi.GitRepoUpdate) {
				require.Len(t, selectedUpdates, 1)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(t, selectKustomizeUpdates(testCase.updates))
		})
	}
}

func TestKustomizerApply(t *testing.T) {
	testCases := []struct {
		name       string
		update     kargoapi.GitRepoUpdate
		kustomizer *kustomizer
		assertions func(t *testing.T, changes []string, err error)
	}{
		{
			name: "error finding image from Freight",
			update: kargoapi.GitRepoUpdate{
				Kustomize: &kargoapi.KustomizePromotionMechanism{
					Images: []kargoapi.KustomizeImageUpdate{
						{Image: "fake-image"},
					},
				},
			},
			kustomizer: &kustomizer{
				findImageFn: func(
					context.Context,
					client.Client,
					string,
					[]kargoapi.FreightRequest,
					*kargoapi.FreightOrigin,
					[]kargoapi.FreightReference,
					string,
				) (*kargoapi.Image, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []string, err error) {
				require.ErrorContains(t, err, "error finding image")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error running kustomize edit set image",
			update: kargoapi.GitRepoUpdate{
				Kustomize: &kargoapi.KustomizePromotionMechanism{
					Images: []kargoapi.KustomizeImageUpdate{
						{Image: "fake-image"},
					},
				},
			},
			kustomizer: &kustomizer{
				findImageFn: func(
					context.Context,
					client.Client,
					string,
					[]kargoapi.FreightRequest,
					*kargoapi.FreightOrigin,
					[]kargoapi.FreightReference,
					string,
				) (*kargoapi.Image, error) {
					return &kargoapi.Image{}, nil
				},
				setImageFn: func(string, string) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []string, err error) {
				require.ErrorContains(t, err, "error updating image")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success using tag",
			update: kargoapi.GitRepoUpdate{
				Kustomize: &kargoapi.KustomizePromotionMechanism{
					Images: []kargoapi.KustomizeImageUpdate{
						{
							Image: "fake-image",
							Path:  "fake-path",
						},
					},
				},
			},
			kustomizer: &kustomizer{
				findImageFn: func(
					context.Context,
					client.Client,
					string,
					[]kargoapi.FreightRequest,
					*kargoapi.FreightOrigin,
					[]kargoapi.FreightReference,
					string,
				) (*kargoapi.Image, error) {
					return &kargoapi.Image{
						RepoURL: "fake-image",
						Tag:     "fake-tag",
					}, nil
				},
				setImageFn: func(string, string) error {
					return nil
				},
			},
			assertions: func(t *testing.T, changes []string, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]string{
						"updated fake-path/kustomization.yaml to use image fake-image:fake-tag",
					},
					changes,
				)
			},
		},
		{
			name: "success using digest",
			update: kargoapi.GitRepoUpdate{
				Kustomize: &kargoapi.KustomizePromotionMechanism{
					Images: []kargoapi.KustomizeImageUpdate{
						{
							Image:     "fake-image",
							Path:      "fake-path",
							UseDigest: true,
						},
					},
				},
			},
			kustomizer: &kustomizer{
				findImageFn: func(
					context.Context,
					client.Client,
					string,
					[]kargoapi.FreightRequest,
					*kargoapi.FreightOrigin,
					[]kargoapi.FreightReference,
					string,
				) (*kargoapi.Image, error) {
					return &kargoapi.Image{
						RepoURL: "fake-image",
						Digest:  "fake-digest",
					}, nil
				},
				setImageFn: func(string, string) error {
					return nil
				},
			},
			assertions: func(t *testing.T, changes []string, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]string{
						"updated fake-path/kustomization.yaml to use image fake-image@fake-digest",
					},
					changes,
				)
			},
		},
	}
	for _, testCase := range testCases {
		stage := &kargoapi.Stage{
			Spec: kargoapi.StageSpec{
				PromotionMechanisms: &kargoapi.PromotionMechanisms{
					GitRepoUpdates: []kargoapi.GitRepoUpdate{testCase.update},
				},
			},
		}
		t.Run(testCase.name, func(t *testing.T) {
			changes, err := testCase.kustomizer.apply(
				context.Background(),
				stage,
				&stage.Spec.PromotionMechanisms.GitRepoUpdates[0],
				[]kargoapi.FreightReference{{
					Images: []kargoapi.Image{
						{
							RepoURL: "fake-image",
							Tag:     "fake-tag",
							Digest:  "fake-digest",
						},
					},
				}},
				"",
				"",
				"",
				git.RepoCredentials{},
			)
			testCase.assertions(t, changes, err)
		})
	}
}
