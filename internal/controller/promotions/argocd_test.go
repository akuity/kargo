package promotions

import (
	"context"
	"testing"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/akuityio/kargo/api/v1alpha1"
)

func TestApplyArgoCDSourceUpdate(t *testing.T) {
	testCases := []struct {
		name       string
		source     argocd.ApplicationSource
		newState   api.EnvironmentState
		update     api.ArgoCDSourceUpdate
		assertions func(
			originalSource argocd.ApplicationSource,
			updatedSource argocd.ApplicationSource,
			err error,
		)
	}{
		{
			name: "update doesn't apply to this source",
			source: argocd.ApplicationSource{
				RepoURL: "fake-url",
			},
			update: api.ArgoCDSourceUpdate{
				RepoURL: "different-fake-url",
			},
			assertions: func(
				originalSource argocd.ApplicationSource,
				updatedSource argocd.ApplicationSource,
				err error,
			) {
				require.NoError(t, err)
				// Source should be entirely unchanged
				require.Equal(t, originalSource, updatedSource)
			},
		},

		{
			name: "update target revision (git)",
			source: argocd.ApplicationSource{
				RepoURL: "fake-url",
			},
			newState: api.EnvironmentState{
				Commits: []api.GitCommit{
					{
						RepoURL: "fake-url",
						ID:      "fake-commit",
					},
				},
			},
			update: api.ArgoCDSourceUpdate{
				RepoURL:              "fake-url",
				UpdateTargetRevision: true,
			},
			assertions: func(
				originalSource argocd.ApplicationSource,
				updatedSource argocd.ApplicationSource,
				err error,
			) {
				require.NoError(t, err)
				// TargetRevision should be updated
				require.Equal(t, "fake-commit", updatedSource.TargetRevision)
				// Everything else should be unchanged
				updatedSource.TargetRevision = originalSource.TargetRevision
				require.Equal(t, originalSource, updatedSource)
			},
		},

		{
			name: "update target revision (helm chart)",
			source: argocd.ApplicationSource{
				RepoURL: "fake-url",
				Chart:   "fake-chart",
			},
			newState: api.EnvironmentState{
				Charts: []api.Chart{
					{
						RegistryURL: "fake-url",
						Name:        "fake-chart",
						Version:     "fake-version",
					},
				},
			},
			update: api.ArgoCDSourceUpdate{
				RepoURL:              "fake-url",
				Chart:                "fake-chart",
				UpdateTargetRevision: true,
			},
			assertions: func(
				originalSource argocd.ApplicationSource,
				updatedSource argocd.ApplicationSource,
				err error,
			) {
				require.NoError(t, err)
				// TargetRevision should be updated
				require.Equal(t, "fake-version", updatedSource.TargetRevision)
				// Everything else should be unchanged
				updatedSource.TargetRevision = originalSource.TargetRevision
				require.Equal(t, originalSource, updatedSource)
			},
		},

		{
			name: "update images with kustomize",
			source: argocd.ApplicationSource{
				RepoURL: "fake-url",
			},
			newState: api.EnvironmentState{
				Images: []api.Image{
					{
						RepoURL: "fake-image-url",
						Tag:     "fake-tag",
					},
					{
						// This one should not be updated because it's not a match for
						// anything in the update instructions
						RepoURL: "another-fake-image-url",
						Tag:     "another-fake-tag",
					},
				},
			},
			update: api.ArgoCDSourceUpdate{
				RepoURL: "fake-url",
				Kustomize: &api.ArgoCDKustomize{
					Images: []string{
						"fake-image-url",
					},
				},
			},
			assertions: func(
				originalSource argocd.ApplicationSource,
				updatedSource argocd.ApplicationSource,
				err error,
			) {
				require.NoError(t, err)
				// Kustomize attributes should be updated
				require.NotNil(t, updatedSource.Kustomize)
				require.Equal(
					t,
					argocd.KustomizeImages{
						argocd.KustomizeImage("fake-image-url=fake-image-url:fake-tag"),
					},
					updatedSource.Kustomize.Images,
				)
				// Everything else should be unchanged
				updatedSource.Kustomize = originalSource.Kustomize
				require.Equal(t, originalSource, updatedSource)
			},
		},

		{
			name: "update images with helm",
			source: argocd.ApplicationSource{
				RepoURL: "fake-url",
			},
			newState: api.EnvironmentState{
				Images: []api.Image{
					{
						RepoURL: "fake-image-url",
						Tag:     "fake-tag",
					},
					{
						// This one should not be updated because it's not a match for
						// anything in the update instructions
						RepoURL: "another-fake-image-url",
						Tag:     "another-fake-tag",
					},
				},
			},
			update: api.ArgoCDSourceUpdate{
				RepoURL: "fake-url",
				Helm: &api.ArgoCDHelm{
					Images: []api.ArgoCDHelmImageUpdate{
						{
							Image: "fake-image-url",
							Key:   "image",
							Value: "Image",
						},
					},
				},
			},
			assertions: func(
				originalSource argocd.ApplicationSource,
				updatedSource argocd.ApplicationSource,
				err error,
			) {
				require.NoError(t, err)
				// Helm attributes should be updated
				require.NotNil(t, updatedSource.Helm)
				require.NotNil(t, updatedSource.Helm.Parameters)
				require.Equal(
					t,
					[]argocd.HelmParameter{
						{
							Name:  "image",
							Value: "fake-image-url:fake-tag",
						},
					},
					updatedSource.Helm.Parameters,
				)
				// Everything else should be unchanged
				updatedSource.Helm = originalSource.Helm
				require.Equal(t, originalSource, updatedSource)
			},
		},
	}
	reconciler := &reconciler{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			updatedSource, err := reconciler.applyArgoCDSourceUpdate(
				testCase.source,
				testCase.newState,
				testCase.update,
			)
			testCase.assertions(testCase.source, updatedSource, err)
		})
	}
}

