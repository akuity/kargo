package controller

import (
	"context"
	"fmt"

	"github.com/akuityio/bookkeeper"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	api "github.com/akuityio/k8sta/api/v1alpha1"
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
	if env.Spec.PromotionMechanisms.ConfigManagement.Bookkeeper != nil {
		if newState, err = e.promoteWithBookkeeper(ctx, env, newState); err != nil {
			return newState, errors.Wrap(err, "error promoting via Bookkeeper")
		}
	} else if env.Spec.PromotionMechanisms.ConfigManagement.Kustomize != nil {
		if newState, err = e.promoteWithKustomize(ctx, env, newState); err != nil {
			return newState, errors.Wrap(err, "error promoting via Kustomize")
		}
	} else if env.Spec.PromotionMechanisms.ConfigManagement.Helm != nil {
		if newState, err = e.promoteWithHelm(ctx, env, newState); err != nil {
			return newState, errors.Wrap(err, "error promoting via Helm")
		}
	}

	if env.Spec.PromotionMechanisms.ArgoCD != nil {
		if newState, err = e.promoteWithArgoCD(ctx, env, newState); err != nil {
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

func (e *environmentReconciler) promoteWithBookkeeper(
	ctx context.Context,
	env *api.Environment,
	newState api.EnvironmentState,
) (api.EnvironmentState, error) {
	// If any of the following is true, this function ought not to have been
	// invoked, but we don't take that on faith.
	if env.Spec.PromotionMechanisms == nil ||
		env.Spec.PromotionMechanisms.ConfigManagement == nil ||
		env.Spec.PromotionMechanisms.ConfigManagement.Bookkeeper == nil ||
		env.Spec.PromotionMechanisms.ConfigManagement.Bookkeeper.TargetBranch == "" { // nolint: lll
		return newState, errors.New(
			"spec contains insufficient instructions to reach new state using " +
				"Bookkeeper",
		)
	}

	images := make([]string, len(newState.Images))
	for i, image := range newState.Images {
		images[i] = fmt.Sprintf("%s:%s", image.RepoURL, image.Tag)
	}
	creds, err := e.getGitRepoCredentialsFn(ctx, newState.GitCommit.RepoURL)
	if err != nil {
		return newState, errors.Wrapf(
			err,
			"error obtaining credentials for git repo %q",
			newState.GitCommit.RepoURL,
		)
	}
	req := bookkeeper.RenderRequest{
		RepoURL: newState.GitCommit.RepoURL,
		RepoCreds: bookkeeper.RepoCredentials{
			Username:      creds.Username,
			Password:      creds.Password,
			SSHPrivateKey: creds.SSHPrivateKey,
		},
		Commit:       newState.GitCommit.ID,
		Images:       images,
		TargetBranch: env.Spec.PromotionMechanisms.ConfigManagement.Bookkeeper.TargetBranch, // nolint: lll
	}
	res, err := e.bookkeeperService.RenderManifests(ctx, req)
	if err != nil {
		return newState,
			errors.Wrap(err, "error rendering manifests via Bookkeeper")
	}

	if res.ActionTaken == bookkeeper.ActionTakenPushedDirectly ||
		res.ActionTaken == bookkeeper.ActionTakenNone {
		newState.GitCommit.HealthCheckID = res.CommitID
	}
	// TODO: This is a fairly large outstanding question. How do we deal with PRs?
	// When a PR is opened, we don't immediately know the

	return newState, nil
}

func (e *environmentReconciler) promoteWithKustomize(
	ctx context.Context,
	env *api.Environment,
	newState api.EnvironmentState,
) (api.EnvironmentState, error) {
	return newState, errors.New("promotion via Kustomize is not yet supported")
}

func (e *environmentReconciler) promoteWithHelm(
	ctx context.Context,
	env *api.Environment,
	newState api.EnvironmentState,
) (api.EnvironmentState, error) {
	return newState, errors.New("promotion via Helm is not yet supported")
}

func (e *environmentReconciler) promoteWithArgoCD(
	ctx context.Context,
	env *api.Environment,
	newState api.EnvironmentState,
) (api.EnvironmentState, error) {
	// If any of the following is true, this function ought not to have been
	// invoked, but we don't take that on faith.
	if env.Spec.PromotionMechanisms == nil ||
		env.Spec.PromotionMechanisms.ArgoCD == nil ||
		len(env.Spec.PromotionMechanisms.ArgoCD.AppUpdates) == 0 {
		return newState, errors.New(
			"spec contains insufficient instructions to reach new state using " +
				"Argo CD",
		)
	}

	for _, appUpdate := range env.Spec.PromotionMechanisms.ArgoCD.AppUpdates {
		if appUpdate.UpdateTargetRevision {
			return newState, errors.Errorf(
				"updating target revision of an Argo CD Application resource is " +
					"not yet supported",
			)
		}
		if appUpdate.RefreshAndSync {
			if err := e.refreshAndSyncArgoCDApp(
				ctx,
				env.Namespace,
				appUpdate.Name,
			); err != nil {
				return newState, err
			}
		}
	}

	return newState, nil
}
