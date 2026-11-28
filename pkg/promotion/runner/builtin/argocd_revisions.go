package builtin

import (
	argocd "github.com/akuity/kargo/pkg/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/pkg/urls"
	"github.com/akuity/kargo/pkg/x/promotion/stepconfig"
)

// getDesiredRevisions returns the desired revisions for all sources of the given
// Application. For a single-source Application, the returned slice will have
// precisely one value. For a multi-source Application, the returned slice will
// have the same length as and will be indexed identically to the Application's
// Sources slice. For any source whose desired revision cannot be determined,
// the slice will contain an empty string at the corresponding index.
func (a *argocdUpdater) getDesiredRevisions(
	update *stepconfig.ArgoCDAppUpdate,
	app *argocd.Application,
) []string {
	if app == nil {
		return nil
	}
	sources := app.Spec.Sources
	if len(sources) == 0 && app.Spec.Source != nil {
		sources = []argocd.ApplicationSource{*app.Spec.Source}
	}
	if len(sources) == 0 {
		return nil
	}
	revisions := make([]string, len(sources))
	for i, src := range sources {
		sourceUpdate := a.findSourceUpdate(update, src)
		// If there is a source update that targets this source, it might be
		// specific about a previous step whose output should be used as the desired
		// revision.
		if sourceUpdate != nil {
			revisions[i] = sourceUpdate.DesiredRevision
		}
	}
	return revisions
}

// findSourceUpdate finds and returns the ArgoCDSourceUpdate that targets the
// given source. If no such update exists, it returns nil. Updates are matched
// to sources the same way applyArgoCDSourceUpdate matches them.
func (a *argocdUpdater) findSourceUpdate(
	update *stepconfig.ArgoCDAppUpdate,
	src argocd.ApplicationSource,
) *stepconfig.ArgoCDAppSourceUpdate {
	for i := range update.Sources {
		sourceUpdate := &update.Sources[i]
		if src.Chart != "" || sourceUpdate.Chart != "" {
			if sourceUpdate.RepoURL == src.RepoURL && sourceUpdate.Chart == src.Chart {
				return sourceUpdate
			}
			continue
		}
		// We're dealing with a git repo, so we should normalize the repo URLs
		// before comparing them.
		if urls.NormalizeGit(sourceUpdate.RepoURL) == urls.NormalizeGit(src.RepoURL) {
			return sourceUpdate
		}
	}
	return nil
}
