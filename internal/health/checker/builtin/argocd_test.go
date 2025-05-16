package builtin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
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
							Health: argocd.HealthStatus{
								Status: argocd.HealthStatusHealthy,
							},
							Sync: argocd.SyncStatus{
								Status:    argocd.SyncStatusCodeSynced,
								Revisions: []string{"fake-version"},
							},
							OperationState: &argocd.OperationState{
								FinishedAt: ptr.To(metav1.Now()),
							},
						},
					},
					&argocd.Application{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testAppNamespace,
							Name:      testAppName2,
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
							Health: argocd.HealthStatus{
								Status: argocd.HealthStatusHealthy,
							},
							Sync: argocd.SyncStatus{
								Status:    argocd.SyncStatusCodeSynced,
								Revisions: []string{"fake-version"},
							},
							OperationState: &argocd.OperationState{
								FinishedAt: ptr.To(metav1.Now()),
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
							Health: argocd.HealthStatus{
								Status: argocd.HealthStatusHealthy,
							},
							Sync: argocd.SyncStatus{
								Status:    argocd.SyncStatusCodeSynced,
								Revisions: []string{"fake-commit"},
							},
							OperationState: &argocd.OperationState{
								FinishedAt: ptr.To(metav1.Now()),
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
					context.Background(),
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
		assertions       func(*testing.T, kargoapi.HealthState, ArgoCDAppStatus, error)
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
					return kubeerr.NewNotFound(schema.GroupResource{}, "")
				},
			},
			assertions: func(
				t *testing.T,
				stageHealth kargoapi.HealthState,
				appStatus ArgoCDAppStatus,
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
			name: "Application has error conditions",
			appStatus: argocd.ApplicationStatus{
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
				require.Equal(t, argocd.HealthStatusHealthy, appStatus.ApplicationStatus.Health.Status)
				require.Equal(t, argocd.SyncStatusCodeSynced, appStatus.ApplicationStatus.Sync.Status)
			},
		},
		{
			name: "no error conditions and no desired revisions",
			appStatus: argocd.ApplicationStatus{
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
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.HealthStateHealthy, stageHealth)
				require.Equal(t, testApp.Namespace, appStatus.Namespace)
				require.Equal(t, testApp.Name, appStatus.Name)
				require.Equal(t, argocd.HealthStatusHealthy, appStatus.ApplicationStatus.Health.Status)
				require.Equal(t, argocd.SyncStatusCodeSynced, appStatus.ApplicationStatus.Sync.Status)
			},
		},
		{
			name: "no error conditions, but revisions out of sync",
			appStatus: argocd.ApplicationStatus{
				Health: argocd.HealthStatus{
					Status: argocd.HealthStatusHealthy,
				},
				Sync: argocd.SyncStatus{
					Status:    argocd.SyncStatusCodeSynced,
					Revisions: []string{"fake-version", "wrong-fake-commit", "another-fake-commit"},
				},
				OperationState: &argocd.OperationState{
					FinishedAt: ptr.To(metav1.Now()),
				},
			},
			desiredRevisions: []string{"fake-version", "fake-commit", "another-fake-commit"},
			assertions: func(
				t *testing.T,
				stageHealth kargoapi.HealthState,
				appStatus ArgoCDAppStatus,
				err error,
			) {
				require.ErrorContains(t, err, "Not all sources of Application")
				require.ErrorContains(t, err, "are synced to the desired revisions")
				require.ErrorContains(t, err, "Source 1 with RepoURL https://example.com/universe/42")
				require.Equal(t, kargoapi.HealthStateUnhealthy, stageHealth)
				require.Equal(t, testApp.Namespace, appStatus.Namespace)
				require.Equal(t, testApp.Name, appStatus.Name)
				require.Equal(t, argocd.HealthStatusHealthy, appStatus.ApplicationStatus.Health.Status)
				require.Equal(t, argocd.SyncStatusCodeSynced, appStatus.ApplicationStatus.Sync.Status)
			},
		},
		{
			name: "no error conditions and revisions in sync",
			appStatus: argocd.ApplicationStatus{
				Health: argocd.HealthStatus{
					Status: argocd.HealthStatusHealthy,
				},
				Sync: argocd.SyncStatus{
					Status:    argocd.SyncStatusCodeSynced,
					Revisions: []string{"fake-version", "fake-commit", "another-fake-commit"},
				},
				OperationState: &argocd.OperationState{
					FinishedAt: &metav1.Time{Time: metav1.Now().Add(-10 * time.Second)},
				},
			},
			desiredRevisions: []string{"fake-version", "fake-commit", "another-fake-commit"},
			assertions: func(
				t *testing.T,
				stageHealth kargoapi.HealthState,
				appStatus ArgoCDAppStatus,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.HealthStateHealthy, stageHealth)
				require.Equal(t, testApp.Namespace, appStatus.Namespace)
				require.Equal(t, testApp.Name, appStatus.Name)
				require.Equal(t, argocd.HealthStatusHealthy, appStatus.ApplicationStatus.Health.Status)
				require.Equal(t, argocd.SyncStatusCodeSynced, appStatus.ApplicationStatus.Sync.Status)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			app := testApp.DeepCopy()
			app.Status = testCase.appStatus
			runner := &argocdChecker{
				argocdClient: fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(app).
					WithInterceptorFuncs(testCase.interceptor).
					Build(),
			}
			stageHealth, appStatus, err := runner.getApplicationHealth(
				context.Background(),
				client.ObjectKey{
					Namespace: app.Namespace,
					Name:      app.Name,
				},
				testCase.desiredRevisions,
			)
			testCase.assertions(t, stageHealth, appStatus, err)
		})
	}

	t.Run("waits for operation cooldown", func(t *testing.T) {
		app := testApp.DeepCopy()
		app.Status = argocd.ApplicationStatus{
			Health: argocd.HealthStatus{
				Status: argocd.HealthStatusProgressing,
			},
			Sync: argocd.SyncStatus{
				Status:    argocd.SyncStatusCodeSynced,
				Revisions: []string{"fake-version", "fake-commit", "another-fake-commit"},
			},
			OperationState: &argocd.OperationState{
				FinishedAt: ptr.To(metav1.Now()),
			},
		}
		var count int
		runner := &argocdChecker{
			argocdClient: fake.NewClientBuilder().WithInterceptorFuncs(interceptor.Funcs{
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
			}).Build(),
		}
		_, _, err := runner.getApplicationHealth(
			context.Background(),
			client.ObjectKey{
				Namespace: testApp.Namespace,
				Name:      testApp.Name,
			},
			[]string{"fake-version", "fake-commit", "another-fake-commit"},
		)
		elapsed := time.Since(app.Status.OperationState.FinishedAt.Time)
		require.NoError(t, err)
		// We wait for 10 seconds after the sync operation has finished. As such,
		// the elapsed time should be greater than 8 seconds, but less than 12
		// seconds. To ensure we do not introduce flakes in the tests.
		require.Greater(t, elapsed, 8*time.Second)
		require.Less(t, elapsed, 12*time.Second)
		require.Equal(t, 2, count)
	})
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
