package stages

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewReconciler(t *testing.T) {
	kubeClient := fake.NewClientBuilder().Build()
	e := newReconciler(
		kubeClient,
		kubeClient,
	)
	require.NotNil(t, e.kargoClient)
	require.NotNil(t, e.argoClient)
	// Assert that all overridable behaviors were initialized to a default:
	// Loop guard:
	require.NotNil(t, e.hasNonTerminalPromotionsFn)
	require.NotNil(t, e.listPromosFn)
	// Health checks:
	require.NotNil(t, e.checkHealthFn)
	require.NotNil(t, e.getArgoCDAppFn)
	// Freight qualification:
	require.NotNil(t, e.getFreightFn)
	require.NotNil(t, e.qualifyFreightFn)
	require.NotNil(t, e.patchFreightStatusFn)
	// Auto-promotion:
	require.NotNil(t, e.isAutoPromotionPermittedFn)
	require.NotNil(t, e.listPromoPoliciesFn)
	require.NotNil(t, e.createPromotionFn)
	// Discovering latest Freight:
	require.NotNil(t, e.getLatestAvailableFreightFn)
	require.NotNil(t, e.getAllFreightFromWarehouseFn)
	require.NotNil(t, e.getLatestFreightFromWarehouseFn)
	require.NotNil(t, e.getAllFreightQualifiedForUpstreamStagesFn)
	require.NotNil(t, e.getLatestFreightQualifiedForUpstreamStagesFn)
	require.NotNil(t, e.listFreightFn)
}

func TestSyncControlFlowStage(t *testing.T) {
	testCases := []struct {
		name       string
		stage      *kargoapi.Stage
		reconciler *reconciler
		assertions func(
			initialStatus kargoapi.StageStatus,
			newStatus kargoapi.StageStatus,
			err error,
		)
	}{
		{
			name: "error listing Freight from Warehouse",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Subscriptions: &kargoapi.Subscriptions{
						Warehouse: "fake-warehouse",
					},
				},
			},
			reconciler: &reconciler{
				getAllFreightFromWarehouseFn: func(
					context.Context,
					string,
					string,
				) ([]kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error finding all Freight from Warehouse",
				)
				require.Contains(t, err.Error(), "something went wrong")
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},
		{
			name: "error listing Freight qualified for upstream Stages",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Subscriptions: &kargoapi.Subscriptions{
						UpstreamStages: []kargoapi.StageSubscription{
							{Name: "fake-stage"},
						},
					},
				},
			},
			reconciler: &reconciler{
				getAllFreightQualifiedForUpstreamStagesFn: func(
					context.Context,
					string,
					[]kargoapi.StageSubscription,
				) ([]kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error finding available Freight for Stage",
				)
				require.Contains(t, err.Error(), "something went wrong")
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},
		{
			name: "error qualifying Freight",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Subscriptions: &kargoapi.Subscriptions{
						Warehouse: "fake-warehouse",
					},
				},
			},
			reconciler: &reconciler{
				getAllFreightFromWarehouseFn: func(
					context.Context,
					string,
					string,
				) ([]kargoapi.Freight, error) {
					return []kargoapi.Freight{{}}, nil
				},
				patchFreightStatusFn: func(
					context.Context,
					*kargoapi.Freight,
					kargoapi.FreightStatus,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error qualifying Freight")
				require.Contains(t, err.Error(), "something went wrong")
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},
		{
			name: "success",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 42,
				},
				Spec: &kargoapi.StageSpec{
					Subscriptions: &kargoapi.Subscriptions{
						Warehouse: "fake-warehouse",
					},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.SimpleFreight{},
					Health:         &kargoapi.Health{},
				},
			},
			reconciler: &reconciler{
				getAllFreightFromWarehouseFn: func(
					context.Context,
					string,
					string,
				) ([]kargoapi.Freight, error) {
					return []kargoapi.Freight{{}}, nil
				},
				patchFreightStatusFn: func(
					context.Context,
					*kargoapi.Freight,
					kargoapi.FreightStatus,
				) error {
					return nil
				},
			},
			assertions: func(
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, int64(42), newStatus.ObservedGeneration) // Set
				require.Nil(t, newStatus.CurrentFreight)                  // Cleared
				require.Nil(t, newStatus.Health)                          // Cleared
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			newStatus, err := testCase.reconciler.syncControlFlowStage(
				context.Background(),
				testCase.stage,
			)
			testCase.assertions(testCase.stage.Status, newStatus, err)
		})
	}
}

