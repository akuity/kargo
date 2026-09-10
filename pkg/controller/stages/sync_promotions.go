package stages

import (
	"context"
	"fmt"
	"slices"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/conditions"
	"github.com/akuity/kargo/pkg/logging"
)

// syncPromotions synchronizes the Promotions for a Stage. It determines the
// current state of the Stage based on the Promotions that are running or have
// completed.
func (r *RegularStageReconciler) syncPromotions(
	ctx context.Context,
	stage *kargoapi.Stage,
	autoPromotionEnabled bool,
) (kargoapi.StageStatus, bool, error) {
	logger := logging.LoggerFromContext(ctx)
	newStatus := *stage.Status.DeepCopy()

	promotions, err := r.getPromotions(ctx, *stage)
	if err != nil {
		conditions.Set(&newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypePromoting,
			Status:             metav1.ConditionUnknown,
			Reason:             "ListPromotionsFailed",
			Message:            err.Error(),
			ObservedGeneration: stage.Generation,
		})

		return newStatus, false, err
	}

	// Without this, a child would occupy the Stage's own single Promotion
	// slot, serializing the very Promotions the request fanned out to run in
	// parallel -- and its terminal phases would replay into the Stage's
	// Freight history.
	promotions = withoutTargetPromotions(promotions)

	// Build a map of origin keys that are currently requested by this Stage,
	// used both to filter new holds and to evict stale ones.
	requestedOrigins := make(map[string]struct{}, len(stage.Spec.RequestedFreight))
	for _, req := range stage.Spec.RequestedFreight {
		requestedOrigins[req.Origin.String()] = struct{}{}
	}

	// While auto-promotion is disabled, holds are meaningless: the Stage's
	// current Freight is held in place by auto-promotion being disabled, not by
	// the holds. Clear them so that re-enabling auto-promotion is a uniform fresh
	// start, consistent with the fact that promoting non-candidate Freight while
	// auto-promotion is disabled never establishes a hold. (Hold establishment in
	// the replay below is gated on autoPromotionEnabled too.)
	if !autoPromotionEnabled {
		newStatus.AutoPromotionHolds = nil
	} else {
		// Drop holds for origins that are no longer requested. This runs before
		// the early-exit below so it takes effect even when there are no
		// Promotions.
		for key := range newStatus.AutoPromotionHolds {
			if _, ok := requestedOrigins[key]; !ok {
				delete(newStatus.AutoPromotionHolds, key)
			}
		}
		if len(newStatus.AutoPromotionHolds) == 0 {
			newStatus.AutoPromotionHolds = nil
		}
	}

	// If there are no Promotions, then we are not promoting any Freight.
	if len(promotions) == 0 {
		logger.Debug("no Promotions found for Stage")

		// Ensure we delete any existing "current" Promotion related information
		// from the Stage status.
		conditions.Delete(&newStatus, kargoapi.ConditionTypePromoting)
		newStatus.CurrentPromotion = nil

		return newStatus, false, nil
	}

	// The Promotion which is currently running on the Stage.
	currentPromo := stage.Status.CurrentPromotion

	summary := r.getPromotionsSummary(promotions, currentPromo)

	// Track if there are any pending promotions that need handling.
	// This is later used to determine if we should issue an immediate
	// requeue.
	// We don't count running promotions here.
	hasNonTerminalPromotions := summary.hasNonTerminalPromotions

	// There is a promotion this stage considers current
	if currentPromo != nil {
		// If current promotion exists and either terminal or not found in promotions,
		// we need to finalize that promotion
		if summary.currentPromotion == nil || summary.currentPromotion.Status.Phase.IsTerminal() {
			newStatus = r.finalizePromotion(
				ctx, stage, requestedOrigins, autoPromotionEnabled, newStatus, summary.terminal)
			return newStatus, hasNonTerminalPromotions, nil
		}

		// CurrentPromotion is still executing, track progress
		newStatus = r.trackPromotion(stage, newStatus, *summary.currentPromotion)
		return newStatus, hasNonTerminalPromotions, nil
	}

	// There is a new promotion in the queue not blocked by freight verification
	if summary.nextPromotion != nil && !r.verificationBlocksPromotion(ctx, stage) {
		// Allow next promotion
		newStatus = r.trackPromotion(stage, newStatus, *summary.nextPromotion)
		return newStatus, hasNonTerminalPromotions, nil
	}

	conditions.Delete(&newStatus, kargoapi.ConditionTypePromoting)
	return newStatus, hasNonTerminalPromotions, nil
}

