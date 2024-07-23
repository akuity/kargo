package argocd

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
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
	EvaluateHealth(
		context.Context,
		*kargoapi.Stage,
	) *kargoapi.Health
}

// applicationHealth is an ApplicationHealthEvaluator implementation.
type applicationHealth struct {
	kargoClient client.Client
	argoClient  client.Client
}

// NewApplicationHealthEvaluator returns a new ApplicationHealthEvaluator.
func NewApplicationHealthEvaluator(kargoClient, argoClient client.Client) ApplicationHealthEvaluator {
	return &applicationHealth{
		kargoClient: kargoClient,
		argoClient:  argoClient,
	}
}

// EvaluateHealth assesses the health of a set of Argo CD Applications.
func (h *applicationHealth) EvaluateHealth(
	ctx context.Context,
	stage *kargoapi.Stage,
) *kargoapi.Health {
	if stage.Spec.PromotionMechanisms == nil ||
		len(stage.Spec.PromotionMechanisms.ArgoCDAppUpdates) == 0 {
		return nil
	}

	if h.argoClient == nil {
		return &kargoapi.Health{
			Status: kargoapi.HealthStateUnknown,
			Issues: []string{
				"Argo CD integration is disabled; cannot assess the health or sync status of Argo CD Applications",
			},
		}
	}

	health := kargoapi.Health{
		Status:     kargoapi.HealthStateHealthy,
		ArgoCDApps: make([]kargoapi.ArgoCDAppStatus, len(stage.Spec.PromotionMechanisms.ArgoCDAppUpdates)),
		Issues:     make([]string, 0),
	}

	for i := range stage.Spec.PromotionMechanisms.ArgoCDAppUpdates {
		update := &stage.Spec.PromotionMechanisms.ArgoCDAppUpdates[i]
		namespace := update.AppNamespace
		if namespace == "" {
			namespace = Namespace()
		}

		health.ArgoCDApps[i] = kargoapi.ArgoCDAppStatus{
			Namespace: namespace,
			Name:      update.AppName,
		}

		state, healthStatus, syncStatus, err := h.GetApplicationHealth(
			ctx,
			stage,
			update,
			types.NamespacedName{
				Namespace: health.ArgoCDApps[i].Namespace,
				Name:      health.ArgoCDApps[i].Name,
			},
		)

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
	stage *kargoapi.Stage,
	update *kargoapi.ArgoCDAppUpdate,
	key types.NamespacedName,
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
	if err := h.argoClient.Get(ctx, key, app); err != nil {
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
	if desiredRevision, err := GetDesiredRevision(
		ctx,
		h.kargoClient,
		stage,
		update,
		app,
		stage.Status.FreightHistory.Current().References(),
	); err != nil {
		return kargoapi.HealthStateUnknown, healthStatus, syncStatus, err
	} else if desiredRevision != "" {
		if healthState, err := stageHealthForAppSync(app, desiredRevision); err != nil {
			return healthState, healthStatus, syncStatus, err
		}

		// If we care about revisions, and recently finished an operation, we
		// should wait for a cooldown period before assessing the health of the
		// application. This is to ensure the health check has a chance to run
		// after the sync operation has finished.
		//
		// xref: https://github.com/akuity/kargo/issues/2196
		//
		// TODO: revisit this when https://github.com/argoproj/argo-cd/pull/18660
		// 	 is merged and released.
		cooldown := app.Status.OperationState.FinishedAt.Time.Add(10 * time.Second)
		if duration := time.Until(cooldown); duration > 0 {
			time.Sleep(duration)

			// Re-fetch the application to get the latest state.
			if err := h.argoClient.Get(ctx, key, app); err != nil {
				err = fmt.Errorf("error finding Argo CD Application %q in namespace %q: %w", key.Name, key.Namespace, err)
				if client.IgnoreNotFound(err) == nil {
					err = fmt.Errorf("unable to find Argo CD Application %q in namespace %q", key.Name, key.Namespace)
				}
				return kargoapi.HealthStateUnknown, healthStatus, syncStatus, err
			}
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
	case app.Operation != nil && app.Operation.Sync != nil,
		app.Status.OperationState == nil || app.Status.OperationState.FinishedAt.IsZero():
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
	case argocd.HealthStatusSuspended:
		err := fmt.Errorf(
			"Argo CD Application %q in namespace %q is suspended",
			app.GetName(),
			app.GetNamespace(),
		)
		// To Kargo, a suspended Application is considered progressing until
		// the suspension is lifted.
		// xref: https://github.com/akuity/kargo/issues/2216
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
