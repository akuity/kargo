package kargo

import (
	"context"
	"fmt"
	"strings"

	"github.com/oklog/ulid/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/user"
	"github.com/akuity/kargo/internal/logging"
)

const (
	// maximum length of the stage name used in the promotion name prefix before it exceeds
	// kubernetes resource name limit of 253
	// 253 - 1 (.) - 26 (ulid) - 1 (.) - 7 (sha) = 218
	maxStageNamePrefixLength = 218
)

// NewPromotion returns a new Promotion from a given stage and freight with our
// naming convention.
func NewPromotion(
	ctx context.Context,
	stage kargoapi.Stage,
	freight string,
) kargoapi.Promotion {
	shortHash := freight
	if len(shortHash) > 7 {
		shortHash = freight[0:7]
	}
	shortStageName := stage.Name
	if len(stage.Name) > maxStageNamePrefixLength {
		shortStageName = shortStageName[0:maxStageNamePrefixLength]
	}

	annotations := make(map[string]string, 1)
	// Put actor information to track on the controller side
	if u, ok := user.InfoFromContext(ctx); ok {
		annotations[kargoapi.AnnotationKeyCreateActor] = kargoapi.FormatEventUserActor(u)
	}

	// ulid.Make() is pseudo-random, not crypto-random, but we don't care.
	// We just want a unique ID that can be sorted lexicographically
	promoName := strings.ToLower(fmt.Sprintf("%s.%s.%s", shortStageName, ulid.Make(), shortHash))

	promotion := kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:        promoName,
			Namespace:   stage.Namespace,
			Annotations: annotations,
		},
		Spec: kargoapi.PromotionSpec{
			Stage:   stage.Name,
			Freight: freight,
		},
	}
	if stage.Spec.PromotionTemplate != nil {
		promotion.Spec.Steps = stage.Spec.PromotionTemplate.Spec.Steps
	}
	return promotion
}

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

	if newVal, newOk := kargoapi.RefreshAnnotationValue(e.ObjectNew.GetAnnotations()); newOk {
		if oldVal, oldOk := kargoapi.RefreshAnnotationValue(e.ObjectOld.GetAnnotations()); oldOk {
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

	if newVal, newOk := kargoapi.ReverifyAnnotationValue(e.ObjectNew.GetAnnotations()); newOk {
		if oldVal, oldOk := kargoapi.ReverifyAnnotationValue(e.ObjectOld.GetAnnotations()); oldOk {
			return !newVal.ForID(oldVal.ID)
		}
		return true
	}
	return false
}

// AbortRequested is a predicate that returns true if the abort annotation has
// been set on a resource, or the ID of the request has changed compared to the
// previous state.
type AbortRequested struct {
	predicate.Funcs
}

// Update returns true if the abort annotation has been set on the new object,
// or if the ID of the request has changed compared to the old object.
func (p AbortRequested) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	if newVal, newOk := kargoapi.AbortAnnotationValue(e.ObjectNew.GetAnnotations()); newOk {
		if oldVal, oldOk := kargoapi.AbortAnnotationValue(e.ObjectOld.GetAnnotations()); oldOk {
			return !newVal.ForID(oldVal.ID)
		}
		return true
	}
	return false
}
