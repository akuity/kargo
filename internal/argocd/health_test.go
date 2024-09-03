package argocd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
)

func TestApplicationHealth_EvaluateHealth(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, argocd.AddToScheme(scheme))

	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}

	testCases := []struct {
		name         string
		applications []client.Object
		stage        *kargoapi.Stage
		assertions   func(*testing.T, *kargoapi.Health)
	}{
		{
			name: "no updates",
			assertions: func(t *testing.T, health *kargoapi.Health) {
				require.Nil(t, health)
			},
			stage: &kargoapi.Stage{},
		},
		{
			name: "single update",
			applications: []client.Object{
				&argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-namespace",
						Name:      "fake-name",
					},
					Spec: argocd.ApplicationSpec{
						Source: &argocd.ApplicationSource{
							RepoURL: "https://example.com",
							Chart:   "fake-chart",
						},
					},
					Status: argocd.ApplicationStatus{
						Health: argocd.HealthStatus{
							Status: argocd.HealthStatusHealthy,
						},
						Sync: argocd.SyncStatus{
							Status:   argocd.SyncStatusCodeSynced,
							Revision: "v1.2.3",
						},
						OperationState: &argocd.OperationState{
							FinishedAt: &metav1.Time{Time: metav1.Now().Add(-10 * time.Second)},
						},
					},
				},
			},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
							{
								AppNamespace: "fake-namespace",
								AppName:      "fake-name",
							},
						},
					},
				},
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						&kargoapi.FreightCollection{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
									Charts: []kargoapi.Chart{
										{
											RepoURL: "https://example.com",
											Name:    "fake-chart",
											Version: "v1.2.3",
										},
									},
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, health *kargoapi.Health) {
				require.NotNil(t, health)

				require.Equal(t, kargoapi.HealthStateHealthy, health.Status)
				require.Len(t, health.Issues, 0)

				require.Len(t, health.ArgoCDApps, 1)
				require.Equal(t, kargoapi.ArgoCDAppStatus{
					Namespace: "fake-namespace",
					Name:      "fake-name",
					HealthStatus: kargoapi.ArgoCDAppHealthStatus{
						Status: kargoapi.ArgoCDAppHealthStateHealthy,
					},
					SyncStatus: kargoapi.ArgoCDAppSyncStatus{
						Status:   kargoapi.ArgoCDAppSyncStateSynced,
						Revision: "v1.2.3",
					},
				}, health.ArgoCDApps[0])
			},
		},
		{
			name: "multiple updates",
			applications: []client.Object{
				&argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-namespace",
						Name:      "fake-name-1",
					},
					Spec: argocd.ApplicationSpec{
						Source: &argocd.ApplicationSource{},
					},
					Status: argocd.ApplicationStatus{
						Health: argocd.HealthStatus{
							Status: argocd.HealthStatusHealthy,
						},
						Sync: argocd.SyncStatus{
							Status: argocd.SyncStatusCodeSynced,
						},
					},
				},
				&argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-namespace",
						Name:      "fake-name-2",
					},
					Spec: argocd.ApplicationSpec{
						Source: &argocd.ApplicationSource{},
					},
				},
			},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
							{
								AppNamespace: "fake-namespace",
								AppName:      "fake-name-1",
							},
							{
								AppNamespace: "fake-namespace",
								AppName:      "fake-name-2",
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, health *kargoapi.Health) {
				require.NotNil(t, health)

				require.Equal(t, kargoapi.HealthStateProgressing, health.Status)
				require.Len(t, health.Issues, 1)
				require.Contains(t, health.Issues[0], "is progressing")

				require.Len(t, health.ArgoCDApps, 2)

				require.Equal(t, kargoapi.ArgoCDAppStatus{
					Namespace: "fake-namespace",
					Name:      "fake-name-1",
					HealthStatus: kargoapi.ArgoCDAppHealthStatus{
						Status: kargoapi.ArgoCDAppHealthStateHealthy,
					},
					SyncStatus: kargoapi.ArgoCDAppSyncStatus{
						Status: kargoapi.ArgoCDAppSyncStateSynced,
					},
				}, health.ArgoCDApps[0])
				require.Equal(t, kargoapi.ArgoCDAppStatus{
					Namespace: "fake-namespace",
					Name:      "fake-name-2",
					HealthStatus: kargoapi.ArgoCDAppHealthStatus{
						Status: kargoapi.ArgoCDAppHealthStateUnknown,
					},
					SyncStatus: kargoapi.ArgoCDAppSyncStatus{
						Status: kargoapi.ArgoCDAppSyncStateUnknown,
					},
				}, health.ArgoCDApps[1])
			},
		},
		{
			name: "update with empty namespace",
			applications: []client.Object{
				&argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-name",
						Namespace: Namespace(),
					},
					Spec: argocd.ApplicationSpec{
						Source: &argocd.ApplicationSource{},
					},
					Status: argocd.ApplicationStatus{
						Health: argocd.HealthStatus{
							Status: argocd.HealthStatusHealthy,
						},
						Sync: argocd.SyncStatus{
							Status: argocd.SyncStatusCodeSynced,
						},
					},
				},
			},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{{
							AppName: "fake-name",
						}},
					},
				},
			},
			assertions: func(t *testing.T, health *kargoapi.Health) {
				require.NotNil(t, health)

				require.Equal(t, kargoapi.HealthStateHealthy, health.Status)
				require.Len(t, health.Issues, 0)
			},
		},
		{
			name: "composite error checking Application health",
			applications: []client.Object{
				&argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-namespace",
						Name:      "fake-name",
					},
					Status: argocd.ApplicationStatus{
						Conditions: []argocd.ApplicationCondition{
							{
								Type: argocd.ApplicationConditionComparisonError,
							},
							{
								Type: argocd.ApplicationConditionInvalidSpecError,
							},
						},
					},
				},
			},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{{
							AppNamespace: "fake-namespace",
							AppName:      "fake-name",
						}},
					},
				},
			},
			assertions: func(t *testing.T, health *kargoapi.Health) {
				require.NotNil(t, health)

				require.Equal(t, kargoapi.HealthStateUnhealthy, health.Status)
				require.Len(t, health.Issues, 2)
				require.Contains(t, health.Issues[0], "ComparisonError")
				require.Contains(t, health.Issues[1], "InvalidSpecError")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(scheme)
			if len(testCase.applications) > 0 {
				c.WithObjects(testCase.applications...)
			}

			h := &applicationHealth{
				argoClient: c.Build(),
			}

			testCase.assertions(
				t,
				h.EvaluateHealth(
					context.Background(),
					testCase.stage,
				),
			)
		})
	}

	t.Run("Argo CD integration disabled", func(t *testing.T) {
		h := &applicationHealth{}
		health := h.EvaluateHealth(
			context.Background(),
			&kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{{}},
					},
				},
			},
		)
		require.NotNil(t, health)
		require.Equal(t, kargoapi.HealthStateUnknown, health.Status)
		require.Len(t, health.Issues, 1)
		require.Contains(t, health.Issues[0], "Argo CD integration is disabled")
	})
}

