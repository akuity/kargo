package stages

import (
	"context"
	"fmt"
	"slices"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	kargoEvent "github.com/akuity/kargo/pkg/event"
	"github.com/akuity/kargo/pkg/logging"
)

// newAutoPromotionHold builds an AutoPromotionHold for origin from the
// hold-intent Promotion promo.
func newAutoPromotionHold(
	promo PromotionObject,
	origin kargoapi.FreightOrigin,
) kargoapi.AutoPromotionHold {
	hold := kargoapi.AutoPromotionHold{
		FreightName:   promo.GetFreightName(),
		Origin:        origin,
		PromotionName: promo.GetName(),
	}
	if actor := promo.GetAnnotations()[kargoapi.AnnotationKeyCreateActor]; actor != "" {
		hold.Actor = actor
	}
	if !promo.GetCreationTimestamp().Time.IsZero() {
		t := promo.GetCreationTimestamp()
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

	promotions, err := r.getPromotionObjectsByStage(ctx, stage)
	if err != nil {
		return nil, err
	}
	fmt.Printf("PROMOS: %v\n", promotions)

	lastPromo := getLastPromoObject(stage)
	for _, req := range stage.Spec.RequestedFreight {
		originKey := req.Origin.String()
		var newest PromotionObject
		var newestIsHold bool
		for i := range promotions {
			promo := promotions[i]
			if promo.GetPhase().IsAborted() {
				fmt.Printf("PROMO ABORTED: %v\n", promo)
				continue
			}
			// Only Promotions newer than the last one syncPromotions recorded can
			// change the durable state. Older ones are already reflected in it, and
			// the Promotion that superseded them may since have been deleted.
			if lastPromo != nil && strings.Compare(promo.GetName(), lastPromo.GetName()) <= 0 {
				fmt.Printf("LASTPROMO NOT NIL: %v\n",lastPromo)
				continue
			}
			annotations := promo.GetAnnotations()
			fmt.Printf("ANNOTATIONS: %v\n PROMO %v\n", annotations, promo)
			isHold := promo.GetAnnotations()[kargoapi.AnnotationKeyAutoPromotionHold] == originKey
			isRelease := promo.GetAnnotations()[kargoapi.AnnotationKeyAutoPromotionResume] == originKey
			if !isHold && !isRelease {
				continue
			}
			if newest == nil || strings.Compare(promo.GetName(), newest.GetName()) > 0 {
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

		if err := r.maybeCreateAutoPromotionObject(ctx, stage, &candidate, origin); err != nil {
			return newStatus, err
		}
	}

	return newStatus, nil
}

func (r *RegularStageReconciler) unprocessedPromotionObjectExists(
	stage *kargoapi.Stage,
	promotions []PromotionObject,
) bool {
	lastPromo := getLastPromoObject(stage)
	for i := range promotions {
		promo := promotions[i]
		promoPhase := promo.GetPhase()
		if !promoPhase.IsTerminal() ||
			(promoPhase.IsSucceeded() &&
				(lastPromo == nil || strings.Compare(promo.GetName(), lastPromo.GetName()) > 0)) {
			return true
		}
	}
	return false
}

func (r *RegularStageReconciler) newestTerminalPromotionObject(
	promotions []PromotionObject,
) PromotionObject {
	if len(promotions) == 0 {
		return nil
	}
	return slices.MaxFunc(promotions, func(lhs, rhs PromotionObject) int {
		if result := lhs.GetCreationTimestamp().Compare(rhs.GetCreationTimestamp().Time); result != 0 {
			return result
		}
		return strings.Compare(lhs.GetName(), rhs.GetName())
	})
}

func (r *RegularStageReconciler) maybeCreateAutoPromotionObject(
	ctx context.Context,
	stage *kargoapi.Stage,
	candidate *kargoapi.Freight,
	origin string,
) error {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"origin", origin,
		"freight", candidate.Name,
	)
	existingPromotionsForFreight, err := r.getPromotionObjectsByStageAndFreight(ctx, stage, candidate.Name)
	if err != nil {
		return fmt.Errorf(
			"error listing existing Promotion Objects for Freight %q in namespace "+
				"%q: %w",
			candidate.Name, stage.Namespace, err,
		)
	}
	// Do not create duplicate work: stand down while any Promotion for
	// this candidate is either still in flight or succeeded with an
	// outcome not yet recorded in Stage status.
	if r.unprocessedPromotionObjectExists(stage, existingPromotionsForFreight) {
		logger.Debug("an unprocessed Promotion already exists for " +
			"Stage and Freight")
		return nil
	}

	newestPromotion := r.newestTerminalPromotionObject(existingPromotionsForFreight)

	if err != nil {
		return fmt.Errorf(
			"error listing existing terminal Promotions for Freight %q in "+
				"namespace %q: %w",
			candidate.Name, stage.Namespace, err,
		)
	}

	if newestPromotion != nil &&
		!newestPromotion.GetPhase().IsSucceeded() {
		logger.Debug(
			"most recent terminal Promotion for Stage and Freight was not "+
				"successful; skipping auto-promotion to avoid an infinite loop",
			"lastPromotion", newestPromotion.GetName(),
			"lastPromotionPhase", newestPromotion.GetPhase(),
		)
		return nil
	}

	if !api.IsTargetAware(stage) {
		return r.createAutoPromotion(ctx, stage, candidate, origin)
	}
	// FIXME: cleanup this change after verifying that it's not necessary
	if len(existingPromotionsForFreight) > 0 {
		logger.Debug("a PromotionRequest already exists for Stage and Freight")
		return nil
	}
	return r.createAutoPromotionRequest(ctx, stage, candidate, origin)
}

func (r *RegularStageReconciler) createAutoPromotion(
	ctx context.Context,
	stage *kargoapi.Stage,
	candidate *kargoapi.Freight,
	origin string,
) error {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"origin", origin,
		"freight", candidate.Name,
	)

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

	// FIXME: add event here

	// No event is recorded. Kargo's promotion events carry a Promotion, and a
	// PromotionRequest has none of its own; the events belong to the child
	// Promotions that its reconciler creates.
	logger.Debug(
		"created PromotionRequest resource",
		"promotionRequest", promotionRequest.Name,
	)
	return nil
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
	currentPromo := getCurrentPromoObject(stage)
	if currentPromo == nil || currentPromo.GetFreightReference() == nil {
		return false
	}
	freightRef := currentPromo.GetFreightReference()
	// Reconcile patches stage.Status back onto the in-memory Stage after each
	// sub-reconciler, so this sees Promotions observed earlier in this pass.
	return freightRef.Name == name && freightRef.Origin.String() == origin
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

func refreshAutoPromotionHolds(
	newStatus kargoapi.StageStatus,
	promo PromotionObject,
	requestedOrigins map[string]struct{},
) kargoapi.StageStatus {
	// A Promotion's hold/resume intent is fixed at creation and is not
	// changed by an involuntary failure, so any terminal Promotion
	// applies its intent. The exception is an Aborted Promotion: the
	// user deliberately canceled it, withdrawing the intent along with
	// it. (newPromos contains only terminal Promotions.) Holds are only
	// maintained while auto-promotion is enabled; when disabled they are
	// cleared above and not re-established here.
	if !promo.GetPhase().IsAborted() {
		if originKey := promo.GetAnnotations()[kargoapi.AnnotationKeyAutoPromotionHold]; originKey != "" {
			if _, requested := requestedOrigins[originKey]; requested {
				if origin, err := kargoapi.ParseFreightOrigin(originKey); err == nil {
					if newStatus.AutoPromotionHolds == nil {
						newStatus.AutoPromotionHolds = make(map[string]kargoapi.AutoPromotionHold)
					}
					newStatus.AutoPromotionHolds[originKey] = newAutoPromotionHold(promo, origin)
				}
			}
		} else if originKey := promo.GetAnnotations()[kargoapi.AnnotationKeyAutoPromotionResume]; originKey != "" {
			delete(newStatus.AutoPromotionHolds, originKey)
		}
	}
	return newStatus
}
