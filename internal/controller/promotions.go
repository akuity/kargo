package controller

import (
	"context"

	"github.com/pkg/errors"

	api "github.com/akuityio/kargo/api/v1alpha1"
)

// TODO: This function could use some tests
func (e *environmentReconciler) promote(
	ctx context.Context,
	promoMechanisms api.PromotionMechanisms,
	newState api.EnvironmentState,
) (api.EnvironmentState, error) {
	var err error
	for _, gitRepoUpdate := range promoMechanisms.GitRepoUpdates {
		if gitRepoUpdate.Bookkeeper != nil {
			if newState, err =
				e.applyBookkeeperUpdate(ctx, newState, gitRepoUpdate); err != nil {
				return newState, errors.Wrap(err, "error promoting via Git")
			}
		} else {
			if newState, err =
				e.applyGitRepoUpdate(ctx, newState, gitRepoUpdate); err != nil {
				return newState, errors.Wrap(err, "error promoting via Git")
			}
		}
	}

	for _, argoCDAppUpdate := range promoMechanisms.ArgoCDAppUpdates {
		if err =
			e.applyArgoCDAppUpdate(ctx, newState, argoCDAppUpdate); err != nil {
			return newState, errors.Wrap(err, "error promoting via Argo CD")
		}
	}

	newState.Health = &api.Health{
		Status:       api.HealthStateUnknown,
		StatusReason: "Health has not yet been assessed",
	}

	e.logger.Debug("completed promotion")

	return newState, nil
}
