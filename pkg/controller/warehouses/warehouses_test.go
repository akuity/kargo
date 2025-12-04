package warehouses

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/conditions"
	"github.com/akuity/kargo/pkg/controller"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/subscription"
)

func TestNewReconciler(t *testing.T) {
	kubeClient := fake.NewClientBuilder().Build()
	minReconciliationInterval := time.Duration(1000)

	e := newReconciler(
		kubeClient,
		&credentials.FakeDB{},
		subscription.MustNewSubscriberRegistry(),
		ReconcilerConfig{MinReconciliationInterval: minReconciliationInterval},
	)
	require.NotNil(t, e.client)
	require.NotNil(t, e.credentialsDB)
	require.NotNil(t, e.subscriberRegistry)
	require.Equal(t, minReconciliationInterval, e.cfg.MinReconciliationInterval)

	// Assert that all overridable behaviors were initialized to a default:
	require.NotNil(t, e.discoverArtifactsFn)
	require.NotNil(t, e.buildFreightFromLatestArtifactsFn)
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
				discoverArtifactsFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) (*kargoapi.DiscoveredArtifacts, error) {
					return nil, errors.New("something went wrong")
				},
				patchStatusFn: func(context.Context, *kargoapi.Warehouse, func(*kargoapi.WarehouseStatus)) error {
					return nil
				},
			},
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
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
				require.Equal(t, int64(1), readyCondition.ObservedGeneration)

				// Ensure that the Healthy condition is set to False.
				healthyCondition := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCondition)
				require.Equal(t, metav1.ConditionFalse, healthyCondition.Status)
				require.Equal(t, "DiscoveryFailed", healthyCondition.Reason)
				require.Contains(t, healthyCondition.Message, "something went wrong")
				require.Equal(t, int64(1), healthyCondition.ObservedGeneration)
				require.Equal(t, int64(1), healthyCondition.ObservedGeneration)

				// Ensure that the Reconciling condition is still set to True.
				reconcilingCondition := conditions.Get(&status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, reconcilingCondition)
				require.Equal(t, metav1.ConditionTrue, reconcilingCondition.Status)
				require.Equal(t, "ScheduledDiscovery", reconcilingCondition.Reason)
				require.Equal(t, int64(1), reconcilingCondition.ObservedGeneration)
				require.Equal(t, int64(1), reconcilingCondition.ObservedGeneration)
			},
		},

		{
			name: "validation error discovered artifacts",
			reconciler: &reconciler{
				discoverArtifactsFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) (*kargoapi.DiscoveredArtifacts, error) {
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
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
			},
			assertions: func(t *testing.T, status kargoapi.WarehouseStatus, err error) {
				require.NoError(t, err)

				require.Len(t, status.GetConditions(), 2)

				// Ensure that the Ready condition is set to False.
				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "MissingCommits", readyCondition.Reason)
				require.Contains(t, readyCondition.Message, "No commits discovered")
				require.Equal(t, int64(1), readyCondition.ObservedGeneration)

				// Ensure that the Healthy condition is set to False.
				healthyCondition := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCondition)
				require.Equal(t, metav1.ConditionFalse, healthyCondition.Status)
				require.Equal(t, "NoCommitsDiscovered", healthyCondition.Reason)
				require.Contains(t, healthyCondition.Message, "No commits discovered")
				require.Equal(t, int64(1), healthyCondition.ObservedGeneration)
			},
		},

		{
			name: "Freight build error",
			reconciler: &reconciler{
				discoverArtifactsFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
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
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
			},
			assertions: func(t *testing.T, status kargoapi.WarehouseStatus, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "failed to build Freight from latest artifacts")

				require.NotNil(t, status.DiscoveredArtifacts)

				require.Len(t, status.GetConditions(), 4)

				// Ensure that the Ready condition is set to False.
				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "FreightBuildFailure", readyCondition.Reason)
				require.Contains(t, readyCondition.Message, "something went wrong")
				require.Equal(t, int64(1), readyCondition.ObservedGeneration)

				// Ensure that the Healthy condition is set to False.
				healthyCondition := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCondition)
				require.Equal(t, metav1.ConditionFalse, healthyCondition.Status)
				require.Equal(t, "FreightBuildFailure", healthyCondition.Reason)
				require.Contains(t, healthyCondition.Message, "something went wrong")
				require.Equal(t, int64(1), healthyCondition.ObservedGeneration)

				// Ensure that the criteria satisfied condition is True.
				criteriaCondition := conditions.Get(&status, kargoapi.ConditionTypeFreightCreationCriteriaSatisfied)
				require.NotNil(t, criteriaCondition)
				require.Equal(t, metav1.ConditionTrue, criteriaCondition.Status)
				require.Equal(t, "CriteriaSatisfied", criteriaCondition.Reason)
				require.Equal(t, int64(1), criteriaCondition.ObservedGeneration)

				// Ensure that the Reconciling condition is still set to True.
				reconcilingCondition := conditions.Get(&status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, reconcilingCondition)
				require.Equal(t, metav1.ConditionTrue, reconcilingCondition.Status)
				require.Equal(t, "FreightCreationInProgress", reconcilingCondition.Reason)
				require.Equal(t, int64(1), reconcilingCondition.ObservedGeneration)
			},
		},

		{
			name: "Freight for latest artifacts already exists",
			reconciler: &reconciler{
				discoverArtifactsFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
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
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
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

				require.Len(t, status.GetConditions(), 4)

				// Ensure that the Ready condition is set to True.
				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionTrue, readyCondition.Status)
				require.Equal(t, "ArtifactsDiscovered", readyCondition.Reason)
				require.Equal(t, int64(1), readyCondition.ObservedGeneration)

				// Ensure that the Healthy condition is set to True.
				healthyCondition := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCondition)
				require.Equal(t, metav1.ConditionTrue, healthyCondition.Status)
				require.Equal(t, "ReconciliationSucceeded", healthyCondition.Reason)
				require.Equal(t, int64(1), healthyCondition.ObservedGeneration)

				// Ensure that the criteria satisfied condition is True.
				criteriaCondition := conditions.Get(&status, kargoapi.ConditionTypeFreightCreationCriteriaSatisfied)
				require.NotNil(t, criteriaCondition)
				require.Equal(t, metav1.ConditionTrue, criteriaCondition.Status)
				require.Equal(t, "CriteriaSatisfied", criteriaCondition.Reason)
				require.Equal(t, int64(1), criteriaCondition.ObservedGeneration)

				// Ensure that the FreightCreated condition is set to False.
				freightCreatedCondition := conditions.Get(&status, kargoapi.ConditionTypeFreightCreated)
				require.NotNil(t, freightCreatedCondition)
				require.Equal(t, metav1.ConditionFalse, freightCreatedCondition.Status)
				require.Equal(t, "AlreadyExists", freightCreatedCondition.Reason)
				require.Equal(t, int64(1), freightCreatedCondition.ObservedGeneration)
			},
		},

		{
			name: "error creating Freight",
			reconciler: &reconciler{
				discoverArtifactsFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
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
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
				Spec: kargoapi.WarehouseSpec{
					FreightCreationPolicy: kargoapi.FreightCreationPolicyAutomatic,
				},
			},
			assertions: func(t *testing.T, status kargoapi.WarehouseStatus, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error creating Freight")
				require.NotNil(t, status.DiscoveredArtifacts)
				require.Empty(t, status.LastFreightID)

				require.Len(t, status.GetConditions(), 4)

				// Ensure that the Ready condition is set to False.
				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "FreightCreationFailure", readyCondition.Reason)
				require.Contains(t, readyCondition.Message, "something went wrong")
				require.Equal(t, int64(1), readyCondition.ObservedGeneration)

				// Ensure that the Healthy condition is set to False.
				healthyCondition := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCondition)
				require.Equal(t, metav1.ConditionFalse, healthyCondition.Status)
				require.Equal(t, "FreightBuildFailure", healthyCondition.Reason)
				require.Contains(t, healthyCondition.Message, "something went wrong")
				require.Equal(t, int64(1), healthyCondition.ObservedGeneration)

				// Ensure that the criteria satisfied condition is True.
				criteriaCondition := conditions.Get(&status, kargoapi.ConditionTypeFreightCreationCriteriaSatisfied)
				require.NotNil(t, criteriaCondition)
				require.Equal(t, metav1.ConditionTrue, criteriaCondition.Status)
				require.Equal(t, "CriteriaSatisfied", criteriaCondition.Reason)
				require.Equal(t, int64(1), criteriaCondition.ObservedGeneration)

				// Ensure that the Reconciling condition is still set to True.
				reconcilingCondition := conditions.Get(&status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, reconcilingCondition)
				require.Equal(t, metav1.ConditionTrue, reconcilingCondition.Status)
				require.Equal(t, "FreightCreationInProgress", reconcilingCondition.Reason)
				require.Equal(t, int64(1), reconcilingCondition.ObservedGeneration)
			},
		},

		{
			name: "automatic Freight creation",
			reconciler: &reconciler{
				discoverArtifactsFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) (*kargoapi.DiscoveredArtifacts, error) {
					return &kargoapi.DiscoveredArtifacts{
						Git: []kargoapi.GitDiscoveryResult{
							{
								RepoURL: "fake-repo",
								Commits: []kargoapi.DiscoveredCommit{{ID: "fake-commit"}},
							},
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
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
				Spec: kargoapi.WarehouseSpec{
					FreightCreationPolicy: kargoapi.FreightCreationPolicyAutomatic,
				},
			},
			assertions: func(t *testing.T, status kargoapi.WarehouseStatus, err error) {
				require.NoError(t, err)
				require.NotNil(t, status.DiscoveredArtifacts)
				require.NotEmpty(t, status.LastFreightID)

				require.Len(t, status.GetConditions(), 4)

				// Ensure that the Ready condition is set to True.
				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionTrue, readyCondition.Status)
				require.Equal(t, "ArtifactsDiscovered", readyCondition.Reason)
				require.Equal(t, int64(1), readyCondition.ObservedGeneration)

				// Ensure that the Healthy condition is set to True.
				healthyCondition := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCondition)
				require.Equal(t, metav1.ConditionTrue, healthyCondition.Status)
				require.Equal(t, "ReconciliationSucceeded", healthyCondition.Reason)
				require.Equal(t, int64(1), healthyCondition.ObservedGeneration)

				// Ensure that the criteria satisfied condition is true.
				criteriaCondition := conditions.Get(&status, kargoapi.ConditionTypeFreightCreationCriteriaSatisfied)
				require.NotNil(t, criteriaCondition)
				require.Equal(t, metav1.ConditionTrue, criteriaCondition.Status)
				require.Equal(t, "CriteriaSatisfied", criteriaCondition.Reason)
				require.Equal(t, int64(1), criteriaCondition.ObservedGeneration)

				// Ensure that the FreightCreated condition is true.
				freightCreatedCondition := conditions.Get(&status, kargoapi.ConditionTypeFreightCreated)
				require.NotNil(t, freightCreatedCondition)
				require.Equal(t, metav1.ConditionTrue, freightCreatedCondition.Status)
				require.Equal(t, "NewFreight", freightCreatedCondition.Reason)
				require.Equal(t, int64(1), freightCreatedCondition.ObservedGeneration)
			},
		},

		{
			name: "manual Freight creation",
			reconciler: &reconciler{
				discoverArtifactsFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) (*kargoapi.DiscoveredArtifacts, error) {
					return &kargoapi.DiscoveredArtifacts{
						Git: []kargoapi.GitDiscoveryResult{
							{
								RepoURL: "fake-repo",
								Commits: []kargoapi.DiscoveredCommit{{ID: "fake-commit"}},
							},
						},
					}, nil
				},
				patchStatusFn: func(context.Context, *kargoapi.Warehouse, func(*kargoapi.WarehouseStatus)) error {
					return nil
				},
			},
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
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
				require.Equal(t, "ArtifactsDiscovered", readyCondition.Reason)
				require.Equal(t, int64(1), readyCondition.ObservedGeneration)

				// Ensure that the Healthy condition is set to True.
				healthyCondition := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCondition)
				require.Equal(t, metav1.ConditionTrue, healthyCondition.Status)
				require.Equal(t, "ReconciliationSucceeded", healthyCondition.Reason)
				require.Equal(t, int64(1), healthyCondition.ObservedGeneration)
			},
		},

		{
			name: "updates refresh request status value",
			reconciler: &reconciler{
				discoverArtifactsFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) (*kargoapi.DiscoveredArtifacts, error) {
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
					Generation: 1,
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
				discoverArtifactsFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) (*kargoapi.DiscoveredArtifacts, error) {
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
				discoverArtifactsFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) (*kargoapi.DiscoveredArtifacts, error) {
					return &kargoapi.DiscoveredArtifacts{
						Git: []kargoapi.GitDiscoveryResult{
							{
								RepoURL: "fake-repo",
								Commits: []kargoapi.DiscoveredCommit{{ID: "fake-commit"}},
							},
						},
					}, nil
				},
				patchStatusFn: func(context.Context, *kargoapi.Warehouse, func(*kargoapi.WarehouseStatus)) error {
					return nil
				},
			},
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
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
				require.Equal(t, "ArtifactsDiscovered", readyCondition.Reason)

				// Ensure that the Healthy condition is set to True.
				healthyCondition := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCondition)
				require.Equal(t, metav1.ConditionTrue, healthyCondition.Status)
				require.Equal(t, "ReconciliationSucceeded", healthyCondition.Reason)
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
			name: "error discovering artifacts",
			reconciler: &reconciler{
				subscriberRegistry: subscription.MustNewSubscriberRegistry(
					subscription.SubscriberRegistration{
						Predicate: func(
							context.Context,
							kargoapi.RepoSubscription,
						) (bool, error) {
							return true, nil
						},
						Value: func(
							context.Context,
							credentials.Database,
						) (subscription.Subscriber, error) {
							return &subscription.MockSubscriber{
								DiscoverArtifactsFn: func(
									context.Context,
									string,
									kargoapi.RepoSubscription,
								) (any, error) {
									return nil, errors.New("something went wrong")
								},
							}, nil
						},
					},
				),
			},
			assertions: func(
				t *testing.T,
				discoveredArtifacts *kargoapi.DiscoveredArtifacts,
				err error,
			) {
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, discoveredArtifacts)
			},
		},
		{
			name: "success -- chart",
			reconciler: &reconciler{
				subscriberRegistry: subscription.MustNewSubscriberRegistry(
					subscription.SubscriberRegistration{
						Predicate: func(
							context.Context,
							kargoapi.RepoSubscription,
						) (bool, error) {
							return true, nil
						},
						Value: func(
							context.Context,
							credentials.Database,
						) (subscription.Subscriber, error) {
							return &subscription.MockSubscriber{
								DiscoverArtifactsFn: func(
									context.Context,
									string,
									kargoapi.RepoSubscription,
								) (any, error) {
									return kargoapi.ChartDiscoveryResult{}, nil
								},
							}, nil
						},
					},
				),
			},
			assertions: func(
				t *testing.T,
				discoveredArtifacts *kargoapi.DiscoveredArtifacts,
				err error,
			) {
				require.NoError(t, err)
				require.Len(t, discoveredArtifacts.Charts, 1)
			},
		},
		{
			name: "success -- git",
			reconciler: &reconciler{
				subscriberRegistry: subscription.MustNewSubscriberRegistry(
					subscription.SubscriberRegistration{
						Predicate: func(
							context.Context,
							kargoapi.RepoSubscription,
						) (bool, error) {
							return true, nil
						},
						Value: func(
							context.Context,
							credentials.Database,
						) (subscription.Subscriber, error) {
							return &subscription.MockSubscriber{
								DiscoverArtifactsFn: func(
									context.Context,
									string,
									kargoapi.RepoSubscription,
								) (any, error) {
									return kargoapi.GitDiscoveryResult{}, nil
								},
							}, nil
						},
					},
				),
			},
			assertions: func(
				t *testing.T,
				discoveredArtifacts *kargoapi.DiscoveredArtifacts,
				err error,
			) {
				require.NoError(t, err)
				require.Len(t, discoveredArtifacts.Git, 1)
			},
		},
		{
			name: "success -- image",
			reconciler: &reconciler{
				subscriberRegistry: subscription.MustNewSubscriberRegistry(
					subscription.SubscriberRegistration{
						Predicate: func(
							context.Context,
							kargoapi.RepoSubscription,
						) (bool, error) {
							return true, nil
						},
						Value: func(
							context.Context,
							credentials.Database,
						) (subscription.Subscriber, error) {
							return &subscription.MockSubscriber{
								DiscoverArtifactsFn: func(
									context.Context,
									string,
									kargoapi.RepoSubscription,
								) (any, error) {
									return kargoapi.ImageDiscoveryResult{}, nil
								},
							}, nil
						},
					},
				),
			},
			assertions: func(
				t *testing.T,
				discoveredArtifacts *kargoapi.DiscoveredArtifacts,
				err error,
			) {
				require.NoError(t, err)
				require.Len(t, discoveredArtifacts.Images, 1)
			},
		},
		{
			name: "success -- generic",
			reconciler: &reconciler{
				subscriberRegistry: subscription.MustNewSubscriberRegistry(
					subscription.SubscriberRegistration{
						Predicate: func(
							context.Context,
							kargoapi.RepoSubscription,
						) (bool, error) {
							return true, nil
						},
						Value: func(
							context.Context,
							credentials.Database,
						) (subscription.Subscriber, error) {
							return &subscription.MockSubscriber{
								DiscoverArtifactsFn: func(
									context.Context,
									string,
									kargoapi.RepoSubscription,
								) (any, error) {
									return kargoapi.DiscoveryResult{}, nil
								},
							}, nil
						},
					},
				),
			},
			assertions: func(
				t *testing.T,
				discoveredArtifacts *kargoapi.DiscoveredArtifacts,
				err error,
			) {
				require.NoError(t, err)
				require.Len(t, discoveredArtifacts.Results, 1)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			discoveredArtifacts, err := testCase.reconciler.discoverArtifacts(
				context.TODO(),
				"fake-project",
				[]kargoapi.RepoSubscription{{}},
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
				Git: []kargoapi.GitDiscoveryResult{{
					RepoURL: "fake-repo",
					Commits: []kargoapi.DiscoveredCommit{},
				}},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "no commits discovered for repository")
				require.Nil(t, freight)
			},
		},
		{
			name: "no images discovered",
			artifacts: &kargoapi.DiscoveredArtifacts{
				Git: []kargoapi.GitDiscoveryResult{{
					RepoURL: "fake-repo",
					Commits: []kargoapi.DiscoveredCommit{{ID: "fake-commit"}},
				}},
				Images: []kargoapi.ImageDiscoveryResult{{
					RepoURL:    "fake-repo",
					References: []kargoapi.DiscoveredImageReference{},
				}},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "no images discovered for repository")
				require.Nil(t, freight)
			},
		},
		{
			name: "no charts discovered",
			artifacts: &kargoapi.DiscoveredArtifacts{
				Git: []kargoapi.GitDiscoveryResult{{
					RepoURL: "fake-repo",
					Commits: []kargoapi.DiscoveredCommit{{ID: "fake-commit"}},
				}},
				Images: []kargoapi.ImageDiscoveryResult{{
					RepoURL:    "fake-repo",
					References: []kargoapi.DiscoveredImageReference{{Tag: "fake-tag"}},
				}},
				Charts: []kargoapi.ChartDiscoveryResult{{
					RepoURL:  "fake-repo",
					Versions: []string{},
				}},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "no versions discovered for chart")
				require.Nil(t, freight)
			},
		},
		{
			name: "no generic artifacts discovered",
			artifacts: &kargoapi.DiscoveredArtifacts{
				Git: []kargoapi.GitDiscoveryResult{{
					RepoURL: "fake-repo",
					Commits: []kargoapi.DiscoveredCommit{{ID: "fake-commit"}},
				}},
				Images: []kargoapi.ImageDiscoveryResult{{
					RepoURL:    "fake-repo",
					References: []kargoapi.DiscoveredImageReference{{Tag: "fake-tag"}},
				}},
				Charts: []kargoapi.ChartDiscoveryResult{{
					RepoURL:  "fake-repo",
					Versions: []string{"fake-version"},
				}},
				Results: []kargoapi.DiscoveryResult{{
					SubscriptionName:   "fake-sub",
					ArtifactReferences: []kargoapi.ArtifactReference{},
				}},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "no versions discovered for subscription")
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
				Results: []kargoapi.DiscoveryResult{
					{
						ArtifactReferences: []kargoapi.ArtifactReference{{
							SubscriptionName: "fake-sub",
							Version:          "v1.0.0",
							Metadata:         &v1.JSON{Raw: []byte("{}")},
						}},
					},
					{
						ArtifactReferences: []kargoapi.ArtifactReference{{
							SubscriptionName: "fake-sub",
							Version:          "v1.1.0",
							Metadata:         &v1.JSON{Raw: []byte("{}")},
						}},
					},
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.NotNil(t, freight)
				require.Len(t, freight.Commits, 2)
				require.Len(t, freight.Images, 2)
				require.Len(t, freight.Charts, 2)
				require.Len(t, freight.Artifacts, 2)
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

				// Ensure that the Ready condition is set to False.
				readyCondition := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "MissingArtifacts", readyCondition.Reason)
				require.Equal(t, "No artifacts discovered", readyCondition.Message)
				require.Equal(t, int64(1), readyCondition.ObservedGeneration)

				// Ensure that the Healthy condition is set to False.
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

				// Ensure that the Ready condition is set to False.
				readyCondition := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "MissingCommits", readyCondition.Reason)
				require.Contains(t, readyCondition.Message, "No commits discovered for Git repository")
				require.Equal(t, int64(1), readyCondition.ObservedGeneration)

				// Ensure that the Healthy condition is set to False.
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

				// Ensure that the Ready condition is set to False.
				readyCondition := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "MissingImageReferences", readyCondition.Reason)
				require.Contains(t, readyCondition.Message, "No references discovered for image repository")
				require.Equal(t, int64(1), readyCondition.ObservedGeneration)

				// Ensure that the Healthy condition is set to False.
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

				// Ensure that the Ready condition is set to False.
				readyCondition := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "MissingChartVersions", readyCondition.Reason)
				require.Contains(t, readyCondition.Message, "No versions discovered for chart")
				require.Equal(t, int64(1), readyCondition.ObservedGeneration)

				// Ensure that the Healthy condition is set to False.
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

				// Ensure that when no validation errors occur, no conditions are set.
				// Conditions reflecting success are managed by the caller.
				require.Empty(t, status.GetConditions())
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

