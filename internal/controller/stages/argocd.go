package stages

import (
	"context"
	"fmt"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argohealth "github.com/argoproj/gitops-engine/pkg/health"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func (r *reconciler) checkHealth(
	ctx context.Context,
	currentFreight kargoapi.Freight,
	argoCDAppUpdates []kargoapi.ArgoCDAppUpdate,
) kargoapi.Health {
	if len(argoCDAppUpdates) == 0 {
		return kargoapi.Health{
			Status: kargoapi.HealthStateUnknown,
			Issues: []string{
				"no spec.promotionMechanisms.argoCDAppUpdates are defined",
			},
		}
	}

	health := kargoapi.Health{
		// We'll start healthy and degrade as we find issues
		Status: kargoapi.HealthStateHealthy,
		Issues: []string{},
	}

	for _, check := range argoCDAppUpdates {
		app, err := r.getArgoCDAppFn(ctx, r.argoClient, check.AppNamespaceOrDefault(), check.AppName)
		if err != nil {
			if health.Status != kargoapi.HealthStateUnhealthy {
				health.Status = kargoapi.HealthStateUnknown
			}
			health.Issues = append(
				health.Issues,
				fmt.Sprintf(
					"error finding Argo CD Application %q in namespace %q: %s",
					check.AppName,
					check.AppNamespaceOrDefault(),
					err,
				),
			)
		} else if app == nil {
			if health.Status != kargoapi.HealthStateUnhealthy {
				health.Status = kargoapi.HealthStateUnknown
			}
			health.Issues = append(
				health.Issues,
				fmt.Sprintf(
					"unable to find Argo CD Application %q in namespace %q",
					check.AppName,
					check.AppNamespaceOrDefault(),
				),
			)
		} else if len(app.Spec.Sources) > 0 {
			if health.Status != kargoapi.HealthStateUnhealthy {
				health.Status = kargoapi.HealthStateUnknown
			}
			health.Issues = append(
				health.Issues,
				fmt.Sprintf(
					"bugs in Argo CD currently prevent a comprehensive assessment of "+
						"the health of multi-source Application %q in namespace %q",
					check.AppName,
					check.AppNamespaceOrDefault(),
				),
			)
		} else {
			var desiredRevision string
			for _, commit := range currentFreight.Commits {
				if commit.RepoURL == app.Spec.Source.RepoURL {
					if commit.HealthCheckCommit != "" {
						desiredRevision = commit.HealthCheckCommit
					} else {
						desiredRevision = commit.ID
					}
				}
			}
			if desiredRevision == "" {
				for _, chart := range currentFreight.Charts {
					if chart.RegistryURL == app.Spec.Source.RepoURL &&
						chart.Name == app.Spec.Source.Chart {
						desiredRevision = chart.Version
					}
				}
			}
			// TODO: currently an stage relies on the Argo CD app being both Healthy
			// and Synced in order for the freight to be healthy. But many users run
			// in a mode where apps are in a perpetual state of drift, and it is
			// unreasonable to expect Sync status will be Synced. We need to switch to
			// perhaps only considering health, and perhaps considering whether or not
			// an operation is in flight. See:
			// https://github.com/akuity/kargo/issues/670
			health = health.Merge(stageHealthForAppHealth(app))
			health = health.Merge(stageHealthForAppSync(app, desiredRevision))
		}
	}

	return health
}

func stageHealthForAppHealth(app *argocd.Application) kargoapi.Health {
	switch app.Status.Health.Status {
	case argohealth.HealthStatusDegraded, argohealth.HealthStatusUnknown:
		return kargoapi.Health{
			Status: kargoapi.HealthStateUnhealthy,
			Issues: []string{fmt.Sprintf("Argo CD Application %q in namespace %q has health state %q",
				app.Name, app.Namespace, app.Status.Health.Status)},
		}
	case argohealth.HealthStatusProgressing, "":
		return kargoapi.Health{
			Status: kargoapi.HealthStateProgressing,
			Issues: []string{fmt.Sprintf("Argo CD Application %q in namespace %q is progressing",
				app.Name, app.Namespace)},
		}
	default:
		return kargoapi.Health{Status: kargoapi.HealthStateHealthy}
	}
}

func stageHealthForAppSync(app *argocd.Application, revision string) kargoapi.Health {
	if revision != "" && app.Status.Sync.Revision != revision {

		if app.Operation != nil && app.Operation.Sync != nil {
			return kargoapi.Health{
				Status: kargoapi.HealthStateProgressing,
				Issues: []string{fmt.Sprintf("Argo CD Application %q in namespace %q is being synced",
					app.Name, app.Namespace)},
			}
		}
		return kargoapi.Health{
			Status: kargoapi.HealthStateUnhealthy,
			Issues: []string{fmt.Sprintf("Argo CD Application %q in namespace %q is not synced to revision %q",
				app.Name, app.Namespace, revision)},
		}
	}
	return kargoapi.Health{Status: kargoapi.HealthStateHealthy}
}