func TestApplicationHealth_GetApplicationHealth(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, argocd.AddToScheme(scheme))

	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
	testOrigin2 := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse2",
	}

	testStageSpec := kargoapi.StageSpec{
		PromotionMechanisms: &kargoapi.PromotionMechanisms{
			ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{{
				Origin:       &testOrigin,
				AppNamespace: "fake-namespace",
				AppName:      "fake-name",
			}},
		},
	}

	testCases := []struct {
		name        string
		application *argocd.Application
		interceptor interceptor.Funcs
		key         types.NamespacedName
		stage       *kargoapi.Stage
		assertions  func(
			*testing.T,
			kargoapi.HealthState,
			kargoapi.ArgoCDAppHealthStatus,
			kargoapi.ArgoCDAppSyncStatus,
			error,
		)
	}{
		{
			name: "Application not found",
			key:  types.NamespacedName{Namespace: "fake-namespace", Name: "fake-name"},
			stage: &kargoapi.Stage{
				Spec: testStageSpec,
			},
			assertions: func(
				t *testing.T,
				state kargoapi.HealthState,
				healthStatus kargoapi.ArgoCDAppHealthStatus,
				syncStatus kargoapi.ArgoCDAppSyncStatus,
				err error,
			) {
				require.ErrorContains(t, err, "unable to find Argo CD Application")

				require.Equal(t, kargoapi.HealthStateUnknown, state)
				require.Equal(t, kargoapi.ArgoCDAppHealthStateUnknown, healthStatus.Status)
				require.Equal(t, kargoapi.ArgoCDAppSyncStateUnknown, syncStatus.Status)
			},
		},
		{
			name: "error getting Application",
			key:  types.NamespacedName{Namespace: "fake-namespace", Name: "fake-name"},
			stage: &kargoapi.Stage{
				Spec: testStageSpec,
			},
			interceptor: interceptor.Funcs{
				Get: func(
					_ context.Context,
					_ client.WithWatch,
					_ client.ObjectKey,
					_ client.Object,
					_ ...client.GetOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				state kargoapi.HealthState,
				healthStatus kargoapi.ArgoCDAppHealthStatus,
				syncStatus kargoapi.ArgoCDAppSyncStatus,
				err error,
			) {
				require.ErrorContains(t, err, "error finding Argo CD Application")
				require.ErrorContains(t, err, "something went wrong")

				require.Equal(t, kargoapi.HealthStateUnknown, state)
				require.Equal(t, kargoapi.ArgoCDAppHealthStateUnknown, healthStatus.Status)
				require.Equal(t, kargoapi.ArgoCDAppSyncStateUnknown, syncStatus.Status)
			},
		},
		{
			name: "Application with error conditions yields Unhealthy state",
			key:  types.NamespacedName{Namespace: "fake-namespace", Name: "fake-name"},
			stage: &kargoapi.Stage{
				Spec: testStageSpec,
			},
			application: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-namespace",
					Name:      "fake-name",
				},
				Status: argocd.ApplicationStatus{
					Conditions: []argocd.ApplicationCondition{
						{
							Type:    argocd.ApplicationConditionComparisonError,
							Message: "fake-error",
						},
						{
							Type:    argocd.ApplicationConditionInvalidSpecError,
							Message: "fake-error",
						},
					},
					Health: argocd.HealthStatus{
						Status:  argocd.HealthStatusHealthy,
						Message: "fake-message",
					},
					Sync: argocd.SyncStatus{
						Status: argocd.SyncStatusCodeSynced,
					},
				},
			},
			assertions: func(
				t *testing.T,
				state kargoapi.HealthState,
				healthStatus kargoapi.ArgoCDAppHealthStatus,
				syncStatus kargoapi.ArgoCDAppSyncStatus,
				err error,
			) {
				require.Error(t, err)
				require.ErrorContains(t, err, `has "ComparisonError" condition`)
				require.ErrorContains(t, err, `has "InvalidSpecError" condition`)

				unwrappedErr, ok := err.(compositeError)
				require.True(t, ok)
				require.Len(t, unwrappedErr.Unwrap(), 2)

				require.Equal(t, kargoapi.HealthStateUnhealthy, state)
				require.Equal(t, kargoapi.ArgoCDAppHealthStatus{
					Status:  kargoapi.ArgoCDAppHealthStateHealthy,
					Message: "fake-message",
				}, healthStatus)
				require.Equal(t, kargoapi.ArgoCDAppSyncStatus{
					Status: kargoapi.ArgoCDAppSyncStateSynced,
				}, syncStatus)
			},
		},
		{
			name: "Desired revision requires Application to be synced",
			key:  types.NamespacedName{Namespace: "fake-namespace", Name: "fake-name"},
			application: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-namespace",
					Name:      "fake-name",
				},
				Spec: argocd.ApplicationSpec{
					Source: &argocd.ApplicationSource{
						RepoURL: "https://example.com/universe/42",
					},
				},
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: argocd.HealthStatusHealthy,
					},
					Sync: argocd.SyncStatus{
						Status:   argocd.SyncStatusCodeSynced,
						Revision: "fake-revision",
					},
					OperationState: &argocd.OperationState{
						FinishedAt: ptr.To(metav1.Now()),
					},
				},
			},
			stage: &kargoapi.Stage{
				Spec: testStageSpec,
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						&kargoapi.FreightCollection{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
									Commits: []kargoapi.GitCommit{{
										RepoURL: "https://example.com/universe/42",
										ID:      "other-fake-revision",
									}},
								},
							},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				state kargoapi.HealthState,
				healthStatus kargoapi.ArgoCDAppHealthStatus,
				syncStatus kargoapi.ArgoCDAppSyncStatus,
				err error,
			) {
				require.ErrorContains(t, err, "Desired revision [\"other-fake-revision\"]")
				require.ErrorContains(t, err, "does not match current revision \"fake-revision\" of Application")

				require.Equal(t, kargoapi.HealthStateUnhealthy, state)
				require.Equal(t, kargoapi.ArgoCDAppHealthStatus{
					Status: kargoapi.ArgoCDAppHealthStateHealthy,
				}, healthStatus)
				require.Equal(t, kargoapi.ArgoCDAppSyncStatus{
					Status:   kargoapi.ArgoCDAppSyncStateSynced,
					Revision: "fake-revision",
				}, syncStatus)
			},
		},
		{
			name: "Desired revisions require Application to be synced",
			key:  types.NamespacedName{Namespace: "fake-namespace", Name: "fake-name"},
			application: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-namespace",
					Name:      "fake-name",
				},
				Spec: argocd.ApplicationSpec{
					Sources: argocd.ApplicationSources{
						argocd.ApplicationSource{
							RepoURL: "https://example.com/universe/42",
						},
						argocd.ApplicationSource{
							RepoURL: "https://example.com/another-universe/42",
						},
						argocd.ApplicationSource{
							Chart:   "fake-chart",
							RepoURL: "https://example.com/",
						},
					},
				},
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: argocd.HealthStatusHealthy,
					},
					Sync: argocd.SyncStatus{
						Status:    argocd.SyncStatusCodeSynced,
						Revisions: []string{"warehouse1-revision1", "warehouse2-revision1", "unsubscribed-revision"},
					},
					OperationState: &argocd.OperationState{
						FinishedAt: ptr.To(metav1.Now()),
					},
				},
			},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
							{
								AppNamespace: "fake-namespace",
								AppName:      "fake-name",
								SourceUpdates: []kargoapi.ArgoCDSourceUpdate{
									{Origin: &testOrigin, RepoURL: "https://example.com/universe/42"},
									{Origin: &testOrigin2, RepoURL: "https://example.com/another-universe/42"},
								},
							}},
					},
				},
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						&kargoapi.FreightCollection{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
									Commits: []kargoapi.GitCommit{{
										RepoURL: "https://example.com/universe/42",
										ID:      "warehouse1-revision1",
									}},
								},
								testOrigin2.String(): {
									Origin: testOrigin2,
									Commits: []kargoapi.GitCommit{{
										RepoURL: "https://example.com/another-universe/42",
										ID:      "warehouse2-new-revision2",
									}},
								},
							},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				state kargoapi.HealthState,
				healthStatus kargoapi.ArgoCDAppHealthStatus,
				syncStatus kargoapi.ArgoCDAppSyncStatus,
				err error,
			) {
				require.ErrorContains(t, err, "Not all sources of Application")
				require.ErrorContains(t, err, "match the desired revisions")
				require.ErrorContains(t, err, "Source 1 with RepoURL https://example.com/another-universe/42")

				require.Equal(t, kargoapi.HealthStateUnhealthy, state)
				require.Equal(t, kargoapi.ArgoCDAppHealthStatus{
					Status: kargoapi.ArgoCDAppHealthStateHealthy,
				}, healthStatus)
				require.Equal(t, kargoapi.ArgoCDAppSyncStatus{
					Status:    kargoapi.ArgoCDAppSyncStateSynced,
					Revisions: []string{"warehouse1-revision1", "warehouse2-revision1", "unsubscribed-revision"},
				}, syncStatus)
			},
		},
		{
			name: "Without a desired revision, Application is Healthy",
			key:  types.NamespacedName{Namespace: "fake-namespace", Name: "fake-name"},
			stage: &kargoapi.Stage{
				Spec: testStageSpec,
			},
			application: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-namespace",
					Name:      "fake-name",
				},
				Spec: argocd.ApplicationSpec{
					Source: &argocd.ApplicationSource{
						RepoURL: "https://example.com/universe/42",
					},
				},
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: argocd.HealthStatusHealthy,
					},
					Sync: argocd.SyncStatus{
						Status:   argocd.SyncStatusCodeSynced,
						Revision: "fake-revision",
					},
				},
			},
			assertions: func(
				t *testing.T,
				state kargoapi.HealthState,
				healthStatus kargoapi.ArgoCDAppHealthStatus,
				syncStatus kargoapi.ArgoCDAppSyncStatus,
				err error,
			) {
				require.NoError(t, err)

				require.Equal(t, kargoapi.HealthStateHealthy, state)
				require.Equal(t, kargoapi.ArgoCDAppHealthStatus{
					Status: kargoapi.ArgoCDAppHealthStateHealthy,
				}, healthStatus)
				require.Equal(t, kargoapi.ArgoCDAppSyncStatus{
					Status:   kargoapi.ArgoCDAppSyncStateSynced,
					Revision: "fake-revision",
				}, syncStatus)
			},
		},
		{
			name: "Application is Healthy",
			key:  types.NamespacedName{Namespace: "fake-namespace", Name: "fake-name"},
			application: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-namespace",
					Name:      "fake-name",
				},
				Spec: argocd.ApplicationSpec{
					Source: &argocd.ApplicationSource{
						RepoURL: "https://example.com/universe/42",
					},
				},
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: argocd.HealthStatusHealthy,
					},
					Sync: argocd.SyncStatus{
						Status:   argocd.SyncStatusCodeSynced,
						Revision: "fake-revision",
					},
					OperationState: &argocd.OperationState{
						FinishedAt: &metav1.Time{Time: metav1.Now().Add(-10 * time.Second)},
					},
				},
			},
			stage: &kargoapi.Stage{
				Spec: testStageSpec,
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						&kargoapi.FreightCollection{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
									Commits: []kargoapi.GitCommit{{
										RepoURL: "https://example.com/universe/42",
										ID:      "fake-revision",
									}},
								},
							},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				state kargoapi.HealthState,
				healthStatus kargoapi.ArgoCDAppHealthStatus,
				syncStatus kargoapi.ArgoCDAppSyncStatus,
				err error,
			) {
				require.NoError(t, err)

				require.Equal(t, kargoapi.HealthStateHealthy, state)
				require.Equal(t, kargoapi.ArgoCDAppHealthStatus{
					Status: kargoapi.ArgoCDAppHealthStateHealthy,
				}, healthStatus)
				require.Equal(t, kargoapi.ArgoCDAppSyncStatus{
					Status:   kargoapi.ArgoCDAppSyncStateSynced,
					Revision: "fake-revision",
				}, syncStatus)
			},
		},
		{
			name: "Multi-source single-warehouse Application is Healthy",
			key:  types.NamespacedName{Namespace: "fake-namespace", Name: "fake-name"},
			application: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-namespace",
					Name:      "fake-name",
				},
				Spec: argocd.ApplicationSpec{
					Sources: argocd.ApplicationSources{
						argocd.ApplicationSource{
							RepoURL: "https://example.com/universe/42",
						},
						argocd.ApplicationSource{
							RepoURL: "https://example.com/another-universe/42",
						},
					},
				},
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: argocd.HealthStatusHealthy,
					},
					Sync: argocd.SyncStatus{
						Status:    argocd.SyncStatusCodeSynced,
						Revisions: []string{"fake-revision1", "fake-revision2"},
					},
					OperationState: &argocd.OperationState{
						FinishedAt: &metav1.Time{Time: metav1.Now().Add(-10 * time.Second)},
					},
				},
			},
			stage: &kargoapi.Stage{
				Spec: testStageSpec,
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						&kargoapi.FreightCollection{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
									Commits: []kargoapi.GitCommit{{
										RepoURL: "https://example.com/universe/42",
										ID:      "fake-revision1",
									}},
								},
							},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				state kargoapi.HealthState,
				healthStatus kargoapi.ArgoCDAppHealthStatus,
				syncStatus kargoapi.ArgoCDAppSyncStatus,
				err error,
			) {
				require.NoError(t, err)

				require.Equal(t, kargoapi.HealthStateHealthy, state)
				require.Equal(t, kargoapi.ArgoCDAppHealthStatus{
					Status: kargoapi.ArgoCDAppHealthStateHealthy,
				}, healthStatus)
				require.Equal(t, kargoapi.ArgoCDAppSyncStatus{
					Status:    kargoapi.ArgoCDAppSyncStateSynced,
					Revisions: []string{"fake-revision1", "fake-revision2"},
				}, syncStatus)
			},
		},
		{
			name: "Multi-source multi-warehouse Application is Healthy",
			key:  types.NamespacedName{Namespace: "fake-namespace", Name: "fake-name"},
			application: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-namespace",
					Name:      "fake-name",
				},
				Spec: argocd.ApplicationSpec{
					Sources: argocd.ApplicationSources{
						argocd.ApplicationSource{
							RepoURL: "https://example.com/universe/42",
						},
						argocd.ApplicationSource{
							RepoURL: "https://example.com/another-universe/42",
						},
					},
				},
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: argocd.HealthStatusHealthy,
					},
					Sync: argocd.SyncStatus{
						Status:    argocd.SyncStatusCodeSynced,
						Revisions: []string{"warehouse1-revision1", "warehouse2-revision2"},
					},
					OperationState: &argocd.OperationState{
						FinishedAt: &metav1.Time{Time: metav1.Now().Add(-10 * time.Second)},
					},
				},
			},
			stage: &kargoapi.Stage{
				Spec: testStageSpec,
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						&kargoapi.FreightCollection{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
									Commits: []kargoapi.GitCommit{{
										RepoURL: "https://example.com/universe/42",
										ID:      "warehouse1-revision1",
									}},
								},
								testOrigin2.String(): {
									Origin: testOrigin2,
									Commits: []kargoapi.GitCommit{{
										RepoURL: "https://example.com/another-universe/42",
										ID:      "warehouse2-revision2",
									}},
								},
							},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				state kargoapi.HealthState,
				healthStatus kargoapi.ArgoCDAppHealthStatus,
				syncStatus kargoapi.ArgoCDAppSyncStatus,
				err error,
			) {
				require.NoError(t, err)

				require.Equal(t, kargoapi.HealthStateHealthy, state)
				require.Equal(t, kargoapi.ArgoCDAppHealthStatus{
					Status: kargoapi.ArgoCDAppHealthStateHealthy,
				}, healthStatus)
				require.Equal(t, kargoapi.ArgoCDAppSyncStatus{
					Status:    kargoapi.ArgoCDAppSyncStateSynced,
					Revisions: []string{"warehouse1-revision1", "warehouse2-revision2"},
				}, syncStatus)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(testCase.interceptor)
			if testCase.application != nil {
				c.WithObjects(testCase.application)
			}

			h := &applicationHealth{
				argoClient: c.Build(),
			}
			state, healthStatus, syncStatus, err := h.GetApplicationHealth(
				context.Background(),
				testCase.stage,
				&testCase.stage.Spec.PromotionMechanisms.ArgoCDAppUpdates[0],
				testCase.key,
			)
			testCase.assertions(t, state, healthStatus, syncStatus, err)
		})
	}

	t.Run("waits for operation cooldown", func(t *testing.T) {
		app := &argocd.Application{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "fake-namespace",
				Name:      "fake-name",
			},
			Spec: argocd.ApplicationSpec{
				Source: &argocd.ApplicationSource{
					RepoURL: "https://example.com/universe/42",
				},
			},
			Status: argocd.ApplicationStatus{
				Health: argocd.HealthStatus{
					Status: argocd.HealthStatusProgressing,
				},
				Sync: argocd.SyncStatus{
					Status:   argocd.SyncStatusCodeSynced,
					Revision: "fake-revision",
				},
				OperationState: &argocd.OperationState{
					FinishedAt: ptr.To(metav1.Now()),
				},
			},
		}

		var count int
		c := fake.NewClientBuilder().WithInterceptorFuncs(interceptor.Funcs{
			Get: func(
				_ context.Context,
				_ client.WithWatch,
				_ client.ObjectKey,
				obj client.Object,
				_ ...client.GetOption,
			) error {
				count++

				appCopy := app.DeepCopy()
				if count > 1 {
					appCopy.Status.Health.Status = argocd.HealthStatusHealthy
				}

				*obj.(*argocd.Application) = *appCopy // nolint: forcetypeassert
				return nil
			},
		})
		h := &applicationHealth{
			argoClient: c.Build(),
		}

		stage := &kargoapi.Stage{
			Spec: testStageSpec,
			Status: kargoapi.StageStatus{
				FreightHistory: kargoapi.FreightHistory{
					&kargoapi.FreightCollection{
						Freight: map[string]kargoapi.FreightReference{
							testOrigin.String(): {
								Origin: testOrigin,
								Commits: []kargoapi.GitCommit{{
									RepoURL: "https://example.com/universe/42",
									ID:      "fake-revision",
								}},
							},
						},
					},
				},
			},
		}
		_, _, _, err := h.GetApplicationHealth(
			context.Background(),
			stage,
			&stage.Spec.PromotionMechanisms.ArgoCDAppUpdates[0],
			types.NamespacedName{
				Namespace: "fake-namespace",
				Name:      "fake-name",
			},
		)
		elapsed := time.Since(app.Status.OperationState.FinishedAt.Time)

		require.NoError(t, err)

		// We wait for 10 seconds after the sync operation has finished.
		// As such, the elapsed time should be greater than 8 seconds,
		// but less than 12 seconds. To ensure we do not introduce
		// flakes in the tests.
		require.Greater(t, elapsed, 8*time.Second)
		require.Less(t, elapsed, 12*time.Second)
		require.Equal(t, 2, count)
	})
}

