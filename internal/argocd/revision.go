package argocd

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/freight"
)

// GetDesiredRevision returns the desired revision for the given
// v1alpha1.Application. If that cannot be determined, an empty string is
// returned.
func GetDesiredRevision(
	ctx context.Context,
	cl client.Client,
	stage *kargoapi.Stage,
	update *kargoapi.ArgoCDAppUpdate,
	app *argocd.Application,
	frght []kargoapi.FreightReference,
) (string, error) {
	// Note that frght was provided as an argument instead of being plucked
	// directly from stage.Status, because this gives us the flexibility to use
	// this function for finding the revision to sync to either in the context of
	// a health check (current freight) or in the context of a promotion (new
	// freight).
	switch {
	case app == nil || app.Spec.Source == nil:
		// Without an Application, we can't determine the desired revision.
		return "", nil
	case app.Spec.Source.Chart != "":
		// This source points to a Helm chart.
		// NB: This has to go first, as the repository URL can also point to
		//     a Helm repository.

		// If there is a source update that targets app.Spec.Source, it might
		// have its own ideas about the desired revision.
		var targetPromoMechanism any
		for i := range update.SourceUpdates {
			sourceUpdate := &update.SourceUpdates[i]
			if sourceUpdate.RepoURL == app.Spec.Source.RepoURL && sourceUpdate.Chart == app.Spec.Source.Chart {
				targetPromoMechanism = sourceUpdate
				break
			}
		}
		if targetPromoMechanism == nil {
			targetPromoMechanism = update
		}
		desiredOrigin := freight.GetDesiredOrigin(stage, targetPromoMechanism)
		chart, err := freight.FindChart(
			ctx,
			cl,
			stage,
			desiredOrigin,
			frght,
			app.Spec.Source.RepoURL,
			app.Spec.Source.Chart,
		)
		if err != nil {
			return "", fmt.Errorf("error chart from repo %q: %w", app.Spec.Source.RepoURL, err)
		}
		if chart == nil {
			return "", nil
		}
		return chart.Version, nil
	case app.Spec.Source.RepoURL != "":
		// This source points to a Git repository.

		// If there is a source update that targets app.Spec.Source, it might
		// have its own ideas about the desired revision.
		var targetPromoMechanism any
		for i := range update.SourceUpdates {
			sourceUpdate := &update.SourceUpdates[i]
			if sourceUpdate.RepoURL == app.Spec.Source.RepoURL {
				targetPromoMechanism = sourceUpdate
				break
			}
		}
		if targetPromoMechanism == nil {
			targetPromoMechanism = update
		}
		desiredOrigin := freight.GetDesiredOrigin(stage, targetPromoMechanism)
		commit, err := freight.FindCommit(
			ctx,
			cl,
			stage,
			desiredOrigin,
			frght,
			app.Spec.Source.RepoURL,
		)
		if err != nil {
			return "",
				fmt.Errorf("error finding commit from repo %q: %w", app.Spec.Source.RepoURL, err)
		}
		if commit == nil {
			return "", nil
		}
		if commit.HealthCheckCommit != "" {
			return commit.HealthCheckCommit, nil
		}
		return commit.ID, nil
	}
	// If we end up here, no desired revision was found.
	return "", nil
}