func TestAuthorizeArgoCDAppUpdate(t *testing.T) {
	testCases := []struct {
		name    string
		app     *argocd.Application
		allowed bool
	}{
		{
			name: "annotations are nil",
			app:  &argocd.Application{},
		},
		{
			name: "annotation is missing",
			app: &argocd.Application{
				ObjectMeta: v1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
		},
		{
			name: "annotation cannot be parsed",
			app: &argocd.Application{
				ObjectMeta: v1.ObjectMeta{
					Annotations: map[string]string{
						authorizedEnvAnnotationKey: "bogus",
					},
				},
			},
		},
		{
			name: "mutation is not allowed",
			app: &argocd.Application{
				ObjectMeta: v1.ObjectMeta{
					Annotations: map[string]string{
						authorizedEnvAnnotationKey: "ns-nope:name-nope",
					},
				},
			},
		},
		{
			name: "mutation is allowed",
			app: &argocd.Application{
				ObjectMeta: v1.ObjectMeta{
					Annotations: map[string]string{
						authorizedEnvAnnotationKey: "ns-yep:name-yep",
					},
				},
			},
			allowed: true,
		},
	}
	reconciler := reconciler{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := reconciler.authorizeArgoCDAppUpdate(
				v1.ObjectMeta{
					Name:      "name-yep",
					Namespace: "ns-yep",
				},
				testCase.app,
			)
			if testCase.allowed {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestApplyArgoCDAppUpdate(t *testing.T) {
	testCases := []struct {
		name           string
		newState       api.EnvironmentState
		update         api.ArgoCDAppUpdate
		getArgoCDAppFn func(
			context.Context,
			client.Client,
			string,
			string,
		) (*argocd.Application, error)
		applyArgoCDSourceUpdateFn func(
			argocd.ApplicationSource,
			api.EnvironmentState,
			api.ArgoCDSourceUpdate,
		) (argocd.ApplicationSource, error)
		patchFn func(
			context.Context,
			client.Object,
			client.Patch,
			...client.PatchOption,
		) error
		assertions func(error)
	}{
		{
			name: "error getting Argo CD App",
			getArgoCDAppFn: func(
				context.Context,
				client.Client,
				string,
				string,
			) (*argocd.Application, error) {
				return nil, errors.New("something went wrong")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(t, err.Error(), "error finding Argo CD Application")
			},
		},

		{
			name: "Argo CD App not found",
			getArgoCDAppFn: func(
				context.Context,
				client.Client,
				string,
				string,
			) (*argocd.Application, error) {
				return nil, nil
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to find Argo CD Application")
			},
		},

		{
			name: "update not allowed",
			getArgoCDAppFn: func(
				context.Context,
				client.Client,
				string,
				string,
			) (*argocd.Application, error) {
				// This is not annotated properly to allow the Environment to mutate it
				return &argocd.Application{}, nil
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"does not permit mutation by Kargo Environment",
				)
			},
		},

		{
			name: "error applying source update (single-source app)",
			update: api.ArgoCDAppUpdate{
				SourceUpdates: []api.ArgoCDSourceUpdate{
					{},
				},
			},
			getArgoCDAppFn: func(
				context.Context,
				client.Client,
				string,
				string,
			) (*argocd.Application, error) {
				return &argocd.Application{
					ObjectMeta: v1.ObjectMeta{
						Annotations: map[string]string{
							authorizedEnvAnnotationKey: "fake-namespace:fake-env",
						},
					},
					Spec: argocd.ApplicationSpec{
						Source: &argocd.ApplicationSource{},
					},
				}, nil
			},
			applyArgoCDSourceUpdateFn: func(
				source argocd.ApplicationSource,
				_ api.EnvironmentState,
				_ api.ArgoCDSourceUpdate,
			) (argocd.ApplicationSource, error) {
				return source, errors.New("something went wrong")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(
					t,
					err.Error(),
					"error updating source of Argo CD Application",
				)
			},
		},

		{
			name: "error applying source update (multi-source app)",
			update: api.ArgoCDAppUpdate{
				SourceUpdates: []api.ArgoCDSourceUpdate{
					{},
				},
			},
			getArgoCDAppFn: func(
				context.Context,
				client.Client,
				string,
				string,
			) (*argocd.Application, error) {
				return &argocd.Application{
					ObjectMeta: v1.ObjectMeta{
						Annotations: map[string]string{
							authorizedEnvAnnotationKey: "fake-namespace:fake-env",
						},
					},
					Spec: argocd.ApplicationSpec{
						Sources: argocd.ApplicationSources{
							{},
						},
					},
				}, nil
			},
			applyArgoCDSourceUpdateFn: func(
				source argocd.ApplicationSource,
				_ api.EnvironmentState,
				_ api.ArgoCDSourceUpdate,
			) (argocd.ApplicationSource, error) {
				return source, errors.New("something went wrong")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(
					t,
					err.Error(),
					"error updating source(s) of Argo CD Application",
				)
			},
		},

		{
			name: "error patching Argo CD App",
			update: api.ArgoCDAppUpdate{
				SourceUpdates: []api.ArgoCDSourceUpdate{
					{},
				},
			},
			getArgoCDAppFn: func(
				context.Context,
				client.Client,
				string,
				string,
			) (*argocd.Application, error) {
				return &argocd.Application{
					ObjectMeta: v1.ObjectMeta{
						Annotations: map[string]string{
							authorizedEnvAnnotationKey: "fake-namespace:fake-env",
						},
					},
					Spec: argocd.ApplicationSpec{
						Source: &argocd.ApplicationSource{},
					},
				}, nil
			},
			applyArgoCDSourceUpdateFn: func(
				source argocd.ApplicationSource,
				_ api.EnvironmentState,
				_ api.ArgoCDSourceUpdate,
			) (argocd.ApplicationSource, error) {
				return source, nil
			},
			patchFn: func(
				context.Context,
				client.Object,
				client.Patch,
				...client.PatchOption,
			) error {
				return errors.New("something went wrong")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(
					t,
					err.Error(),
					"error patching Argo CD Application",
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reconciler := reconciler{
				getArgoCDAppFn:            testCase.getArgoCDAppFn,
				applyArgoCDSourceUpdateFn: testCase.applyArgoCDSourceUpdateFn,
				patchFn:                   testCase.patchFn,
			}
			testCase.assertions(
				reconciler.applyArgoCDAppUpdate(
					context.Background(),
					v1.ObjectMeta{
						Name:      "fake-env",
						Namespace: "fake-namespace",
					},
					testCase.newState,
					testCase.update,
				),
			)
		})
	}
}

func TestBuildKustomizeImagesForArgoCDAppSource(t *testing.T) {
	images := []api.Image{
		{
			RepoURL: "fake-url",
			Tag:     "fake-tag",
		},
		{
			RepoURL: "another-fake-url",
			Tag:     "another-fake-tag",
		},
	}
	imageUpdates := []string{
		"fake-url",
		"another-fake-url",
		"image-that-is-not-in-list",
	}
	result := buildKustomizeImagesForArgoCDAppSource(images, imageUpdates)
	require.Equal(
		t,
		argocd.KustomizeImages{
			"fake-url=fake-url:fake-tag",
			"another-fake-url=another-fake-url:another-fake-tag",
		},
		result,
	)
}

func TestBuildHelmParamChangesForArgoCDAppSource(t *testing.T) {
	images := []api.Image{
		{
			RepoURL: "fake-url",
			Tag:     "fake-tag",
		},
		{
			RepoURL: "another-fake-url",
			Tag:     "another-fake-tag",
		},
	}
	imageUpdates := []api.ArgoCDHelmImageUpdate{
		{
			Image: "fake-url",
			Key:   "fake-key",
			Value: "Image",
		},
		{
			Image: "another-fake-url",
			Key:   "another-fake-key",
			Value: "Tag",
		},
		{
			Image: "image-that-is-not-in-list",
			Key:   "fake-key",
			Value: "Tag",
		},
	}
	result := buildHelmParamChangesForArgoCDAppSource(images, imageUpdates)
	require.Equal(
		t,
		map[string]string{
			"fake-key":         "fake-url:fake-tag",
			"another-fake-key": "another-fake-tag",
		},
		result,
	)
}
