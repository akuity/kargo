package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/controller/promotionrequests"
	"github.com/akuity/kargo/pkg/database"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/user"
)

// createPromotionRequest records the intent to promote the named Freight to
// the Targets the Stage governs, and returns the resulting PromotionRequest.
//
// Neither the Target lookup nor the write is authorized here. The promote
// check on the Stage that every caller has already passed IS the
// authorization decision for this request; which Targets the Stage governs is
// a detail of carrying it out. Resolving Targets as the user would also break
// the kargo-promoter role, which holds the promote verb but no permission to
// list Targets.
//
// The Stage MUST be target-aware. Callers should gate on api.IsTargetAware.
func (s *server) createPromotionRequest(
	ctx context.Context,
	stage *kargoapi.Stage,
	freightName string,
) (*kargoapi.PromotionRequest, error) {
	if s.store == nil {
		return nil, errDatabaseNotConfigured
	}

	rows, err := s.store.ListTargets(ctx, stage.Namespace)
	if err != nil {
		return nil, fmt.Errorf("error listing Targets in Project %q: %w", stage.Namespace, err)
	}
	targets, err := database.TargetsFromRows(rows, stage.Namespace)
	if err != nil {
		return nil, err
	}
	governed, err := api.FilterTargetsForStage(stage, targets)
	if err != nil {
		return nil, libhttp.Error(
			fmt.Errorf("error resolving Targets governed by Stage %q: %w", stage.Name, err),
			http.StatusBadRequest,
		)
	}

	promotionRequest := api.NewPromotionRequest(stage, freightName, governed)
	if u, ok := user.InfoFromContext(ctx); ok {
		api.SetCreateActorAnnotation(promotionRequest, api.FormatEventUserActor(u))
	}

	snapshot, err := s.store.CreatePromotionRequest(ctx, database.NewPromotionRequestCreate(promotionRequest))
	if err != nil {
		if errors.Is(err, database.ErrNotMirrored) {
			// The Stage, the Freight or a Target exists in Kubernetes but has not
			// reached the database yet. That resolves itself within moments.
			return nil, libhttp.Error(
				fmt.Errorf(
					"cannot promote to Stage %q yet because the database has not caught up "+
						"with Kubernetes; retry shortly: %w",
					stage.Name, err,
				),
				http.StatusConflict,
			)
		}
		return nil, fmt.Errorf("error creating PromotionRequest for Stage %q: %w", stage.Name, err)
	}

	// Announce the request so the PromotionRequest reconciler takes it up now
	// rather than at its next resync. The resync is the safety net, so a
	// failure to announce is not a failure to promote.
	if err = promotionrequests.PublishCreated(s.natsConn, snapshot); err != nil {
		logging.LoggerFromContext(ctx).Error(err, "error announcing PromotionRequest")
	}

	// No event is recorded: Kargo's promotion events carry a Promotion, and a
	// PromotionRequest has none of its own.
	created := database.PromotionRequestFromSnapshot(snapshot)
	return &created, nil
}