func Test_stageHealthForAppSync(t *testing.T) {
	tests := []struct {
		name       string
		app        *argocd.Application
		revisions  []string
		assertions func(*testing.T, kargoapi.HealthState, error)
	}{
		{
			name: "empty revision",
			app: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-namespace",
					Name:      "fake-name",
				},
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{},
				},
			},
			assertions: func(t *testing.T, state kargoapi.HealthState, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.HealthStateHealthy, state)
			},
		},
		{
			name:      "ongoing sync operation",
			revisions: []string{"fake-revision"},
			app: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-namespace",
					Name:      "fake-name",
				},
				Operation: &argocd.Operation{
					Sync: &argocd.SyncOperation{},
				},
			},
			assertions: func(t *testing.T, state kargoapi.HealthState, err error) {
				require.ErrorContains(t, err, "is being synced")
				require.Equal(t, kargoapi.HealthStateUnknown, state)
			},
		},
		{
			name:      "no operation state",
			revisions: []string{"fake-revision"},
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Revision: "fake-revision",
					},
				},
			},
			assertions: func(t *testing.T, state kargoapi.HealthState, err error) {
				require.ErrorContains(t, err, "is being synced")
				require.Equal(t, kargoapi.HealthStateUnknown, state)
			},
		},
		{
			name:      "operation state without finished time",
			revisions: []string{"fake-revision"},
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Revision: "fake-revision",
					},
					OperationState: &argocd.OperationState{},
				},
			},
			assertions: func(t *testing.T, state kargoapi.HealthState, err error) {
				require.ErrorContains(t, err, "is being synced")
				require.Equal(t, kargoapi.HealthStateUnknown, state)
			},
		},
		{
			name:      "sync revision mismatch",
			revisions: []string{"fake-revision"},
			app: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-namespace",
					Name:      "fake-name",
				},
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Revision: "other-fake-revision",
					},
					OperationState: &argocd.OperationState{
						FinishedAt: ptr.To(metav1.Now()),
					},
				},
			},
			assertions: func(t *testing.T, state kargoapi.HealthState, err error) {
				require.ErrorContains(t, err, "Desired revision [\"fake-revision\"]")
				require.ErrorContains(t, err, "does not match current revision \"other-fake-revision\"")
				require.Equal(t, kargoapi.HealthStateUnhealthy, state)
			},
		},
		{
			name:      "synced",
			revisions: []string{"fake-revision"},
			app: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-namespace",
					Name:      "fake-name",
				},
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Revision: "fake-revision",
					},
					OperationState: &argocd.OperationState{
						FinishedAt: &metav1.Time{Time: metav1.Now().Add(-10 * time.Second)},
					},
				},
			},
			assertions: func(t *testing.T, state kargoapi.HealthState, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.HealthStateHealthy, state)
			},
		},
	}
	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := stageHealthForAppSync(ctx, tt.app, tt.revisions)
			tt.assertions(t, got, err)
		})
	}
}

