package server

import (
	"context"
	"net/http"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	libhttp "github.com/akuity/kargo/pkg/http"
)

var errDatabaseNotConfigured = libhttp.ErrorStr(
	"database is not configured",
	http.StatusNotImplemented,
)

// governedTargets returns the Targets the Stage governs, read from the
// database. It runs no SubjectAccessReview of its own: the promote verb that
// callers have already checked IS the authorization decision, and which
// Targets the Stage governs is a detail of carrying it out. Resolving as the
// user would also break the kargo-promoter role, which holds the promote verb
// but no permission to list Targets.
func (s *server) governedTargets(
	ctx context.Context,
	stage *kargoapi.Stage,
) ([]kargoapi.Target, error) {
	if s.listTargetsFn == nil {
		return nil, errDatabaseNotConfigured
	}
	targets, err := s.listTargetsFn(ctx, stage.Namespace)
	if err != nil {
		return nil, err
	}
	targets, err = api.FilterTargetsForStage(stage, targets)
	if err != nil {
		return nil, libhttp.Error(err, http.StatusBadRequest)
	}
	return targets, nil
}
