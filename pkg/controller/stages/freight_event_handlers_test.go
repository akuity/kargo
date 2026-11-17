package stages

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllertest"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
)

const freightEventTestNamespace = "fake-project"

func drainStageRequests(q *controllertest.Queue) []reconcile.Request {
	var reqs []reconcile.Request
	for q.Len() > 0 {
		req, _ := q.Get()
		reqs = append(reqs, req)
		q.Done(req)
	}
	return reqs
}

func stageRequestingFreight(name string, req kargoapi.FreightRequest) *kargoapi.Stage {
	return &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Namespace: freightEventTestNamespace, Name: name},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{req},
			// A PromotionTemplate with steps makes this a regular (non-control-flow)
			// Stage, which is what the regular reconciler's enqueuers act on.
			PromotionTemplate: &kargoapi.PromotionTemplate{
				Spec: kargoapi.PromotionTemplateSpec{
					Steps: []kargoapi.PromotionStep{{Uses: "fake-step"}},
				},
			},
		},
	}
}

func Test_freightCreatedStageEnqueuer_Generic(t *testing.T) {
	t.Parallel()
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	// Subscribed to Warehouse wh1 directly.
	subscriber := stageRequestingFreight("subscriber", kargoapi.FreightRequest{
		Origin:  kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "wh1"},
		Sources: kargoapi.FreightSources{Direct: true},
	})
	other := stageRequestingFreight("other", kargoapi.FreightRequest{
		Origin:  kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "wh2"},
		Sources: kargoapi.FreightSources{Direct: true},
	})

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(subscriber, other).
		WithIndex(&kargoapi.Stage{}, indexer.StagesByWarehouseField, indexer.StagesByWarehouse).
		Build()

	enqueuer := &freightCreatedStageEnqueuer[*kargoapi.Freight]{kargoClient: c}

	t.Run("enqueues Stages subscribed to the Warehouse", func(t *testing.T) {
		q := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}
		enqueuer.Generic(t.Context(), event.TypedGenericEvent[*kargoapi.Freight]{
			Object: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{Namespace: freightEventTestNamespace, Name: "f"},
				Origin:     kargoapi.FreightOrigin{Name: "wh1"},
			},
		}, q)
		assert.ElementsMatch(t, []reconcile.Request{{
			NamespacedName: types.NamespacedName{Namespace: freightEventTestNamespace, Name: "subscriber"},
		}}, drainStageRequests(q))
	})

	t.Run("no Stage subscribes to the origin -> nothing enqueued", func(t *testing.T) {
		q := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}
		enqueuer.Generic(t.Context(), event.TypedGenericEvent[*kargoapi.Freight]{
			Object: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{Namespace: freightEventTestNamespace, Name: "f"},
				Origin:     kargoapi.FreightOrigin{Name: "unconsumed"},
			},
		}, q)
		assert.Empty(t, drainStageRequests(q))
	})
}

func Test_freightVerifiedStageEnqueuer_Generic(t *testing.T) {
	t.Parallel()
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	// Sources Freight verified in upstream Stage "up".
	downstream := stageRequestingFreight("downstream", kargoapi.FreightRequest{
		Origin:  kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "wh1"},
		Sources: kargoapi.FreightSources{Stages: []string{"up"}},
	})

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(downstream).
		WithIndex(&kargoapi.Stage{}, indexer.StagesByUpstreamStagesField, indexer.StagesByUpstreamStages).
		Build()

	enqueuer := &freightVerifiedStageEnqueuer[*kargoapi.Freight]{kargoClient: c}

	t.Run("enqueues Stages downstream of the verified Stage", func(t *testing.T) {
		q := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}
		enqueuer.Generic(t.Context(), event.TypedGenericEvent[*kargoapi.Freight]{
			Object: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{Namespace: freightEventTestNamespace, Name: "f"},
				Status: kargoapi.FreightStatus{
					VerifiedIn: map[string]kargoapi.VerifiedStage{"up": {}},
				},
			},
		}, q)
		assert.ElementsMatch(t, []reconcile.Request{{
			NamespacedName: types.NamespacedName{Namespace: freightEventTestNamespace, Name: "downstream"},
		}}, drainStageRequests(q))
	})

	t.Run("no Stage is downstream of the Stage -> nothing enqueued", func(t *testing.T) {
		q := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}
		enqueuer.Generic(t.Context(), event.TypedGenericEvent[*kargoapi.Freight]{
			Object: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{Namespace: freightEventTestNamespace, Name: "f"},
				Status: kargoapi.FreightStatus{
					VerifiedIn: map[string]kargoapi.VerifiedStage{"unrelated": {}},
				},
			},
		}, q)
		assert.Empty(t, drainStageRequests(q))
	})
}

