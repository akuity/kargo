package controller

import (
	"context"
	"errors"
	"testing"

	"github.com/argoproj-labs/argocd-image-updater/pkg/image"
	"github.com/argoproj-labs/argocd-image-updater/pkg/registry"
	"github.com/argoproj-labs/argocd-image-updater/pkg/tag"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	api "github.com/akuityio/kargo/api/v1alpha1"
)

func TestGetLatestImages(t *testing.T) {
	testCases := []struct {
		name        string
		spec        api.EnvironmentSpec
		repoCredsFn func(
			context.Context,
			string,
			api.ImageSubscription,
			*registry.RegistryEndpoint,
		) (image.Credential, error)
		tagsFn func(
			*registry.RegistryEndpoint,
			*image.ContainerImage,
			registry.RegistryClient,
			*image.VersionConstraint,
		) (*tag.ImageTagList, error)
		newestTagFn func(
			*image.ContainerImage,
			*image.VersionConstraint,
			*tag.ImageTagList,
		) (*tag.ImageTag, error)
		assertions func([]api.Image, error)
	}{
		{
			name: "spec has no subscriptions",
			assertions: func(images []api.Image, err error) {
				require.NoError(t, err)
				require.Nil(t, images)
			},
		},
		{
			name: "spec has no upstream repo subscriptions",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{},
			},
			assertions: func(images []api.Image, err error) {
				require.NoError(t, err)
				require.Nil(t, images)
			},
		},
		{
			name: "spec has no image repo subscriptions",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
			},
			assertions: func(images []api.Image, err error) {
				require.NoError(t, err)
				require.Nil(t, images)
			},
		},
		{
			name: "error parsing image platform",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{
						Images: []api.ImageSubscription{
							{
								RepoURL:  "fake-url",
								Platform: "bogus", // This will force an error
							},
						},
					},
				},
			},
			assertions: func(images []api.Image, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error parsing platform")
				require.Nil(t, images)
			},
		},
		{
			name: "error getting image repo credentials",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{
						Images: []api.ImageSubscription{
							{
								RepoURL: "fake-url",
							},
						},
					},
				},
			},
			repoCredsFn: func(
				context.Context,
				string,
				api.ImageSubscription,
				*registry.RegistryEndpoint,
			) (image.Credential, error) {
				return image.Credential{}, errors.New("something went wrong")
			},
			assertions: func(images []api.Image, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error getting credentials for image")
				require.Contains(t, err.Error(), "something went wrong")
				require.Nil(t, images)
			},
		},
		{
			name: "error fetching tags",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{
						Images: []api.ImageSubscription{
							{
								RepoURL: "fake-url",
							},
						},
					},
				},
			},
			repoCredsFn: func(
				context.Context,
				string,
				api.ImageSubscription,
				*registry.RegistryEndpoint,
			) (image.Credential, error) {
				return image.Credential{}, nil
			},
			tagsFn: func(
				*registry.RegistryEndpoint,
				*image.ContainerImage,
				registry.RegistryClient,
				*image.VersionConstraint,
			) (*tag.ImageTagList, error) {
				return nil, errors.New("something went wrong")
			},
			assertions: func(images []api.Image, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error fetching tags for image")
				require.Contains(t, err.Error(), "something went wrong")
				require.Nil(t, images)
			},
		},
		{
			name: "error finding newest tag",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{
						Images: []api.ImageSubscription{
							{
								RepoURL: "fake-url",
							},
						},
					},
				},
			},
			repoCredsFn: func(
				context.Context,
				string,
				api.ImageSubscription,
				*registry.RegistryEndpoint,
			) (image.Credential, error) {
				return image.Credential{}, nil
			},
			tagsFn: func(
				*registry.RegistryEndpoint,
				*image.ContainerImage,
				registry.RegistryClient,
				*image.VersionConstraint,
			) (*tag.ImageTagList, error) {
				return tag.NewImageTagList(), nil
			},
			newestTagFn: func(
				*image.ContainerImage,
				*image.VersionConstraint,
				*tag.ImageTagList,
			) (*tag.ImageTag, error) {
				return nil, errors.New("something went wrong")
			},
			assertions: func(images []api.Image, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error finding newest tag")
				require.Contains(t, err.Error(), "something went wrong")
				require.Nil(t, images)
			},
		},
		{
			name: "no suitable image version found",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{
						Images: []api.ImageSubscription{
							{
								RepoURL: "fake-url",
							},
						},
					},
				},
			},
			repoCredsFn: func(
				context.Context,
				string,
				api.ImageSubscription,
				*registry.RegistryEndpoint,
			) (image.Credential, error) {
				return image.Credential{}, nil
			},
			tagsFn: func(
				*registry.RegistryEndpoint,
				*image.ContainerImage,
				registry.RegistryClient,
				*image.VersionConstraint,
			) (*tag.ImageTagList, error) {
				return tag.NewImageTagList(), nil
			},
			newestTagFn: func(
				*image.ContainerImage,
				*image.VersionConstraint,
				*tag.ImageTagList,
			) (*tag.ImageTag, error) {
				return nil, nil
			},
			assertions: func(images []api.Image, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "found no suitable version of image")
				require.Nil(t, images)
			},
		},
		{
			name: "success",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{
						Images: []api.ImageSubscription{
							{
								RepoURL: "fake-url",
							},
						},
					},
				},
			},
			repoCredsFn: func(
				context.Context,
				string,
				api.ImageSubscription,
				*registry.RegistryEndpoint,
			) (image.Credential, error) {
				return image.Credential{}, nil
			},
			tagsFn: func(
				*registry.RegistryEndpoint,
				*image.ContainerImage,
				registry.RegistryClient,
				*image.VersionConstraint,
			) (*tag.ImageTagList, error) {
				return tag.NewImageTagList(), nil
			},
			newestTagFn: func(
				*image.ContainerImage,
				*image.VersionConstraint,
				*tag.ImageTagList,
			) (*tag.ImageTag, error) {
				return &tag.ImageTag{
					TagName: "fake-tag",
				}, nil
			},
			assertions: func(images []api.Image, err error) {
				require.NoError(t, err)
				require.Len(t, images, 1)
				require.Equal(
					t,
					api.Image{
						RepoURL: "fake-url",
						Tag:     "fake-tag",
					},
					images[0],
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reconciler := environmentReconciler{
				logger:                    log.New(),
				getImageRepoCredentialsFn: testCase.repoCredsFn,
				getImageTagsFn:            testCase.tagsFn,
				getNewestImageTagFn:       testCase.newestTagFn,
			}
			reconciler.logger.SetLevel(log.ErrorLevel)
			env := &api.Environment{
				Spec: testCase.spec,
			}
			testCase.assertions(reconciler.getLatestImages(context.Background(), env))
		})
	}
}