func (r *RegularStageReconciler) verificationBlocksPromotion(ctx context.Context, stage *kargoapi.Stage) bool {
	logger := logging.LoggerFromContext(ctx)
	// If the current Freight exists and has a non-terminal verification, wait
	// for it to complete regardless of health state to ensure we capture the
	// results.
	if curFreight := stage.Status.FreightHistory.Current(); curFreight != nil {
		if curFreight.HasNonTerminalVerification() {
			logger.Debug(
				"current Freight has a non-terminal verification: " +
					"wait for it to complete before allowing new promotions to start",
			)
			return true
		}

		// If we are in a healthy state, the current Freight needs to be verified
		// before we can allow the next Promotion to start. If we are unhealthy
		// or the verification failed, then we can allow the next Promotion to
		// start immediately as the expectation is that the Promotion can fix the
		// issue.
		if stage.Status.Health == nil || stage.Status.Health.Status == kargoapi.HealthStateHealthy {
			curVI := curFreight.VerificationHistory.Current()
			if curVI == nil || !curVI.Phase.IsTerminal() {
				logger.Debug("current Freight needs to be verified before allowing new promotions to start")
				return true
			}
		}
	}
	return false
}

func (r *RegularStageReconciler) trackPromotion(
	stage *kargoapi.Stage,
	newStatus kargoapi.StageStatus,
	promotion kargoapi.Promotion,
) kargoapi.StageStatus {
	conditions.Set(&newStatus, &metav1.Condition{
		Type:   kargoapi.ConditionTypePromoting,
		Status: metav1.ConditionTrue,
		Reason: "ActivePromotion",
		Message: fmt.Sprintf(
			"Promotion %q is currently %s",
			promotion.Name, promotion.Status.Phase,
		),
		ObservedGeneration: stage.Generation,
	})

	newStatus.CurrentPromotion = &kargoapi.PromotionReference{
		Name: promotion.Name,
	}
	if freight := promotion.Status.Freight; freight != nil {
		newStatus.CurrentPromotion.Freight = freight.DeepCopy()
	}
	return newStatus
}

