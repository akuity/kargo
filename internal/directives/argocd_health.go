package directives

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libargocd "github.com/akuity/kargo/internal/argocd"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
)

const applicationStatusesKey = "applicationStatuses"

// ArgoCDHealthConfig is the configuration for a health check to be executed by
// the argocd-update directive.
type ArgoCDHealthConfig struct {
	// Apps is a list health checks to perform on specific Argo CD Applications.
	Apps []ArgoCDAppHealthCheck `json:"apps"`
}

// ArgoCDAppHealthCheck is the configuration for a health check on a single Argo
// CD Application.
type ArgoCDAppHealthCheck struct {
	// Name is the name of the Argo CD Application to check.
	Name string `json:"name"`
	// Namespace is the namespace of the Argo CD Application to check. If empty,
	// the default Argo CD namespace is used.
	Namespace string `json:"namespace,omitempty"`
	// DesiredRevisions is a list of desired revisions for the Argo CD Application
	// to be synced to.
	DesiredRevisions []string `json:"desiredRevisions,omitempty"`
}

// ArgoCDAppStatus describes the current state of a single ArgoCD Application.
type ArgoCDAppStatus struct {
	// Namespace is the namespace of the ArgoCD Application.
	Namespace string
	// Name is the name of the ArgoCD Application.
	Name                     string
	argocd.ApplicationStatus `json:",inline"`
}

// compositeError is an interface for wrapped standard errors produced by
// errors.Join.
type compositeError interface {
	// Unwrap returns the wrapped errors.
	Unwrap() []error
}

// RunHealthCheckStep implements the Directive interface.
func (a *argocdUpdater) RunHealthCheckStep(
	ctx context.Context,
	healthCtx *HealthCheckStepContext,
) HealthCheckStepResult {
	cfg, err := ConfigToStruct[ArgoCDHealthConfig](healthCtx.Config)
	if err != nil {
		return HealthCheckStepResult{
			Status: kargoapi.HealthStateUnknown,
			Issues: []string{
				fmt.Sprintf(
					"could not convert config into %s health check config: %s",
					a.Name(), err.Error(),
				),
			},
		}
	}
	return a.runHealthCheckStep(ctx, healthCtx, cfg)
}

