package types

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

type PromotionGate interface {
	Name() string
	Evaluate(context.Context, PromotionInput) (*Decision, error)
}

// PromotionInput contains the information a PromotionGate uses to decide
// whether a Promotion may be created.
type PromotionInput struct {
	Stage   *kargoapi.Stage
	Freight *kargoapi.Freight
}

// FreightRequest returns the Stage's requested-freight entry that applies to
// this input's Freight, matched by origin, or nil if the Stage does not
// request Freight from that origin. The applicable request carries the Stage's
// policy for the Freight (direct sources, upstream Stages, availability
// strategy, and soak requirement); gates read it from here rather than
// re-deriving the origin match themselves.
func (i PromotionInput) FreightRequest() *kargoapi.FreightRequest {
	if i.Stage == nil || i.Freight == nil {
		return nil
	}
	for idx := range i.Stage.Spec.RequestedFreight {
		request := &i.Stage.Spec.RequestedFreight[idx]
		if request.Origin.Equals(&i.Freight.Origin) {
			return request
		}
	}
	return nil
}
