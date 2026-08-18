package builtin

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocd "github.com/akuity/kargo/pkg/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/pkg/health"
)

func Test_argocdUpdater_check(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, argocd.AddToScheme(scheme))

	const testAppNamespace = "fake-namespace"
	const testAppName1 = "fake-app"
	const testAppName2 = "another-fake-app"

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, health.Result)
	}{
		{
			name: "Argo CD integration disabled",
			assertions: func(t *testing.T, res health.Result) {
				require.Equal(t, kargoapi.HealthStateUnknown, res.Status)
				require.Len(t, res.Issues, 1)
				require.Contains(t, res.Issues[0], "Argo CD integration is disabled")
			},
		},
		{
			name: "composite error checking Application health",
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(
					&argocd.Application{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testAppNamespace,
							Name:      testAppName1,
						},
						Spec: argocd.ApplicationSpec{
							Sources: []argocd.ApplicationSource{{}},
						},
						Status: argocd.ApplicationStatus{
							OperationState: &argocd.OperationState{
								Phase: argocd.OperationSucceeded,
								FinishedAt: &metav1.Time{
									Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
								},
							},
							ReconciledAt: &metav1.Time{
								Time: time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC),
							},
							Health: argocd.HealthStatus{
								Status: argocd.HealthStatusHealthy,
							},
							Sync: argocd.SyncStatus{
								Status:    argocd.SyncStatusCodeSynced,
								Revisions: []string{"fake-version"},
							},
						},
					},
					&argocd.Application{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testAppNamespace,
							Name:      testAppName2,
						},
						Status: argocd.ApplicationStatus{
							OperationState: &argocd.OperationState{
								Phase: argocd.OperationSucceeded,
								FinishedAt: &metav1.Time{
									Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
								},
							},
							ReconciledAt: &metav1.Time{
								Time: time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC),
							},
							Conditions: []argocd.ApplicationCondition{
								{Type: argocd.ApplicationConditionComparisonError},
								{Type: argocd.ApplicationConditionInvalidSpecError},
							},
						},
					},
				).
				Build(),
			assertions: func(t *testing.T, res health.Result) {
				require.Equal(t, kargoapi.HealthStateUnhealthy, res.Status)
				require.Contains(t, res.Output, applicationStatusesKey)
				require.Len(t, res.Issues, 2)
				require.Contains(t, res.Issues[0], testAppName1)
				require.Contains(t, res.Issues[0], "ComparisonError")
				require.Contains(t, res.Issues[1], testAppName1)
				require.Contains(t, res.Issues[1], "InvalidSpecError")
			},
		},
		{
			name: "all apps healthy and synced",
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(
					&argocd.Application{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testAppNamespace,
							Name:      testAppName1,
						},
						Spec: argocd.ApplicationSpec{
							Sources: []argocd.ApplicationSource{{}},
						},
						Status: argocd.ApplicationStatus{
							OperationState: &argocd.OperationState{
								Phase: argocd.OperationSucceeded,
								FinishedAt: &metav1.Time{
									Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
								},
							},
							ReconciledAt: &metav1.Time{
								Time: time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC),
							},
							Health: argocd.HealthStatus{
								Status: argocd.HealthStatusHealthy,
							},
							Sync: argocd.SyncStatus{
								Status:    argocd.SyncStatusCodeSynced,
								Revisions: []string{"fake-version"},
							},
						},
					},
					&argocd.Application{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testAppNamespace,
							Name:      testAppName2,
						},
						Spec: argocd.ApplicationSpec{
							Sources: []argocd.ApplicationSource{{}},
						},
						Status: argocd.ApplicationStatus{
							OperationState: &argocd.OperationState{
								Phase: argocd.OperationSucceeded,
								FinishedAt: &metav1.Time{
									Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
								},
							},
							ReconciledAt: &metav1.Time{
								Time: time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC),
							},
							Health: argocd.HealthStatus{
								Status: argocd.HealthStatusHealthy,
							},
							Sync: argocd.SyncStatus{
								Status:    argocd.SyncStatusCodeSynced,
								Revisions: []string{"fake-commit"},
							},
						},
					},
				).
				Build(),
			assertions: func(t *testing.T, res health.Result) {
				require.Equal(t, kargoapi.HealthStateHealthy, res.Status)
				require.Contains(t, res.Output, applicationStatusesKey)
				require.Empty(t, res.Issues)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			runner := &argocdChecker{
				argocdClient: testCase.client,
			}
			testCase.assertions(
				t,
				runner.check(
					t.Context(),
					ArgoCDHealthInput{
						Apps: []ArgoCDAppHealthCheck{
							{
								Namespace:        testAppNamespace,
								Name:             testAppName1,
								DesiredRevisions: []string{"fake-version"},
							},
							{
								Namespace:        testAppNamespace,
								Name:             testAppName2,
								DesiredRevisions: []string{"fake-commit"},
							},
						},
					},
				),
			)
		})
	}
}

