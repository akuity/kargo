package argocd

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
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
	logger := logging.LoggerFromContext(ctx)
	logger.Debug("About to evaluate ArgoCD application health.")

	if stage.Spec.PromotionMechanisms == nil ||
		len(stage.Spec.PromotionMechanisms.ArgoCDAppUpdates) == 0 {
		logger.Debug("No updates to process, skipping.")
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

	logger.Debug("About to evaluate health of applications.", "count",
		len(stage.Spec.PromotionMechanisms.ArgoCDAppUpdates))
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

		logger.Debug("About to get health of application.", "appName", update.AppName)
		state, healthStatus, syncStatus, err := h.GetApplicationHealth(
			ctx,
			stage,
			update,
			types.NamespacedName{
				Namespace: health.ArgoCDApps[i].Namespace,
				Name:      health.ArgoCDApps[i].Name,
			},
		)

		logger.Debug("Got application health status.", "appName", update.AppName,
			"healthStatus", healthStatus.Status, "syncStatus", syncStatus.Status, "state", state)

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

	logger := logging.LoggerFromContext(ctx)
	logger.Debug("About to get application health.")

	app := &argocd.Application{}
	if err := h.argoClient.Get(ctx, key, app); err != nil {
		err = fmt.Errorf("error finding Argo CD Application %q in namespace %q: %w", key.Name, key.Namespace, err)
		if client.IgnoreNotFound(err) == nil {
			err = fmt.Errorf("unable to find Argo CD Application %q in namespace %q", key.Name, key.Namespace)
		}
		return kargoapi.HealthStateUnknown, healthStatus, syncStatus, err
	}

	logger.Debug("Successfully received application health from ArgoCD.", "key", key.Name, "namespace", key.Namespace)

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
	logger.Debug("About to check for app conditions")
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
		logger.Error(errors.Join(issues...), "Application has conditions, considering Unhealthy.")
		return kargoapi.HealthStateUnhealthy, healthStatus, syncStatus, errors.Join(issues...)
	}

	// If we have a desired revision, we should confirm the Argo CD Application
	// is syncing to it. We do not further care about the cluster being in sync
	// with the desired revision, as some applications may be out of sync by
	// default.
	if desiredRevisions, err := GetDesiredRevisions(
		ctx,
		h.kargoClient,
		stage,
		update,
		app,
		stage.Status.FreightHistory.Current().References(),
	); err != nil {
		logger.Error(err, "Error getting desired revision, assuming Unknown health state.")
		return kargoapi.HealthStateUnknown, healthStatus, syncStatus, err
	} else if len(desiredRevisions) > 0 {
		if healthState, err := stageHealthForAppSync(ctx, app, desiredRevisions); err != nil {
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
		if app.Status.OperationState != nil {
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
	}

	// With all the above checks passed, we can now assume the Argo CD
	// Application's health state is reliable.
	healthState, err := stageHealthForAppHealth(ctx, app)
	return healthState, healthStatus, syncStatus, err
}

// stageHealthForAppSync returns the v1alpha1.HealthState for an Argo CD
// Application based on its sync status.
func stageHealthForAppSync(
	ctx context.Context,
	app *argocd.Application,
	revisions []string) (kargoapi.HealthState, error) {
	logger := logging.LoggerFromContext(ctx).WithValues("appName", app.GetName(), "revisions", revisions)
	logger.Debug("About to determine stage health based on app sync status.")
	switch {
	case revisions == nil || (len(revisions) == 1 && revisions[0] == ""):
		logger.Debug("Desired revision not set, assuming healthy.")
		return kargoapi.HealthStateHealthy, nil
	case app.Operation != nil && app.Operation.Sync != nil,
		app.Status.OperationState == nil || app.Status.OperationState.FinishedAt.IsZero():
		logger.Debug("Application in sync operation, assuming Unknown.")
		err := fmt.Errorf(
			"Argo CD Application %q in namespace %q is being synced",
			app.GetName(),
			app.GetNamespace(),
		)
		return kargoapi.HealthStateUnknown, err

	default:
		// Apps may have multiple revisions in the list of revisions, so we need to check in both places.

		// Trivial case where app has only a single source and revision is set.
		singleSourceRevision := app.Status.Sync.Revision
		if !app.IsMultisource() {
			if len(revisions) == 1 && revisions[0] == singleSourceRevision {
				return kargoapi.HealthStateHealthy, nil
			}

			msg := fmt.Sprintf(
				"Desired revision %q does not match current revision %q of Application %q "+
					"in namespace %q, assuming unhealthy.",
				revisions,
				singleSourceRevision,
				app.GetName(),
				app.GetNamespace(),
			)

			return kargoapi.HealthStateUnhealthy, errors.New(msg)
		}

		multiSourceRevisions := app.Status.Sync.Revisions
		// Apps with multiple sources pointed at the same Git repository can only have the same revision
		// for all sources because ArgoCD does not support the alternative.
		// We follow ArgoCD shadow-array implementation here that preserves the order of app.spec.sources for
		// the revisions.

		misaligned_sources := make([]string, 0)
		for i, r := range multiSourceRevisions {

			// An empty desired revision means we are not managing the corresponding source.
			if revisions[i] == "" {
				continue
			}

			// Multi-source applications are considered healthy if all the desired source revisions match
			// the source-specific synced revisions.
			if r != revisions[i] {
				msg := fmt.Sprintf(
					"Source %d with RepoURL %v of Application %q in namespace %q does not match the desired revision %q.",
					i,
					app.Spec.Sources[i].RepoURL,
					app.GetName(),
					app.GetNamespace(),
					revisions[i],
				)
				misaligned_sources = append(misaligned_sources, msg)
				logger.Debug(msg)
			}
		}

		if len(misaligned_sources) > 0 {
			msg := fmt.Sprintf("Not all sources of Application %q in namespace %q "+
				"match the desired revisions, assuming unhealthy. Issues: %s",
				app.GetName(), app.GetNamespace(), strings.Join(misaligned_sources[:], " "))
			return kargoapi.HealthStateUnhealthy, errors.New(msg)
		}

		logger.Debug("Found all desired revisions in list of revisions of multi-source application, app is healthy.",
			"desiredRevisions", revisions, "currentRevisions", multiSourceRevisions)
		return kargoapi.HealthStateHealthy, nil
	}
}

// stageHealthForAppHealth returns the v1alpha1.HealthState for an Argo CD
// Application based on its health status.
func stageHealthForAppHealth(ctx context.Context, app *argocd.Application) (kargoapi.HealthState, error) {
	logger := logging.LoggerFromContext(ctx).WithValues("appName", app.GetName())
	switch app.Status.Health.Status {
	case argocd.HealthStatusProgressing, "":
		logger.Debug("Application in progress or health status not set, assuming progressing.")
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
		logger.Debug("Application is healthy.")
		return kargoapi.HealthStateHealthy, nil
	default:
		logger.Debug("Application is unhealthy.", "healthStatus", app.Status.Health.Status)
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
