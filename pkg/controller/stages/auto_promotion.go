package stages

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	kargoEvent "github.com/akuity/kargo/pkg/event"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/logging"
)

// newAutoPromotionHold builds an AutoPromotionHold for origin from the
// hold-intent Promotion promo.
func newAutoPromotionHold(
	promo *kargoapi.Promotion,
	origin kargoapi.FreightOrigin,
) kargoapi.AutoPromotionHold {
	hold := kargoapi.AutoPromotionHold{
		FreightName:   promo.Spec.Freight,
		Origin:        origin,
		PromotionName: promo.Name,
	}
	if actor := promo.Annotations[kargoapi.AnnotationKeyCreateActor]; actor != "" {
		hold.Actor = actor
	}
	if !promo.CreationTimestamp.IsZero() {
		t := promo.CreationTimestamp
		hold.CreatedAt = &t
	}
	return hold
}

// computeEffectiveAutoPromotionHolds returns the set of auto-promotion holds in
// effect for the Stage right now. It starts from the durable holds in
// Status.AutoPromotionHolds -- preserving a hold whose establishing Promotion
// has been garbage-collected -- and overlays the newest non-aborted intent for
// each requested origin: a hold-intent Promotion holds the origin, a
// release-intent Promotion clears it, and the newest of the two wins. It
// returns an empty map while auto-promotion is disabled, when holds are
// meaningless. It does not mutate stage.
func (r *RegularStageReconciler) computeEffectiveAutoPromotionHolds(
	ctx context.Context,
	stage *kargoapi.Stage,
	autoPromotionEnabled bool,
) (map[string]kargoapi.AutoPromotionHold, error) {
	if !autoPromotionEnabled {
		return nil, nil
	}

	effective := make(map[string]kargoapi.AutoPromotionHold, len(stage.Status.AutoPromotionHolds))
	for key, hold := range stage.Status.AutoPromotionHolds {
		effective[key] = hold
	}

	promotions, err := r.getPromotions(ctx, *stage)
	if err != nil {
		return nil, err
	}

	promotions = withoutTargetPromotions(promotions)

	lastPromo := stage.Status.LastPromotion
	for _, req := range stage.Spec.RequestedFreight {
		originKey := req.Origin.String()
		var newest *kargoapi.Promotion
		var newestIsHold bool
		for i := range promotions {
			promo := &promotions[i]
			if promo.Status.Phase == kargoapi.PromotionPhaseAborted {
				continue
			}
			// Only Promotions newer than the last one syncPromotions recorded can
			// change the durable state. Older ones are already reflected in it, and
			// the Promotion that superseded them may since have been deleted.
			if lastPromo != nil && strings.Compare(promo.Name, lastPromo.Name) <= 0 {
				continue
			}
			isHold := promo.Annotations[kargoapi.AnnotationKeyAutoPromotionHold] == originKey
			isRelease := promo.Annotations[kargoapi.AnnotationKeyAutoPromotionResume] == originKey
			if !isHold && !isRelease {
				continue
			}
			if newest == nil || strings.Compare(promo.Name, newest.Name) > 0 {
				newest = promo
				newestIsHold = isHold
			}
		}
		switch {
		case newest == nil:
			// No in-flight intent for this origin; leave the durable state as-is.
		case newestIsHold:
			effective[originKey] = newAutoPromotionHold(newest, req.Origin)
		default:
			delete(effective, originKey)
		}
	}
	return effective, nil
}

