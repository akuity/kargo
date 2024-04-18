package argocd

import (
	"context"
	"errors"
	"fmt"
	"path"
	"slices"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/git"
)

// healthErrorConditions are the v1alpha1.ApplicationConditionType conditions
// that indicate an Argo CD Application is unhealthy.
var healthErrorConditions = []argocd.ApplicationConditionType{
	argocd.ApplicationConditionComparisonError,
	argocd.ApplicationConditionInvalidSpecError,
}

// compositeError is an interface for wrapped standard errors produced by
// errors.Join.
type compositeError interface {
	// Unwrap returns the wrapped errors.
	Unwrap() []error
}

// ApplicationHealthEvaluator is an interface for evaluating the health of
// Argo CD Applications.
type ApplicationHealthEvaluator interface {
	EvaluateHealth(context.Context, kargoapi.FreightReference, []kargoapi.ArgoCDAppUpdate) *kargoapi.Health
}

// applicationHealth is an ApplicationHealthEvaluator implementation.
type applicationHealth struct {
	Client client.Client
}

// NewApplicationHealthEvaluator returns a new ApplicationHealthEvaluator.
func NewApplicationHealthEvaluator(c client.Client) ApplicationHealthEvaluator {
	return &applicationHealth{Client: c}
}

// EvaluateHealth assesses the health of a set of Argo CD Applications.
func (h *applicationHealth) EvaluateHealth(
	ctx context.Context,
	freight kargoapi.FreightReference,
	updates []kargoapi.ArgoCDAppUpdate,
) *kargoapi.Health {
	if len(updates) == 0 {
		return nil
	}

	if h.Client == nil {
		return &kargoapi.Health{
			Status: kargoapi.HealthStateUnknown,
			Issues: []string{
				"Argo CD integration is disabled; cannot assess the health or sync status of Argo CD Applications",
			},
		}
	}

	health := kargoapi.Health{
		Status:     kargoapi.HealthStateHealthy,
		ArgoCDApps: make([]kargoapi.ArgoCDAppStatus, len(updates)),
		Issues:     make([]string, 0),
	}

	for i, update := range updates {
		namespace := update.AppNamespace
		if namespace == "" {
			namespace = Namespace()
		}

		health.ArgoCDApps[i] = kargoapi.ArgoCDAppStatus{
			Namespace: namespace,
			Name:      update.AppName,
		}

		state, healthStatus, syncStatus, err := h.GetApplicationHealth(ctx, types.NamespacedName{
			Namespace: health.ArgoCDApps[i].Namespace,
			Name:      health.ArgoCDApps[i].Name,
		}, freight)

		health.Status = health.Status.Merge(state)
		health.ArgoCDApps[i].HealthStatus = healthStatus
		health.ArgoCDApps[i].SyncStatus = syncStatus

		if err != nil {
			if cErr, ok := err.(compositeError); ok {
				for _, e := range cErr.Unwrap() {
					health.Issues = append(health.Issues, e.Error())
				}
			} else {
				health.Issues = append(health.Issues, err.Error())
			}
		}
	}

	return &health
}

// GetApplicationHealth assesses the health of an Argo CD Application by looking
// at its conditions, health status, and sync status. Based on these, it returns
// an overall health state, the Argo CD Application's health status, and its sync
// status. If it can not (fully) assess the health of the Argo CD Application, it
// returns an error with a message explaining why.
func (h *applicationHealth) GetApplicationHealth(
	ctx context.Context,
	key types.NamespacedName,
	freight kargoapi.FreightReference,
) (kargoapi.HealthState, kargoapi.ArgoCDAppHealthStatus, kargoapi.ArgoCDAppSyncStatus, error) {
	var (
		healthStatus = kargoapi.ArgoCDAppHealthStatus{
			Status: kargoapi.ArgoCDAppHealthStateUnknown,
		}
		syncStatus = kargoapi.ArgoCDAppSyncStatus{
			Status: kargoapi.ArgoCDAppSyncStateUnknown,
		}
	)

	app := &argocd.Application{}
	if err := h.Client.Get(ctx, key, app); err != nil {
		err = fmt.Errorf("error finding Argo CD Application %q in namespace %q: %w", key.Name, key.Namespace, err)
		if client.IgnoreNotFound(err) == nil {
			err = fmt.Errorf("unable to find Argo CD Application %q in namespace %q", key.Name, key.Namespace)
		}
		return kargoapi.HealthStateUnknown, healthStatus, syncStatus, err
	}

	// Mirror the health and sync status of the Argo CD Application.
	if app.Status.Health.Status != "" {
		healthStatus = kargoapi.ArgoCDAppHealthStatus{
			Status:  kargoapi.ArgoCDAppHealthState(app.Status.Health.Status),
			Message: app.Status.Health.Message,
		}
	}
	if app.Status.Sync.Status != "" {
		syncStatus = kargoapi.ArgoCDAppSyncStatus{
			Status:    kargoapi.ArgoCDAppSyncState(app.Status.Sync.Status),
			Revision:  app.Status.Sync.Revision,
			Revisions: app.Status.Sync.Revisions,
		}
	}

	// TODO: We should re-evaluate this soon. It may have been fixed in recent
	//       versions.
	// TODO(hidde): Do we have an upstream reference for this?
	if len(app.Spec.Sources) > 0 {
		err := fmt.Errorf(
			"bugs in Argo CD currently prevent a comprehensive assessment of "+
				"the health of multi-source Application %q in namespace %q",
			key.Name,
			key.Namespace,
		)
		return kargoapi.HealthStateUnknown, healthStatus, syncStatus, err
	}

	// Check for any error conditions. If these are found, the application is
	// considered unhealthy as they may indicate a problem which can result in
	// e.g. the health status result to become unreliable.
	if errConditions := filterAppConditions(app, healthErrorConditions...); len(errConditions) > 0 {
		issues := make([]error, len(errConditions))
		for _, condition := range errConditions {
			issues = append(issues, fmt.Errorf(
				"Argo CD Application %q in namespace %q has %q condition: %s",
				key.Name,
				key.Namespace,
				condition.Type,
				condition.Message,
			))
		}
		return kargoapi.HealthStateUnhealthy, healthStatus, syncStatus, errors.Join(issues...)
	}

	// If we have a desired revision, we should confirm the Argo CD Application
	// is syncing to it. We do not further care about the cluster being in sync
	// with the desired revision, as some applications may be out of sync by
	// default.
	if desiredRevision := getDesiredRevision(app, freight); desiredRevision != "" {
		if healthState, err := stageHealthForAppSync(app, desiredRevision); err != nil {
			return healthState, healthStatus, syncStatus, err
		}
	}

	// With all the above checks passed, we can now assume the Argo CD
	// Application's health state is reliable.
	healthState, err := stageHealthForAppHealth(app)
	return healthState, healthStatus, syncStatus, err
}

