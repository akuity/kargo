package argocd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/freight"
	"github.com/akuity/kargo/internal/logging"
)

// GetDesiredRevision returns the desired revisions for the given
// v1alpha1.Application. The array indices shadow the application's
// sources. If desired revision for a source cannot be determined,
// empty string is returned at the corresponding index.
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
		return nil, nil
	}

	appLogger := logger.WithValues("appName", app.Name, "namespace", app.Namespace)
	appLogger.Debug("Getting desired revision for application", "app", app, "spec", app.Spec)

	// Note that frght was provided as an argument instead of being plucked
	// directly from stage.Status, because this gives us the flexibility to use
	// this function for finding the revision to sync to either in the context of
	// a health check (current freight) or in the context of a promotion (new
	// freight).

	hasSource := app.Spec.Source != nil
	hasSources := app.Spec.Sources != nil
	if !hasSource && !hasSources {
		appLogger.Debug("Application has no sources, cannot determine desired revision.")
		return nil, nil
	}

	sources := app.Spec.Sources
	if hasSource {
		sources = []argocd.ApplicationSource{*app.Spec.Source}
	}

	var revisions = make([]string, len(sources))

	// ArgoCD revisions array items correspond to the app.spec.sources at the same index location.
	// We match that logic here and return the desired revision for each source,
	// or empty string if one is not found.
	for i, src := range sources {
		s := src

		syncedRevisionOfSource := ""
		if app.Status.Sync.Revisions != nil && i < len(app.Status.Sync.Revisions) {
			syncedRevisionOfSource = app.Status.Sync.Revisions[i]
		} else {
			syncedRevisionOfSource = app.Status.Sync.Revision
		}

		// If there is a source update that targets the current source, it might
		// have its own ideas about the desired revision.
		targetUpdate := getTargetUpdate(ctx, update, &s)
		desiredOrigin := freight.GetDesiredOrigin(ctx, stage, targetUpdate)
		if desiredOrigin == nil {
			appLogger.WithValues("source", s,
				"revision", syncedRevisionOfSource).Debug("Could not determine desired origin" +
				" for application source.")
			revisions[i] = ""
			continue
		}

		logger.Debug("Resolved origin for application source", "origin", desiredOrigin)
		desiredRevision, err := getDesiredRevisionForSource(ctx, cl, stage, &s, desiredOrigin, frght)

		if err != nil {
			return nil, err
		}

		if desiredRevision != "" {
			appLogger.Trace("Found desired revision for application source.", "revision", desiredRevision, "source", &s)
		}
		revisions[i] = desiredRevision
	}

	appLogger.Debug("Found desired revisions for multi-source application.", "revisions", revisions)
	return revisions, nil
}

func getDesiredRevisionForSource(
	ctx context.Context,
	cl client.Client,
	stage *kargoapi.Stage,
	s *argocd.ApplicationSource,
	desiredOrigin *kargoapi.FreightOrigin,
	frght []kargoapi.FreightReference) (string, error) {

	logger := logging.LoggerFromContext(ctx)
	sourceLogger := logger.WithValues("source", s)
	sourceLogger.Debug("Getting desired revision for application source", "source", s)

	repoURL := s.RepoURL
	chartName := s.Chart
	if !strings.Contains(repoURL, "://") {
		// In Argo CD ApplicationSource, if a repo URL specifies no protocol and a
		// chart name is set (already confirmed at this point), we can assume that
		// the repo URL is an OCI registry URL. Kargo Warehouses and Freight,
		// however, do use oci:// at the beginning of such URLs.
		//
		// Additionally, where OCI is concerned, an ApplicationSource's repoURL is
		// really a registry URL, and the chart name is a repository within that
		// registry. Warehouses and Freight, however, handle things more correctly
		// where a repoURL points directly to a repository and chart name is
		// irrelevant / blank. We need to account for this when we search our
		// Freight for the chart.
		repoURL = fmt.Sprintf(
			"oci://%s/%s",
			strings.TrimSuffix(repoURL, "/"),
			chartName,
		)
		chartName = ""
	}

	switch {
	case chartName != "":
		// This source points to a Helm chart.
		// NB: This has to go first, as the repository URL can also point to
		//     a Helm repository.
		sourceLogger.Debug("Application source is a Helm chart")

		chart, err := freight.FindChart(
			ctx,
			cl,
			stage,
			desiredOrigin,
			frght,
			repoURL,
			chartName,
		)
		if err != nil {
			sourceLogger.Error(err, "Error finding chart from repo")
			return "", fmt.Errorf("error chart from repo %q: %w", repoURL, err)
		}
		if chart == nil {
			sourceLogger.Debug("Chart not found")
			return "", nil
		}
		return chart.Version, nil

	case repoURL != "":
		// This source points to a Git repository.
		sourceLogger.Debug("Application source is a Git repository")
		commit, err := freight.FindCommit(
			ctx,
			cl,
			stage,
			desiredOrigin,
			frght,
			repoURL,
		)
		if err != nil {
			sourceLogger.Error(err, "Error finding commit from repo")
			return "",
				fmt.Errorf("error finding commit from repo %q: %w", repoURL, err)
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

func getTargetUpdate(
	ctx context.Context,
	update *kargoapi.ArgoCDAppUpdate,
	s *argocd.ApplicationSource,
) any {
	sourceLogger := logging.LoggerFromContext(ctx).WithValues("source", s)
	sourceLogger.Debug("Resolving origin for application source")

	// If there is a source update that targets app.Spec.Source, it might
	// have its own ideas about the desired revision.
	// Default to the ArgoCDAppUpdate if no source update targets app.Spec.Source.
	for i := range update.SourceUpdates {
		sourceUpdate := &update.SourceUpdates[i]
		sourceLogger.Trace("Checking source update", "sourceUpdate", sourceUpdate)

		if sourceUpdate.RepoURL == s.RepoURL && (sourceUpdate.Chart == "" || sourceUpdate.Chart == s.Chart) {
			sourceLogger.Debug("Source update matching application source found", "sourceUpdate", sourceUpdate)
			return sourceUpdate
		}
	}

	sourceLogger.Debug("No source update matching application source found, using ArgoCDAppUpdate.")
	return update
}