func TestReconcile(t *testing.T) {
	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	tests := []struct {
		name       string
		reconciler func() *reconciler
		req        ctrl.Request
		assertions func(*testing.T, ctrl.Result, error)
	}{
		{
			name: "Shard mismatch",
			reconciler: func() *reconciler {
				return &reconciler{
					shardPredicate: controller.ResponsibleFor[kargoapi.Warehouse]{
						ShardName:           "right-shard",
						IsDefaultController: false,
					},
					client: fake.NewClientBuilder().
						WithScheme(testScheme).
						WithObjects(
							&kargoapi.Warehouse{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-warehouse",
									Namespace: "test-namespace",
									Labels: map[string]string{
										kargoapi.LabelKeyShard: "wrong-shard",
									},
								},
							},
						).Build(),
					cfg: ReconcilerConfig{ShardName: "right-shard"},
					// Intentionally not setting any xFns because we should never reach them
					// because we will exit before any reconciliation logic is executed.
				}
			},
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test-warehouse",
					Namespace: "test-namespace",
				},
			},
			assertions: func(t *testing.T, r ctrl.Result, err error) {
				require.NoError(t, err)
				require.True(t, r.IsZero(), "expected no further reconciliation after shard mismatch")
			},
		},
		{
			name: "Shard match",
			reconciler: func() *reconciler {
				return &reconciler{
					shardPredicate: controller.ResponsibleFor[kargoapi.Warehouse]{
						ShardName:           "right-shard",
						IsDefaultController: false,
					},
					client: fake.NewClientBuilder().
						WithScheme(testScheme).
						WithObjects(
							&kargoapi.Warehouse{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-warehouse",
									Namespace: "test-namespace",
									Labels: map[string]string{
										kargoapi.LabelKeyShard: "right-shard",
									},
								},
							},
						).Build(),
					cfg: ReconcilerConfig{
						ShardName:                 "right-shard",
						MinReconciliationInterval: 5 * time.Minute,
					},
					discoverArtifactsFn: func(
						context.Context, string,
						[]kargoapi.RepoSubscription,
					) (*kargoapi.DiscoveredArtifacts, error) {
						return &kargoapi.DiscoveredArtifacts{}, nil
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
						return nil
					},
					patchStatusFn: func(
						context.Context,
						*kargoapi.Warehouse,
						func(*kargoapi.WarehouseStatus),
					) error {
						return nil
					},
				}
			},
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test-warehouse",
					Namespace: "test-namespace",
				},
			},
			assertions: func(t *testing.T, r ctrl.Result, err error) {
				require.NoError(t, err)
				require.False(t, r.IsZero(), "expected further reconciliation after shard match")
			},
		},
		// TODO(fuskovic): TestReconcile was initially added as part of
		// https://github.com/akuity/kargo/pull/4677. We should add more test cases
		// here to cover logic outside of the scope of shard predicate checks.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.reconciler()
			logger := logging.NewLoggerOrDie(logging.DebugLevel, logging.DefaultFormat)
			ctx := logging.ContextWithLogger(t.Context(), logger)
			result, err := r.Reconcile(ctx, tt.req)
			tt.assertions(t, result, err)
		})
	}
}

