package argocd

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/freight"
)

// GetDesiredRevisions returns the desired revisions for the given
// v1alpha1.Application. If that cannot be determined, an empty slice is
// returned.
func GetDesiredRevisions(
	ctx context.Context,
	cl client.Client,
	stage *kargoapi.Stage,
	update *kargoapi.ArgoCDAppUpdate,
	app *argocd.Application,
	frght []kargoapi.FreightReference,
) ([]string, error) {
	revisions := []string{}
	if app == nil || (app.Spec.Source == nil && app.Spec.Sources == nil) {
		// Without an Application, we can't determine the desired revision.
		return revisions, nil
	}
	sources := app.Spec.Sources
	if sources == nil {
		sources = []argocd.ApplicationSource{*app.Spec.Source}
	}
	for _, source := range sources {
		// Note that frght was provided as an argument instead of being plucked
		// directly from stage.Status, because this gives us the flexibility to use
		// this function for finding the revision to sync to either in the context of
		// a health check (current freight) or in the context of a promotion (new
		// freight).
		switch {
		case source.Chart != "":
			// This source points to a Helm chart.
			// NB: This has to go first, as the repository URL can also point to
			//     a Helm repository.

			// If there is a source update that targets app.Spec.Source, it might
			// have its own ideas about the desired revision.
			var targetPromoMechanism any
			for i := range update.SourceUpdates {
				sourceUpdate := &update.SourceUpdates[i]
				if sourceUpdate.RepoURL == source.RepoURL && sourceUpdate.Chart == source.Chart {
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
				source.RepoURL,
				source.Chart,
			)
			if err != nil {
				return revisions, fmt.Errorf("error finding chart from repo %q: %w", source.RepoURL, err)
			}
			if chart == nil {
				revisions = append(revisions, "")
				continue
			}
			revisions = append(revisions, chart.Version)
		case source.RepoURL != "":
			// This source points to a Git repository.

			// If there is a source update that targets app.Spec.Source, it might
			// have its own ideas about the desired revision.
			var targetPromoMechanism any
			for i := range update.SourceUpdates {
				sourceUpdate := &update.SourceUpdates[i]
				if sourceUpdate.RepoURL == source.RepoURL {
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
				source.RepoURL,
			)
			if err != nil {
				return revisions,
					fmt.Errorf("error finding commit from repo %q: %w", source.RepoURL, err)
			}
			if commit == nil {
				revisions = append(revisions, "")
				continue
			}
			if commit.HealthCheckCommit != "" {
				revisions = append(revisions, commit.HealthCheckCommit)
				continue
			}
			revisions = append(revisions, commit.ID)
		}
	}
	// Return revisions if any were found
	return revisions, nil
}