func TestSyncNormalStage(t *testing.T) {
	noNonTerminalPromotionsFn := func(
		context.Context,
		string,
		string,
	) (bool, error) {
		return false, nil
	}

	testCases := []struct {
		name       string
		stage      *kargoapi.Stage
		reconciler *reconciler
		assertions func(
			initialStatus kargoapi.StageStatus,
			newStatus kargoapi.StageStatus,
			err error,
		)
	}{
		{
			name:  "error checking for non-terminal promotions",
			stage: &kargoapi.Stage{},
			reconciler: &reconciler{
				hasNonTerminalPromotionsFn: func(
					context.Context,
					string,
					string,
				) (bool, error) {
					return false, errors.New("something went wrong")
				},
			},
			assertions: func(
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name:  "non-terminal promotions found",
			stage: &kargoapi.Stage{},
			reconciler: &reconciler{
				hasNonTerminalPromotionsFn: func(
					context.Context,
					string,
					string,
				) (bool, error) {
					return true, nil
				},
			},
			assertions: func(
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "error qualifying Freight",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.SimpleFreight{},
				},
			},
			reconciler: &reconciler{
				hasNonTerminalPromotionsFn: noNonTerminalPromotionsFn,
				checkHealthFn: func(
					context.Context,
					kargoapi.SimpleFreight,
					[]kargoapi.ArgoCDAppUpdate,
				) *kargoapi.Health {
					return nil
				},
				qualifyFreightFn: func(context.Context, string, string, string) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(t, err.Error(), "error qualifying Freight")
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "auto-promotion not possible",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Subscriptions: &kargoapi.Subscriptions{
						UpstreamStages: []kargoapi.StageSubscription{
							{Name: "fake-stage"},
							{Name: "another-fake-stage"},
						},
					},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.SimpleFreight{},
				},
			},
			reconciler: &reconciler{
				hasNonTerminalPromotionsFn: noNonTerminalPromotionsFn,
				checkHealthFn: func(
					context.Context,
					kargoapi.SimpleFreight,
					[]kargoapi.ArgoCDAppUpdate,
				) *kargoapi.Health {
					return nil
				},
				qualifyFreightFn: func(context.Context, string, string, string) error {
					return nil
				},
			},
			assertions: func(
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "error checking if auto-promotion is permitted",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Subscriptions: &kargoapi.Subscriptions{
						Warehouse: "fake-warehouse",
					},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.SimpleFreight{},
				},
			},
			reconciler: &reconciler{
				hasNonTerminalPromotionsFn: noNonTerminalPromotionsFn,
				checkHealthFn: func(
					context.Context,
					kargoapi.SimpleFreight,
					[]kargoapi.ArgoCDAppUpdate,
				) *kargoapi.Health {
					return nil
				},
				qualifyFreightFn: func(context.Context, string, string, string) error {
					return nil
				},
				isAutoPromotionPermittedFn: func(
					context.Context,
					string,
					string,
				) (bool, error) {
					return false, errors.New("something went wrong")
				},
			},
			assertions: func(
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(
					t,
					err.Error(),
					"error checking if auto-promotion is permitted",
				)
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "auto-promotion is not permitted",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Subscriptions: &kargoapi.Subscriptions{
						Warehouse: "fake-warehouse",
					},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.SimpleFreight{},
				},
			},
			reconciler: &reconciler{
				hasNonTerminalPromotionsFn: noNonTerminalPromotionsFn,
				checkHealthFn: func(
					context.Context,
					kargoapi.SimpleFreight,
					[]kargoapi.ArgoCDAppUpdate,
				) *kargoapi.Health {
					return nil
				},
				qualifyFreightFn: func(context.Context, string, string, string) error {
					return nil
				},
				isAutoPromotionPermittedFn: func(
					context.Context,
					string,
					string,
				) (bool, error) {
					return false, nil
				},
			},
			assertions: func(
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "error getting latest Freight",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Subscriptions: &kargoapi.Subscriptions{
						Warehouse: "fake-warehouse",
					},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.SimpleFreight{},
				},
			},
			reconciler: &reconciler{
				hasNonTerminalPromotionsFn: noNonTerminalPromotionsFn,
				checkHealthFn: func(
					context.Context,
					kargoapi.SimpleFreight,
					[]kargoapi.ArgoCDAppUpdate,
				) *kargoapi.Health {
					return nil
				},
				qualifyFreightFn: func(context.Context, string, string, string) error {
					return nil
				},
				isAutoPromotionPermittedFn: func(
					context.Context,
					string,
					string,
				) (bool, error) {
					return true, nil
				},
				getLatestAvailableFreightFn: func(
					context.Context,
					string,
					kargoapi.Subscriptions,
				) (*kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(
					t,
					err.Error(),
					"error finding latest Freight for Stage",
				)
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "no Freight found",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Subscriptions: &kargoapi.Subscriptions{
						Warehouse: "fake-warehouse",
					},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.SimpleFreight{},
				},
			},
			reconciler: &reconciler{
				hasNonTerminalPromotionsFn: noNonTerminalPromotionsFn,
				checkHealthFn: func(
					context.Context,
					kargoapi.SimpleFreight,
					[]kargoapi.ArgoCDAppUpdate,
				) *kargoapi.Health {
					return nil
				},
				qualifyFreightFn: func(context.Context, string, string, string) error {
					return nil
				},
				isAutoPromotionPermittedFn: func(
					context.Context,
					string,
					string,
				) (bool, error) {
					return true, nil
				},
				getLatestAvailableFreightFn: func(
					context.Context,
					string,
					kargoapi.Subscriptions,
				) (*kargoapi.Freight, error) {
					return nil, nil
				},
			},
			assertions: func(
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "Stage already has latest Freight",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Subscriptions: &kargoapi.Subscriptions{
						Warehouse: "fake-warehouse",
					},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.SimpleFreight{
						ID: "fake-freight-id",
					},
				},
			},
			reconciler: &reconciler{
				hasNonTerminalPromotionsFn: noNonTerminalPromotionsFn,
				checkHealthFn: func(
					context.Context,
					kargoapi.SimpleFreight,
					[]kargoapi.ArgoCDAppUpdate,
				) *kargoapi.Health {
					return nil
				},
				qualifyFreightFn: func(context.Context, string, string, string) error {
					return nil
				},
				isAutoPromotionPermittedFn: func(
					context.Context,
					string,
					string,
				) (bool, error) {
					return true, nil
				},
				getLatestAvailableFreightFn: func(
					context.Context,
					string,
					kargoapi.Subscriptions,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name: "fake-freight-id",
						},
					}, nil
				},
			},
			assertions: func(
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "Promotion already exists",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Subscriptions: &kargoapi.Subscriptions{
						Warehouse: "fake-warehouse",
					},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
			},
			reconciler: &reconciler{
				hasNonTerminalPromotionsFn: noNonTerminalPromotionsFn,
				checkHealthFn: func(
					context.Context,
					kargoapi.SimpleFreight,
					[]kargoapi.ArgoCDAppUpdate,
				) *kargoapi.Health {
					return nil
				},
				qualifyFreightFn: func(context.Context, string, string, string) error {
					return nil
				},
				isAutoPromotionPermittedFn: func(
					context.Context,
					string,
					string,
				) (bool, error) {
					return true, nil
				},
				getLatestAvailableFreightFn: func(
					context.Context,
					string,
					kargoapi.Subscriptions,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name: "fake-freight-id",
						},
					}, nil
				},
				listPromosFn: func(
					_ context.Context,
					obj client.ObjectList,
					_ ...client.ListOption,
				) error {
					promos, ok := obj.(*kargoapi.PromotionList)
					require.True(t, ok)
					promos.Items = []kargoapi.Promotion{{}}
					return nil
				},
			},
			assertions: func(
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "error creating Promotion",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Subscriptions: &kargoapi.Subscriptions{
						Warehouse: "fake-warehouse",
					},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
			},
			reconciler: &reconciler{
				hasNonTerminalPromotionsFn: noNonTerminalPromotionsFn,
				checkHealthFn: func(
					context.Context,
					kargoapi.SimpleFreight,
					[]kargoapi.ArgoCDAppUpdate,
				) *kargoapi.Health {
					return nil
				},
				qualifyFreightFn: func(context.Context, string, string, string) error {
					return nil
				},
				isAutoPromotionPermittedFn: func(
					context.Context,
					string,
					string,
				) (bool, error) {
					return true, nil
				},
				getLatestAvailableFreightFn: func(
					context.Context,
					string,
					kargoapi.Subscriptions,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name: "fake-freight-id",
						},
					}, nil
				},
				listPromosFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
				createPromotionFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(t, err.Error(), "error creating Promotion of Stage")
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "success",
			// Note: In this final case, we will also assert than anything that should
			// have been cleared or modified in the Stage's status was.
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 42,
				},
				Spec: &kargoapi.StageSpec{
					Subscriptions: &kargoapi.Subscriptions{
						Warehouse: "fake-warehouse",
					},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					CurrentPromotion: &kargoapi.PromotionInfo{},
					CurrentFreight:   &kargoapi.SimpleFreight{},
				},
			},
			reconciler: &reconciler{
				hasNonTerminalPromotionsFn: noNonTerminalPromotionsFn,
				checkHealthFn: func(
					context.Context,
					kargoapi.SimpleFreight,
					[]kargoapi.ArgoCDAppUpdate,
				) *kargoapi.Health {
					return &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					}
				},
				qualifyFreightFn: func(context.Context, string, string, string) error {
					return nil
				},
				isAutoPromotionPermittedFn: func(
					context.Context,
					string,
					string,
				) (bool, error) {
					return true, nil
				},
				getLatestAvailableFreightFn: func(
					context.Context,
					string,
					kargoapi.Subscriptions,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name: "fake-freight-id",
						},
					}, nil
				},
				listPromosFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
				createPromotionFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return nil
				},
			},
			assertions: func(
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, int64(42), newStatus.ObservedGeneration) // Set
				require.NotNil(t, newStatus.Health)                       // Set
				require.Nil(t, newStatus.CurrentPromotion)                // Cleared
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			newStatus, err :=
				testCase.reconciler.syncNormalStage(context.Background(), testCase.stage)
			testCase.assertions(testCase.stage.Status, newStatus, err)
		})
	}
}