// autoPromoteFreight automatically promotes the candidate Freight for each
// requested origin, unless auto-promotion is disabled or the origin has an
// effective auto-promotion hold.
func (r *RegularStageReconciler) autoPromoteFreight(
	ctx context.Context,
	stage *kargoapi.Stage,
	autoPromotionEnabled bool,
) (kargoapi.StageStatus, error) {
	logger := logging.LoggerFromContext(ctx)
	newStatus := *stage.Status.DeepCopy()
	newStatus.AutoPromotionEnabled = autoPromotionEnabled

	// If the Stage has no requested Freight, then there is nothing to promote.
	// NB: This should not happen in practice, as a Stage cannot exist without
	// requested Freight.
	if len(stage.Spec.RequestedFreight) == 0 {
		return newStatus, nil
	}

	logger.Debug("checked auto-promotion policy for Stage", "enabled", autoPromotionEnabled)
	if !autoPromotionEnabled {
		// Nothing to promote. The durable and effective hold maps are cleared by
		// syncPromotions and computeEffectiveAutoPromotionHolds respectively when
		// auto-promotion is disabled, not here.
		return newStatus, nil
	}

	availableFreight, err := api.ListFreightAvailableToStage(ctx, r.client, stage)
	if err != nil {
		return newStatus, fmt.Errorf(
			"error listing available Freight for Stage %q: %w",
			stage.Name, err,
		)
	}
	candidates := api.SelectAutoPromotionCandidates(ctx, stage, availableFreight)
	// If the Stage has no current Freight, any candidate is new to it.
	currentFreight := newStatus.FreightHistory.Current()

	// Check if there is any new Freight which can be auto-promoted.
	for _, req := range stage.Spec.RequestedFreight {
		origin := req.Origin.String()
		// Never create an auto-promotion for an origin with an effective hold.
		if _, held := stage.Status.EffectiveAutoPromotionHolds[origin]; held {
			logger.Debug("auto-promotion is blocked by an auto-promotion hold", "origin", origin)
			continue
		}

		candidate, exists := candidates[origin]
		if !exists {
			logger.Debug("no Freight from origin available for auto-promotion", "origin", origin)
			continue
		}

		freightLogger := logger.WithValues("origin", origin, "freight", candidate.Name)

		// Only proceed if the candidate Freight is not already current in the Stage.
		if freightCollectionHasFreight(currentFreight, origin, candidate.Name) {
			freightLogger.Debug("Stage already has candidate Freight for origin")
			continue
		}
		if stageAwaitingFreightForOrigin(stage, origin, candidate.Name) {
			freightLogger.Debug("Stage is already awaiting candidate Freight for origin")
			continue
		}

		if api.IsTargetAware(stage) {
			if err = r.createAutoPromotionRequest(ctx, stage, &candidate, origin); err != nil {
				return newStatus, err
			}
		} else {
			if err = r.createAutoPromotion(ctx, stage, &candidate, origin); err != nil {
				return newStatus, err
			}
		}
	}

	return newStatus, nil
}

func (r *RegularStageReconciler) createAutoPromotion(
	ctx context.Context,
	stage *kargoapi.Stage,
	candidate *kargoapi.Freight,
	origin string) error {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"origin", origin,
		"freight", candidate.Name,
	)
	// Do not create duplicate work: stand down while any Promotion for
	// this candidate is either still in flight or succeeded with an
	// outcome not yet recorded in Stage status.
	var unprocessedPromotionExists bool
	unprocessedPromotionExists, err := r.unprocessedPromotionExistsForStageFreight(
		ctx,
		stage,
		candidate.Name,
	)
	if err != nil {
		return fmt.Errorf(
			"error listing existing Promotions for Freight %q in namespace "+
				"%q: %w",
			candidate.Name, stage.Namespace, err,
		)
	}
	if unprocessedPromotionExists {
		logger.Debug("an unprocessed Promotion already exists for " +
			"Stage and Freight")
		return nil
	}

	var newestPromotion *kargoapi.Promotion
	newestPromotion, err = r.newestTerminalPromotionForStageFreight(
		ctx,
		stage,
		candidate.Name,
	)
	if err != nil {
		return fmt.Errorf(
			"error listing existing terminal Promotions for Freight %q in "+
				"namespace %q: %w",
			candidate.Name, stage.Namespace, err,
		)
	}
	if newestPromotion != nil &&
		newestPromotion.Status.Phase != kargoapi.PromotionPhaseSucceeded {
		logger.Debug(
			"most recent terminal Promotion for Stage and Freight was not "+
				"successful; skipping auto-promotion to avoid an infinite loop",
			"lastPromotion", newestPromotion.Name,
			"lastPromotionPhase", newestPromotion.Status.Phase,
		)
		return nil
	}

	// Auto-promote the candidate Freight and record an event. Create a minimal
	// Promotion. The defaulting webhook fills in the rest from the Stage's
	// PromotionTemplate.
	promotion := api.NewMinimalPromotion(stage, candidate.Name)
	if err := r.client.Create(ctx, promotion); err != nil {
		// An admission webhook may deny the create. Tolerate this as a
		// non-error: nothing is persisted, so this reconcile moves on, and a
		// later one re-derives and re-attempts the auto-promotion once the
		// denying policy no longer applies. Any other error is still fatal to
		// the reconcile.
		//
		// Deliberately not recorded as an event. A policy that holds keeps
		// denying, so this branch is taken on every reconcile for as long as
		// it does; an event per occurrence would report one unchanging
		// condition thousands of times and bury the Project's event feed. A
		// condition belongs in status, and Kargo Enterprise reports the one
		// it knows about in Stage.status.promotionSchedule.
		if apierrors.IsForbidden(err) {
			logger.Debug(
				"auto-promotion was denied by an admission webhook",
				"error", err.Error(),
			)
			return nil
		}
		return fmt.Errorf(
			"error creating Promotion for Freight %q in namespace %q: %w",
			candidate.Name, stage.Namespace, err,
		)
	}
	evt := kargoEvent.NewPromotionCreated(
		fmt.Sprintf("Automatically promoted Freight from origin %q for Stage %q",
			origin,
			promotion.Spec.Stage),
		api.FormatEventControllerActor(r.cfg.Name()),
		promotion,
		candidate,
	)
	if err := r.eventSender.Send(ctx, evt); err != nil {
		logger.Error(err, "failed to send promotion event")
	}
	logger.Debug(
		"created Promotion resource",
		"promotion", promotion.Name,
	)
	return nil

}