func Test_freightCreationCriteriaSatisfied(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		freightCreationCriteria *kargoapi.FreightCreationCriteria
		artifacts               *kargoapi.DiscoveredArtifacts
		expected                bool
		errExpected             bool
	}{
		{
			name:                    "nil freight creation criteria",
			freightCreationCriteria: nil,
			artifacts:               &kargoapi.DiscoveredArtifacts{},
			expected:                true,
		},
		{
			name:                    "empty criteria expression",
			freightCreationCriteria: new(kargoapi.FreightCreationCriteria),
			artifacts:               &kargoapi.DiscoveredArtifacts{},
			expected:                true,
		},
		{
			name: "no artifacts discovered",
			freightCreationCriteria: &kargoapi.FreightCreationCriteria{
				Expression: "doesntmatter",
			},
			artifacts: &kargoapi.DiscoveredArtifacts{},
			expected:  true,
		},
		{
			name: "invalid criteria expression",
			freightCreationCriteria: &kargoapi.FreightCreationCriteria{
				Expression: "invalid.expression",
			},
			artifacts: &kargoapi.DiscoveredArtifacts{
				Git: []kargoapi.GitDiscoveryResult{
					{RepoURL: "doesntmatter"},
				},
			},
			expected:    false,
			errExpected: true,
		},
		{
			name: "success - commit tags match",
			freightCreationCriteria: &kargoapi.FreightCreationCriteria{
				Expression: "commitFrom('site/repo/frontend').Tag == commitFrom('site/repo/backend').Tag",
			},
			artifacts: &kargoapi.DiscoveredArtifacts{
				Git: []kargoapi.GitDiscoveryResult{
					{
						RepoURL: "site/repo/frontend",
						Commits: []kargoapi.DiscoveredCommit{
							{
								Tag:         `abc123`,
								CreatorDate: &metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
							},
							// this is the one that should be picked
							{
								Tag:         `def456`,
								CreatorDate: &metav1.Time{Time: time.Now()},
							},
						},
					},
					{
						RepoURL: "site/repo/backend",
						Commits: []kargoapi.DiscoveredCommit{
							{
								Tag:         `abc123`,
								CreatorDate: &metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
							},
							// this is the one that should be picked
							{
								Tag:         `def456`,
								CreatorDate: &metav1.Time{Time: time.Now()},
							},
						},
					},
				},
			},
			expected:    true,
			errExpected: false,
		},
		{
			name: "success - commit tags do not match",
			freightCreationCriteria: &kargoapi.FreightCreationCriteria{
				Expression: "commitFrom('site/repo/frontend').Tag == commitFrom('site/repo/backend').Tag",
			},
			artifacts: &kargoapi.DiscoveredArtifacts{
				Git: []kargoapi.GitDiscoveryResult{
					{
						RepoURL: "site/repo/frontend",
						Commits: []kargoapi.DiscoveredCommit{
							{
								Tag:         `abc123`,
								CreatorDate: &metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
							},
						},
					},
					{
						RepoURL: "site/repo/backend",
						Commits: []kargoapi.DiscoveredCommit{
							{
								Tag:         `def456`,
								CreatorDate: &metav1.Time{Time: time.Now()},
							},
						},
					},
				},
			},
			expected:    false,
			errExpected: false,
		},
		{
			name: "success - image tags match",
			freightCreationCriteria: &kargoapi.FreightCreationCriteria{
				Expression: "imageFrom('site/repo/frontend').Tag == imageFrom('site/repo/backend').Tag",
			},
			artifacts: &kargoapi.DiscoveredArtifacts{
				Images: []kargoapi.ImageDiscoveryResult{
					{
						RepoURL: "site/repo/frontend",
						References: []kargoapi.DiscoveredImageReference{
							{Tag: `v1.0.0`},
							// this is the one that should be picked
							{Tag: `v1.1.0`},
						},
					},
					{
						RepoURL: "site/repo/backend",
						References: []kargoapi.DiscoveredImageReference{
							{Tag: `v1.0.0`},
							// this is the one that should be picked
							{Tag: `v1.1.0`},
						},
					},
				},
			},
			expected:    true,
			errExpected: false,
		},
		{
			name: "success - image tags do not match",
			freightCreationCriteria: &kargoapi.FreightCreationCriteria{
				Expression: "imageFrom('site/repo/frontend').Tag == imageFrom('site/repo/backend').Tag",
			},
			artifacts: &kargoapi.DiscoveredArtifacts{
				Images: []kargoapi.ImageDiscoveryResult{
					{
						RepoURL: "site/repo/frontend",
						References: []kargoapi.DiscoveredImageReference{
							{Tag: `v1.0.0`},
						},
					},
					{
						RepoURL: "site/repo/backend",
						References: []kargoapi.DiscoveredImageReference{
							{Tag: `v1.1.0`},
						},
					},
				},
			},
			expected:    false,
			errExpected: false,
		},
		{
			name: "success - chart versions match with repo URL only",
			freightCreationCriteria: &kargoapi.FreightCreationCriteria{
				Expression: "chartFrom('site/repo/frontend').Version == chartFrom('site/repo/backend').Version",
			},
			artifacts: &kargoapi.DiscoveredArtifacts{
				Charts: []kargoapi.ChartDiscoveryResult{
					{
						RepoURL:  "site/repo/frontend",
						Versions: []string{"v1.0.0", "v1.1.0"},
					},
					{
						RepoURL:  "site/repo/backend",
						Versions: []string{"v1.0.0", "v1.1.0"},
					},
				},
			},
			expected:    true,
			errExpected: false,
		},
		{
			name: "success - chart versions match with repo URL and optional name",
			freightCreationCriteria: &kargoapi.FreightCreationCriteria{
				Expression: `chartFrom('site/repo/frontend', 'some-name').Version == 
				chartFrom('site/repo/backend', 'some-other-name').Version`,
			},
			artifacts: &kargoapi.DiscoveredArtifacts{
				Charts: []kargoapi.ChartDiscoveryResult{
					{
						Name:     "some-name",
						RepoURL:  "site/repo/frontend",
						Versions: []string{"v1.0.0", "v1.1.0"},
					},
					{
						Name:     "some-other-name",
						RepoURL:  "site/repo/backend",
						Versions: []string{"v1.0.0", "v1.1.0"},
					},
				},
			},
			expected:    true,
			errExpected: false,
		},
		{
			name: "success - chart versions do not match with repo URL only",
			freightCreationCriteria: &kargoapi.FreightCreationCriteria{
				Expression: "chartFrom('site/repo/frontend').Version == chartFrom('site/repo/backend').Version",
			},
			artifacts: &kargoapi.DiscoveredArtifacts{
				Charts: []kargoapi.ChartDiscoveryResult{
					{
						RepoURL:  "site/repo/frontend",
						Versions: []string{"v1.0.0"},
					},
					{
						RepoURL:  "site/repo/backend",
						Versions: []string{"v1.1.0"},
					},
				},
			},
			expected:    false,
			errExpected: false,
		},
		{
			name: "success - chart versions do not match with repo URL and optional name",
			freightCreationCriteria: &kargoapi.FreightCreationCriteria{
				Expression: `chartFrom('site/repo/frontend', 'some-name').Version ==
						chartFrom('site/repo/backend', 'some-other-name').Version`,
			},
			artifacts: &kargoapi.DiscoveredArtifacts{
				Charts: []kargoapi.ChartDiscoveryResult{
					{
						Name:     "some-name",
						RepoURL:  "site/repo/frontend",
						Versions: []string{"v1.0.0"},
					},
					{
						Name:     "some-other-name",
						RepoURL:  "site/repo/backend",
						Versions: []string{"v1.1.0"},
					},
				},
			},
			expected:    false,
			errExpected: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logger, err := logging.NewLogger(logging.ErrorLevel, logging.DefaultFormat)
			require.NoError(t, err)
			ctx := logging.ContextWithLogger(t.Context(), logger)
			result, err := freightCreationCriteriaSatisfied(ctx, tc.freightCreationCriteria, tc.artifacts)
			require.Equal(t, tc.expected, result)
			if tc.errExpected {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}

}
