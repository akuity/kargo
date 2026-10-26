package stages

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	fakeevent "github.com/akuity/kargo/pkg/event/fake"
	"github.com/akuity/kargo/pkg/health"
	"github.com/akuity/kargo/pkg/indexer"
)

// TestRegularStageReconciler_Reconcile_statusPatchesPerPass counts the Stage
// status patches each pass writes as one Promotion goes through a Stage.
func TestRegularStageReconciler_Reconcile_statusPatchesPerPass(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	const ns = "fake-project"
	origin := kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "w"}
	stageKey := types.NamespacedName{Namespace: ns, Name: "test-stage"}
	promoKey := types.NamespacedName{Namespace: ns, Name: "test-stage.01"}

	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "f1"},
		Origin:     origin,
	}
	promo := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         ns,
			Name:              promoKey.Name,
			CreationTimestamp: metav1.Now(),
		},
		Spec:   kargoapi.PromotionSpec{Stage: stageKey.Name, Freight: freight.Name},
		Status: kargoapi.PromotionStatus{Phase: kargoapi.PromotionPhasePending},
	}
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:  ns,
			Name:       stageKey.Name,
			Finalizers: []string{kargoapi.FinalizerName},
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{{
				Origin:  origin,
				Sources: kargoapi.FreightSources{Direct: true},
			}},
			PromotionTemplate: &kargoapi.PromotionTemplate{
				Spec: kargoapi.PromotionTemplateSpec{
					Steps: []kargoapi.PromotionStep{{Uses: "compose-output"}},
				},
			},
		},
	}

	var patches int
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(stage, freight, promo).
		WithStatusSubresource(&kargoapi.Stage{}, &kargoapi.Freight{}, &kargoapi.Promotion{}).
		WithIndex(&kargoapi.Promotion{}, indexer.PromotionsByStageField, indexer.PromotionsByStage).
		WithIndex(&kargoapi.Freight{}, indexer.FreightByWarehouseField, indexer.FreightByWarehouse).
		WithIndex(&kargoapi.Freight{}, indexer.FreightByCurrentStagesField, indexer.FreightByCurrentStages).
		WithIndex(&kargoapi.Freight{}, indexer.FreightByVerifiedStagesField, indexer.FreightByVerifiedStages).
		WithIndex(&kargoapi.Freight{}, indexer.FreightApprovedForStagesField, indexer.FreightApprovedForStages).
		WithIndex(
			&kargoapi.Promotion{},
			indexer.PromotionsByStageAndFreightField,
			indexer.PromotionsByStageAndFreight,
		).
		WithIndex(
			&kargoapi.PromotionRequest{},
			indexer.PromotionRequestsByStageField,
			indexer.PromotionRequestsByStage,
		).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourcePatch: func(
				ctx context.Context,
				c client.Client,
				sub string,
				obj client.Object,
				patch client.Patch,
				opts ...client.SubResourcePatchOption,
			) error {
				if _, ok := obj.(*kargoapi.Stage); ok {
					patches++
				}
				return c.SubResource(sub).Patch(ctx, obj, patch, opts...)
			},
		}).
		Build()

	r := &RegularStageReconciler{
		client:        c,
		eventSender:   &fakeevent.Sender{},
		healthChecker: &health.MockAggregatingChecker{},
	}

	setPhase := func(phase kargoapi.PromotionPhase) {
		p := &kargoapi.Promotion{}
		require.NoError(t, c.Get(t.Context(), promoKey, p))
		p.Status.Phase = phase
		if phase == kargoapi.PromotionPhaseSucceeded {
			p.Status.Freight = &kargoapi.FreightReference{Name: freight.Name, Origin: origin}
			p.Status.FreightCollection = &kargoapi.FreightCollection{}
			p.Status.FreightCollection.UpdateOrPush(*p.Status.Freight)
			now := metav1.Now()
			p.Status.StartedAt, p.Status.FinishedAt = &now, &now
		}
		require.NoError(t, c.Status().Update(t.Context(), p))
	}
	pass := func() int {
		patches = 0
		_, err := r.Reconcile(t.Context(), ctrl.Request{NamespacedName: stageKey})
		require.NoError(t, err)
		return patches
	}

	counts := map[string]int{}
	counts["pending"] = pass()
	setPhase(kargoapi.PromotionPhaseRunning)
	counts["running"] = pass()
	setPhase(kargoapi.PromotionPhaseSucceeded)
	counts["succeeded"] = pass()
	counts["settled"] = pass()
	counts["steady"] = pass()
	t.Logf("Stage status patches per pass: %v", counts)

	// Acknowledging the current Promotion is flushed early, then the pass's
	// result: two patches.
	assert.Equal(t, 2, counts["pending"])
	for _, k := range []string{"running", "succeeded", "settled"} {
		assert.LessOrEqual(t, counts[k], 1, k)
	}
	assert.Zero(t, counts["steady"])

	// The end state is the same as when every sub-reconciler patched.
	s := &kargoapi.Stage{}
	require.NoError(t, c.Get(t.Context(), stageKey, s))
	assert.Nil(t, s.Status.CurrentPromotion)
	require.NotNil(t, s.Status.LastPromotion)
	assert.Equal(t, promoKey.Name, s.Status.LastPromotion.Name)
	require.NotNil(t, s.Status.FreightHistory.Current())
	assert.True(t, s.Status.FreightHistory.Current().Includes(freight.Name))
}

func Test_acknowledgesNewPromotion(t *testing.T) {
	a := &kargoapi.PromotionReference{Name: "a"}
	b := &kargoapi.PromotionReference{Name: "b"}
	assert.False(t, acknowledgesNewPromotion(nil, nil))
	assert.True(t, acknowledgesNewPromotion(nil, a))
	assert.True(t, acknowledgesNewPromotion(a, b))
	assert.False(t, acknowledgesNewPromotion(a, &kargoapi.PromotionReference{Name: "a"}))
	// Clearing the current Promotion waits for the end of the pass.
	assert.False(t, acknowledgesNewPromotion(a, nil))
}