func (r *RegularStageReconciler) finalizePromotion(
	ctx context.Context,
	stage *kargoapi.Stage,
	requestedOrigins map[string]struct{},
	autoPromotionEnabled bool,
	newStatus kargoapi.StageStatus,
	terminalPromotions []kargoapi.Promotion,
) kargoapi.StageStatus {
	logger := logging.LoggerFromContext(ctx)
	// Update the conditions to reflect that we are no longer promoting.
	conditions.Delete(&newStatus, kargoapi.ConditionTypePromoting)
	newStatus.CurrentPromotion = nil

	// The last Promotion which ran on the Stage.
	lastPromo := stage.Status.LastPromotion

	// Gather terminal Promotions newer than the last processed one, sorted
	// oldest-to-newest so holds are applied in chronological order and
	// Freight history entries are appended oldest-first (GC removes oldest
	// first).
	var newPromos []*kargoapi.Promotion
	for i := range terminalPromotions {
		promo := &terminalPromotions[i]
		if lastPromo != nil {
			// We can break here since we know that all subsequent Promotions
			// will be older than the last Promotion we saw.
			// NB: This makes use of the fact that Promotion names are
			// generated, and contain a timestamp component which will ensure
			// that they can be sorted in a consistent order.
			if strings.Compare(promo.Name, lastPromo.Name) <= 0 {
				break
			}
		}
		if promo.Status.Phase.IsTerminal() {
			newPromos = append(newPromos, promo)
		}
	}
	slices.SortFunc(newPromos, func(a, b *kargoapi.Promotion) int {
		return strings.Compare(a.Name, b.Name)
	})

	// Replay new Promotions in chronological order to update hold state and
	// Stage status. Hold/release intent is applied in order so a later
	// release correctly supersedes an earlier hold and vice versa. Holds
	// persist in status even after their establishing Promotion is GC'd.
	for _, promo := range newPromos {
		// A Promotion's hold/resume intent is fixed at creation and is not
		// changed by an involuntary failure, so any terminal Promotion
		// applies its intent. The exception is an Aborted Promotion: the
		// user deliberately canceled it, withdrawing the intent along with
		// it. (newPromos contains only terminal Promotions.) Holds are only
		// maintained while auto-promotion is enabled; when disabled they are
		// cleared above and not re-established here.
		if autoPromotionEnabled && promo.Status.Phase != kargoapi.PromotionPhaseAborted {
			if originKey := promo.Annotations[kargoapi.AnnotationKeyAutoPromotionHold]; originKey != "" {
				if _, requested := requestedOrigins[originKey]; requested {
					if origin, err := kargoapi.ParseFreightOrigin(originKey); err == nil {
						if newStatus.AutoPromotionHolds == nil {
							newStatus.AutoPromotionHolds = make(map[string]kargoapi.AutoPromotionHold)
						}
						newStatus.AutoPromotionHolds[originKey] = newAutoPromotionHold(promo, origin)
					}
				}
			} else if originKey := promo.Annotations[kargoapi.AnnotationKeyAutoPromotionResume]; originKey != "" {
				delete(newStatus.AutoPromotionHolds, originKey)
			}
		}
		ref := kargoapi.PromotionReference{
			Name:       promo.Name,
			Status:     promo.Status.DeepCopy(),
			FinishedAt: promo.Status.FinishedAt,
		}
		if promo.Status.Freight != nil {
			ref.Freight = promo.Status.Freight.DeepCopy()
		}
		// A Promotion that was aborted before it ever reached Running never
		// had the chance to build a FreightCollection. Recording it as-is
		// would make the Stage forget Freight origins the previous
		// lastPromotion had already collected, permanently breaking any
		// subsequent Promotion's ability to inherit them.
		if promo.Status.StartedAt == nil && ref.Status.FreightCollection == nil &&
			newStatus.LastPromotion != nil && newStatus.LastPromotion.Status != nil {
			ref.Status.FreightCollection = newStatus.LastPromotion.Status.FreightCollection
		}
		newStatus.LastPromotion = &ref
		if promo.Status.Phase == kargoapi.PromotionPhaseSucceeded {
			// If the Promotion was successful, then we should add the Freight
			// to the history of successfully promoted Freight.
			newStatus.FreightHistory.Record(ref.Status.FreightCollection)

			// Erase any health checks that were performed for the previous
			// Freight, as they are no longer relevant.
			newStatus.Health = nil
			conditions.Set(&newStatus, &metav1.Condition{
				Type:               kargoapi.ConditionTypeHealthy,
				Status:             metav1.ConditionUnknown,
				Reason:             "WaitingForHealthCheck",
				Message:            "Waiting for health check to be performed after successful promotion",
				ObservedGeneration: stage.Generation,
			})

			// Set verified condition to unknown to indicate that the
			// new Freight needs to be verified.
			conditions.Set(&newStatus, &metav1.Condition{
				Type:               kargoapi.ConditionTypeVerified,
				Status:             metav1.ConditionUnknown,
				Reason:             "WaitingForVerification",
				Message:            "Waiting for verification to be performed after successful promotion",
				ObservedGeneration: stage.Generation,
			})

			// Annotate the Stage with the latest information related to
			// ArgoCD Applications. This is used to provide deep links to the
			// ArgoCD UI for the Stage in the Kargo UI.
			//
			// NB: If the Promotion did not involve any ArgoCD Applications,
			// then the annotation will be removed.
			if err := api.AnnotateStageWithArgoCDContext(
				ctx,
				r.client,
				promo,
				client.ObjectKeyFromObject(stage),
			); err != nil {
				// Let the error be logged, but do not return it as it is not
				// critical to the operation of the Stage.
				logger.Error(err, "failed to annotate Stage with ArgoCD context")
			}
		}
	}
	return newStatus
}

type promotionsSummary struct {
	terminal                 []kargoapi.Promotion
	currentPromotion         *kargoapi.Promotion
	hasNonTerminalPromotions bool
	nextPromotion            *kargoapi.Promotion
}

func (r *RegularStageReconciler) getPromotionsSummary(
	promotions []kargoapi.Promotion,
	currentPromoRef *kargoapi.PromotionReference,
) promotionsSummary {
	summary := promotionsSummary{}
	running := []kargoapi.Promotion{}
	pending := []kargoapi.Promotion{}
	for _, promo := range promotions {
		if currentPromoRef != nil && promo.Name == currentPromoRef.Name {
			summary.currentPromotion = &promo
		}
		if promo.Status.Phase == kargoapi.PromotionPhaseRunning {
			summary.hasNonTerminalPromotions = true
			running = append(running, promo)
		} else if promo.Status.Phase.IsTerminal() {
			summary.terminal = append(summary.terminal, promo)
		} else {
			summary.hasNonTerminalPromotions = true
			pending = append(pending, promo)
		}
	}

	// nextPromotion is either first one running or first one pending
	// If it's running, it should match currentPromotion, but we don't validate that
	if len(running) > 0 {
		summary.nextPromotion = new(slices.MinFunc(running, func(a, b kargoapi.Promotion) int {
			return strings.Compare(a.Name, b.Name)
		}))
	} else if len(pending) > 0 {
		summary.nextPromotion = new(slices.MinFunc(pending, func(a, b kargoapi.Promotion) int {
			return strings.Compare(a.Name, b.Name)
		}))
	}

	slices.SortFunc(summary.terminal, func(a, b kargoapi.Promotion) int {
		return strings.Compare(b.Name, a.Name)
	})
	return summary
}
