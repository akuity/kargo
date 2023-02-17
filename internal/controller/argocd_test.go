package controller

import (
	"context"
	"testing"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argoHealth "github.com/argoproj/gitops-engine/pkg/health"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/akuityio/kargo/api/v1alpha1"
)

func TestCheckHealth(t *testing.T) {
	testCases := []struct {
		name           string
		state          api.EnvironmentState
		appChecks      []api.ArgoCDAppCheck
		getArgoCDAppFn func(
			context.Context,
			client.Client,
			string,
			string,
		) (*argocd.Application, error)
		assertions func(api.Health)
	}{
		{
			name: "healthchecks do not include any Argo CD Apps",
			assertions: func(health api.Health) {
				require.Equal(t,
					api.Health{
						Status: api.HealthStateUnknown,
						StatusReason: "spec.healthChecks contains insufficient " +
							"instructions to assess Environment health",
					},
					health,
				)
			},
		},

		{
			name: "error finding Argo CD App",
			appChecks: []api.ArgoCDAppCheck{
				{
					AppName:      "fake-app",
					AppNamespace: "fake-namespace",
				},
			},
			getArgoCDAppFn: func(
				context.Context,
				client.Client,
				string,
				string,
			) (*argocd.Application, error) {
				return nil, errors.New("something went wrong")
			},
			assertions: func(health api.Health) {
				require.Equal(t, api.HealthStateUnknown, health.Status)
				require.Contains(
					t,
					health.StatusReason,
					"error finding Argo CD Application",
				)
				require.Contains(t, health.StatusReason, "something went wrong")
			},
		},

		{
			name: "Argo CD App not found",
			appChecks: []api.ArgoCDAppCheck{
				{
					AppName:      "fake-app",
					AppNamespace: "fake-namespace",
				},
			},
			getArgoCDAppFn: func(
				context.Context,
				client.Client,
				string,
				string,
			) (*argocd.Application, error) {
				return nil, nil
			},
			assertions: func(health api.Health) {
				require.Equal(t, api.HealthStateUnknown, health.Status)
				require.Contains(
					t,
					health.StatusReason,
					"unable to find Argo CD Application",
				)
			},
		},

		{
			name: "Argo CD App is multi-source",
			// This doesn't require there to actually BE multiple sources. Simply
			// using the sources field instead of the source fields should be enough
			// to trigger this case.
			appChecks: []api.ArgoCDAppCheck{
				{
					AppName:      "fake-app",
					AppNamespace: "fake-namespace",
				},
			},
			getArgoCDAppFn: func(
				context.Context,
				client.Client,
				string,
				string,
			) (*argocd.Application, error) {
				return &argocd.Application{
					Spec: argocd.ApplicationSpec{
						Sources: argocd.ApplicationSources{
							{},
						},
					},
				}, nil
			},
			assertions: func(health api.Health) {
				require.Equal(t, api.HealthStateUnknown, health.Status)
				require.Contains(
					t,
					health.StatusReason,
					"bugs in Argo CD currently prevent a comprehensive assessment of "+
						"the health of multi-source Application",
				)
			},
		},

		{
			name: "Argo CD App is not healthy",
			appChecks: []api.ArgoCDAppCheck{
				{
					AppName:      "fake-app",
					AppNamespace: "fake-namespace",
				},
			},
			getArgoCDAppFn: func(
				context.Context,
				client.Client,
				string,
				string,
			) (*argocd.Application, error) {
				return &argocd.Application{
					Status: argocd.ApplicationStatus{
						Health: argocd.HealthStatus{
							Status: argoHealth.HealthStatusDegraded,
						},
					},
				}, nil
			},
			assertions: func(health api.Health) {
				require.Equal(t, api.HealthStateUnhealthy, health.Status)
				require.Contains(t, health.StatusReason, "has health state")
				require.Contains(
					t,
					health.StatusReason,
					argoHealth.HealthStatusDegraded,
				)
			},
		},

		{
			name: "Argo CD App not synced",
			appChecks: []api.ArgoCDAppCheck{
				{
					AppName:      "fake-app",
					AppNamespace: "fake-namespace",
				},
			},
			getArgoCDAppFn: func(
				context.Context,
				client.Client,
				string,
				string,
			) (*argocd.Application, error) {
				return &argocd.Application{
					Spec: argocd.ApplicationSpec{
						Source: &argocd.ApplicationSource{},
					},
					Status: argocd.ApplicationStatus{
						Health: argocd.HealthStatus{
							Status: argoHealth.HealthStatusHealthy,
						},
						Sync: argocd.SyncStatus{
							Status: argocd.SyncStatusCodeOutOfSync,
						},
					},
				}, nil
			},
			assertions: func(health api.Health) {
				require.Equal(t, api.HealthStateUnhealthy, health.Status)
				require.Contains(
					t,
					health.StatusReason,
					"is not synced to revision",
				)
			},
		},

		{
			name: "Argo CD App healthy and synced",
			state: api.EnvironmentState{
				Commits: []api.GitCommit{
					{
						RepoURL: "fake-url",
						ID:      "fake-commit",
					},
				},
			},
			appChecks: []api.ArgoCDAppCheck{
				{
					AppName:      "fake-app",
					AppNamespace: "fake-namespace",
				},
			},
			getArgoCDAppFn: func(
				context.Context,
				client.Client,
				string,
				string,
			) (*argocd.Application, error) {
				return &argocd.Application{
					Spec: argocd.ApplicationSpec{
						Source: &argocd.ApplicationSource{
							RepoURL: "fake-url",
						},
					},
					Status: argocd.ApplicationStatus{
						Health: argocd.HealthStatus{
							Status: argoHealth.HealthStatusHealthy,
						},
						Sync: argocd.SyncStatus{
							Status:   argocd.SyncStatusCodeSynced,
							Revision: "fake-commit",
						},
					},
				}, nil
			},
			assertions: func(health api.Health) {
				require.Equal(t, api.HealthStateHealthy, health.Status)
				require.Empty(t, health.StatusReason)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reconciler := &environmentReconciler{
				getArgoCDAppFn: testCase.getArgoCDAppFn,
			}
			testCase.assertions(
				reconciler.checkHealth(
					context.Background(),
					testCase.state,
					api.HealthChecks{
						ArgoCDAppChecks: testCase.appChecks,
					},
				),
			)
		})
	}
}

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
	reconciler := &environmentReconciler{}
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
			reconciler := environmentReconciler{
				getArgoCDAppFn:            testCase.getArgoCDAppFn,
				applyArgoCDSourceUpdateFn: testCase.applyArgoCDSourceUpdateFn,
				patchFn:                   testCase.patchFn,
			}
			testCase.assertions(
				reconciler.applyArgoCDAppUpdate(
					context.Background(),
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