func TestHasNonTerminalPromotions(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(bool, error)
	}{
		{
			name: "error listing Promotions",
			reconciler: &reconciler{
				listPromosFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(_ bool, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "has non-terminal Promotions",
			reconciler: &reconciler{
				listPromosFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					promos, ok := objList.(*kargoapi.PromotionList)
					require.True(t, ok)
					promos.Items = []kargoapi.Promotion{{}}
					return nil
				},
			},
			assertions: func(result bool, err error) {
				require.NoError(t, err)
				require.True(t, result)
			},
		},
		{
			name: "does not have non-terminal Promotions",
			reconciler: &reconciler{
				listPromosFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					promos, ok := objList.(*kargoapi.PromotionList)
					require.True(t, ok)
					promos.Items = []kargoapi.Promotion{}
					return nil
				},
			},
			assertions: func(result bool, err error) {
				require.NoError(t, err)
				require.False(t, result)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := testCase.reconciler.hasNonTerminalPromotions(
				context.Background(),
				"fake-namespace",
				"fake-stage",
			)
			testCase.assertions(result, err)
		})
	}
}

func TestQualifyFreight(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(error)
	}{
		{
			name: "error getting Freight",
			reconciler: &reconciler{
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(t, err.Error(), "error finding Freight")
			},
		},
		{
			name: "Freight not found",
			reconciler: &reconciler{
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return nil, nil
				},
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "found no Freight")
			},
		},
		{
			name: "Freight already qualified for Stage",
			reconciler: &reconciler{
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						Status: kargoapi.FreightStatus{
							Qualifications: map[string]kargoapi.Qualification{
								"fake-stage": {},
							},
						},
					}, nil
				},
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "error Patching Freight status",
			reconciler: &reconciler{
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				patchFreightStatusFn: func(
					context.Context,
					*kargoapi.Freight,
					kargoapi.FreightStatus,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "success",
			reconciler: &reconciler{
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				patchFreightStatusFn: func(
					context.Context,
					*kargoapi.Freight,
					kargoapi.FreightStatus,
				) error {
					return nil
				},
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.reconciler.qualifyFreight(
					context.Background(),
					"fake-namespace",
					"fake-freight",
					"fake-stage",
				),
			)
		})
	}
}

