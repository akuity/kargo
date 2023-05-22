package environments

import (
	"context"
	"fmt"

	api "github.com/akuity/kargo/api/v1alpha1"
	libArgoCD "github.com/akuity/kargo/internal/argocd"
)

func (r *reconciler) checkHealth(
	ctx context.Context,
	currentState api.EnvironmentState,
	argoCDAppUpdates []api.ArgoCDAppUpdate,
) api.Health {
	if len(argoCDAppUpdates) == 0 {
		return api.Health{
			Status:       api.HealthStateUnknown,
			StatusReason: "no spec.promotionMechanisms.argoCDAppUpdates are defined",
		}
	}

	for _, check := range argoCDAppUpdates {
		app, err :=
			r.getArgoCDAppFn(ctx, r.client, check.AppNamespace, check.AppName)
		if err != nil {
			return api.Health{
				Status: api.HealthStateUnknown,
				StatusReason: fmt.Sprintf(
					"error finding Argo CD Application %q in namespace %q: %s",
					check.AppName,
					check.AppNamespace,
					err,
				),
			}
		}
		if app == nil {
			return api.Health{
				Status: api.HealthStateUnknown,
				StatusReason: fmt.Sprintf(
					"unable to find Argo CD Application %q in namespace %q",
					check.AppName,
					check.AppNamespace,
				),
			}
		}

		if len(app.Spec.Sources) > 0 {
			return api.Health{
				Status: api.HealthStateUnknown,
				StatusReason: fmt.Sprintf(
					"bugs in Argo CD currently prevent a comprehensive assessment of "+
						"the health of multi-source Application %q in namespace %q",
					check.AppName,
					check.AppNamespace,
				),
			}
		}

		var desiredRevision string
		for _, commit := range currentState.Commits {
			if commit.RepoURL == app.Spec.Source.RepoURL {
				if commit.HealthCheckCommit != "" {
					desiredRevision = commit.HealthCheckCommit
				} else {
					desiredRevision = commit.ID
				}
			}
		}
		if desiredRevision == "" {
			for _, chart := range currentState.Charts {
				if chart.RegistryURL == app.Spec.Source.RepoURL &&
					chart.Name == app.Spec.Source.Chart {
					desiredRevision = chart.Version
				}
			}
		}

		if healthy, reason := libArgoCD.IsApplicationHealthyAndSynced(
			app,
			desiredRevision,
		); !healthy {
			return api.Health{
				Status:       api.HealthStateUnhealthy,
				StatusReason: reason,
			}
		}
	}

	return api.Health{
		Status: api.HealthStateHealthy,
	}
}
