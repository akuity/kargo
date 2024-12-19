package directives

import (
	"fmt"

	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
)

// getDesiredRevisions returns the desired revisions for all sources of the given
// Application. For a single-source Application, the returned slice will have
// precisely one value. For a multi-source Application, the returned slice will
// have the same length as and will be indexed identically to the Application's
// Sources slice. For any source whose desired revision cannot be determined,
// the slice will contain an empty string at the corresponding index.
func (a *argocdUpdater) getDesiredRevisions(
	stepCtx *PromotionStepContext,
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
			revisions[i] = sourceUpdate.DesiredRevision
			if revisions[i] == "" {
				var err error
				if revisions[i], err = getCommitFromStep(
					stepCtx.SharedState,
					sourceUpdate.DesiredCommitFromStep,
				); err != nil {
					return nil, err
				}
			}
		}
	}
	return revisions, nil
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
	commitAny, exists := stepOutputMap[stateKeyCommit]
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
