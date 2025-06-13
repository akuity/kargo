package warehouses

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/conditions"
	"github.com/akuity/kargo/internal/credentials"
)

func TestNewReconciler(t *testing.T) {
	kubeClient := fake.NewClientBuilder().Build()
	minReconciliationInterval := time.Duration(1000)

	e := newReconciler(
		kubeClient,
		&credentials.FakeDB{},
		minReconciliationInterval,
	)
	require.NotNil(t, e.client)
	require.NotNil(t, e.credentialsDB)
	require.Equal(t, minReconciliationInterval, e.minReconciliationInterval)
	require.NotEmpty(t, e.imageSourceURLFnsByBaseURL)

	// Assert that all overridable behaviors were initialized to a default:
	require.NotNil(t, e.discoverArtifactsFn)
	require.NotNil(t, e.discoverCommitsFn)
	require.NotNil(t, e.discoverImagesFn)
	require.NotNil(t, e.discoverChartsFn)
	require.NotNil(t, e.buildFreightFromLatestArtifactsFn)
	require.NotNil(t, e.listCommitsFn)
	require.NotNil(t, e.listTagsFn)
	require.NotNil(t, e.discoverBranchHistoryFn)
	require.NotNil(t, e.discoverTagsFn)
	require.NotNil(t, e.getDiffPathsForCommitIDFn)
	require.NotNil(t, e.createFreightFn)
	require.NotNil(t, e.patchStatusFn)
}