func Test_freightCurrentlyInStageEnqueuer_Generic(t *testing.T) {
	t.Parallel()
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	// Sources Freight from upstream Stage "up".
	downstream := stageRequestingFreight("downstream", kargoapi.FreightRequest{
		Origin:  kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "wh1"},
		Sources: kargoapi.FreightSources{Stages: []string{"up"}},
	})

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(downstream).
		WithIndex(&kargoapi.Stage{}, indexer.StagesByUpstreamStagesField, indexer.StagesByUpstreamStages).
		Build()

	enqueuer := &freightCurrentlyInStageEnqueuer[*kargoapi.Freight]{kargoClient: c}

	t.Run("enqueues downstream when current in AND verified in the Stage", func(t *testing.T) {
		q := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}
		enqueuer.Generic(t.Context(), event.TypedGenericEvent[*kargoapi.Freight]{
			Object: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{Namespace: freightEventTestNamespace, Name: "f"},
				Status: kargoapi.FreightStatus{
					CurrentlyIn: map[string]kargoapi.CurrentStage{"up": {}},
					VerifiedIn:  map[string]kargoapi.VerifiedStage{"up": {}},
				},
			},
		}, q)
		assert.ElementsMatch(t, []reconcile.Request{{
			NamespacedName: types.NamespacedName{Namespace: freightEventTestNamespace, Name: "downstream"},
		}}, drainStageRequests(q))
	})

	t.Run("does not enqueue when current but not verified in the Stage", func(t *testing.T) {
		q := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}
		enqueuer.Generic(t.Context(), event.TypedGenericEvent[*kargoapi.Freight]{
			Object: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{Namespace: freightEventTestNamespace, Name: "f"},
				Status: kargoapi.FreightStatus{
					CurrentlyIn: map[string]kargoapi.CurrentStage{"up": {}},
				},
			},
		}, q)
		assert.Empty(t, drainStageRequests(q))
	})

	t.Run("no Stage is downstream of the Stage -> nothing enqueued", func(t *testing.T) {
		q := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}
		enqueuer.Generic(t.Context(), event.TypedGenericEvent[*kargoapi.Freight]{
			Object: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{Namespace: freightEventTestNamespace, Name: "f"},
				Status: kargoapi.FreightStatus{
					CurrentlyIn: map[string]kargoapi.CurrentStage{"unrelated": {}},
					VerifiedIn:  map[string]kargoapi.VerifiedStage{"unrelated": {}},
				},
			},
		}, q)
		assert.Empty(t, drainStageRequests(q))
	})
}

func Test_freightApprovedStageEnqueuer_Generic(t *testing.T) {
	t.Parallel()
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	enqueuer := &freightApprovedStageEnqueuer[*kargoapi.Freight]{kargoClient: c}

	t.Run("enqueues the Stage the Freight was approved for", func(t *testing.T) {
		q := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}
		enqueuer.Generic(t.Context(), event.TypedGenericEvent[*kargoapi.Freight]{
			Object: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{Namespace: freightEventTestNamespace, Name: "f"},
				Status: kargoapi.FreightStatus{
					ApprovedFor: map[string]kargoapi.ApprovedStage{"stage-a": {}},
				},
			},
		}, q)
		assert.ElementsMatch(t, []reconcile.Request{{
			NamespacedName: types.NamespacedName{Namespace: freightEventTestNamespace, Name: "stage-a"},
		}}, drainStageRequests(q))
	})

	t.Run("no approval -> nothing enqueued", func(t *testing.T) {
		q := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}
		enqueuer.Generic(t.Context(), event.TypedGenericEvent[*kargoapi.Freight]{
			Object: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{Namespace: freightEventTestNamespace, Name: "f"},
			},
		}, q)
		assert.Empty(t, drainStageRequests(q))
	})
}