func TestIsAutoPromotionPermitted(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(bool, error)
	}{
		{
			name: "error listing PromotionPolicies",
			reconciler: &reconciler{
				listPromoPoliciesFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(_ bool, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(
					t,
					err.Error(),
					"error listing PromotionPolicies for Stage",
				)
			},
		},
		{
			name: "no PromotionPolicies found",
			reconciler: &reconciler{
				listPromoPoliciesFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					policies, ok := objList.(*kargoapi.PromotionPolicyList)
					require.True(t, ok)
					policies.Items = []kargoapi.PromotionPolicy{}
					return nil
				},
			},
			assertions: func(result bool, err error) {
				require.NoError(t, err)
				require.False(t, result)
			},
		},
		{
			name: "more than one PromotionPolicy found",
			reconciler: &reconciler{
				listPromoPoliciesFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					policies, ok := objList.(*kargoapi.PromotionPolicyList)
					require.True(t, ok)
					policies.Items = []kargoapi.PromotionPolicy{{}, {}}
					return nil
				},
			},
			assertions: func(result bool, err error) {
				require.NoError(t, err)
				require.False(t, result)
			},
		},
		{
			name: "not permitted",
			reconciler: &reconciler{
				listPromoPoliciesFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					policies, ok := objList.(*kargoapi.PromotionPolicyList)
					require.True(t, ok)
					policies.Items = []kargoapi.PromotionPolicy{{}}
					return nil
				},
			},
			assertions: func(result bool, err error) {
				require.NoError(t, err)
				require.False(t, result)
			},
		},
		{
			name: "permitted",
			reconciler: &reconciler{
				listPromoPoliciesFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					policies, ok := objList.(*kargoapi.PromotionPolicyList)
					require.True(t, ok)
					policies.Items = []kargoapi.PromotionPolicy{
						{
							EnableAutoPromotion: true,
						},
					}
					return nil
				},
			},
			assertions: func(result bool, err error) {
				require.NoError(t, err)
				require.True(t, result)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.reconciler.isAutoPromotionPermitted(
					context.Background(),
					"fake-namespace",
					"fake-stage",
				),
			)
		})
	}
}