func TestSyncWarehouse(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		warehouse  *kargoapi.Warehouse
		assertions func(*testing.T, kargoapi.WarehouseStatus, error)
	}{
		{
			name: "error discovering latest artifacts",
			reconciler: &reconciler{
				discoverArtifactsFn: func(context.Context, *kargoapi.Warehouse) (*kargoapi.DiscoveredArtifacts, error) {
					return nil, errors.New("something went wrong")
				},
				patchStatusFn: func(context.Context, *kargoapi.Warehouse, func(*kargoapi.WarehouseStatus)) error {
					return nil
				},
			},
			warehouse: &kargoapi.Warehouse{
				Status: kargoapi.WarehouseStatus{
					DiscoveredArtifacts: &kargoapi.DiscoveredArtifacts{},
				},
			},
			assertions: func(t *testing.T, status kargoapi.WarehouseStatus, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error discovering artifacts")

				// Ensure previous discovered artifacts are preserved.
				require.NotNil(t, status.DiscoveredArtifacts)

				require.Len(t, status.GetConditions(), 3)

				// Ensure that the Ready condition is set to False.
				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "DiscoveryFailure", readyCondition.Reason)
				require.Contains(t, readyCondition.Message, "something went wrong")

				// Ensure that the Reconciling condition is still set to True.
				reconcilingCondition := conditions.Get(&status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, reconcilingCondition)
				require.Equal(t, metav1.ConditionTrue, reconcilingCondition.Status)

				// Ensure that the Healthy condition is set to False.
				healthyCondition := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCondition)
				require.Equal(t, metav1.ConditionFalse, healthyCondition.Status)
				require.Equal(t, "DiscoveryFailed", healthyCondition.Reason)
				require.Contains(t, healthyCondition.Message, "something went wrong")
			},
		},

		{
			name: "validation error discovered artifacts",
			reconciler: &reconciler{
				discoverArtifactsFn: func(context.Context, *kargoapi.Warehouse) (*kargoapi.DiscoveredArtifacts, error) {
					return &kargoapi.DiscoveredArtifacts{
						Git: []kargoapi.GitDiscoveryResult{
							{RepoURL: "fake-repo", Commits: nil},
						},
					}, nil
				},
				patchStatusFn: func(context.Context, *kargoapi.Warehouse, func(*kargoapi.WarehouseStatus)) error {
					return nil
				},
			},
			warehouse: &kargoapi.Warehouse{},
			assertions: func(t *testing.T, status kargoapi.WarehouseStatus, err error) {
				require.NoError(t, err)

				require.Len(t, status.GetConditions(), 2)

				// Ensure that the Ready condition is set to False.
				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "MissingCommits", readyCondition.Reason)
				require.Contains(t, readyCondition.Message, "No commits discovered")

				// Ensure that the Healthy condition is set to False.
				healthyCondition := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCondition)
				require.Equal(t, metav1.ConditionFalse, healthyCondition.Status)
				require.Equal(t, "NoCommitsDiscovered", healthyCondition.Reason)
				require.Contains(t, healthyCondition.Message, "No commits discovered")
			},
		},

		{
			name: "Freight build error",
			reconciler: &reconciler{
				discoverArtifactsFn: func(
					context.Context,
					*kargoapi.Warehouse,
				) (*kargoapi.DiscoveredArtifacts, error) {
					return &kargoapi.DiscoveredArtifacts{
						Git: []kargoapi.GitDiscoveryResult{
							{RepoURL: "fake-repo", Commits: []kargoapi.DiscoveredCommit{{ID: "fake-commit"}}},
						},
					}, nil
				},
				buildFreightFromLatestArtifactsFn: func(
					string,
					*kargoapi.DiscoveredArtifacts,
				) (*kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
				patchStatusFn: func(context.Context, *kargoapi.Warehouse, func(*kargoapi.WarehouseStatus)) error {
					return nil
				},
			},
			warehouse: &kargoapi.Warehouse{},
			assertions: func(t *testing.T, status kargoapi.WarehouseStatus, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "failed to build Freight from latest artifacts")

				require.NotNil(t, status.DiscoveredArtifacts)

				require.Len(t, status.GetConditions(), 3)

				// Ensure that the Ready condition is set to False.
				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "FreightBuildFailure", readyCondition.Reason)
				require.Contains(t, readyCondition.Message, "something went wrong")

				// Ensure that the Reconciling condition is still set to True.
				reconcilingCondition := conditions.Get(&status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, reconcilingCondition)
				require.Equal(t, metav1.ConditionTrue, reconcilingCondition.Status)
			},
		},

		{
			name: "Freight for latest artifacts already exists",
			reconciler: &reconciler{
				discoverArtifactsFn: func(
					context.Context,
					*kargoapi.Warehouse,
				) (*kargoapi.DiscoveredArtifacts, error) {
					return &kargoapi.DiscoveredArtifacts{
						Git: []kargoapi.GitDiscoveryResult{
							{RepoURL: "fake-repo", Commits: []kargoapi.DiscoveredCommit{{ID: "fake-commit"}}},
						},
					}, nil
				},
				buildFreightFromLatestArtifactsFn: func(
					string,
					*kargoapi.DiscoveredArtifacts,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-freight",
							Namespace: "fake-namespace",
						},
					}, nil
				},
				createFreightFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return apierrors.NewAlreadyExists(
						schema.GroupResource{
							Group:    kargoapi.GroupVersion.Group,
							Resource: "Warehouse",
						},
						"fake-freight",
					)
				},
				patchStatusFn: func(context.Context, *kargoapi.Warehouse, func(*kargoapi.WarehouseStatus)) error {
					return nil
				},
			},
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					FreightCreationPolicy: kargoapi.FreightCreationPolicyAutomatic,
				},
			},
			assertions: func(t *testing.T, status kargoapi.WarehouseStatus, err error) {
				require.NoError(t, err)
				require.NotNil(t, status.DiscoveredArtifacts)
				// Ensure that even if the Freight already exists, the status
				// is still updated with the latest Freight.
				require.NotEmpty(t, status.LastFreightID)
			},
		},

		{
			name: "error creating Freight",
			reconciler: &reconciler{
				discoverArtifactsFn: func(
					context.Context,
					*kargoapi.Warehouse,
				) (*kargoapi.DiscoveredArtifacts, error) {
					return &kargoapi.DiscoveredArtifacts{
						Git: []kargoapi.GitDiscoveryResult{
							{RepoURL: "fake-repo", Commits: []kargoapi.DiscoveredCommit{{ID: "fake-commit"}}},
						},
					}, nil
				},
				buildFreightFromLatestArtifactsFn: func(
					string,
					*kargoapi.DiscoveredArtifacts,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				createFreightFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return errors.New("something went wrong")
				},
				patchStatusFn: func(context.Context, *kargoapi.Warehouse, func(*kargoapi.WarehouseStatus)) error {
					return nil
				},
			},
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					FreightCreationPolicy: kargoapi.FreightCreationPolicyAutomatic,
				},
			},
			assertions: func(t *testing.T, status kargoapi.WarehouseStatus, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error creating Freight")
				require.NotNil(t, status.DiscoveredArtifacts)
				require.Empty(t, status.LastFreightID)

				require.Len(t, status.GetConditions(), 3)

				// Ensure that the Ready condition is set to False.
				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "FreightCreationFailure", readyCondition.Reason)
				require.Contains(t, readyCondition.Message, "something went wrong")

				// Ensure that the Reconciling condition is still set to True.
				reconcilingCondition := conditions.Get(&status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, reconcilingCondition)
				require.Equal(t, metav1.ConditionTrue, reconcilingCondition.Status)
			},
		},

		{
			name: "automatic Freight creation",
			reconciler: &reconciler{
				discoverArtifactsFn: func(context.Context, *kargoapi.Warehouse) (*kargoapi.DiscoveredArtifacts, error) {
					return &kargoapi.DiscoveredArtifacts{
						Git: []kargoapi.GitDiscoveryResult{
							{RepoURL: "fake-repo", Commits: []kargoapi.DiscoveredCommit{{ID: "fake-commit"}}},
						},
					}, nil
				},
				buildFreightFromLatestArtifactsFn: func(
					string,
					*kargoapi.DiscoveredArtifacts,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-freight",
							Namespace: "fake-namespace",
						},
					}, nil
				},
				createFreightFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return nil
				},
				patchStatusFn: func(context.Context, *kargoapi.Warehouse, func(*kargoapi.WarehouseStatus)) error {
					return nil
				},
			},
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					FreightCreationPolicy: kargoapi.FreightCreationPolicyAutomatic,
				},
			},
			assertions: func(t *testing.T, status kargoapi.WarehouseStatus, err error) {
				require.NoError(t, err)
				require.NotNil(t, status.DiscoveredArtifacts)
				require.NotEmpty(t, status.LastFreightID)

				require.Len(t, status.GetConditions(), 2)

				// Ensure that the Ready condition is set to True.
				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionTrue, readyCondition.Status)
				require.Equal(t, "ArtifactsDiscovered", readyCondition.Reason)

				// Ensure that the Healthy condition is set to True.
				healthyCondition := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCondition)
				require.Equal(t, metav1.ConditionTrue, healthyCondition.Status)
			},
		},

		{
			name: "manual Freight creation",
			reconciler: &reconciler{
				discoverArtifactsFn: func(context.Context, *kargoapi.Warehouse) (*kargoapi.DiscoveredArtifacts, error) {
					return &kargoapi.DiscoveredArtifacts{
						Git: []kargoapi.GitDiscoveryResult{
							{RepoURL: "fake-repo", Commits: []kargoapi.DiscoveredCommit{{ID: "fake-commit"}}},
						},
					}, nil
				},
				patchStatusFn: func(context.Context, *kargoapi.Warehouse, func(*kargoapi.WarehouseStatus)) error {
					return nil
				},
			},
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					FreightCreationPolicy: kargoapi.FreightCreationPolicyManual,
				},
			},
			assertions: func(t *testing.T, status kargoapi.WarehouseStatus, err error) {
				require.NoError(t, err)
				require.NotNil(t, status.DiscoveredArtifacts)
				require.Empty(t, status.LastFreightID)

				require.Len(t, status.GetConditions(), 2)

				// Ensure that the Ready condition is set to True.
				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionTrue, readyCondition.Status)

				// Ensure that the Healthy condition is set to True.
				healthyCondition := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCondition)
				require.Equal(t, metav1.ConditionTrue, healthyCondition.Status)
			},
		},

		{
			name: "updates refresh request status value",
			reconciler: &reconciler{
				discoverArtifactsFn: func(context.Context, *kargoapi.Warehouse) (*kargoapi.DiscoveredArtifacts, error) {
					return &kargoapi.DiscoveredArtifacts{}, nil
				},
				patchStatusFn: func(context.Context, *kargoapi.Warehouse, func(*kargoapi.WarehouseStatus)) error {
					return nil
				},
			},
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyRefresh: "new",
					},
				},
				Spec: kargoapi.WarehouseSpec{
					FreightCreationPolicy: kargoapi.FreightCreationPolicyManual,
				},
				Status: kargoapi.WarehouseStatus{
					LastHandledRefresh: "old",
				},
			},
			assertions: func(t *testing.T, status kargoapi.WarehouseStatus, err error) {
				require.NoError(t, err)
				require.Equal(t, "new", status.LastHandledRefresh)
			},
		},

		{
			name: "updates observed generation",
			reconciler: &reconciler{
				discoverArtifactsFn: func(context.Context, *kargoapi.Warehouse) (*kargoapi.DiscoveredArtifacts, error) {
					return &kargoapi.DiscoveredArtifacts{}, nil
				},
				patchStatusFn: func(context.Context, *kargoapi.Warehouse, func(*kargoapi.WarehouseStatus)) error {
					return nil
				},
			},
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 2,
				},
				Spec: kargoapi.WarehouseSpec{
					FreightCreationPolicy: kargoapi.FreightCreationPolicyManual,
				},
				Status: kargoapi.WarehouseStatus{
					ObservedGeneration: 1,
				},
			},
			assertions: func(t *testing.T, status kargoapi.WarehouseStatus, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(2), status.ObservedGeneration)
			},
		},

		{
			name: "clears previous transient error conditions",
			reconciler: &reconciler{
				discoverArtifactsFn: func(context.Context, *kargoapi.Warehouse) (*kargoapi.DiscoveredArtifacts, error) {
					return &kargoapi.DiscoveredArtifacts{
						Git: []kargoapi.GitDiscoveryResult{
							{RepoURL: "fake-repo", Commits: []kargoapi.DiscoveredCommit{{ID: "fake-commit"}}},
						},
					}, nil
				},
				patchStatusFn: func(context.Context, *kargoapi.Warehouse, func(*kargoapi.WarehouseStatus)) error {
					return nil
				},
			},
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					FreightCreationPolicy: kargoapi.FreightCreationPolicyManual,
				},
				Status: kargoapi.WarehouseStatus{
					Conditions: []metav1.Condition{
						{
							Type:    kargoapi.ConditionTypeReady,
							Status:  metav1.ConditionFalse,
							Reason:  "DiscoveryFailure",
							Message: "something went wrong",
						},
						{
							Type:    kargoapi.ConditionTypeHealthy,
							Status:  metav1.ConditionFalse,
							Reason:  "DiscoveryFailed",
							Message: "something went wrong",
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.WarehouseStatus, err error) {
				require.NoError(t, err)
				require.Len(t, status.GetConditions(), 2)

				// Ensure that the Ready condition is set to True.
				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionTrue, readyCondition.Status)

				// Ensure that the Healthy condition is set to True.
				healthyCondition := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCondition)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			status, err := testCase.reconciler.syncWarehouse(context.TODO(), testCase.warehouse)
			testCase.assertions(t, status, err)
		})
	}
}

