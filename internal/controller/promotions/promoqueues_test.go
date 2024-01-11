package promotions

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

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
	testPromos    = kargoapi.PromotionList{
		Items: []kargoapi.Promotion{
			// foo stage. two have same creation timestamp but different names
			*newPromo(testNamespace, "d", "foo", "", after),
			*newPromo(testNamespace, "b", "foo", "", now),
			*newPromo(testNamespace, "c", "foo", "", now),
			*newPromo(testNamespace, "a", "foo", "", before),
			// bar stage. two are Running (possibly because of bad bookkeeping).
			// one needs to be deduplicated. one promo is invalid
			*newPromo(testNamespace, "x", "bar", "", before),
			*newPromo(testNamespace, "x", "bar", "", before),
			*newPromo(testNamespace, "y", "bar", kargoapi.PromotionPhaseRunning, now),
			*newPromo(testNamespace, "z", "bar", "", after),
			{
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
	phase kargoapi.PromotionPhase,
	creationTimestamp metav1.Time,
) *kargoapi.Promotion {
	return &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
			Name:              name,
			Namespace:         namespace,
		},
		Spec: &kargoapi.PromotionSpec{
			Stage: stage,
		},
		Status: kargoapi.PromotionStatus{
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

func TestTryBegin(t *testing.T) {
	pqs := promoQueues{
		activePromoByStage:        map[types.NamespacedName]string{},
		pendingPromoQueuesByStage: map[types.NamespacedName]runtime.PriorityQueue{},
	}
	pqs.initializeQueues(context.Background(), testPromos)

	ctx := context.TODO()

	// 1. nil promotion
	require.False(t, pqs.tryBegin(ctx, nil))

	// 2. invalid promotion
	require.False(t, pqs.tryBegin(ctx, &kargoapi.Promotion{}))

	// 3. Try to begin promos not first in queue
	for _, promoName := range []string{"b", "c", "d"} {
		require.False(t, pqs.tryBegin(ctx, newPromo(testNamespace, promoName, "foo", "", now)))
		require.Equal(t, "", pqs.activePromoByStage[fooStageKey])
		require.Equal(t, 4, pqs.pendingPromoQueuesByStage[fooStageKey].Depth())
	}

	// 4. Now try to begin highest priority. this should succeed
	require.True(t, pqs.tryBegin(ctx, newPromo(testNamespace, "a", "foo", "", now)))
	require.Equal(t, "a", pqs.activePromoByStage[fooStageKey])
	require.Equal(t, 3, pqs.pendingPromoQueuesByStage[fooStageKey].Depth())

	// 5. Begin an already active promo, this should be a no-op
	require.True(t, pqs.tryBegin(ctx, newPromo(testNamespace, "a", "foo", "", now)))
	require.Equal(t, "a", pqs.activePromoByStage[fooStageKey])
	require.Equal(t, 3, pqs.pendingPromoQueuesByStage[fooStageKey].Depth())

	// 5. Begin a promo with something else active, this should be a no-op
	require.False(t, pqs.tryBegin(ctx, newPromo(testNamespace, "b", "foo", "", now)))
	require.Equal(t, "a", pqs.activePromoByStage[fooStageKey])
	require.Equal(t, 3, pqs.pendingPromoQueuesByStage[fooStageKey].Depth())
}

func TestConclude(t *testing.T) {
	pqs := promoQueues{
		activePromoByStage:        map[types.NamespacedName]string{},
		pendingPromoQueuesByStage: map[types.NamespacedName]runtime.PriorityQueue{},
	}
	pqs.initializeQueues(context.Background(), testPromos)

	ctx := context.TODO()

	// Test setup
	require.True(t, pqs.tryBegin(ctx, newPromo(testNamespace, "a", "foo", "", now)))

	// 1. conclude something not even active. it should be a no-op
	pqs.conclude(ctx, fooStageKey, "not-active")
	require.Equal(t, "a", pqs.activePromoByStage[fooStageKey])

	// 2. Conclude the active one
	pqs.conclude(ctx, fooStageKey, "a")
	require.Equal(t, "", pqs.activePromoByStage[fooStageKey])

	// 3. Conclude the same key, should be a noop
	pqs.conclude(ctx, fooStageKey, "a")
	require.Equal(t, "", pqs.activePromoByStage[fooStageKey])
}
