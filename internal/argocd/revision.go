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

var NO_REVISIONS = []string{}

func GetIntersection(revisions []string, desired []string) []string {
	var intersection []string
	hash := make(map[string]bool)
	for _, e := range revisions {
		hash[e] = true
	}
	for _, e := range desired {
		// If elements present in the hashmap then append intersection list.
		if hash[e] {
			intersection = append(intersection, e)
		}
	}
	return intersection
}

// GetDesiredRevision returns the desired revision for the given
// v1alpha1.Application. If that cannot be determined, an empty string is
// returned.
func GetDesiredRevisions(
	ctx context.Context,
	cl client.Client,
	stage *kargoapi.Stage,
	update *kargoapi.ArgoCDAppUpdate,
	app *argocd.Application,
	frght []kargoapi.FreightReference,
) ([]string, error) {

	logger := logging.LoggerFromContext(ctx)
	if app == nil {
		err := errors.New("Application is nil")
		logger.Error(err, "Cannot determine desired revision for application, bailing.")
		return NO_REVISIONS, nil
	}

	appLogger := logger.WithValues("application", app.Name, "namespace", app.Namespace)
	appLogger.Debug("Getting desired revision for application", "app", app, "spec", app.Spec)

	// Note that frght was provided as an argument instead of being plucked
	// directly from stage.Status, because this gives us the flexibility to use
	// this function for finding the revision to sync to either in the context of
	// a health check (current freight) or in the context of a promotion (new
	// freight).

	// An application may have one or more sources.
	if !app.IsMultisource() {
		if app.Spec.Source == nil {
			err := errors.New("Single-source application source is nil")
			appLogger.Error(err, "Cannot determine desired revision for application, bailing.")
			return NO_REVISIONS, nil
		}

		appLogger.Debug("Application source is not nil, checking.", "source", app.Spec.Source)
		desiredRevision, err := getRevisionFromSource(ctx, cl, stage, update, app.Spec.Source, frght)

		if err != nil {
			return NO_REVISIONS, err
		}

		if desiredRevision != "" {
			return []string{desiredRevision}, nil
		}
	}

	// An application may have more than one source that points to the same Git repository,
	// eg. a Helm and a vanilla manifest source.
	// In that situation it doesn't matter which source's target revision is returned as ArgoCD does not support
	// different target revisions for sources targeting the same repository.
	var revisions []string
	for i := range app.Spec.Sources {
		s := &app.Spec.Sources[i]
		desiredRevision, err := getRevisionFromSource(ctx, cl, stage, update, s, frght)

		if err != nil {
			return NO_REVISIONS, err
		}

		if desiredRevision != "" {
			appLogger.Trace("Found desired revision for application source.", "revision", desiredRevision, "source", s)
			revisions = append(revisions, desiredRevision)
		}
	}
	if len(revisions) > 0 {
		appLogger.Debug("Found desired revision(s) for multi-source application.", "revisions", revisions)
		return revisions, nil
	}

	// If we end up here, no desired revision was found from any of the possible sources.
	appLogger.Debug("Could not determine desired revision for application from any sources.")
	return NO_REVISIONS, nil
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

	// If there is a source update that targets app.Spec.Source, it might
	// have its own ideas about the desired revision.
	desiredOrigin := getDesiredOriginForSource(ctx, stage, update, s)

	switch {
	case s.Chart != "":
		// NB: This has to go first, as the repository URL can also point to
		//     a Helm repository.
		sourceLogger.Debug("Application source is a Helm chart")

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
			sourceLogger.Error(err, "Error finding chart from repo")
			return "", fmt.Errorf("error chart from repo %q: %w", s.RepoURL, err)
		}
		if chart == nil {
			sourceLogger.Debug("Chart not found")
			return "", nil
		}
		return chart.Version, nil

	case s.RepoURL != "":
		sourceLogger.Debug("Application source is a Git repository")
		commit, err := freight.FindCommit(
			ctx,
			cl,
			stage,
			desiredOrigin,
			frght,
			s.RepoURL,
		)
		if err != nil {
			sourceLogger.Error(err, "Error finding commit from repo")
			return "",
				fmt.Errorf("error finding commit from repo %q: %w", s.RepoURL, err)
		}
		if commit == nil {
			sourceLogger.Debug("Commit not found")
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

func getDesiredOriginForSource(
	ctx context.Context,
	stage *kargoapi.Stage,
	update *kargoapi.ArgoCDAppUpdate,
	s *argocd.ApplicationSource,
) *kargoapi.FreightOrigin {
	sourceLogger := logging.LoggerFromContext(ctx).WithValues("source", s)
	sourceLogger.Debug("Resolving origin for application source")

	// If there is a source update that targets app.Spec.Source, it might
	// have its own ideas about the desired revision.
	// Default to the ArgoCDAppUpdate if no source update targets app.Spec.Source.
	var targetPromoMechanism any

	for i := range update.SourceUpdates {
		sourceUpdate := &update.SourceUpdates[i]
		sourceLogger.Trace("Checking source update", "sourceUpdate", sourceUpdate)
		if sourceUpdate.RepoURL != s.RepoURL {
			sourceLogger.Debug("Source update does not match application source, skipping", "sourceUpdate", sourceUpdate)
			continue
		}

		if s.Chart != "" && sourceUpdate.Chart != s.Chart {
			sourceLogger.Debug("Source update chart does not match application source chart, skipping",
				"sourceUpdate", sourceUpdate)
			continue
		}

		sourceLogger.Debug("Source update matching application source found", "sourceUpdate", sourceUpdate)
		targetPromoMechanism = sourceUpdate
		break
	}

	if targetPromoMechanism == nil {
		sourceLogger.Debug("No source update matching application source found, using ArgoCDAppUpdate")
		targetPromoMechanism = update
	}

	origin := freight.GetDesiredOrigin(ctx, stage, targetPromoMechanism)
	if origin == nil {
		sourceLogger.Debug("Could not determine desired origin for application source")
		return nil
	}

	sourceLogger.Debug("Resolved origin for application source", "origin", origin)
	return origin
}