func Test_argocdUpdater_getApplicationHealth(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, argocd.AddToScheme(scheme))

	testApp := &argocd.Application{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "fake-namespace",
			Name:      "fake-name",
		},
		Spec: argocd.ApplicationSpec{
			Sources: []argocd.ApplicationSource{
				{
					RepoURL: "https://example.com",
					Chart:   "fake-chart",
				},
				{
					RepoURL: "https://example.com/universe/42",
				},
				{
					RepoURL: "https://example.com/another-universe/42",
				},
			},
		},
	}

	testCases := []struct {
		name             string
		appStatus        argocd.ApplicationStatus
		interceptor      interceptor.Funcs
		desiredRevisions []string
		assertions       func(
			*testing.T,
			kargoapi.HealthState,
			ArgoCDAppStatus,
			client.Client,
			error,
		)
	}{
		{
			name: "Application not found",
			interceptor: interceptor.Funcs{
				Get: func(
					context.Context,
					client.WithWatch,
					client.ObjectKey,
					client.Object,
					...client.GetOption,
				) error {
					// return not found error
					return apierrors.NewNotFound(schema.GroupResource{}, "")
				},
			},
			assertions: func(
				t *testing.T,
				stageHealth kargoapi.HealthState,
				appStatus ArgoCDAppStatus,
				_ client.Client,
				err error,
			) {
				require.ErrorContains(t, err, "unable to find Argo CD Application")
				require.Equal(t, kargoapi.HealthStateUnknown, stageHealth)
				require.Equal(t, argocd.HealthStatusUnknown, appStatus.Health.Status)
				require.Equal(t, argocd.SyncStatusCodeUnknown, appStatus.Sync.Status)
			},
		},
		{
			name: "error getting Application",
			interceptor: interceptor.Funcs{
				Get: func(
					context.Context,
					client.WithWatch,
					client.ObjectKey,
					client.Object,
					...client.GetOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				stageHealth kargoapi.HealthState,
				appStatus ArgoCDAppStatus,
				_ client.Client,
				err error,
			) {
				require.ErrorContains(t, err, "error finding Argo CD Application")
				require.ErrorContains(t, err, "something went wrong")
				require.Equal(t, kargoapi.HealthStateUnknown, stageHealth)
				require.Equal(t, argocd.HealthStatusUnknown, appStatus.Health.Status)
				require.Equal(t, argocd.SyncStatusCodeUnknown, appStatus.Sync.Status)
			},
		},
		{
			name: "Application not reconciled after last operation",
			appStatus: argocd.ApplicationStatus{
				OperationState: &argocd.OperationState{
					Phase: argocd.OperationSucceeded,
					FinishedAt: &metav1.Time{
						Time: time.Date(2024, 1, 1, 0, 0, 10, 0, time.UTC),
					},
				},
				Health: argocd.HealthStatus{Status: argocd.HealthStatusHealthy},
			},
			assertions: func(
				t *testing.T,
				stageHealth kargoapi.HealthState,
				appStatus ArgoCDAppStatus,
				c client.Client,
				err error,
			) {
				require.Error(t, err)
				require.ErrorContains(t, err, "was not reconciled after")
				require.ErrorContains(t, err, "Application health status not trusted")
				require.Equal(t, kargoapi.HealthStateUnknown, stageHealth)
				require.Equal(t, testApp.Namespace, appStatus.Namespace)
				require.Equal(t, testApp.Name, appStatus.Name)
				require.Equal(t, argocd.HealthStatusHealthy, appStatus.Health.Status)
				// Verify hard refresh was requested
				app := &argocd.Application{}
				require.NoError(t, c.Get(
					context.Background(),
					client.ObjectKey{
						Namespace: testApp.Namespace,
						Name:      testApp.Name,
					},
					app,
				))
				require.Equal(
					t,
					string(argocd.RefreshTypeHard),
					app.Annotations[argocd.AnnotationKeyRefresh],
				)
			},
		},
		{
			name: "Application reconciled before last operation completed",
			appStatus: argocd.ApplicationStatus{
				OperationState: &argocd.OperationState{
					Phase: argocd.OperationSucceeded,
					FinishedAt: &metav1.Time{
						Time: time.Date(2024, 1, 1, 0, 0, 10, 0, time.UTC),
					},
				},
				ReconciledAt: &metav1.Time{
					Time: time.Date(2024, 1, 1, 0, 0, 5, 0, time.UTC),
				},
				Health: argocd.HealthStatus{Status: argocd.HealthStatusHealthy},
			},
			assertions: func(
				t *testing.T,
				stageHealth kargoapi.HealthState,
				appStatus ArgoCDAppStatus,
				c client.Client,
				err error,
			) {
				require.Error(t, err)
				require.ErrorContains(t, err, "was not reconciled after")
				require.ErrorContains(t, err, "Application health status not trusted")
				require.Equal(t, kargoapi.HealthStateUnknown, stageHealth)
				require.Equal(t, testApp.Namespace, appStatus.Namespace)
				require.Equal(t, testApp.Name, appStatus.Name)
				require.Equal(t, argocd.HealthStatusHealthy, appStatus.Health.Status)
				// Verify hard refresh was requested
				app := &argocd.Application{}
				require.NoError(t, c.Get(
					context.Background(),
					client.ObjectKey{
						Namespace: testApp.Namespace,
						Name:      testApp.Name,
					},
					app,
				))
				require.Equal(
					t,
					string(argocd.RefreshTypeHard),
					app.Annotations[argocd.AnnotationKeyRefresh],
				)
			},
		},
		{
			name: "Application has error conditions",
			appStatus: argocd.ApplicationStatus{
				OperationState: &argocd.OperationState{
					Phase: argocd.OperationSucceeded,
					FinishedAt: &metav1.Time{
						Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					},
				},
				ReconciledAt: &metav1.Time{
					Time: time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC),
				},
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
			assertions: func(
				t *testing.T,
				stageHealth kargoapi.HealthState,
				appStatus ArgoCDAppStatus,
				_ client.Client,
				err error,
			) {
				require.Error(t, err)
				require.ErrorContains(t, err, `has "ComparisonError" condition`)
				require.ErrorContains(t, err, `has "InvalidSpecError" condition`)
				unwrappedErr, ok := err.(compositeError)
				require.True(t, ok)
				require.Len(t, unwrappedErr.Unwrap(), 2)
				require.Equal(t, kargoapi.HealthStateUnhealthy, stageHealth)
				require.Equal(t, testApp.Namespace, appStatus.Namespace)
				require.Equal(t, testApp.Name, appStatus.Name)
				require.Equal(t, argocd.HealthStatusHealthy, appStatus.Health.Status)
				require.Equal(t, argocd.SyncStatusCodeSynced, appStatus.Sync.Status)
			},
		},
		{
			name: "Application is not Healthy",
			appStatus: argocd.ApplicationStatus{
				OperationState: &argocd.OperationState{
					Phase: argocd.OperationSucceeded,
					FinishedAt: &metav1.Time{
						Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					},
				},
				ReconciledAt: &metav1.Time{
					Time: time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC),
				},
				Health: argocd.HealthStatus{
					Status:  argocd.HealthStatusDegraded,
					Message: "fake-message",
				},
				Sync: argocd.SyncStatus{
					Status: argocd.SyncStatusCodeSynced,
				},
			},
			assertions: func(
				t *testing.T,
				stageHealth kargoapi.HealthState,
				appStatus ArgoCDAppStatus,
				_ client.Client,
				err error,
			) {
				require.ErrorContains(t, err, "Argo CD Application")
				require.ErrorContains(t, err, "has health state")
				require.ErrorContains(t, err, string(argocd.HealthStatusDegraded))
				require.Equal(t, kargoapi.HealthStateUnhealthy, stageHealth)
				require.Equal(t, testApp.Namespace, appStatus.Namespace)
				require.Equal(t, testApp.Name, appStatus.Name)
				require.Equal(t, argocd.HealthStatusDegraded, appStatus.Health.Status)
				require.Equal(t, argocd.SyncStatusCodeSynced, appStatus.Sync.Status)
			},
		},
		{
			name: "Application is Healthy and check has no desired revisions",
			appStatus: argocd.ApplicationStatus{
				OperationState: &argocd.OperationState{
					Phase: argocd.OperationSucceeded,
					FinishedAt: &metav1.Time{
						Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					},
				},
				ReconciledAt: &metav1.Time{
					Time: time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC),
				},
				Health: argocd.HealthStatus{
					Status:  argocd.HealthStatusHealthy,
					Message: "fake-message",
				},
				Sync: argocd.SyncStatus{
					Status: argocd.SyncStatusCodeSynced,
				},
			},
			assertions: func(
				t *testing.T,
				stageHealth kargoapi.HealthState,
				appStatus ArgoCDAppStatus,
				_ client.Client,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.HealthStateHealthy, stageHealth)
				require.Equal(t, testApp.Namespace, appStatus.Namespace)
				require.Equal(t, testApp.Name, appStatus.Name)
				require.Equal(t, argocd.HealthStatusHealthy, appStatus.Health.Status)
				require.Equal(t, argocd.SyncStatusCodeSynced, appStatus.Sync.Status)
			},
		},
		{
			name: "Application is Healthy, but not synced to desired revisions",
			appStatus: argocd.ApplicationStatus{
				OperationState: &argocd.OperationState{
					Phase: argocd.OperationSucceeded,
					FinishedAt: &metav1.Time{
						Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					},
				},
				ReconciledAt: &metav1.Time{
					Time: time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC),
				},
				Health: argocd.HealthStatus{
					Status: argocd.HealthStatusHealthy,
				},
				Sync: argocd.SyncStatus{
					Status:    argocd.SyncStatusCodeSynced,
					Revisions: []string{"fake-version", "wrong-fake-commit", "another-fake-commit"},
				},
			},
			desiredRevisions: []string{"fake-version", "fake-commit", "another-fake-commit"},
			assertions: func(
				t *testing.T,
				stageHealth kargoapi.HealthState,
				appStatus ArgoCDAppStatus,
				_ client.Client,
				err error,
			) {
				require.ErrorContains(t, err, "Not all sources of Application")
				require.ErrorContains(t, err, "are synced to the desired revisions")
				require.ErrorContains(t, err, "Source 1 with RepoURL https://example.com/universe/42")
				require.Equal(t, kargoapi.HealthStateUnhealthy, stageHealth)
				require.Equal(t, testApp.Namespace, appStatus.Namespace)
				require.Equal(t, testApp.Name, appStatus.Name)
				require.Equal(t, argocd.HealthStatusHealthy, appStatus.Health.Status)
				require.Equal(t, argocd.SyncStatusCodeSynced, appStatus.Sync.Status)
			},
		},
		{
			name: "Application is Healthy and synced to desired revisions",
			appStatus: argocd.ApplicationStatus{
				OperationState: &argocd.OperationState{
					Phase: argocd.OperationSucceeded,
					FinishedAt: &metav1.Time{
						Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					},
				},
				ReconciledAt: &metav1.Time{
					Time: time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC),
				},
				Health: argocd.HealthStatus{
					Status: argocd.HealthStatusHealthy,
				},
				Sync: argocd.SyncStatus{
					Status:    argocd.SyncStatusCodeSynced,
					Revisions: []string{"fake-version", "fake-commit", "another-fake-commit"},
				},
			},
			desiredRevisions: []string{"fake-version", "fake-commit", "another-fake-commit"},
			assertions: func(
				t *testing.T,
				stageHealth kargoapi.HealthState,
				appStatus ArgoCDAppStatus,
				_ client.Client,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.HealthStateHealthy, stageHealth)
				require.Equal(t, testApp.Namespace, appStatus.Namespace)
				require.Equal(t, testApp.Name, appStatus.Name)
				require.Equal(t, argocd.HealthStatusHealthy, appStatus.Health.Status)
				require.Equal(t, argocd.SyncStatusCodeSynced, appStatus.Sync.Status)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			app := testApp.DeepCopy()
			app.Status = testCase.appStatus
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(app).
				WithInterceptorFuncs(testCase.interceptor).
				Build()
			runner := &argocdChecker{argocdClient: fakeClient}
			stageHealth, appStatus, err := runner.getApplicationHealth(
				t.Context(),
				client.ObjectKey{
					Namespace: app.Namespace,
					Name:      app.Name,
				},
				testCase.desiredRevisions,
			)
			testCase.assertions(t, stageHealth, appStatus, fakeClient, err)
		})
	}
}

