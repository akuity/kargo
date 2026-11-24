package server

import (
	"context"
	"net/http"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
)

var errDatabaseNotConfigured = libhttp.ErrorStr(
	"database is not configured",
	http.StatusNotImplemented,
)

func (s *server) getListTargetsFn() (func(context.Context, string) ([]kargoapi.Target, error), error) {
	if s.listTargetsFn == nil {
		return nil, errDatabaseNotConfigured
	}
	return s.listTargetsFn, nil
}

// createPromotionRollout promotes the Freight to the Targets the Stage
// governs, or to the named ones among them, through a PromotionRollout, at
// the Stage's pace or the one given in its place, and returns the rollout as
// stored. It is what the promote endpoints do for a target-aware Stage when
// this server creates rollouts, in place of a PromotionRequest. The Stage
// must be target-aware and the Freight available to it, and the caller must
// have been found allowed to promote to the Stage: that is the whole
// authorization decision, as it is for a PromotionRequest. The rollout is
// written to the database without a check of its own for the same reason.
func (s *server) createPromotionRollout(
	ctx context.Context,
	stage *kargoapi.Stage,
	freight *kargoapi.Freight,
	targetNames []string,
	updateStrategy *kargoapi.TargetUpdateStrategy,
) (*kargoapi.PromotionRollout, error) {
	if s.rolloutCreator == nil {
		return nil, errDatabaseNotConfigured
	}
	return s.rolloutCreator.Create(ctx, stage, freight, targetNames, updateStrategy)
}