func TestDiscoverArtifacts(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(*testing.T, *kargoapi.DiscoveredArtifacts, error)
	}{
		{
			name: "error discovering commits",
			reconciler: &reconciler{
				discoverCommitsFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.GitDiscoveryResult, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, discoveredArtifacts *kargoapi.DiscoveredArtifacts, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error discovering commits")
				require.Nil(t, discoveredArtifacts)
			},
		},
		{
			name: "error discovering images",
			reconciler: &reconciler{
				discoverCommitsFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.GitDiscoveryResult, error) {
					return []kargoapi.GitDiscoveryResult{}, nil
				},
				discoverImagesFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.ImageDiscoveryResult, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, discoveredArtifacts *kargoapi.DiscoveredArtifacts, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error discovering images")
				require.Nil(t, discoveredArtifacts)
			},
		},
		{
			name: "error discovering charts",
			reconciler: &reconciler{
				discoverCommitsFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.GitDiscoveryResult, error) {
					return []kargoapi.GitDiscoveryResult{}, nil
				},
				discoverImagesFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.ImageDiscoveryResult, error) {
					return []kargoapi.ImageDiscoveryResult{}, nil
				},
				discoverChartsFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.ChartDiscoveryResult, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, discoveredArtifacts *kargoapi.DiscoveredArtifacts, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error discovering charts")
				require.Nil(t, discoveredArtifacts)
			},
		},
		{
			name: "success",
			reconciler: &reconciler{
				discoverCommitsFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.GitDiscoveryResult, error) {
					return []kargoapi.GitDiscoveryResult{
						{RepoURL: "fake-repo", Commits: []kargoapi.DiscoveredCommit{
							{ID: "fake-commit"},
						}},
					}, nil
				},
				discoverImagesFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.ImageDiscoveryResult, error) {
					return []kargoapi.ImageDiscoveryResult{
						{RepoURL: "fake-repo", References: []kargoapi.DiscoveredImageReference{
							{Tag: "fake-tag"},
						}},
					}, nil
				},
				discoverChartsFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.ChartDiscoveryResult, error) {
					return []kargoapi.ChartDiscoveryResult{
						{RepoURL: "fake-repo", Versions: []string{
							"fake-version",
						}},
					}, nil
				},
			},
			assertions: func(t *testing.T, discoveredArtifacts *kargoapi.DiscoveredArtifacts, err error) {
				require.NoError(t, err)
				require.Len(t, discoveredArtifacts.Git, 1)
				require.Len(t, discoveredArtifacts.Images, 1)
				require.Len(t, discoveredArtifacts.Charts, 1)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			discoveredArtifacts, err := testCase.reconciler.discoverArtifacts(
				context.TODO(),
				&kargoapi.Warehouse{},
			)
			testCase.assertions(t, discoveredArtifacts, err)
		})
	}
}

