package argocd

import (
	"context"
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/freight"
	"github.com/akuity/kargo/internal/logging"
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

	logger := logging.LoggerFromContext(ctx)
	if app == nil {
		err := errors.New("Application is nil")
		logger.Error(err, "Cannot determine desired revision for application, bailing.")
		return "", nil
	}

	appLogger := logger.WithValues("application", app.Name, "namespace", app.Namespace)
	appLogger.Debug("Getting desired revision for application", "app", app, "spec", app.Spec)

	// Note that frght was provided as an argument instead of being plucked
	// directly from stage.Status, because this gives us the flexibility to use
	// this function for finding the revision to sync to either in the context of
	// a health check (current freight) or in the context of a promotion (new
	// freight).

	// An application may have one or more sources.
	if app.Spec.Source != nil {
		appLogger.Debug("Application source is not nil, checking.", "source", app.Spec.Source)
		desiredRevision, err := getRevisionFromSource(ctx, cl, stage, update, app.Spec.Source, frght)

		if err != nil {
			return "", err
		}

		if desiredRevision != "" {
			return desiredRevision, nil
		}
	}

	// An application may have more than one source that points to the same Git repository,
	// eg. a Helm and a vanilla manifest source.
	// In that situation it doesn't matter which source's target revision is returned as ArgoCD does not support
	// different target revisions for sources targeting the same repository.
	for i := range app.Spec.Sources {
		desiredRevision, err := getRevisionFromSource(ctx, cl, stage, update, &app.Spec.Sources[i], frght)

		if err != nil {
			return "", err
		}

		if desiredRevision != "" {
			return desiredRevision, nil
		}
	}

	// If we end up here, no desired revision was found from any of the possible sources.
	appLogger.Debug("Could not determine desired revision for application from any sources.")
	return "", nil
}

func getRevisionFromSource(
	ctx context.Context,
	cl client.Client,
	stage *kargoapi.Stage,
	update *kargoapi.ArgoCDAppUpdate,
	s *argocd.ApplicationSource,
	frght []kargoapi.FreightReference) (string, error) {

	logger := logging.LoggerFromContext(ctx)
	sourceLogger := logger.WithValues("source", s)
	sourceLogger.Debug("Getting desired revision for application source", "source", s)

	switch {
	case s.Chart != "":
		// This source points to a Helm chart.
		// NB: This has to go first, as the repository URL can also point to
		//     a Helm repository.
		sourceLogger.Debug("Application source is a Helm chart", "repoURL", s.RepoURL, "chart", s.Chart)

		// If there is a source update that targets app.Spec.Source, it might
		// have its own ideas about the desired revision.
		var targetPromoMechanism any
		for i := range update.SourceUpdates {
			sourceUpdate := &update.SourceUpdates[i]
			if sourceUpdate.RepoURL == s.RepoURL && sourceUpdate.Chart == s.Chart {
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
			s.RepoURL,
			s.Chart,
		)
		if err != nil {
			return "", fmt.Errorf("error chart from repo %q: %w", s.RepoURL, err)
		}
		if chart == nil {
			return "", nil
		}
		return chart.Version, nil

	case s.RepoURL != "":
		// This source points to a Git repository.

		// If there is a source update that targets app.Spec.Source, it might
		// have its own ideas about the desired revision.
		var targetPromoMechanism any
		for i := range update.SourceUpdates {
			sourceUpdate := &update.SourceUpdates[i]
			if sourceUpdate.RepoURL == s.RepoURL {
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
			s.RepoURL,
		)
		if err != nil {
			return "",
				fmt.Errorf("error finding commit from repo %q: %w", s.RepoURL, err)
		}
		if commit == nil {
			return "", nil
		}
		if commit.HealthCheckCommit != "" {
			return commit.HealthCheckCommit, nil
		}
		return commit.ID, nil
	}
	sourceLogger.Debug("Could not determine desired revision for application from this source.")
	return "", nil
}
