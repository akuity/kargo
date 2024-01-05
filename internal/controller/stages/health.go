package stages

import (
	"context"
	"fmt"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
)

func (r *reconciler) checkHealth(
	ctx context.Context,
	currentFreight kargoapi.FreightReference,
	argoCDAppUpdates []kargoapi.ArgoCDAppUpdate,
) *kargoapi.Health {
	if len(argoCDAppUpdates) == 0 {
		return nil
	}

	h := kargoapi.Health{
		// We'll start healthy and degrade as we find issues
		Status:     kargoapi.HealthStateHealthy,
		ArgoCDApps: make([]kargoapi.ArgoCDAppStatus, len(argoCDAppUpdates)),
		Issues:     []string{},
	}

	for i, updates := range argoCDAppUpdates {
		h.ArgoCDApps[i] = kargoapi.ArgoCDAppStatus{
			Namespace: updates.AppNamespaceOrDefault(),
			Name:      updates.AppName,
		}

		app, err := r.getArgoCDAppFn(
			ctx,
			r.argocdClient,
			updates.AppNamespaceOrDefault(),
			updates.AppName,
		)

		if err != nil {
			h.ArgoCDApps[i].HealthStatus = kargoapi.ArgoCDAppHealthStatus{
				Status: kargoapi.ArgoCDAppHealthStateUnknown,
			}
			h.ArgoCDApps[i].SyncStatus = kargoapi.ArgoCDAppSyncStatus{
				Status: kargoapi.ArgoCDAppSyncStateUnknown,
			}
			h.Status = h.Status.Merge(kargoapi.HealthStateUnknown)
			h.Issues = append(
				h.Issues,
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
			h.ArgoCDApps[i].HealthStatus = kargoapi.ArgoCDAppHealthStatus{
				Status: kargoapi.ArgoCDAppHealthStateUnknown,
			}
			h.ArgoCDApps[i].SyncStatus = kargoapi.ArgoCDAppSyncStatus{
				Status: kargoapi.ArgoCDAppSyncStateUnknown,
			}
			h.Status = h.Status.Merge(kargoapi.HealthStateUnknown)
			h.Issues = append(
				h.Issues,
				fmt.Sprintf(
					"unable to find Argo CD Application %q in namespace %q",
					updates.AppName,
					updates.AppNamespaceOrDefault(),
				),
			)
			continue
		}

		h.ArgoCDApps[i].HealthStatus = kargoapi.ArgoCDAppHealthStatus{
			Status:  kargoapi.ArgoCDAppHealthState(app.Status.Health.Status),
			Message: app.Status.Health.Message,
		}
		h.ArgoCDApps[i].SyncStatus = kargoapi.ArgoCDAppSyncStatus{
			Status:    kargoapi.ArgoCDAppSyncState(app.Status.Sync.Status),
			Revision:  app.Status.Sync.Revision,
			Revisions: app.Status.Sync.Revisions,
		}

		// TODO: We should re-evaluate this soon. It may have been fixed in recent
		// versions.
		if len(app.Spec.Sources) > 0 {
			h.Status = h.Status.Merge(kargoapi.HealthStateUnknown)
			h.Issues = append(
				h.Issues,
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
		h.Status = h.Status.Merge(stageHealth)
		if issue != "" {
			h.Issues = append(h.Issues, issue)
		}

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
			h.Status = h.Status.Merge(stageHealth)
			if issue != "" {
				h.Issues = append(h.Issues, issue)
			}
		}

	}

	return &h
}

func stageHealthForAppHealth(
	app *argocd.Application,
) (kargoapi.HealthState, string) {
	switch app.Status.Health.Status {
	case argocd.HealthStatusProgressing, "":
		return kargoapi.HealthStateProgressing,
			fmt.Sprintf(
				"Argo CD Application %q in namespace %q is progressing",
				app.Name,
				app.Namespace,
			)
	case argocd.HealthStatusHealthy:
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
