package argocd

import (
	"path"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/git"
)

// GetDesiredRevision returns the desired revision for the given
// v1alpha1.Application by traversing the given v1alpha1.FreightReference for
// a matching source. If no match is found, an empty string is returned.
func GetDesiredRevision(app *argocd.Application, freight kargoapi.FreightReference) string {
	switch {
	case app == nil || app.Spec.Source == nil:
		// Without an Application, we can't determine the desired revision.
		return ""
	case app.Spec.Source.Chart != "":
		// This source points to a Helm chart.
		// NB: This has to go first, as the repository URL can also point to
		//     a Helm repository.
		sourceChart := path.Join(app.Spec.Source.RepoURL, app.Spec.Source.Chart)
		for _, chart := range freight.Charts {
			// Join accounts for the possibility that chart.Name is empty.
			if path.Join(chart.RepoURL, chart.Name) == sourceChart {
				return chart.Version
			}
		}
	case app.Spec.Source.RepoURL != "":
		// This source points to a Git repository.
		sourceGitRepoURL := git.NormalizeURL(app.Spec.Source.RepoURL)
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
	// If we end up here, no desired revision was found.
	return ""
}
