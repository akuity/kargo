package stages

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
)

func TestRegularStageReconciler_syncTargetPromotions(t *testing.T) {
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Name: "stage"},
		Spec: kargoapi.StageSpec{
			TargetSelectors: []metav1.LabelSelector{{}},
		},
	}
	freight := &kargoapi.FreightCollection{
		Freight: map[string]kargoapi.FreightReference{
			"Warehouse/warehouse": {
				Name:   "freight",
				Origin: kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "warehouse"},
			},
		},
	}
	testCases := []struct {
		name       string
		promotions []kargoapi.Promotion
		assertions func(*testing.T, kargoapi.StageStatus, bool, error)
	}{
		{
			name: "waits for every child",
			promotions: []kargoapi.Promotion{
				targetPromotion("one", kargoapi.PromotionPhaseSucceeded, freight),
				targetPromotion("two", kargoapi.PromotionPhaseRunning, freight),
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, pending bool, err error) {
				require.NoError(t, err)
				require.True(t, pending)
				require.Nil(t, status.FreightHistory.Current())
				require.Nil(t, status.CurrentPromotion)
			},
		},
		{
			name: "records Freight only when all children succeed",
			promotions: []kargoapi.Promotion{
				targetPromotion("one", kargoapi.PromotionPhaseSucceeded, freight),
				targetPromotion("two", kargoapi.PromotionPhaseSucceeded, freight),
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, pending bool, err error) {
				require.NoError(t, err)
				require.False(t, pending)
				require.NotNil(t, status.FreightHistory.Current())
				require.True(t, status.FreightHistory.Current().Includes("freight"))
				require.Nil(t, status.LastPromotion)
			},
		},
		{
			name: "does not record Freight when any child fails",
			promotions: []kargoapi.Promotion{
				targetPromotion("one", kargoapi.PromotionPhaseSucceeded, freight),
				targetPromotion("two", kargoapi.PromotionPhaseFailed, freight),
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, pending bool, err error) {
				require.NoError(t, err)
				require.False(t, pending)
				require.Nil(t, status.FreightHistory.Current())
			},
		},
	}

	reconciler := &RegularStageReconciler{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			status, pending, err := reconciler.syncTargetPromotions(stage, testCase.promotions)
			testCase.assertions(t, status, pending, err)
		})
	}
}

func TestRegularStageReconciler_createTargetPromotions(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Name: "stage", Namespace: "project"},
	}
	existing := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing",
			Namespace: "project",
			Labels: map[string]string{
				kargoapi.LabelKeyPromotionBatch: "batch",
			},
		},
		Spec: kargoapi.PromotionSpec{
			Stage:   stage.Name,
			Freight: "freight",
			Target:  "one",
		},
	}
	kargoClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(existing).
		WithIndex(
			&kargoapi.Promotion{},
			indexer.PromotionsByStageAndFreightField,
			indexer.PromotionsByStageAndFreight,
		).
		Build()
	reconciler := &RegularStageReconciler{client: kargoClient}

	err := reconciler.createTargetPromotions(
		context.Background(),
		stage,
		"freight",
		[]kargoapi.Target{
			{ObjectMeta: metav1.ObjectMeta{Name: "one"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "two"}},
		},
	)
	require.NoError(t, err)

	promotions := &kargoapi.PromotionList{}
	require.NoError(t, kargoClient.List(context.Background(), promotions, client.InNamespace("project")))
	require.Len(t, promotions.Items, 2)
	for _, promotion := range promotions.Items {
		require.Equal(t, "batch", promotion.Labels[kargoapi.LabelKeyPromotionBatch])
	}
}

func targetPromotion(
	target string,
	phase kargoapi.PromotionPhase,
	freight *kargoapi.FreightCollection,
) kargoapi.Promotion {
	return kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:   target,
			Labels: map[string]string{kargoapi.LabelKeyPromotionBatch: "batch"},
		},
		Spec: kargoapi.PromotionSpec{Freight: "freight", Target: target},
		Status: kargoapi.PromotionStatus{
			Phase:             phase,
			FreightCollection: freight.DeepCopy(),
		},
	}
}