// stageHealthForAppSync returns the v1alpha1.HealthState for an Argo CD
// Application based on its sync status.
func stageHealthForAppSync(app *argocd.Application, revision string) (kargoapi.HealthState, error) {
	switch {
	case revision == "":
		return kargoapi.HealthStateHealthy, nil
	// TODO(hidde): When https://github.com/akuity/kargo/pull/1753 is merged,
	//  this can rely on the operation state from the Status.
	case app.Operation != nil && app.Operation.Sync != nil:
		err := fmt.Errorf(
			"Argo CD Application %q in namespace %q is being synced",
			app.GetName(),
			app.GetNamespace(),
		)
		return kargoapi.HealthStateUnknown, err
	case app.Status.Sync.Revision != revision:
		err := fmt.Errorf(
			"Argo CD Application %q in namespace %q is out of sync",
			app.GetName(),
			app.GetNamespace(),
		)
		return kargoapi.HealthStateUnhealthy, err
	default:
		return kargoapi.HealthStateHealthy, nil
	}
}

// stageHealthForAppHealth returns the v1alpha1.HealthState for an Argo CD
// Application based on its health status.
func stageHealthForAppHealth(app *argocd.Application) (kargoapi.HealthState, error) {
	switch app.Status.Health.Status {
	case argocd.HealthStatusProgressing, "":
		err := fmt.Errorf(
			"Argo CD Application %q in namespace %q is progressing",
			app.GetName(),
			app.GetNamespace(),
		)
		return kargoapi.HealthStateProgressing, err
	case argocd.HealthStatusHealthy:
		return kargoapi.HealthStateHealthy, nil
	default:
		err := fmt.Errorf(
			"Argo CD Application %q in namespace %q has health state %q",
			app.GetName(),
			app.GetNamespace(),
			app.Status.Health.Status,
		)
		return kargoapi.HealthStateUnhealthy, err
	}
}

// filterAppConditions returns a slice of v1alpha1.ApplicationCondition that
// match the provided types.
func filterAppConditions(
	app *argocd.Application,
	t ...argocd.ApplicationConditionType,
) []argocd.ApplicationCondition {
	errs := make([]argocd.ApplicationCondition, 0, len(app.Status.Conditions))
	for _, condition := range app.Status.Conditions {
		if slices.Contains(t, condition.Type) {
			errs = append(errs, condition)
		}
	}
	return errs
}

// getDesiredRevision returns the desired revision for an Argo CD Application
// based on the provided v1alpha1.FreightReference.
// TODO(hidde): This function can be removed when
//
//	https://github.com/akuity/kargo/pull/1753 is merged.
func getDesiredRevision(app *argocd.Application, freight kargoapi.FreightReference) string {
	if app.Spec.Source.Chart == "" {
		// This source points to a git repository
		sourceGitRepoURL := git.NormalizeURL(app.Spec.Source.RepoURL)
		for _, commit := range freight.Commits {
			if git.NormalizeURL(commit.RepoURL) == sourceGitRepoURL {
				if commit.HealthCheckCommit != "" {
					return commit.HealthCheckCommit
				}
				return commit.ID
			}
		}
	} else {
		// This source points to a Helm chart
		for _, chart := range freight.Charts {
			// path.Join accounts for the possibility that chart.Name is empty
			if path.Join(chart.RepoURL, chart.Name) == path.Join(app.Spec.Source.RepoURL, app.Spec.Source.Chart) {
				return chart.Version
			}
		}
	}
	return ""
}
