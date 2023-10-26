package promotions

import (
	"context"
	"sync"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/runtime"
	"github.com/akuity/kargo/internal/logging"
)

// promoQueues is a data structure to hold priority queues of all Stages
// as well as the "active" promotion for each stage
type promoQueues struct {
	// activePromoByStage holds the active promotion for a given stage (if any)
	activePromoByStage map[types.NamespacedName]string
	// pendingPromoQueuesByStage holds a priority queue of promotions, per Stage. We allow one
	// promotion to run at a time, ordered by creationTimestamp.
	pendingPromoQueuesByStage map[types.NamespacedName]runtime.PriorityQueue
	// promoQueuesByStageMu protects access to the above maps
	promoQueuesByStageMu sync.RWMutex
}

func newPriorityQueue() runtime.PriorityQueue {
	// We can safely ignore errors here because the only error that can happen
	// involves initializing the queue with a nil priority function, which we
	// know we aren't doing.
	pq, _ := runtime.NewPriorityQueue(func(left, right client.Object) bool {
		if left.GetCreationTimestamp().Time.Equal(
			right.GetCreationTimestamp().Time,
		) {
			return left.GetName() < right.GetName()
		}
		return left.GetCreationTimestamp().Time.
			Before(right.GetCreationTimestamp().Time)
	})
	return pq
}

// initializeQueues adds the promotion list to relevant priority queues.
// This is intended to be invoked ONCE and the caller MUST ensure that.
func (pqs *promoQueues) initializeQueues(ctx context.Context, promos kargoapi.PromotionList) {
	pqs.promoQueuesByStageMu.Lock()
	defer pqs.promoQueuesByStageMu.Unlock()
	logger := logging.LoggerFromContext(ctx)
	for _, promo := range promos.Items {
		promo := promo // This is to sidestep implicit memory aliasing in this for loop
		if promo.Status.Phase.IsTerminal() || promo.Spec == nil {
			continue
		}
		stage := types.NamespacedName{
			Namespace: promo.Namespace,
			Name:      promo.Spec.Stage,
		}
		pq, ok := pqs.pendingPromoQueuesByStage[stage]
		if !ok {
			pq = newPriorityQueue()
			pqs.pendingPromoQueuesByStage[stage] = pq
		}
		if promo.Status.Phase == kargoapi.PromotionPhaseRunning {
			if pqs.activePromoByStage[stage] == "" {
				pqs.activePromoByStage[stage] = promo.Name
			}
			continue
		}
		pq.Push(&promo)
		logger.WithFields(log.Fields{
			"promotion": promo.Name,
			"namespace": promo.Namespace,
			"stage":     promo.Spec.Stage,
			"phase":     promo.Status.Phase,
		}).Debug("pushed Promotion onto Stage-specific Promotion queue")
	}
	if logger.Logger.IsLevelEnabled(log.DebugLevel) {
		for stage, pq := range pqs.pendingPromoQueuesByStage {
			logger.WithFields(log.Fields{
				"stage":     stage.Name,
				"namespace": stage.Namespace,
				"depth":     pq.Depth(),
			}).Debug("Stage-specific Promotion queue initialized")
		}
	}
}

// tryActivate tries to mark the given Pending promotion as the active one so it can reconcile.
// Returns true if the promo is already active or became active as a result of this call.
// Returns false if it should not reconcile (another promo is active, or next in line).
func (pqs *promoQueues) tryActivate(ctx context.Context, promo *kargoapi.Promotion) bool {
	if promo == nil || promo.Spec == nil {
		return false
	}
	stageKey := types.NamespacedName{
		Namespace: promo.Namespace,
		Name:      promo.Spec.Stage,
	}
	logger := logging.LoggerFromContext(ctx)

	pqs.promoQueuesByStageMu.Lock()
	defer pqs.promoQueuesByStageMu.Unlock()

	pq, ok := pqs.pendingPromoQueuesByStage[stageKey]
	if !ok {
		// PriorityQueue for the stage has not been been initialized
		pq = newPriorityQueue()
		pqs.pendingPromoQueuesByStage[stageKey] = pq
	}

	// Push this promo to the queue in case it doesn't exist in the queue. Note that we
	// deduplicate pushes on the same object, so this is safe to call repeatedly
	if pq.Push(promo) {
		logger.Debug("promo added to priority queue")
	}

	if activePromoName := pqs.activePromoByStage[stageKey]; activePromoName != "" {
		// There is already an active promo. It's either this promo or someone else.
		return activePromoName == promo.Name
	}

	// If we get here, the Stage does not have any Promotions Running against it.
	// Now check if it this promo is the one that should run next.
	first := pq.Peek()
	if first == nil {
		// This promo exists but nothing exists in the PriorityQueue. This should not happen.
		// But since there appears to be no other promos, allow this one to become the active one.
		pqs.activePromoByStage[stageKey] = promo.Name
		logger.Debug("activated promo (empty queue)")
		return true
	}
	if first.GetNamespace() == promo.Namespace && first.GetName() == promo.Name {
		// This promo is the first in the queue. Mark it as active and pop it off the pending queue.
		popped := pq.Pop()
		pqs.activePromoByStage[stageKey] = popped.GetName()
		logger.Debug("activated promo")
		return true
	}
	return false
}

// deactivate removes the active entry for the given stage key.
// This should only be called after the active promotion has become terminal.
func (pqs *promoQueues) deactivate(ctx context.Context, stageKey types.NamespacedName, promoName string) {
	pqs.promoQueuesByStageMu.RLock()
	defer pqs.promoQueuesByStageMu.RUnlock()
	if pqs.activePromoByStage[stageKey] == promoName {
		logger := logging.LoggerFromContext(ctx).WithFields(log.Fields{
			"namespace": stageKey.Namespace,
			"promotion": promoName,
		})
		delete(pqs.activePromoByStage, stageKey)
		logger.Debug("deactivated promo")
	}
}
