package api

import (
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// NewFreightCollectionForStage builds the FreightCollection a Stage would be
// running once the given Freight is promoted to it: the Freight itself, plus
// the Freight from each of the Stage's other requested origins, carried over
// unchanged from what the Stage is running now.
//
// Carried-over Freight comes from the Stage's last Promotion when there is one
// with a collection, and otherwise from the Stage's freight history. The last
// Promotion is preferred so that Promotions to an ordinary Stage keep building
// on one another exactly as they always have. A Stage that promotes through
// PromotionRequests never records a last Promotion (its Promotions are children
// of a request and take no part in its own flow), so for it the history is the
// source; without that fallback a multi-origin Stage's collection would name
// only the promoted origin and the Stage would appear to drop the rest.
//
// Origins the Stage no longer requests are not carried over, so the result
// never contains Freight the Stage has stopped asking for. The same function
// builds the collection for a Promotion and for a PromotionRequest, so a
// request and its child Promotions agree by construction.
func NewFreightCollectionForStage(
	stage *kargoapi.Stage,
	freight kargoapi.FreightReference,
) *kargoapi.FreightCollection {
	collection := &kargoapi.FreightCollection{}
	if len(stage.Spec.RequestedFreight) > 1 {
		if inherited := inheritableFreight(stage); inherited != nil {
			for _, req := range stage.Spec.RequestedFreight {
				if carried, ok := inherited[req.Origin.String()]; ok {
					collection.UpdateOrPush(carried)
				}
			}
		}
	}
	collection.UpdateOrPush(freight)
	return collection
}

// inheritableFreight returns the Freight, by origin, that a new collection for
// the Stage carries over: the last Promotion's collection when there is one,
// otherwise the Stage's current collection. Nil when there is neither.
func inheritableFreight(stage *kargoapi.Stage) map[string]kargoapi.FreightReference {
	if last := stage.Status.LastPromotion; last != nil &&
		last.Status != nil &&
		last.Status.FreightCollection != nil &&
		last.Status.FreightCollection.Freight != nil {
		return last.Status.FreightCollection.Freight
	}
	if current := stage.Status.FreightHistory.Current(); current != nil {
		return current.Freight
	}
	return nil
}
