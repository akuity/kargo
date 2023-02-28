package controller

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	api "github.com/akuityio/kargo/api/v1alpha1"
)

// TODO: This function could use some tests
func (e *environmentReconciler) promote(
	ctx context.Context,
	env *api.Environment,
	newState api.EnvironmentState,
) (api.EnvironmentState, error) {
	var err error
	for _, gitRepoUpdate := range env.Spec.PromotionMechanisms.GitRepoUpdates {
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

	for _, argoCDAppUpdate := range env.Spec.PromotionMechanisms.ArgoCDAppUpdates { // nolint: lll
		if err =
			e.applyArgoCDAppUpdate(ctx, newState, argoCDAppUpdate); err != nil {
			return newState, errors.Wrap(err, "error promoting via Argo CD")
		}
	}

	newState.Health = &api.Health{
		Status:       api.HealthStateUnknown,
		StatusReason: "Health has not yet been assessed",
	}

	e.logger.WithFields(log.Fields{
		"namespace": env.Namespace,
		"name":      env.Name,
		"state":     newState,
	}).Debug("completed promotion")

	return newState, nil
}