func TestBuildFreightFromLatestArtifacts(t *testing.T) {
	testCases := []struct {
		name       string
		artifacts  *kargoapi.DiscoveredArtifacts
		assertions func(*testing.T, *kargoapi.Freight, error)
	}{
		{
			name:      "no artifacts discovered",
			artifacts: nil,
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "no artifacts discovered")
				require.Nil(t, freight)
			},
		},
		{
			name: "no commits discovered",
			artifacts: &kargoapi.DiscoveredArtifacts{
				Git: []kargoapi.GitDiscoveryResult{
					{RepoURL: "fake-repo", Commits: []kargoapi.DiscoveredCommit{}},
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "no commits discovered for repository")
				require.Nil(t, freight)
			},
		},
		{
			name: "no images discovered",
			artifacts: &kargoapi.DiscoveredArtifacts{
				Git: []kargoapi.GitDiscoveryResult{
					{RepoURL: "fake-repo", Commits: []kargoapi.DiscoveredCommit{{ID: "fake-commit"}}},
				},
				Images: []kargoapi.ImageDiscoveryResult{
					{RepoURL: "fake-repo", References: []kargoapi.DiscoveredImageReference{}},
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "no images discovered for repository")
				require.Nil(t, freight)
			},
		},
		{
			name: "no charts discovered",
			artifacts: &kargoapi.DiscoveredArtifacts{
				Git: []kargoapi.GitDiscoveryResult{
					{RepoURL: "fake-repo", Commits: []kargoapi.DiscoveredCommit{{ID: "fake-commit"}}},
				},
				Images: []kargoapi.ImageDiscoveryResult{
					{RepoURL: "fake-repo", References: []kargoapi.DiscoveredImageReference{{Tag: "fake-tag"}}},
				},
				Charts: []kargoapi.ChartDiscoveryResult{
					{RepoURL: "fake-repo", Versions: []string{}},
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "no versions discovered for chart")
				require.Nil(t, freight)
			},
		},
		{
			name: "success",
			artifacts: &kargoapi.DiscoveredArtifacts{
				Git: []kargoapi.GitDiscoveryResult{
					{RepoURL: "fake-repo", Commits: []kargoapi.DiscoveredCommit{{ID: "fake-commit"}}},
					{RepoURL: "fake-repo", Commits: []kargoapi.DiscoveredCommit{{ID: "fake-commit"}}},
				},
				Images: []kargoapi.ImageDiscoveryResult{
					{RepoURL: "fake-repo", References: []kargoapi.DiscoveredImageReference{{Tag: "fake-tag"}}},
					{RepoURL: "fake-repo", References: []kargoapi.DiscoveredImageReference{{Tag: "fake-tag"}}},
				},
				Charts: []kargoapi.ChartDiscoveryResult{
					{RepoURL: "fake-repo", Versions: []string{"fake-version"}},
					{RepoURL: "fake-repo", Versions: []string{"fake-version"}},
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.NotNil(t, freight)
				require.Len(t, freight.Commits, 2)
				require.Len(t, freight.Images, 2)
				require.Len(t, freight.Charts, 2)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			freight, err := (&reconciler{}).buildFreightFromLatestArtifacts(
				"fake-namespace",
				testCase.artifacts,
			)
			testCase.assertions(t, freight, err)
		})
	}
}