func Test_stageHealthForAppHealth(t *testing.T) {
	tests := []struct {
		name       string
		app        *argocd.Application
		assertions func(*testing.T, kargoapi.HealthState, error)
	}{
		{
			name: "progressing",
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: argocd.HealthStatusProgressing,
					},
				},
			},
			assertions: func(t *testing.T, state kargoapi.HealthState, err error) {
				require.ErrorContains(t, err, "is progressing")
				require.Equal(t, kargoapi.HealthStateProgressing, state)
			},
		},
		{
			name: "progressing (due to suspension)",
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: argocd.HealthStatusSuspended,
					},
				},
			},
			assertions: func(t *testing.T, state kargoapi.HealthState, err error) {
				require.ErrorContains(t, err, "is suspended")
				require.Equal(t, kargoapi.HealthStateProgressing, state)
			},
		},
		{
			name: "empty health status",
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{},
				},
			},
			assertions: func(t *testing.T, state kargoapi.HealthState, err error) {
				require.ErrorContains(t, err, "is progressing")
				require.Equal(t, kargoapi.HealthStateProgressing, state)
			},
		},
		{
			name: "healthy",
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: argocd.HealthStatusHealthy,
					},
				},
			},
			assertions: func(t *testing.T, state kargoapi.HealthState, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.HealthStateHealthy, state)
			},
		},
		{
			name: "degraded",
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: argocd.HealthStatusDegraded,
					},
				},
			},
			assertions: func(t *testing.T, state kargoapi.HealthState, err error) {
				require.ErrorContains(t, err, "has health state")
				require.Equal(t, kargoapi.HealthStateUnhealthy, state)
			},
		},
	}
	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := stageHealthForAppHealth(ctx, tt.app)
			tt.assertions(t, got, err)
		})
	}
}

