package controller

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	api "github.com/akuityio/kargo/api/v1alpha1"
)

func (e *environmentReconciler) promote(
	ctx context.Context,
	env *api.Environment,
	newState api.EnvironmentState,
) (api.EnvironmentState, error) {
	if env.Spec.PromotionMechanisms == nil {
		return newState,
			errors.New("spec contains insufficient instructions to reach new state")
	}

	var err error
	if env.Spec.PromotionMechanisms.Git != nil {
		if env.Spec.PromotionMechanisms.Git.Bookkeeper != nil {
			if newState, err =
				e.promoteWithBookkeeper(ctx, env, newState); err != nil {
				return newState, errors.Wrap(err, "error promoting via Bookkeeper")
			}
		} else if env.Spec.PromotionMechanisms.Git.Kustomize != nil {
			if newState, err =
				e.promoteWithKustomize(ctx, env, newState); err != nil {
				return newState, errors.Wrap(err, "error promoting via Kustomize")
			}
		} else if env.Spec.PromotionMechanisms.Git.Helm != nil {
			if newState, err =
				e.promoteWithHelm(ctx, env, newState); err != nil {
				return newState, errors.Wrap(err, "error promoting via Helm")
			}
		}
	}

	if env.Spec.PromotionMechanisms.ArgoCD != nil {
		if err = e.promoteWithArgoCD(ctx, env, newState); err != nil {
			return newState, errors.Wrap(err, "error promoting via Argo CD")
		}
	}

	e.logger.WithFields(log.Fields{
		"namespace": env.Namespace,
		"name":      env.Name,
		"state":     newState.ID,
		"git":       newState.GitCommit,
		"images":    newState.Images,
	}).Debug("completed promotion")

	return newState, nil
}