// createAutoPromotionRequest creates a PromotionRequest expressing the intent
// to promote the candidate Freight to the Targets that the target-aware Stage
// governs. Those Targets are resolved once, as the request is built, and
// recorded on it; the request does not promote anything itself.
//
// The guard against duplicate work here is deliberately stricter than the one
// autoPromoteFreight applies to Promotions: a PromotionRequest is created only when
// no PromotionRequest for this Stage and Freight exists at all, in any phase.
// Stage status now records the current and last PromotionRequest, but those are
// mirrors of a request's own phase, not of a Stage having absorbed its outcome:
// a PromotionRequest promotes nothing itself, so there is still no equivalent of
// "succeeded, but the outcome is not yet recorded in status" to reason about --
// and absent a guard that holds unconditionally, every reconcile would create
// another PromotionRequest.
//
// The guard is confined to auto-promotion. Promoting the same Freight to the
// same Stage again deliberately -- rolling back to it, say -- goes through the
// API server, which creates a PromotionRequest unconditionally.
func (r *RegularStageReconciler) createAutoPromotionRequest(
	ctx context.Context,
	stage *kargoapi.Stage,
	candidate *kargoapi.Freight,
	origin string,
) error {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"origin", origin,
		"freight", candidate.Name,
	)

	exists, err := r.promotionRequestExistsForStageFreight(ctx, stage, candidate.Name)
	if err != nil {
		return fmt.Errorf(
			"error listing existing PromotionRequests for Freight %q in namespace %q: %w",
			candidate.Name, stage.Namespace, err,
		)
	}
	if exists {
		logger.Debug("a PromotionRequest already exists for Stage and Freight")
		return nil
	}

	promotionRequest, err := api.NewPromotionRequest(ctx, r.client, stage, candidate.Name)
	if err != nil {
		return fmt.Errorf(
			"error building PromotionRequest for Freight %q in namespace %q: %w",
			candidate.Name, stage.Namespace, err,
		)
	}

	if err = r.client.Create(ctx, promotionRequest); err != nil {
		// Tolerate an admission denial exactly as the Promotion path does:
		// nothing is persisted, so a later reconcile re-attempts once the
		// denying policy no longer applies.
		if apierrors.IsForbidden(err) {
			logger.Debug(
				"auto-promotion was denied by an admission webhook",
				"error", err.Error(),
			)
			return nil
		}
		return fmt.Errorf(
			"error creating PromotionRequest for Freight %q in namespace %q: %w",
			candidate.Name, stage.Namespace, err,
		)
	}

	// No event is recorded. Kargo's promotion events carry a Promotion, and a
	// PromotionRequest has none of its own; the events belong to the child
	// Promotions that its reconciler creates.
	logger.Debug(
		"created PromotionRequest resource",
		"promotionRequest", promotionRequest.Name,
	)
	return nil
}

// promotionRequestExistsForStageFreight reports whether any PromotionRequest exists for
// the given Stage and Freight, in any phase.
func (r *RegularStageReconciler) promotionRequestExistsForStageFreight(
	ctx context.Context,
	stage *kargoapi.Stage,
	freightName string,
) (bool, error) {
	promotionRequests := &kargoapi.PromotionRequestList{}
	if err := r.client.List(
		ctx,
		promotionRequests,
		client.InNamespace(stage.Namespace),
		client.MatchingFieldsSelector{
			Selector: fields.OneTermEqualSelector(
				indexer.PromotionRequestsByStageAndFreightField,
				indexer.StageAndFreightKey(stage.Name, freightName),
			),
		},
	); err != nil {
		return false, err
	}
	return len(promotionRequests.Items) > 0, nil
}

