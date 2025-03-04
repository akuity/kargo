package kargo

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/logging"
)

func NewPromoWentTerminalPredicate(logger *logging.Logger) PromoWentTerminal[*kargoapi.Promotion] {
	return PromoWentTerminal[*kargoapi.Promotion]{
		logger: logger,
	}
}

// PromoWentTerminal is a predicate that returns true if a promotion went terminal.
// Used by stage reconciler to enqueue a stage when it's associated promo is complete.
// Also used by promo reconciler to enqueue the next highest priority promotion.
type PromoWentTerminal[T any] struct {
	predicate.Funcs
	logger *logging.Logger
}

func (p PromoWentTerminal[T]) Create(event.TypedCreateEvent[T]) bool {
	return false
}

func (p PromoWentTerminal[T]) Delete(e event.TypedDeleteEvent[T]) bool {
	promo := any(e.Object).(*kargoapi.Promotion) // nolint: forcetypeassert
	// if promo is deleted but was non-terminal, we want to enqueue the
	// Stage so it can reset status.currentPromotion, as well as the
	// enqueue the next priority Promo for reconciliation
	return !promo.Status.Phase.IsTerminal()
}

func (p PromoWentTerminal[T]) Generic(event.TypedGenericEvent[T]) bool {
	// we should never get here
	return true
}

// Update implements default TypedUpdateEvent filter for checking if a promotion
// went terminal
func (p PromoWentTerminal[T]) Update(e event.TypedUpdateEvent[T]) bool {
	oldPromo := any(e.ObjectOld).(*kargoapi.Promotion) // nolint: forcetypeassert
	if oldPromo == nil {
		p.logger.Error(
			nil, "Update event has no old object to update",
			"event", e,
		)
		return false
	}
	newPromo := any(e.ObjectNew).(*kargoapi.Promotion) // nolint: forcetypeassert
	if newPromo == nil {
		p.logger.Error(
			nil, "Update event has no new object for update",
			"event", e,
		)
		return false
	}
	if newPromo.Status.Phase.IsTerminal() && !oldPromo.Status.Phase.IsTerminal() {
		return true
	}
	return false
}

func NewPromoPhaseChangedPredicate(logger *logging.Logger) PromoPhaseChanged[*kargoapi.Promotion] {
	return PromoPhaseChanged[*kargoapi.Promotion]{
		logger: logger,
	}
}

// PromoPhaseChanged is a predicate that returns true if the phase of a promotion
// has changed. It can be used to trigger the reconciliation of an associated
// object when the phase of a Promotion changes. A concrete example is to trigger
// the reconciliation of a Stage when the phase of a Promotion for that Stage
// changes, so that the Stage can update the last Promotion reference in its
// status.
type PromoPhaseChanged[T any] struct {
	predicate.Funcs
	logger *logging.Logger
}

func (p PromoPhaseChanged[T]) Create(event.TypedCreateEvent[T]) bool {
	return false
}

func (p PromoPhaseChanged[T]) Delete(e event.TypedDeleteEvent[T]) bool {
	promo := any(e.Object).(*kargoapi.Promotion) // nolint: forcetypeassert
	// If a Promotion is deleted while it is non-terminal, we want to enqueue
	// the associated Stage so that it can reset its status.currentPromotion.
	return !promo.Status.Phase.IsTerminal()
}

func (p PromoPhaseChanged[T]) Generic(event.TypedGenericEvent[T]) bool {
	return false
}

func (p PromoPhaseChanged[T]) Update(e event.TypedUpdateEvent[T]) bool {
	oldPromo := any(e.ObjectOld).(*kargoapi.Promotion) // nolint: forcetypeassert
	if oldPromo == nil {
		p.logger.Error(
			nil, "Update event has no old object for update",
			"event", e,
		)
		return false
	}
	newPromo := any(e.ObjectNew).(*kargoapi.Promotion) // nolint: forcetypeassert
	if newPromo == nil {
		p.logger.Error(
			nil, "Update event has no new object for update",
			"event", e,
		)
		return false
	}
	return newPromo.Status.Phase != oldPromo.Status.Phase
}

// RefreshRequested is a predicate that returns true if the refresh annotation
// has been set on a resource, or the value of the annotation has changed
// compared to the previous state.
type RefreshRequested struct {
	predicate.Funcs
}

// Update returns true if the refresh annotation has been set on the new object,
// or if the value of the annotation has changed compared to the old object.
func (p RefreshRequested) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	if newVal, newOk := api.RefreshAnnotationValue(e.ObjectNew.GetAnnotations()); newOk {
		if oldVal, oldOk := api.RefreshAnnotationValue(e.ObjectOld.GetAnnotations()); oldOk {
			return newVal != oldVal
		}
		return true
	}
	return false
}

// ReverifyRequested is a predicate that returns true if the reverify annotation
// has been set on a resource, or the ID of the request has changed compared to
// the previous state.
type ReverifyRequested struct {
	predicate.Funcs
}

// Update returns true if the reverify annotation has been set on the new object,
// or if the ID of the request has changed compared to the old object.
func (r ReverifyRequested) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	if newVal, newOk := api.ReverifyAnnotationValue(e.ObjectNew.GetAnnotations()); newOk {
		if oldVal, oldOk := api.ReverifyAnnotationValue(e.ObjectOld.GetAnnotations()); oldOk {
			return !newVal.ForID(oldVal.ID)
		}
		return true
	}
	return false
}

// VerificationAbortRequested is a predicate that returns true if the abort annotation has
// been set on a resource, or the ID of the request has changed compared to the
// previous state.
type VerificationAbortRequested struct {
	predicate.Funcs
}

// Update returns true if the abort annotation has been set on the new object,
// or if the ID of the request has changed compared to the old object.
func (p VerificationAbortRequested) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	if newVal, newOk := api.AbortVerificationAnnotationValue(e.ObjectNew.GetAnnotations()); newOk {
		if oldVal, oldOk := api.AbortVerificationAnnotationValue(e.ObjectOld.GetAnnotations()); oldOk {
			return !newVal.ForID(oldVal.ID)
		}
		return true
	}
	return false
}

// PromotionAbortRequested is a predicate that returns true if the abort
// annotation has been set on a resource, or the action of the request has
// changed compared to the previous state.
type PromotionAbortRequested struct {
	predicate.Funcs
}

// Update returns true if the abort annotation has been set on the new object,
// or if the action of the request has changed compared to the old object.
func (p PromotionAbortRequested) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	if newVal, newOk := api.AbortPromotionAnnotationValue(e.ObjectNew.GetAnnotations()); newOk {
		if oldVal, oldOk := api.AbortPromotionAnnotationValue(e.ObjectOld.GetAnnotations()); oldOk {
			return oldVal.Action != newVal.Action
		}
		return true
	}
	return false
}
