package promotionrollouts

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/database/promotionrolloutstore"
)

// Creator creates PromotionRollouts. The promote endpoints and the create
// endpoint share one, so that a rollout is built the same way whichever
// path asks for it.
type Creator struct {
	store promotionrolloutstore.Store
}

// NewCreator returns a Creator over the store.
func NewCreator(store promotionrolloutstore.Store) *Creator {
	return &Creator{store: store}
}

// Create records the intent to promote the Freight to the Targets the Stage
// governs, and returns the rollout as stored. Which Targets those are is
// not decided here: the rollout reconciler resolves the Stage's selectors
// once, when it first sees the rollout, so the rollout is returned with no
// Targets yet. The Stage must be target-aware and the Freight available to
// it: the caller has checked both, and that the caller may promote to the
// Stage, which is the whole authorization decision. actor is recorded as
// the rollout's creator.
func (cr *Creator) Create(
	ctx context.Context,
	stage *kargoapi.Stage,
	freight *kargoapi.Freight,
	actor string,
) (*kargoapi.PromotionRollout, error) {
	rollout := api.NewPromotionRollout(stage, freight.Name)
	row := promotionrolloutstore.FromSpec(rollout.Name, rollout.Spec)
	row.CreatedBy = actor
	stored, err := cr.store.Create(ctx, stage.Namespace, row)
	if err != nil {
		return nil, storeError(err, rollout.Name)
	}
	return promotionrolloutstore.ToAPI(stored), nil
}