func TestValidateDiscoveredArtifacts(t *testing.T) {
	testCases := []struct {
		name       string
		warehouse  *kargoapi.Warehouse
		newStatus  *kargoapi.WarehouseStatus
		assertions func(*testing.T, bool, *kargoapi.WarehouseStatus)
	}{
		{
			name: "no artifacts",
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
			},
			newStatus: &kargoapi.WarehouseStatus{
				DiscoveredArtifacts: &kargoapi.DiscoveredArtifacts{},
			},
			assertions: func(t *testing.T, result bool, status *kargoapi.WarehouseStatus) {
				require.False(t, result)

				require.Len(t, status.GetConditions(), 2)

				readyCondition := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "MissingArtifacts", readyCondition.Reason)
				require.Equal(t, "No artifacts discovered", readyCondition.Message)
				require.Equal(t, int64(1), readyCondition.ObservedGeneration)

				healthyCondition := conditions.Get(status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCondition)
				require.Equal(t, metav1.ConditionFalse, healthyCondition.Status)
				require.Equal(t, "MissingArtifacts", healthyCondition.Reason)
				require.Equal(t, "No artifacts discovered", healthyCondition.Message)
				require.Equal(t, int64(1), healthyCondition.ObservedGeneration)
			},
		},
		{
			name: "Git repository with no commits",
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
			},
			newStatus: &kargoapi.WarehouseStatus{
				DiscoveredArtifacts: &kargoapi.DiscoveredArtifacts{
					Git: []kargoapi.GitDiscoveryResult{
						{RepoURL: "https://github.com/example/repo"},
					},
				},
			},
			assertions: func(t *testing.T, result bool, status *kargoapi.WarehouseStatus) {
				require.False(t, result)

				require.Len(t, status.GetConditions(), 2)

				readyCondition := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "MissingCommits", readyCondition.Reason)
				require.Contains(t, readyCondition.Message, "No commits discovered for Git repository")
				require.Equal(t, int64(1), readyCondition.ObservedGeneration)

				healthyCondition := conditions.Get(status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCondition)
				require.Equal(t, metav1.ConditionFalse, healthyCondition.Status)
				require.Equal(t, "NoCommitsDiscovered", healthyCondition.Reason)
				require.Contains(t, healthyCondition.Message, "No commits discovered for Git repository")
				require.Equal(t, int64(1), healthyCondition.ObservedGeneration)
			},
		},
		{
			name: "image repository with no references",
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
			},
			newStatus: &kargoapi.WarehouseStatus{
				DiscoveredArtifacts: &kargoapi.DiscoveredArtifacts{
					Images: []kargoapi.ImageDiscoveryResult{
						{RepoURL: "docker.io/example/image"},
					},
				},
			},
			assertions: func(t *testing.T, result bool, status *kargoapi.WarehouseStatus) {
				require.False(t, result)

				require.Len(t, status.GetConditions(), 2)

				readyCondition := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "MissingImageReferences", readyCondition.Reason)
				require.Contains(t, readyCondition.Message, "No references discovered for image repository")
				require.Equal(t, int64(1), readyCondition.ObservedGeneration)

				healthyCondition := conditions.Get(status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCondition)
				require.Equal(t, metav1.ConditionFalse, healthyCondition.Status)
				require.Equal(t, "NoImageReferencesDiscovered", healthyCondition.Reason)
				require.Contains(t, healthyCondition.Message, "No references discovered for image repository")
				require.Equal(t, int64(1), healthyCondition.ObservedGeneration)
			},
		},
		{
			name: "chart repository with no versions",
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
			},
			newStatus: &kargoapi.WarehouseStatus{
				DiscoveredArtifacts: &kargoapi.DiscoveredArtifacts{
					Charts: []kargoapi.ChartDiscoveryResult{
						{RepoURL: "https://charts.example.com", Name: "mychart"},
					},
				},
			},
			assertions: func(t *testing.T, result bool, status *kargoapi.WarehouseStatus) {
				require.False(t, result)

				require.Len(t, status.GetConditions(), 2)

				readyCondition := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "MissingChartVersions", readyCondition.Reason)
				require.Contains(t, readyCondition.Message, "No versions discovered for chart")
				require.Equal(t, int64(1), readyCondition.ObservedGeneration)

				healthyCondition := conditions.Get(status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCondition)
				require.Equal(t, metav1.ConditionFalse, healthyCondition.Status)
				require.Equal(t, "NoChartVersionsDiscovered", healthyCondition.Reason)
				require.Contains(t, healthyCondition.Message, "No versions discovered for chart")
				require.Equal(t, int64(1), healthyCondition.ObservedGeneration)
			},
		},
		{
			name: "successful discovery with all artifact types",
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
			},
			newStatus: &kargoapi.WarehouseStatus{
				DiscoveredArtifacts: &kargoapi.DiscoveredArtifacts{
					Git: []kargoapi.GitDiscoveryResult{
						{RepoURL: "https://github.com/example/repo1", Commits: []kargoapi.DiscoveredCommit{
							{ID: "abc123"},
						}},
						{RepoURL: "https://github.com/example/repo2", Commits: []kargoapi.DiscoveredCommit{
							{ID: "def456"}, {ID: "ghi789"},
						}},
					},
					Images: []kargoapi.ImageDiscoveryResult{
						{RepoURL: "docker.io/example/image1", References: []kargoapi.DiscoveredImageReference{
							{Tag: "1.0.0"}, {Tag: "1.1.0"},
						}},
					},
					Charts: []kargoapi.ChartDiscoveryResult{
						{RepoURL: "https://charts.example.com", Name: "mychart", Versions: []string{"1.0.0", "1.1.0"}},
					},
				},
			},
			assertions: func(t *testing.T, result bool, status *kargoapi.WarehouseStatus) {
				require.True(t, result)

				require.Len(t, status.GetConditions(), 1)

				healthyCondition := conditions.Get(status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCondition)
				require.Equal(t, metav1.ConditionTrue, healthyCondition.Status)
				require.Equal(t, "ArtifactsDiscovered", healthyCondition.Reason)
				require.Contains(
					t,
					healthyCondition.Message,
					"Successfully discovered 3 commits, 2 images, and 2 charts from 4 subscriptions",
				)
				require.Equal(t, int64(1), healthyCondition.ObservedGeneration)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validateDiscoveredArtifacts(tc.warehouse, tc.newStatus)
			tc.assertions(t, result, tc.newStatus)
		})
	}
}

