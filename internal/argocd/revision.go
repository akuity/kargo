package argocd

import (
	"context"
	"errors"
	"path"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/git"
	"github.com/akuity/kargo/internal/logging"
)

// GetDesiredRevision returns the desired revision for the given
// v1alpha1.Application by traversing the given v1alpha1.FreightReference for
// a matching source. If no match is found, an empty string is returned.
func GetDesiredRevision(ctx context.Context, app *argocd.Application, freight kargoapi.FreightReference) string {
	logger := logging.LoggerFromContext(ctx)

	if app == nil {
		logger.Error(errors.New("Application is nil"), "Cannot determine desired revision for application, bailing.")
		return ""
	}

	appLogger := logger.WithValues("application", app.Name, "namespace", app.Namespace)
	appLogger.Debug("Getting desired revision for application", "app", app, "spec", app.Spec)

	// An application may have one or more sources.
	if app.Spec.Source != nil {
		appLogger.Debug("Application source is not nil, checking.", "source", app.Spec.Source)
		desiredRevision := GetDesiredRevisionFromSource(ctx, app.Spec.Source, freight)
		if desiredRevision != "" {
			return desiredRevision
		}
	}

	// An application may have more than one source that points to the same Git repository,
	// eg. a Helm and a vanilla manifest source.
	// In that situation it doesn't matter which source's target revision is returned as ArgoCD does not support
	// different target revisions for sources targeting the same repository.
	for i := range app.Spec.Sources {
		desiredRevision := GetDesiredRevisionFromSource(ctx, &app.Spec.Sources[i], freight)
		if desiredRevision != "" {
			return desiredRevision
		}
	}

	// If we end up here, no desired revision was found from any of the possible sources.
	appLogger.Debug("Could not determine desired revision for application from any sources.")
	return ""
}

func GetDesiredRevisionFromSource(
	ctx context.Context,
	s *argocd.ApplicationSource,
	freight kargoapi.FreightReference) string {

	logger := logging.LoggerFromContext(ctx)
	sourceLogger := logger.WithValues("source", s)
	sourceLogger.Debug("Getting desired revision for application source", "source", s)

	switch {
	case s.Chart != "":
		// This source points to a Helm chart.
		// NB: This has to go first, as the repository URL can also point to
		//     a Helm repository.
		sourceChart := path.Join(s.RepoURL, s.Chart)
		sourceLogger.Debug("Application source is a Helm chart", "repoURL", s.RepoURL, "chart", s.Chart)
		for _, chart := range freight.Charts {
			// Join accounts for the possibility that chart.Name is empty.
			if path.Join(chart.RepoURL, chart.Name) == sourceChart {
				return chart.Version
			}
		}
	case s.RepoURL != "":
		// This source points to a directory in a Git repository.
		sourceLogger.Debug("Application source is a Git repository", "repoURL", s.RepoURL, "path", s.Path)
		sourceGitRepoURL := git.NormalizeURL(s.RepoURL)
		for _, commit := range freight.Commits {
			if git.NormalizeURL(commit.RepoURL) != sourceGitRepoURL {
				continue
			}
			if commit.HealthCheckCommit != "" {
				return commit.HealthCheckCommit
			}
			return commit.ID
		}
	}
	sourceLogger.Debug("Could not determine desired revision for application from this source.")
	return ""
}
