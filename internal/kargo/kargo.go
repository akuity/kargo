package kargo

import (
	"fmt"
	"strings"

	"github.com/oklog/ulid/v2"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

const (
	// maximum length of the stage name used in the promotion name prefix before it exceeds
	// kubernetes resource name limit of 253
	// 253 - 1 (.) - 26 (ulid) - 1 (.) - 7 (sha) = 218
	maxStageNamePrefixLength = 218
)

// NewPromotion returns a new Promotion from a given stage and freight with our
// naming convention.
func NewPromotion(stage kargoapi.Stage, freight string) kargoapi.Promotion {
	shortHash := freight
	if len(shortHash) > 7 {
		shortHash = freight[0:7]
	}
	shortStageName := stage.Name
	if len(stage.Name) > maxStageNamePrefixLength {
		shortStageName = shortStageName[0:maxStageNamePrefixLength]
	}

	// ulid.Make() is pseudo-random, not crypto-random, but we don't care.
	// We just want a unique ID that can be sorted lexicographically
	promoName := strings.ToLower(fmt.Sprintf("%s.%s.%s", shortStageName, ulid.Make(), shortHash))

	promotion := kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      promoName,
			Namespace: stage.Namespace,
		},
		Spec: &kargoapi.PromotionSpec{
			Stage:   stage.Name,
			Freight: freight,
		},
	}
	return promotion
}

func NewPromoWentTerminalPredicate(logger *log.Entry) PromoWentTerminal {
	return PromoWentTerminal{
		logger: logger,
	}
}

// PromoWentTerminal is a predicate that returns true if a promotion went terminal.
// Used by stage reconciler to enqueue a stage when it's associated promo is complete.
// Also used by promo reconciler to enqueue the next highest priority promotion.
type PromoWentTerminal struct {
	predicate.Funcs
	logger *log.Entry
}

func (p PromoWentTerminal) Create(_ event.CreateEvent) bool {
	return false
}

func (p PromoWentTerminal) Delete(e event.DeleteEvent) bool {
	promo, ok := e.Object.(*kargoapi.Promotion)
	// if promo is deleted but was non-terminal, we want to enqueue the
	// Stage so it can reset status.currentPromotion, as well as the
	// enqueue the next priority Promo for reconciliation
	return ok && !promo.Status.Phase.IsTerminal()
}

func (p PromoWentTerminal) Generic(_ event.GenericEvent) bool {
	// we should never get here
	return true
}

// Update implements default UpdateEvent filter for checking if a promotion went terminal
func (p PromoWentTerminal) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		p.logger.Errorf("Update event has no old object to update: %v", e)
		return false
	}
	if e.ObjectNew == nil {
		p.logger.Errorf("Update event has no new object for update: %v", e)
		return false
	}
	newPromo, ok := e.ObjectNew.(*kargoapi.Promotion)
	if !ok {
		p.logger.Errorf("Failed to convert new promo: %v", e.ObjectNew)
		return false
	}
	oldPromo, ok := e.ObjectOld.(*kargoapi.Promotion)
	if !ok {
		p.logger.Errorf("Failed to convert old promo: %v", e.ObjectOld)
		return false
	}
	if newPromo.Status.Phase.IsTerminal() && !oldPromo.Status.Phase.IsTerminal() {
		return true
	}
	return false
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
