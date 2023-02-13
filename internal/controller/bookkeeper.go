package controller

import (
	"context"
	"fmt"

	"github.com/akuityio/bookkeeper"
	"github.com/pkg/errors"

	api "github.com/akuityio/kargo/api/v1alpha1"
)

func (e *environmentReconciler) promoteWithBookkeeper(
	ctx context.Context,
	env *api.Environment,
	newState api.EnvironmentState,
) (api.EnvironmentState, error) {
	// If any of the following is true, this function ought not to have been
	// invoked, but we don't take that on faith.
	if env == nil ||
		env.Spec.PromotionMechanisms == nil ||
		env.Spec.PromotionMechanisms.Git == nil ||
		env.Spec.PromotionMechanisms.Git.Bookkeeper == nil ||
		env.Spec.PromotionMechanisms.Git.Bookkeeper.TargetBranch == "" { // nolint: lll
		return newState, nil
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
		TargetBranch: env.Spec.PromotionMechanisms.Git.Bookkeeper.TargetBranch,
	}
	res, err := e.renderManifestsWithBookkeeperFn(ctx, req)
	if err != nil {
		return newState,
			errors.Wrap(err, "error rendering manifests via Bookkeeper")
	}

	if res.ActionTaken == bookkeeper.ActionTakenPushedDirectly ||
		res.ActionTaken == bookkeeper.ActionTakenNone {
		newState.HealthCheckCommit = res.CommitID
	}
	// TODO: This is a fairly large outstanding question. How do we deal with PRs?
	// When a PR is opened, we don't immediately know the

	return newState, nil
}