func (a *argocdUpdater) runHealthCheckStep(
	ctx context.Context,
	healthCtx *HealthCheckStepContext,
	healthCfg ArgoCDHealthConfig,
) HealthCheckStepResult {
	if healthCtx.ArgoCDClient == nil {
		return HealthCheckStepResult{
			Status: kargoapi.HealthStateUnknown,
			Issues: []string{
				"Argo CD integration is disabled on this controller; cannot assess " +
					"the health or sync status of Argo CD Applications",
			},
		}
	}
	health := HealthCheckStepResult{
		Status: kargoapi.HealthStateHealthy,
		Issues: make([]string, 0),
	}
	appStatuses := make([]ArgoCDAppStatus, len(healthCfg.Apps))
	for i, appHealthCheck := range healthCfg.Apps {
		namespace := appHealthCheck.Namespace
		if namespace == "" {
			namespace = libargocd.Namespace()
		}
		appStatuses[i] = ArgoCDAppStatus{
			Namespace: namespace,
			Name:      appHealthCheck.Name,
		}
		var state kargoapi.HealthState
		var err error
		state, appStatuses[i], err = a.getApplicationHealth(
			ctx,
			healthCtx,
			client.ObjectKey{
				Namespace: namespace,
				Name:      appHealthCheck.Name,
			},
			appHealthCheck.DesiredRevisions,
		)
		health.Status = health.Status.Merge(state)
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
	health.Output = map[string]any{
		applicationStatusesKey: appStatuses,
	}
	return health
}

// healthErrorConditions are the v1alpha1.ApplicationConditionType conditions
// that indicate an Argo CD Application is unhealthy.
var healthErrorConditions = []argocd.ApplicationConditionType{
	argocd.ApplicationConditionComparisonError,
	argocd.ApplicationConditionInvalidSpecError,
}

// getApplicationHealth assesses the health of an Argo CD Application by looking
// at its conditions, health status, and sync status. Based on these, it returns
// an overall health state, the Argo CD Application's health status, and its sync
// status. If it can not (fully) assess the health of the Argo CD Application, it
// returns an error with a message explaining why.
func (a *argocdUpdater) getApplicationHealth(
	ctx context.Context,
	healthCtx *HealthCheckStepContext,
	appKey client.ObjectKey,
	desiredRevisions []string,
) (kargoapi.HealthState, ArgoCDAppStatus, error) {
	appStatus := ArgoCDAppStatus{
		Namespace: appKey.Namespace,
		Name:      appKey.Name,
		ApplicationStatus: argocd.ApplicationStatus{
			Health: argocd.HealthStatus{
				Status: argocd.HealthStatusUnknown,
			},
			Sync: argocd.SyncStatus{
				Status: argocd.SyncStatusCodeUnknown,
			},
		},
	}
	app := &argocd.Application{}
	if err := healthCtx.ArgoCDClient.Get(ctx, appKey, app); err != nil {
		if kubeerr.IsNotFound(err) {
			err = fmt.Errorf(
				"unable to find Argo CD Application %q in namespace %q",
				appKey.Name, appKey.Namespace,
			)
		} else {
			err = fmt.Errorf(
				"error finding Argo CD Application %q in namespace %q: %w",
				appKey.Name, appKey.Namespace, err,
			)
		}
		return kargoapi.HealthStateUnknown, appStatus, err
	}

	// Reflect the health and sync status of the Argo CD Application.
	appStatus.ApplicationStatus = app.Status

	// Check for any error conditions. If these are found, the application is
	// considered unhealthy as they may indicate a problem which can result in
	// e.g. the health status result to become unreliable.
	if errConditions := a.filterAppConditions(app, healthErrorConditions...); len(errConditions) > 0 {
		issues := make([]error, len(errConditions))
		for i, condition := range errConditions {
			issues[i] = fmt.Errorf(
				"Argo CD Application %q in namespace %q has %q condition: %s",
				appKey.Name,
				appKey.Namespace,
				condition.Type,
				condition.Message,
			)
		}
		return kargoapi.HealthStateUnhealthy, appStatus, errors.Join(issues...)
	}

	if len(desiredRevisions) > 0 {
		if stageHealth, err := a.stageHealthForAppSync(app, desiredRevisions); err != nil {
			return stageHealth, appStatus, err
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
			cooldown := time.Now()
			if !app.Status.OperationState.FinishedAt.IsZero() {
				cooldown = app.Status.OperationState.FinishedAt.Time
			}
			cooldown = cooldown.Add(10 * time.Second)
			if duration := time.Until(cooldown); duration > 0 {
				time.Sleep(duration)
				// Re-fetch the application to get the latest state.
				if err := healthCtx.ArgoCDClient.Get(ctx, appKey, app); err != nil {
					if kubeerr.IsNotFound(err) {
						err = fmt.Errorf(
							"unable to find Argo CD Application %q in namespace %q",
							appKey.Name, appKey.Namespace,
						)
					} else {
						err = fmt.Errorf(
							"error finding Argo CD Application %q in namespace %q: %w",
							appKey.Name, appKey.Namespace, err,
						)
					}
					return kargoapi.HealthStateUnknown, appStatus, err
				}
			}
		}
	}

	// With all the above checks passed, we can now assume the Argo CD
	// Application's health state is reliable.
	stageHealth, err := a.stageHealthForAppHealth(app)
	return stageHealth, appStatus, err
}

// stageHealthForAppSync returns the v1alpha1.HealthState for an Argo CD
// Application based on its sync status.
func (a *argocdUpdater) stageHealthForAppSync(
	app *argocd.Application,
	desiredRevisions []string,
) (kargoapi.HealthState, error) {
	if !slices.ContainsFunc(desiredRevisions, func(rev string) bool { return rev != "" }) {
		// We have no idea what this App should be synced to, so it does not
		// negatively impact Stage health.
		return kargoapi.HealthStateHealthy, nil
	}
	if (app.Operation != nil && app.Operation.Sync != nil) ||
		app.Status.OperationState == nil || app.Status.OperationState.FinishedAt.IsZero() {
		// A sync appears to be in progress
		return kargoapi.HealthStateUnknown, fmt.Errorf(
			"Argo CD Application %q in namespace %q is being synced",
			app.Name, app.Namespace,
		)
	}
	sources := app.Spec.Sources
	if len(sources) == 0 && app.Spec.Source != nil {
		sources = []argocd.ApplicationSource{*app.Spec.Source}
	}
	if len(sources) != len(desiredRevisions) {
		// This really shouldn't happen because the sources would have been
		// consulted when determining the desired revisions.
		return kargoapi.HealthStateUnknown, fmt.Errorf(
			"Argo CD Application %q in namespace %q has %d sources but %d desired revisions",
			app.Name, app.Namespace, len(sources), len(desiredRevisions),
		)
	}
	observedRevisions := app.Status.Sync.Revisions
	if len(observedRevisions) == 0 {
		observedRevisions = []string{app.Status.Sync.Revision}
	}
	if len(observedRevisions) != len(desiredRevisions) {
		// This really shouldn't happen.
		return kargoapi.HealthStateUnknown, fmt.Errorf(
			"Argo CD Application %q in namespace %q has %d observed revisions but %d desired revisions",
			app.Name, app.Namespace, len(observedRevisions), len(desiredRevisions),
		)
	}
	// Aggregate issues for all sources
	issues := make([]string, 0)
	for i, observedRevision := range observedRevisions {
		desiredRevision := desiredRevisions[i]
		if desiredRevision == "" {
			// We have no idea what this source should be synced to, so it does not
			// negatively impact Stage health.
			continue
		}
		if observedRevision != desiredRevision {
			issues = append(
				issues,
				fmt.Sprintf(
					"Source %d with RepoURL %s of Application %q in namespace %q does not match the desired revision %q.",
					i, sources[i].RepoURL, app.Name, app.Namespace, desiredRevision,
				),
			)
		}
	}
	if len(issues) > 0 {
		return kargoapi.HealthStateUnhealthy, fmt.Errorf(
			"Not all sources of Application %q in namespace %q "+
				"are synced to the desired revisions. Issues: %s",
			app.Name, app.Namespace, strings.Join(issues, "; "),
		)
	}
	return kargoapi.HealthStateHealthy, nil
}

// stageHealthForAppHealth returns the v1alpha1.HealthState for an Argo CD
// Application based on its health status.
func (a *argocdUpdater) stageHealthForAppHealth(
	app *argocd.Application,
) (kargoapi.HealthState, error) {
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
func (a *argocdUpdater) filterAppConditions(
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