func Test_argocdUpdater_stageHealthForAppSync(t *testing.T) {
	testCases := []struct {
		name       string
		app        *argocd.Application
		revisions  []string
		assertions func(*testing.T, kargoapi.HealthState, error)
	}{
		{
			name: "empty revisions list",
			assertions: func(t *testing.T, health kargoapi.HealthState, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.HealthStateHealthy, health)
			},
		},
		{
			name:      "all revisions are empty string",
			revisions: []string{"", ""},
			assertions: func(t *testing.T, health kargoapi.HealthState, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.HealthStateHealthy, health)
			},
		},
		{
			name:      "ongoing sync operation",
			revisions: []string{"fake-revision"},
			app: &argocd.Application{
				Operation: &argocd.Operation{
					Sync: &argocd.SyncOperation{},
				},
			},
			assertions: func(t *testing.T, health kargoapi.HealthState, err error) {
				require.ErrorContains(t, err, "is being synced")
				require.Equal(t, kargoapi.HealthStateUnknown, health)
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
			assertions: func(t *testing.T, health kargoapi.HealthState, err error) {
				require.ErrorContains(t, err, "is being synced")
				require.Equal(t, kargoapi.HealthStateUnknown, health)
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
			assertions: func(t *testing.T, health kargoapi.HealthState, err error) {
				require.ErrorContains(t, err, "is being synced")
				require.Equal(t, kargoapi.HealthStateUnknown, health)
			},
		},
		{
			name:      "sync revision mismatch",
			revisions: []string{"fake-revision", "another-fake-revision"},
			app: &argocd.Application{
				Spec: argocd.ApplicationSpec{
					Sources: []argocd.ApplicationSource{{}, {}},
				},
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Revisions: []string{"fake-revision", "wrong-fake-revision"},
					},
					OperationState: &argocd.OperationState{
						FinishedAt: ptr.To(metav1.Now()),
					},
				},
			},
			assertions: func(t *testing.T, health kargoapi.HealthState, err error) {
				require.ErrorContains(t, err, "Not all sources of Application")
				require.ErrorContains(t, err, "are synced to the desired revisions")
				require.Equal(t, kargoapi.HealthStateUnhealthy, health)
			},
		},
		{
			name:      "synced",
			revisions: []string{"fake-revision", "another-fake-revision"},
			app: &argocd.Application{
				Spec: argocd.ApplicationSpec{
					Sources: []argocd.ApplicationSource{{}, {}},
				},
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Revisions: []string{"fake-revision", "another-fake-revision"},
					},
					OperationState: &argocd.OperationState{
						FinishedAt: ptr.To(metav1.Now()),
					},
				},
			},
			assertions: func(t *testing.T, state kargoapi.HealthState, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.HealthStateHealthy, state)
			},
		},
	}

	runner := &argocdChecker{}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			health, err := runner.stageHealthForAppSync(
				testCase.app,
				testCase.revisions,
			)
			testCase.assertions(t, health, err)
		})
	}
}