func TestShouldDiscoverArtifacts(t *testing.T) {
	now := metav1.Now()

	tests := []struct {
		name         string
		warehouse    *kargoapi.Warehouse
		refreshToken string
		expected     bool
	}{
		{
			name: "no discovered artifacts",
			warehouse: &kargoapi.Warehouse{
				Status: kargoapi.WarehouseStatus{
					DiscoveredArtifacts: nil,
				},
			},
			expected: true,
		},
		{
			name: "discovered artifacts with zero time",
			warehouse: &kargoapi.Warehouse{
				Status: kargoapi.WarehouseStatus{
					DiscoveredArtifacts: &kargoapi.DiscoveredArtifacts{
						DiscoveredAt: metav1.Time{},
					},
				},
			},
			expected: true,
		},
		{
			name: "Warehouse updated since last discovery",
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 2,
				},
				Status: kargoapi.WarehouseStatus{
					ObservedGeneration: 1,
					DiscoveredArtifacts: &kargoapi.DiscoveredArtifacts{
						DiscoveredAt: now,
					},
				},
			},
			expected: true,
		},
		{
			name: "manual refresh requested",
			warehouse: &kargoapi.Warehouse{
				Status: kargoapi.WarehouseStatus{
					LastHandledRefresh: "old-token",
					DiscoveredArtifacts: &kargoapi.DiscoveredArtifacts{
						DiscoveredAt: now,
					},
				},
			},
			refreshToken: "new-token",
			expected:     true,
		},
		{
			name: "interval passed since last discovery",
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					Interval: metav1.Duration{Duration: time.Hour},
				},
				Status: kargoapi.WarehouseStatus{
					DiscoveredArtifacts: &kargoapi.DiscoveredArtifacts{
						DiscoveredAt: metav1.NewTime(now.Add(-2 * time.Hour)),
					},
				},
			},
			expected: true,
		},
		{
			name: "no need to discover artifacts",
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Spec: kargoapi.WarehouseSpec{
					Interval: metav1.Duration{Duration: time.Hour},
				},
				Status: kargoapi.WarehouseStatus{
					ObservedGeneration: 1,
					LastHandledRefresh: "token",
					DiscoveredArtifacts: &kargoapi.DiscoveredArtifacts{
						DiscoveredAt: metav1.NewTime(now.Add(-30 * time.Minute)),
					},
				},
			},
			refreshToken: "token",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldDiscoverArtifacts(tt.warehouse, tt.refreshToken)
			require.Equal(t, tt.expected, result)
		})
	}
}
