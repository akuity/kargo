package promotions

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/akuity/kargo/api/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/runtime"
)

var (
	now    = metav1.Now()
	before = metav1.Time{Time: now.Add(time.Second * -1)}
	after  = metav1.Time{Time: now.Add(time.Second)}

	fooStageKey = types.NamespacedName{Namespace: testNamespace, Name: "foo"}
	barStageKey = types.NamespacedName{Namespace: testNamespace, Name: "bar"}

	testNamespace = "default"
	testPromos    = v1alpha1.PromotionList{
		Items: []v1alpha1.Promotion{
			// foo stage. two have same creation timestamp but different names
			*newPromo(testNamespace, "d", "foo", "", after),
			*newPromo(testNamespace, "b", "foo", "", now),
			*newPromo(testNamespace, "c", "foo", "", now),
			*newPromo(testNamespace, "a", "foo", "", before),
			// bar stage. two are Running (possibly because of bad bookkeeping).
			// one needs to be deduplicated. one promo is invalid
			*newPromo(testNamespace, "x", "bar", "", before),
			*newPromo(testNamespace, "x", "bar", "", before),
			*newPromo(testNamespace, "y", "bar", v1alpha1.PromotionPhaseRunning, now),
			*newPromo(testNamespace, "z", "bar", "", after),
			kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: now,
					Name:              "w",
					Namespace:         testNamespace,
				},
			},
		},
	}
)

func newPromo(namespace, name, stage string,
	phase v1alpha1.PromotionPhase,
	creationTimestamp metav1.Time,
) *kargoapi.Promotion {
	return &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
			Name:              name,
			Namespace:         namespace,
		},
		Spec: &v1alpha1.PromotionSpec{
			Stage: stage,
		},
		Status: v1alpha1.PromotionStatus{
			Phase: phase,
		},
	}
}

func TestInitializeQueues(t *testing.T) {
	pqs := promoQueues{
		activePromoByStage:        map[types.NamespacedName]string{},
		pendingPromoQueuesByStage: map[types.NamespacedName]runtime.PriorityQueue{},
	}
	pqs.initializeQueues(context.Background(), testPromos)

	// foo stage checks
	require.Equal(t, "", pqs.activePromoByStage[fooStageKey])
	require.Equal(t, 4, pqs.pendingPromoQueuesByStage[fooStageKey].Depth())
	require.Equal(t, "a", pqs.pendingPromoQueuesByStage[fooStageKey].Pop().GetName())
	require.Equal(t, "b", pqs.pendingPromoQueuesByStage[fooStageKey].Pop().GetName())
	require.Equal(t, "c", pqs.pendingPromoQueuesByStage[fooStageKey].Pop().GetName())
	require.Equal(t, "d", pqs.pendingPromoQueuesByStage[fooStageKey].Pop().GetName())
	require.Nil(t, pqs.pendingPromoQueuesByStage[fooStageKey].Pop())

	// bar stage checks
	require.Equal(t, "y", pqs.activePromoByStage[barStageKey])
	// We expect 2 instead of 4 (one was deduped, one went to activePromoByStage)
	require.Equal(t, 2, pqs.pendingPromoQueuesByStage[barStageKey].Depth())
	require.Equal(t, "x", pqs.pendingPromoQueuesByStage[barStageKey].Pop().GetName())
	require.Equal(t, "z", pqs.pendingPromoQueuesByStage[barStageKey].Pop().GetName())
	require.Nil(t, pqs.pendingPromoQueuesByStage[barStageKey].Pop())
}

func TestNewPromotionsQueue(t *testing.T) {
	// runtime.PriorityQueue is already tested pretty well, so what we mainly
	// want to assert here is that our function for establishing relative priority
	// is correct.
	pq := newPriorityQueue()

	// The last added should be the first out if our priority logic is correct
	now := time.Now()
	for i := 0; i < 100; i++ {
		added := pq.Push(&kargoapi.Promotion{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%d", i),
				CreationTimestamp: metav1.NewTime(
					now.Add(-1 * time.Duration(i) * time.Minute),
				),
			},
		})
		require.True(t, added)
	}

	// Verify objects are prioritized by creation time
	var lastTime *time.Time
	for {
		object := pq.Pop()
		if object == nil {
			break
		}
		promo := object.(*kargoapi.Promotion) // nolint: forcetypeassert
		if lastTime != nil {
			require.Greater(t, promo.CreationTimestamp.Time, *lastTime)
		}
		lastTime = &promo.CreationTimestamp.Time
	}
}

func TestTryActivate(t *testing.T) {
	pqs := promoQueues{
		activePromoByStage:        map[types.NamespacedName]string{},
		pendingPromoQueuesByStage: map[types.NamespacedName]runtime.PriorityQueue{},
	}
	pqs.initializeQueues(context.Background(), testPromos)

	ctx := context.TODO()

	// 1. nil promotion
	assert.False(t, pqs.tryActivate(ctx, nil))

	// 2. invalid promotion
	assert.False(t, pqs.tryActivate(ctx, &kargoapi.Promotion{}))

	// 3. Try to activate promos not first in queue
	for _, promoName := range []string{"b", "c", "d"} {
		assert.False(t, pqs.tryActivate(ctx, newPromo(testNamespace, promoName, "foo", "", now)))
		assert.Equal(t, "", pqs.activePromoByStage[fooStageKey])
		assert.Equal(t, 4, pqs.pendingPromoQueuesByStage[fooStageKey].Depth())
	}

	// 4. Now try to activate highest priority. this should succeed
	assert.True(t, pqs.tryActivate(ctx, newPromo(testNamespace, "a", "foo", "", now)))
	assert.Equal(t, "a", pqs.activePromoByStage[fooStageKey])
	assert.Equal(t, 3, pqs.pendingPromoQueuesByStage[fooStageKey].Depth())
}

func TestDeactivate(t *testing.T) {
	pqs := promoQueues{
		activePromoByStage:        map[types.NamespacedName]string{},
		pendingPromoQueuesByStage: map[types.NamespacedName]runtime.PriorityQueue{},
	}
	pqs.initializeQueues(context.Background(), testPromos)

	ctx := context.TODO()

	// Test setup
	assert.True(t, pqs.tryActivate(ctx, newPromo(testNamespace, "a", "foo", "", now)))

	// 1. deactivate something not even active. it should be a no-op
	pqs.deactivate(ctx, fooStageKey, "not-active")
	assert.Equal(t, "a", pqs.activePromoByStage[fooStageKey])

	// 2. Deactivate the active one
	pqs.deactivate(ctx, fooStageKey, "a")
	assert.Equal(t, "", pqs.activePromoByStage[fooStageKey])
}