func TestGetLatestAvailableFreight(t *testing.T) {
	testCases := []struct {
		name       string
		subs       kargoapi.Subscriptions
		reconciler *reconciler
		assertions func(*kargoapi.Freight, error)
	}{
		{
			name: "error getting latest Freight from Warehouse",
			subs: kargoapi.Subscriptions{
				Warehouse: "fake-warehouse",
			},
			reconciler: &reconciler{
				getLatestFreightFromWarehouseFn: func(
					context.Context,
					string,
					string,
				) (*kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(_ *kargoapi.Freight, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(t, err.Error(), "error checking Warehouse")
			},
		},
		{
			name: "found no Freight from Warehouse",
			subs: kargoapi.Subscriptions{
				Warehouse: "fake-warehouse",
			},
			reconciler: &reconciler{
				getLatestFreightFromWarehouseFn: func(
					context.Context,
					string,
					string,
				) (*kargoapi.Freight, error) {
					return nil, nil
				},
			},
			assertions: func(freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Nil(t, freight)
			},
		},
		{
			name: "success getting latest Freight from Warehouse",
			subs: kargoapi.Subscriptions{
				Warehouse: "fake-warehouse",
			},
			reconciler: &reconciler{
				getLatestFreightFromWarehouseFn: func(
					context.Context,
					string,
					string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
			},
			assertions: func(freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.NotNil(t, freight)
			},
		},
		{
			name: "error getting latest qualified Freight for Stage",
			subs: kargoapi.Subscriptions{
				UpstreamStages: []kargoapi.StageSubscription{{}},
			},
			reconciler: &reconciler{
				getLatestFreightQualifiedForUpstreamStagesFn: func(
					context.Context,
					string,
					[]kargoapi.StageSubscription,
				) (*kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(_ *kargoapi.Freight, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(
					t,
					err.Error(),
					"error finding Freight qualified for Stage",
				)
			},
		},
		{
			name: "found no qualified Freight for Stage",
			subs: kargoapi.Subscriptions{
				UpstreamStages: []kargoapi.StageSubscription{{}},
			},
			reconciler: &reconciler{
				getLatestFreightQualifiedForUpstreamStagesFn: func(
					context.Context,
					string,
					[]kargoapi.StageSubscription,
				) (*kargoapi.Freight, error) {
					return nil, nil
				},
			},
			assertions: func(freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Nil(t, freight)
			},
		},
		{
			name: "success getting latest qualified Freight for Stage",
			subs: kargoapi.Subscriptions{
				UpstreamStages: []kargoapi.StageSubscription{{}},
			},
			reconciler: &reconciler{
				getLatestFreightQualifiedForUpstreamStagesFn: func(
					context.Context,
					string,
					[]kargoapi.StageSubscription,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
			},
			assertions: func(freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.NotNil(t, freight)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.reconciler.getLatestAvailableFreight(
					context.Background(),
					"fake-namespace",
					testCase.subs,
				),
			)
		})
	}
}

func TestGetAllFreightFromWarehouse(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func([]kargoapi.Freight, error)
	}{
		{
			name: "error listing Freight",
			reconciler: &reconciler{
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(_ []kargoapi.Freight, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(t, err.Error(), "error listing Freight for Warehouse")
			},
		},
		{
			name: "no Freight found",
			reconciler: &reconciler{
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
			},
			assertions: func(freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Nil(t, freight)
			},
		},
		{
			name: "success",
			reconciler: &reconciler{
				listFreightFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					freight, ok := objList.(*kargoapi.FreightList)
					require.True(t, ok)
					freight.Items = []kargoapi.Freight{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "older-freight",
								CreationTimestamp: metav1.Time{
									Time: time.Now().Add(-time.Hour),
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "newer-freight",
								CreationTimestamp: metav1.Time{
									Time: time.Now(),
								},
							},
						},
					}
					return nil
				},
			},
			assertions: func(freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Len(t, freight, 2)
				// Be sure they've been sorted
				require.Equal(t, "newer-freight", freight[0].Name)
				require.Equal(t, "older-freight", freight[1].Name)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.reconciler.getAllFreightFromWarehouse(
					context.Background(),
					"fake-namespace",
					"fake-warehouse",
				),
			)
		})
	}
}

func TestGetLatestFreightFromWarehouse(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(*kargoapi.Freight, error)
	}{
		{
			name: "error getting all Freight from Warehouse",
			reconciler: &reconciler{
				getAllFreightFromWarehouseFn: func(
					context.Context,
					string,
					string,
				) ([]kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(_ *kargoapi.Freight, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "no Freight found from Warehouse",
			reconciler: &reconciler{
				getAllFreightFromWarehouseFn: func(
					context.Context,
					string,
					string,
				) ([]kargoapi.Freight, error) {
					return nil, nil
				},
			},
			assertions: func(freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Nil(t, freight)
			},
		},
		{
			name: "success",
			reconciler: &reconciler{
				getAllFreightFromWarehouseFn: func(
					context.Context,
					string,
					string,
				) ([]kargoapi.Freight, error) {
					return []kargoapi.Freight{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "newer-freight",
								CreationTimestamp: metav1.Time{
									Time: time.Now(),
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "older-freight",
								CreationTimestamp: metav1.Time{
									Time: time.Now().Add(-time.Hour),
								},
							},
						},
					}, nil
				},
			},
			assertions: func(freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.NotNil(t, freight)
				// Be sure we got the latest
				require.Equal(t, "newer-freight", freight.Name)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.reconciler.getLatestFreightFromWarehouse(
					context.Background(),
					"fake-namespace",
					"fake-warehouse",
				),
			)
		})
	}
}

func TestGetAllFreightQualifiedForUpstreamStages(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func([]kargoapi.Freight, error)
	}{
		{
			name: "error listing Freight",
			reconciler: &reconciler{
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(_ []kargoapi.Freight, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(
					t,
					err.Error(),
					"error listing Freight qualified for Stage",
				)
			},
		},
		{
			name: "no Freight found",
			reconciler: &reconciler{
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
			},
			assertions: func(freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Nil(t, freight)
			},
		},
		{
			name: "success",
			reconciler: &reconciler{
				listFreightFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					freight, ok := objList.(*kargoapi.FreightList)
					require.True(t, ok)
					freight.Items = []kargoapi.Freight{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "older-freight",
								CreationTimestamp: metav1.Time{
									Time: time.Now().Add(-time.Hour),
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "newer-freight",
								CreationTimestamp: metav1.Time{
									Time: time.Now(),
								},
							},
						},
					}
					return nil
				},
			},
			assertions: func(freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.NotNil(t, freight)
				// Be sure they've been sorted// Be sure we got the newest one
				require.Equal(t, "newer-freight", freight[0].Name)
				require.Equal(t, "older-freight", freight[1].Name)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.reconciler.getAllFreightQualifiedForUpstreamStages(
					context.Background(),
					"fake-namespace",
					[]kargoapi.StageSubscription{
						{
							Name: "fake-stage",
						},
					},
				),
			)
		})
	}
}

