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
	currentFreight *kargoapi.Freight,
	argoCDAppUpdates []kargoapi.ArgoCDAppUpdate,
) *kargoapi.Health {
	if len(argoCDAppUpdates) == 0 {
		return nil
	}

	health := kargoapi.Health{
		// We'll start healthy and degrade as we find issues
		Status:     kargoapi.HealthStateHealthy,
		ArgoCDApps: make([]kargoapi.ArgoCDAppStatus, len(argoCDAppUpdates)),
		Issues:     []string{},
	}

	for i, updates := range argoCDAppUpdates {
		health.ArgoCDApps[i] = kargoapi.ArgoCDAppStatus{
			Namespace: updates.AppNamespaceOrDefault(),
			Name:      updates.AppName,
		}

		app, err := r.getArgoCDAppFn(
			ctx,
			r.argoClient,
			updates.AppNamespaceOrDefault(),
			updates.AppName,
		)

		if err != nil {
			health.ArgoCDApps[i].HealthStatus = kargoapi.ArgoCDAppHealthStatus{
				Status: kargoapi.ArgoCDAppHealthStateUnknown,
			}
			health.ArgoCDApps[i].SyncStatus = kargoapi.ArgoCDAppSyncStatus{
				Status: kargoapi.ArgoCDAppSyncStateUnknown,
			}
			health.Status = health.Status.Merge(kargoapi.HealthStateUnknown)
			health.Issues = append(
				health.Issues,
				fmt.Sprintf(
					"error finding Argo CD Application %q in namespace %q: %s",
					updates.AppName,
					updates.AppNamespaceOrDefault(),
					err,
				),
			)
			continue
		}

		if app == nil {
			health.ArgoCDApps[i].HealthStatus = kargoapi.ArgoCDAppHealthStatus{
				Status: kargoapi.ArgoCDAppHealthStateUnknown,
			}
			health.ArgoCDApps[i].SyncStatus = kargoapi.ArgoCDAppSyncStatus{
				Status: kargoapi.ArgoCDAppSyncStateUnknown,
			}
			health.Status = health.Status.Merge(kargoapi.HealthStateUnknown)
			health.Issues = append(
				health.Issues,
				fmt.Sprintf(
					"unable to find Argo CD Application %q in namespace %q",
					updates.AppName,
					updates.AppNamespaceOrDefault(),
				),
			)
			continue
		}

		health.ArgoCDApps[i].HealthStatus = kargoapi.ArgoCDAppHealthStatus{
			Status:  kargoapi.ArgoCDAppHealthState(app.Status.Health.Status),
			Message: app.Status.Health.Message,
		}
		health.ArgoCDApps[i].SyncStatus = kargoapi.ArgoCDAppSyncStatus{
			Status:    kargoapi.ArgoCDAppSyncState(app.Status.Sync.Status),
			Revision:  app.Status.Sync.Revision,
			Revisions: app.Status.Sync.Revisions,
		}

		// TODO: We should re-evaluate this soon. It may have been fixed in recent
		// versions.
		if len(app.Spec.Sources) > 0 {
			health.Status = health.Status.Merge(kargoapi.HealthStateUnknown)
			health.Issues = append(
				health.Issues,
				fmt.Sprintf(
					"bugs in Argo CD currently prevent a comprehensive assessment of "+
						"the health of multi-source Application %q in namespace %q",
					updates.AppName,
					updates.AppNamespaceOrDefault(),
				),
			)
			continue
		}

		stageHealth, issue := stageHealthForAppHealth(app)
		health.Status = health.Status.Merge(stageHealth)
		if issue != "" {
			health.Issues = append(health.Issues, issue)
		}

		if currentFreight != nil {
			var desiredRevision string
			for _, commit := range currentFreight.Commits {
				if commit.RepoURL == app.Spec.Source.RepoURL {
					if commit.HealthCheckCommit != "" {
						desiredRevision = commit.HealthCheckCommit
					} else {
						desiredRevision = commit.ID
					}
				}
				break
			}
			if desiredRevision == "" {
				for _, chart := range currentFreight.Charts {
					if chart.RegistryURL == app.Spec.Source.RepoURL &&
						chart.Name == app.Spec.Source.Chart {
						desiredRevision = chart.Version
						break
					}
				}
			}
			if desiredRevision != "" {
				stageHealth, issue = stageHealthForAppSync(app, desiredRevision)
				health.Status = health.Status.Merge(stageHealth)
				if issue != "" {
					health.Issues = append(health.Issues, issue)
				}
			}
		}
	}

	return &health
}

func stageHealthForAppHealth(
	app *argocd.Application,
) (kargoapi.HealthState, string) {
	switch app.Status.Health.Status {
	case argohealth.HealthStatusProgressing, "":
		return kargoapi.HealthStateProgressing,
			fmt.Sprintf(
				"Argo CD Application %q in namespace %q is progressing",
				app.Name,
				app.Namespace,
			)
	case argohealth.HealthStatusHealthy:
		return kargoapi.HealthStateHealthy, ""
	default:
		return kargoapi.HealthStateUnhealthy,
			fmt.Sprintf(
				"Argo CD Application %q in namespace %q has health state %q",
				app.Name,
				app.Namespace,
				app.Status.Health.Status,
			)
	}
}

func stageHealthForAppSync(
	app *argocd.Application,
	revision string,
) (kargoapi.HealthState, string) {
	if revision != "" && app.Status.Sync.Revision != revision {
		if app.Operation != nil && app.Operation.Sync != nil {
			return kargoapi.HealthStateProgressing,
				fmt.Sprintf(
					"Argo CD Application %q in namespace %q is being synced",
					app.Name,
					app.Namespace,
				)
		}
		return kargoapi.HealthStateUnhealthy,
			fmt.Sprintf(
				"Argo CD Application %q in namespace %q is not synced to revision %q",
				app.Name,
				app.Namespace,
				revision,
			)
	}
	return kargoapi.HealthStateHealthy, ""
}