// stageAwaitingFreightForOrigin reports whether this reconcile pass has already
// observed a Promotion for the named Freight and origin. autoPromoteFreight uses
// it to avoid creating a duplicate Promotion before the status patch from
// syncPromotions reaches the API server.
func stageAwaitingFreightForOrigin(
	stage *kargoapi.Stage,
	origin string,
	name string,
) bool {
	if stage.Status.CurrentPromotion == nil ||
		stage.Status.CurrentPromotion.Freight == nil {
		return false
	}
	// Reconcile patches stage.Status back onto the in-memory Stage after each
	// sub-reconciler, so this sees Promotions observed earlier in this pass.
	return stage.Status.CurrentPromotion.Freight.Name == name &&
		stage.Status.CurrentPromotion.Freight.Origin.String() == origin
}

// unprocessedPromotionExistsForStageFreight reports whether a Promotion for
// this Stage and Freight exists whose outcome syncPromotions has not yet
// recorded: one that is still non-terminal, or one that SUCCEEDED after this
// reconciliation's view of the Stage was computed (i.e., is newer than
// status.lastPromotion). autoPromoteFreight uses it to avoid creating
// duplicate work for the same candidate. The second case matters because a
// fast Promotion can go from pending to succeeded in the interval between
// syncPromotions observing it and autoPromoteFreight acting; a
// non-terminal-only check misses it, and a succeeded Promotion for the
// candidate is deliberately not otherwise disqualifying. Once the next
// reconciliation records the success, the candidate-is-already-current check
// takes over. Promotions that reached any other terminal phase are handled
// by the newest-terminal-not-successful check regardless of whether they
// have been recorded, so they never block here.
func (r *RegularStageReconciler) unprocessedPromotionExistsForStageFreight(
	ctx context.Context,
	stage *kargoapi.Stage,
	freightName string,
) (bool, error) {
	promotions := &kargoapi.PromotionList{}
	if err := r.client.List(
		ctx,
		promotions,
		client.InNamespace(stage.Namespace),
		client.MatchingFieldsSelector{
			Selector: fields.OneTermEqualSelector(
				indexer.PromotionsByStageAndFreightField,
				indexer.StageAndFreightKey(stage.Name, freightName),
			),
		},
	); err != nil {
		return false, err
	}
	promotions.Items = withoutTargetPromotions(promotions.Items)
	lastPromo := stage.Status.LastPromotion
	for i := range promotions.Items {
		promo := &promotions.Items[i]
		if !promo.Status.Phase.IsTerminal() ||
			(promo.Status.Phase == kargoapi.PromotionPhaseSucceeded &&
				(lastPromo == nil || strings.Compare(promo.Name, lastPromo.Name) > 0)) {
			return true, nil
		}
	}
	return false, nil
}

// newestTerminalPromotionForStageFreight returns the newest completed Promotion
// for this Stage and Freight. autoPromoteFreight uses it to avoid retrying
// terminal failures in a loop.
func (r *RegularStageReconciler) newestTerminalPromotionForStageFreight(
	ctx context.Context,
	stage *kargoapi.Stage,
	freightName string,
) (*kargoapi.Promotion, error) {
	promotions := &kargoapi.PromotionList{}
	if err := r.client.List(
		ctx,
		promotions,
		client.InNamespace(stage.Namespace),
		client.MatchingFieldsSelector{
			Selector: fields.AndSelectors(
				fields.OneTermEqualSelector(
					indexer.PromotionsByStageAndFreightField,
					indexer.StageAndFreightKey(stage.Name, freightName),
				),
				fields.OneTermEqualSelector(
					indexer.PromotionsByTerminalField,
					strconv.FormatBool(true),
				),
			),
		},
	); err != nil {
		return nil, err
	}
	promotions.Items = withoutTargetPromotions(promotions.Items)
	if len(promotions.Items) == 0 {
		return nil, nil
	}
	slices.SortFunc(promotions.Items, func(lhs, rhs kargoapi.Promotion) int {
		if result := rhs.CreationTimestamp.Compare(lhs.CreationTimestamp.Time); result != 0 {
			return result
		}
		return strings.Compare(rhs.Name, lhs.Name)
	})
	return &promotions.Items[0], nil
}

// freightCollectionHasFreight checks a single origin in a FreightCollection.
func freightCollectionHasFreight(
	collection *kargoapi.FreightCollection,
	origin string,
	name string,
) bool {
	if collection == nil || len(collection.Freight) == 0 {
		return false
	}
	freightRef, ok := collection.Freight[origin]
	return ok && freightRef.Name == name
}
