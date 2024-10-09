package directives

import (
	"context"
	"fmt"
	"strings"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/freight"
)

// getDesiredRevisions returns the desired revisions for all sources of the given
// Application. For a single-source Application, the returned slice will have
// precisely one value. For a multi-source Application, the returned slice will
// have the same length as and will be indexed identically to the Application's
// Sources slice. For any source whose desired revision cannot be determined,
// the slice will contain an empty string at the corresponding index.
func (a *argocdUpdater) getDesiredRevisions(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	stepCfg *ArgoCDUpdateConfig,
	update *ArgoCDAppUpdate,
	app *argocd.Application,
) ([]string, error) {
	if app == nil {
		return nil, nil
	}
	sources := app.Spec.Sources
	if len(sources) == 0 && app.Spec.Source != nil {
		sources = []argocd.ApplicationSource{*app.Spec.Source}
	}
	if len(sources) == 0 {
		return nil, nil
	}
	revisions := make([]string, len(sources))
	for i, src := range sources {
		sourceUpdate := a.findSourceUpdate(update, src)
		// If there is a source update that targets this source, it might be
		// specific about a previous step whose output should be used as the desired
		// revision.
		if sourceUpdate != nil {
			var err error
			if revisions[i], err = getCommitFromStep(
				stepCtx.SharedState,
				sourceUpdate.DesiredCommitFromStep,
			); err != nil {
				return nil, err
			}
			if revisions[i] != "" {
				continue
			}
		}
		var desiredOrigin *kargoapi.FreightOrigin
		// If there is a source update that targets this source, it might be
		// specific about which origin the desired revision should come from.
		if sourceUpdate != nil {
			desiredOrigin = getDesiredOrigin(stepCfg, sourceUpdate)
		} else {
			desiredOrigin = getDesiredOrigin(stepCfg, update)
		}
		desiredRevision, err := a.getDesiredRevisionForSource(
			ctx,
			stepCtx,
			&src,
			desiredOrigin,
		)
		if err != nil {
			return nil, err
		}
		revisions[i] = desiredRevision
	}
	return revisions, nil
}

func (a *argocdUpdater) getDesiredRevisionForSource(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	src *argocd.ApplicationSource,
	desiredOrigin *kargoapi.FreightOrigin,
) (string, error) {
	switch {
	case src.Chart != "":
		// This source points to a Helm chart.
		repoURL := src.RepoURL
		chartName := src.Chart
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
		chart, err := freight.FindChart(
			ctx,
			stepCtx.KargoClient,
			stepCtx.Project,
			stepCtx.FreightRequests,
			desiredOrigin,
			stepCtx.Freight.References(),
			repoURL,
			chartName,
		)
		if err != nil {
			return "",
				fmt.Errorf("error finding chart from repo %q: %w", repoURL, err)
		}
		if chart == nil {
			return "", nil
		}
		return chart.Version, nil
	case src.RepoURL != "":
		// This source points to a Git repository.
		commit, err := freight.FindCommit(
			ctx,
			stepCtx.KargoClient,
			stepCtx.Project,
			stepCtx.FreightRequests,
			desiredOrigin,
			stepCtx.Freight.References(),
			src.RepoURL,
		)
		if err != nil {
			return "",
				fmt.Errorf("error finding commit from repo %q: %w", src.RepoURL, err)
		}
		if commit == nil {
			return "", nil
		}
		return commit.ID, nil
	}
	// If we end up here, no desired revision was found.
	return "", nil
}

// findSourceUpdate finds and returns the ArgoCDSourceUpdate that targets the
// given source. If no such update exists, it returns nil.
func (a *argocdUpdater) findSourceUpdate(
	update *ArgoCDAppUpdate,
	src argocd.ApplicationSource,
) *ArgoCDAppSourceUpdate {
	for i := range update.Sources {
		sourceUpdate := &update.Sources[i]
		if sourceUpdate.RepoURL == src.RepoURL && sourceUpdate.Chart == src.Chart {
			return sourceUpdate
		}
	}
	return nil
}

func getCommitFromStep(sharedState State, stepAlias string) (string, error) {
	if stepAlias == "" {
		return "", nil
	}
	stepOutput, exists := sharedState.Get(stepAlias)
	if !exists {
		return "", fmt.Errorf("no output found from step with alias %q", stepAlias)
	}
	stepOutputMap, ok := stepOutput.(map[string]any)
	if !ok {
		return "",
			fmt.Errorf("output from step with alias %q is not a map[string]any", stepAlias)
	}
	commitAny, exists := stepOutputMap[commitKey]
	if !exists {
		return "",
			fmt.Errorf("no commit found in output from step with alias %q", stepAlias)
	}
	commit, ok := commitAny.(string)
	if !ok {
		return "", fmt.Errorf(
			"commit in output from step with alias %q is not a string",
			stepAlias,
		)
	}
	return commit, nil
}