func TestGetLatestFreightQualifiedForUpstreamStages(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(*kargoapi.Freight, error)
	}{
		{
			name: "error getting all Freight qualified for upstream Stages",
			reconciler: &reconciler{
				getAllFreightQualifiedForUpstreamStagesFn: func(
					context.Context,
					string,
					[]kargoapi.StageSubscription,
				) ([]kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(_ *kargoapi.Freight, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "no Freight qualified for upstream Stages",
			reconciler: &reconciler{
				getAllFreightQualifiedForUpstreamStagesFn: func(
					context.Context,
					string,
					[]kargoapi.StageSubscription,
				) ([]kargoapi.Freight, error) {
					return nil, nil
				},
			},
			assertions: func(freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Nil(t, freight)
			},
		},
		{
			name: "success",
			reconciler: &reconciler{
				getAllFreightQualifiedForUpstreamStagesFn: func(
					context.Context,
					string,
					[]kargoapi.StageSubscription,
				) ([]kargoapi.Freight, error) {
					return []kargoapi.Freight{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "newer-freight",
								CreationTimestamp: metav1.Time{
									Time: time.Now(),
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "older-freight",
								CreationTimestamp: metav1.Time{
									Time: time.Now().Add(-time.Hour),
								},
							},
						},
					}, nil
				},
			},
			assertions: func(freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.NotNil(t, freight)
				// Be sure we got the latest
				require.Equal(t, "newer-freight", freight.Name)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.reconciler.getLatestFreightQualifiedForUpstreamStages(
					context.Background(),
					"fake-namespace",
					[]kargoapi.StageSubscription{},
				),
			)
		})
	}
}