func Test_argocdUpdater_stageHealthForAppHealth(t *testing.T) {
	testCases := []struct {
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

	runner := &argocdChecker{}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := runner.stageHealthForAppHealth(testCase.app)
			testCase.assertions(t, got, err)
		})
	}
}

func Test_argocdUpdater_filterAppConditions(t *testing.T) {
	testCases := []struct {
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
		},
		{
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

	runner := (&argocdChecker{})

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				runner.filterAppConditions(
					&argocd.Application{
						Status: argocd.ApplicationStatus{
							Conditions: testCase.conditions,
						},
					},
					testCase.types...,
				),
			)
		})
	}
}

func Test_sanitizeAppStatus(t *testing.T) {
	original := argocd.ApplicationStatus{
		Health: argocd.HealthStatus{
			Status: argocd.HealthStatusHealthy,
		},
		Sync: argocd.SyncStatus{
			Status:    argocd.SyncStatusCodeSynced,
			Revisions: []string{"fake-revision"},
		},
		ReconciledAt: &metav1.Time{
			Time: time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC),
		},
		OperationState: &argocd.OperationState{
			Phase: argocd.OperationSucceeded,
			FinishedAt: &metav1.Time{
				Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		Conditions: []argocd.ApplicationCondition{
			{
				Type:    argocd.ApplicationConditionComparisonError,
				Message: "fake-message",
				LastTransitionTime: &metav1.Time{
					Time: time.Date(2024, 1, 1, 0, 0, 2, 0, time.UTC),
				},
			},
		},
	}

	// Snapshot the input so mutations through shared memory can be detected.
	snapshot := *original.DeepCopy()

	sanitized := sanitizeAppStatus(original)

	// Volatile fields are removed.
	require.Nil(t, sanitized.ReconciledAt)
	require.Len(t, sanitized.Conditions, 1)
	require.Nil(t, sanitized.Conditions[0].LastTransitionTime)

	// Meaningful fields are preserved. Compare against the snapshot rather
	// than the input so a mutation through shared memory cannot make an
	// assertion compare a value to itself.
	require.Equal(t, snapshot.Health, sanitized.Health)
	require.Equal(t, snapshot.Sync, sanitized.Sync)
	require.Equal(t, snapshot.OperationState, sanitized.OperationState)
	require.Equal(t, snapshot.Conditions[0].Type, sanitized.Conditions[0].Type)
	require.Equal(t, snapshot.Conditions[0].Message, sanitized.Conditions[0].Message)

	// The input is entirely unmodified and shares no memory with the result.
	require.Equal(t, snapshot, original)
	require.NotSame(t, original.OperationState, sanitized.OperationState)
}

// Test_sanitizeAppStatus_accountsForAllTimestampFields walks the
// v1alpha1.ApplicationStatus type and fails when it contains a timestamp
// field that has not been explicitly accounted for below. This forces whoever
// adds a timestamp-bearing field to the vendored type to decide whether
// sanitizeAppStatus must strip it. See the comment on ApplicationStatus.
func Test_sanitizeAppStatus_accountsForAllTimestampFields(t *testing.T) {
	accountedFor := map[string]bool{
		// Stripped: ticks on every Argo CD refresh cycle.
		"ReconciledAt": true,
		// Stripped: re-stamped when a condition clears and later recurs.
		"Conditions[].LastTransitionTime": true,
		// Kept: changes only when an operation completes.
		"OperationState.FinishedAt": true,
	}

	timeType := reflect.TypeFor[metav1.Time]()
	var timestampFields []string
	ancestors := map[reflect.Type]bool{}
	var walk func(reflect.Type, string)
	walk = func(typ reflect.Type, path string) {
		for typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Slice {
			if typ.Kind() == reflect.Slice {
				path += "[]"
			}
			typ = typ.Elem()
		}
		if typ == timeType {
			timestampFields = append(timestampFields, path)
			return
		}
		if typ.Kind() != reflect.Struct || ancestors[typ] {
			return
		}
		ancestors[typ] = true
		for field := range typ.Fields() {
			fieldPath := field.Name
			if path != "" {
				fieldPath = path + "." + field.Name
			}
			walk(field.Type, fieldPath)
		}
		delete(ancestors, typ)
	}
	walk(reflect.TypeFor[argocd.ApplicationStatus](), "")

	for _, field := range timestampFields {
		require.True(
			t, accountedFor[field],
			"timestamp field %q is not accounted for; decide whether sanitizeAppStatus must strip it",
			field,
		)
	}
	require.Len(t, timestampFields, len(accountedFor))
}

// Test_argocdChecker_check_outputStableAcrossRefreshes verifies that a routine
// Argo CD refresh -- which advances reconciledAt and re-stamps condition
// lastTransitionTimes without any meaningful change to the Application -- does
// not alter the health check output. The output is persisted in Stage status
// with a diff-based patch, so any instability here produces a status patch and
// a watch event for every Stage on every reconciliation.
func Test_argocdChecker_check_outputStableAcrossRefreshes(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, argocd.AddToScheme(scheme))

	appBefore := &argocd.Application{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "fake-namespace",
			Name:      "fake-app",
		},
		Spec: argocd.ApplicationSpec{
			Sources: []argocd.ApplicationSource{{}},
		},
		Status: argocd.ApplicationStatus{
			OperationState: &argocd.OperationState{
				Phase: argocd.OperationSucceeded,
				FinishedAt: &metav1.Time{
					Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			ReconciledAt: &metav1.Time{
				Time: time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC),
			},
			Health: argocd.HealthStatus{
				Status: argocd.HealthStatusHealthy,
			},
			Sync: argocd.SyncStatus{
				Status:    argocd.SyncStatusCodeSynced,
				Revisions: []string{"fake-revision"},
			},
			Conditions: []argocd.ApplicationCondition{
				{
					Type:    argocd.ApplicationConditionComparisonError,
					Message: "fake-message",
					LastTransitionTime: &metav1.Time{
						Time: time.Date(2024, 1, 1, 0, 0, 2, 0, time.UTC),
					},
				},
			},
		},
	}

	// Same Application after a routine refresh: reconciledAt advanced and the
	// condition was re-stamped, but nothing else changed.
	appAfter := appBefore.DeepCopy()
	appAfter.Status.ReconciledAt = &metav1.Time{
		Time: time.Date(2024, 1, 1, 0, 5, 1, 0, time.UTC),
	}
	appAfter.Status.Conditions[0].LastTransitionTime = &metav1.Time{
		Time: time.Date(2024, 1, 1, 0, 5, 2, 0, time.UTC),
	}

	input := ArgoCDHealthInput{
		Apps: []ArgoCDAppHealthCheck{
			{
				Namespace:        "fake-namespace",
				Name:             "fake-app",
				DesiredRevisions: []string{"fake-revision"},
			},
		},
	}

	var results []health.Result
	for _, app := range []*argocd.Application{appBefore, appAfter} {
		checker := &argocdChecker{
			argocdClient: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(app).
				Build(),
		}
		results = append(results, checker.check(t.Context(), input))
	}

	// Guard against vacuous stability: the output must reflect the real,
	// sanitized Application status.
	require.Equal(t, kargoapi.HealthStateUnhealthy, results[0].Status)
	require.Contains(t, results[0].Output, applicationStatusesKey)
	appStatuses, ok := results[0].Output[applicationStatusesKey].([]ArgoCDAppStatus)
	require.True(t, ok)
	require.Len(t, appStatuses, 1)
	require.Equal(t, argocd.HealthStatusHealthy, appStatuses[0].Health.Status)
	require.Equal(t, argocd.SyncStatusCodeSynced, appStatuses[0].Sync.Status)
	require.Nil(t, appStatuses[0].ReconciledAt)
	require.Len(t, appStatuses[0].Conditions, 1)
	require.Nil(t, appStatuses[0].Conditions[0].LastTransitionTime)

	require.Equal(t, results[0], results[1])
}
