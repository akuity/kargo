package stages

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/promotion/dispatch"
)

// groomPromotions retires shadowed auto-forward Promotions so that out-of-order
// dispatch stays correct. It is the enforcement-layer counterpart to the
// dispatch gate: the gate holds stale auto-forwards, grooming retires them.
//
// It maintains the invariant that at most one auto-forward is live per
// (Stage, origin) by stamping the supersede-intent annotation on the redundant
// ones; the Promotion controller finalizes the transition to the terminal
// Superseded phase. Grooming never writes Promotion status itself, and it only
// ever touches auto-forwards — manual-forwards and rollbacks are explicit,
// non-fungible operator actions and are left untouched.
//
// Two rules are applied per origin (supersede mode, the default):
//
//   - G1: among Pending auto-forwards, keep the newest and supersede the rest
//     (they target older Freight; the newest already covers them).
//   - G1b: if a live (Pending or Running) manual-forward exists for the origin,
//     supersede every Pending auto-forward for it — the manual is a deliberate
//     operator choice that an auto-forward must not run after and undo.
//
// Grooming does not mutate Stage status; it returns the Stage's status
// unchanged so it composes cleanly as a sub-reconciler.
func (r *RegularStageReconciler) groomPromotions(
	ctx context.Context,
	stage *kargoapi.Stage,
) (kargoapi.StageStatus, error) {
	logger := logging.LoggerFromContext(ctx)

	promotions := &kargoapi.PromotionList{}
	if err := r.client.List(
		ctx,
		promotions,
		client.InNamespace(stage.Namespace),
		client.MatchingFieldsSelector{
			Selector: fields.OneTermEqualSelector(indexer.PromotionsByStageField, stage.Name),
		},
	); err != nil {
		return stage.Status, fmt.Errorf(
			"failed to list Promotions for Stage %q in namespace %q: %w",
			stage.Name, stage.Namespace, err,
		)
	}

	// Bucket the live Promotions by origin. Resolving the origin requires the
	// Freight (a FreightReference carries no origin on the Promotion itself),
	// so grooming is Freight-aware by design -- the O(pending) GetFreight cost
	// the gate deliberately avoids, accepted here.
	pendingAutos := map[string][]*kargoapi.Promotion{}
	// newestManual maps an origin to the newest live manual-forward for it.
	newestManual := map[string]*kargoapi.Promotion{}

	// The Promotion the gate just selected for dispatch is off-limits: it is on
	// its way to Running, so grooming must never retire it out from under the
	// gate's decision.
	var currentPromo string
	if stage.Status.CurrentPromotion != nil {
		currentPromo = stage.Status.CurrentPromotion.Name
	}

	for i := range promotions.Items {
		promo := &promotions.Items[i]

		// Only live Promotions matter: Pending autos are candidates for
		// retirement; Pending or Running manuals can displace them.
		running := promo.Status.Phase == kargoapi.PromotionPhaseRunning
		if !isPendingPhase(promo.Status.Phase) && !running {
			continue
		}

		// Never touch the Promotion the gate is dispatching.
		if promo.Name == currentPromo {
			continue
		}

		class := dispatch.ClassOf(promo)
		if class == dispatch.ClassRollback {
			// Rollbacks are never coalesced and never displace autos here
			// (AX8); the gate handles their priority.
			continue
		}

		origin, err := r.freightOrigin(ctx, stage.Namespace, promo.Spec.Freight)
		if err != nil {
			// Fail open for this Promotion: an unresolvable Freight (e.g. GC'd)
			// simply means we cannot scope it to an origin, so we leave it be.
			logger.Debug(
				"skipping Promotion during grooming; could not resolve Freight origin",
				"promotion", promo.Name,
				"freight", promo.Spec.Freight,
				"error", err,
			)
			continue
		}

		switch class {
		case dispatch.ClassAutoForward:
			if isPendingPhase(promo.Status.Phase) {
				pendingAutos[origin] = append(pendingAutos[origin], promo)
			}
		case dispatch.ClassManualForward:
			// G1b fires only for a manual that deliberately promoted a
			// NON-candidate Freight for this origin -- the #3016 case, marked at
			// creation by the auto-promotion hold-intent annotation the webhook
			// stamps (value == the origin). A manual promoting the current
			// candidate carries resume intent instead and poses no regression
			// risk, so it never displaces autos; those are handled by G1 and the
			// gate's guards. Keying on the same annotation that (on success)
			// establishes the #6334 hold keeps grooming and the hold in lockstep:
			// grooming retires the already-created rival, the hold suppresses its
			// recreation.
			if promo.Annotations[kargoapi.AnnotationKeyAutoPromotionHold] != origin {
				continue
			}
			if cur := newestManual[origin]; cur == nil || promo.Name > cur.Name {
				newestManual[origin] = promo
			}
		}
	}

	var superseded int
	for origin, autos := range pendingAutos {
		// G1b: a live manual-forward displaces every competing Pending auto for
		// the origin.
		if manual := newestManual[origin]; manual != nil {
			for _, auto := range autos {
				if err := r.supersede(ctx, auto, manual.Name); err != nil {
					return stage.Status, err
				}
				superseded++
			}
			continue
		}

		// G1: keep the newest Pending auto (highest ULID) and supersede the
		// rest. For auto-forwards creation order tracks Freight order, so the
		// newest necessarily targets Freight at least as new as the others.
		if len(autos) < 2 {
			continue
		}
		newest := autos[0]
		for _, auto := range autos[1:] {
			if auto.Name > newest.Name {
				newest = auto
			}
		}
		for _, auto := range autos {
			if auto.Name == newest.Name {
				continue
			}
			if err := r.supersede(ctx, auto, newest.Name); err != nil {
				return stage.Status, err
			}
			superseded++
		}
	}

	if superseded > 0 {
		logger.Debug("groomed shadowed auto-forward Promotions", "count", superseded)
	}

	return stage.Status, nil
}

// supersede stamps the supersede-intent annotation on the given Promotion,
// skipping those already carrying it so grooming stays idempotent.
func (r *RegularStageReconciler) supersede(
	ctx context.Context,
	promo *kargoapi.Promotion,
	supersededBy string,
) error {
	if _, ok := api.SupersedePromotionAnnotationValue(promo.GetAnnotations()); ok {
		return nil
	}
	return api.SupersedePromotion(
		ctx,
		r.client,
		types.NamespacedName{Namespace: promo.Namespace, Name: promo.Name},
		supersededBy,
	)
}

// freightOrigin resolves the origin key ("Kind/name") of the named Freight in
// the given namespace.
func (r *RegularStageReconciler) freightOrigin(
	ctx context.Context,
	namespace, name string,
) (string, error) {
	freight, err := api.GetFreight(ctx, r.client, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	})
	if err != nil {
		return "", err
	}
	if freight == nil {
		// nolint:staticcheck
		return "", fmt.Errorf("Freight %q in namespace %q not found", name, namespace)
	}
	return freight.Origin.String(), nil
}