func Test_filterAppConditions(t *testing.T) {
	tests := []struct {
		name       string
		conditions []argocd.ApplicationCondition
		types      []argocd.ApplicationConditionType
		assertions func(*testing.T, []argocd.ApplicationCondition)
	}{
		{
			name: "no conditions",
			assertions: func(t *testing.T, conditions []argocd.ApplicationCondition) {
				require.Len(t, conditions, 0)
			},
		},
		{
			name: "single matching condition",
			conditions: []argocd.ApplicationCondition{
				{
					Type: argocd.ApplicationConditionComparisonError,
				},
			},
			types: []argocd.ApplicationConditionType{
				argocd.ApplicationConditionComparisonError,
			},
			assertions: func(t *testing.T, conditions []argocd.ApplicationCondition) {
				require.Len(t, conditions, 1)
				require.Equal(t, argocd.ApplicationConditionComparisonError, conditions[0].Type)
			},
		},
		{
			name: "multiple matching conditions",
			conditions: []argocd.ApplicationCondition{
				{
					Type: argocd.ApplicationConditionComparisonError,
				},
				{
					Type: argocd.ApplicationConditionInvalidSpecError,
				},
				{
					Type: argocd.ApplicationConditionComparisonError,
				},
				{
					Type: "SomeOtherType",
				},
			},
			types: []argocd.ApplicationConditionType{
				argocd.ApplicationConditionComparisonError,
				"SomeOtherType",
			},
			assertions: func(t *testing.T, conditions []argocd.ApplicationCondition) {
				require.Len(t, conditions, 3)
				require.Equal(t, argocd.ApplicationConditionComparisonError, conditions[0].Type)
				require.Equal(t, argocd.ApplicationConditionComparisonError, conditions[1].Type)
				require.Equal(t, argocd.ApplicationConditionType("SomeOtherType"), conditions[2].Type)
			},
		}, {
			name: "no matching conditions",
			conditions: []argocd.ApplicationCondition{
				{
					Type: argocd.ApplicationConditionComparisonError,
				},
				{
					Type: argocd.ApplicationConditionInvalidSpecError,
				},
			},
			types: []argocd.ApplicationConditionType{
				"NonMatchingType",
			},
			assertions: func(t *testing.T, conditions []argocd.ApplicationCondition) {
				require.Len(t, conditions, 0)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &argocd.Application{
				Status: argocd.ApplicationStatus{
					Conditions: tt.conditions,
				},
			}
			got := filterAppConditions(app, tt.types...)
			tt.assertions(t, got)
		})
	}
}
