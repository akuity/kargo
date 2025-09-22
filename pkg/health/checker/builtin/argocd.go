package builtin

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libargocd "github.com/akuity/kargo/pkg/argocd"
	argocd "github.com/akuity/kargo/pkg/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/pkg/health"
)

const applicationStatusesKey = "applicationStatuses"

var appHealthCooldownDuration = 10 * time.Second

// ArgoCDHealthInput is the input for a health check associated with the the
// argocd-update step.
type ArgoCDHealthInput struct {
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

type argocdChecker struct {
	argocdClient client.Client
}

// newArgocdChecker returns a implementation of the Checker interface that
// monitors the health and sync state of Argo CD Application resources.
func newArgocdChecker(argocdClient client.Client) *argocdChecker {
	return &argocdChecker{
		argocdClient: argocdClient,
	}
}

// Name implements the Checker interface.
func (a *argocdChecker) Name() string {
	// Note: The promotion.StepRunner for the argocd-update step has historically
	// registered a health check with the same name, so we continue to do that for
	// backwards compatibility, but newer Checkers need not follow this convention
	// of promotion.StepRunner and Checker names matching.
	return "argocd-update"
}

// Check implements the Checker interface.
func (a *argocdChecker) Check(
	ctx context.Context,
	_ string,
	_ string,
	criteria health.Criteria,
) health.Result {
	cfg, err := health.InputToStruct[ArgoCDHealthInput](criteria.Input)
	if err != nil {
		return health.Result{
			Status: kargoapi.HealthStateUnknown,
			Issues: []string{
				fmt.Sprintf(
					"could not convert opaque input into %s health check input: %s",
					a.Name(), err.Error(),
				),
			},
		}
	}
	return a.check(ctx, cfg)
}

func (a *argocdChecker) check(
	ctx context.Context,
	input ArgoCDHealthInput,
) health.Result {
	if a.argocdClient == nil {
		return health.Result{
			Status: kargoapi.HealthStateUnknown,
			Issues: []string{
				"Argo CD integration is disabled on this controller; cannot assess " +
					"the health or sync status of Argo CD Applications",
			},
		}
	}
	res := health.Result{
		Status: kargoapi.HealthStateHealthy,
		Issues: make([]string, 0),
	}
	appStatuses := make([]ArgoCDAppStatus, len(input.Apps))
	for i, appHealthCheck := range input.Apps {
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
			client.ObjectKey{
				Namespace: namespace,
				Name:      appHealthCheck.Name,
			},
			appHealthCheck.DesiredRevisions,
		)
		res.Status = res.Status.Merge(state)
		if err != nil {
			if cErr, ok := err.(compositeError); ok {
				for _, e := range cErr.Unwrap() {
					res.Issues = append(res.Issues, e.Error())
				}
			} else {
				res.Issues = append(res.Issues, err.Error())
			}
		}
	}
	res.Output = map[string]any{
		applicationStatusesKey: appStatuses,
	}
	return res
}

// healthErrorConditions are the v1alpha1.ApplicationConditionType conditions
// that indicate an Argo CD Application is unhealthy.
var healthErrorConditions = []argocd.ApplicationConditionType{
	argocd.ApplicationConditionComparisonError,
	argocd.ApplicationConditionInvalidSpecError,
}

// getApplicationHealth assesses the health of an Argo CD Application by looking
// at its conditions, health status, and sync status. Based on these, it returns
// an overall health state and the Argo CD Application's health status. If it
// can not (fully) assess the health of the Argo CD Application, it returns an
// error with a message explaining why.
func (a *argocdChecker) getApplicationHealth(
	ctx context.Context,
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
	if err := a.argocdClient.Get(ctx, appKey, app); err != nil {
		if apierrors.IsNotFound(err) {
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

	if appStatus.OperationState != nil && appStatus.OperationState.Phase != argocd.OperationSucceeded {
		// This App is in a transitional state. By Kargo Standards, the Stage cannot
		// be Healthy.
		return kargoapi.HealthStateUnknown,
			appStatus,
			fmt.Errorf(
				"last operation of Argo CD Application %q in namespace %q has "+
					"status %q; Application health status not trusted",
				appKey.Name,
				appKey.Namespace,
				appStatus.OperationState.Phase,
			)
	}

	// Argo CD has separate reconciliation loops for operations (like syncing) and
	// assessing App health. This means that in the moments immediately following
	// a completed operation, App health may reflect state from PRIOR to the
	// operation. If that status indicates the App is in a Healthy state, but
	// would have indicated otherwise had it not been stale, it creates the
	// possibility of an overly-optimistic assessment of Stage health. If that
	// occurs following a Promotion, it can prompt a verification process to kick
	// of prematurely.
	//
	// To work around this, we will not immediately trust App health if the most
	// recently completed operation completed fewer than ten seconds ago. In such
	// a case, we will deem Stage health Unknown and return an error. This will
	// cause the Stage to be queued for re-reconciliation with a progressive
	// backoff. This will continue until the App's health status is considered
	// reliable.
	//
	// TODO(krancour): This workaround can be revisited if/when
	// https://github.com/argoproj/argo-cd/pull/21120 is merged, as (for newer
	// versions of Argo CD, at least) it will allow us to accurately determine
	// whether an App's health was last assessed before or after its most recent
	// operation completed.
	if app.Status.OperationState != nil && app.Status.OperationState.FinishedAt != nil {
		if time.Since(app.Status.OperationState.FinishedAt.Time) < appHealthCooldownDuration {
			return kargoapi.HealthStateUnknown,
				appStatus,
				fmt.Errorf(
					"last operation of Argo CD Application %q in namespace %q completed "+
						"less than %s ago; Application health status not trusted",
					appKey.Name,
					appKey.Namespace,
					appHealthCooldownDuration,
				)
		}
	}

	// If we get to here, we assume the Argo CD Application's health state is
	// reliable.

	// Check for any error conditions. If these are found, the application is
	// considered unhealthy as they may indicate a problem which can result in
	// e.g. the health status result to become unreliable.
	if errConditions := a.filterAppConditions(app, healthErrorConditions...); len(errConditions) > 0 {
		issues := make([]error, len(errConditions))
		for i, condition := range errConditions {
			// nolint:staticcheck
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

	stageHealth, err := a.stageHealthForAppHealth(app)
	if err != nil || stageHealth != kargoapi.HealthStateHealthy {
		// If there was an error or the App is not Healthy, we're done.
		return stageHealth, appStatus, err
	}

	// If we get to here, the App is Healthy and, so far, the Stage appears to be
	// as well. If desiredRevisions are known, however, this needs to be factored
	// into Stage health. Assess how App sources being synced to the correct or
	// incorrect revisions affects overall Stage health:
	stageHealth, err = a.stageHealthForAppSync(app, desiredRevisions)
	return stageHealth, appStatus, err
}

// stageHealthForAppSync returns the v1alpha1.HealthState for an Argo CD
// Application based on its sync status.
func (a *argocdChecker) stageHealthForAppSync(
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
		// nolint:staticcheck
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
		// nolint:staticcheck
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
		// nolint:staticcheck
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
		// nolint:staticcheck
		return kargoapi.HealthStateUnhealthy, fmt.Errorf(
			"Not all sources of Application %q in namespace %q "+
				"are synced to the desired revisions. Issues: %s",
			app.Name, app.Namespace, strings.Join(issues, "; "),
		)
	}
	return kargoapi.HealthStateHealthy, nil
}

// stageHealthForAppHealth assesses how the specified Argo CD Application's
// health affects Stage heathy. All results apart from Healthy will also include
// an error.
func (a *argocdChecker) stageHealthForAppHealth(
	app *argocd.Application,
) (kargoapi.HealthState, error) {
	switch app.Status.Health.Status {
	case argocd.HealthStatusProgressing, "":
		// nolint:staticcheck
		err := fmt.Errorf(
			"Argo CD Application %q in namespace %q is progressing",
			app.GetName(),
			app.GetNamespace(),
		)
		return kargoapi.HealthStateProgressing, err
	case argocd.HealthStatusSuspended:
		// nolint:staticcheck
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
		// nolint:staticcheck
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
func (a *argocdChecker) filterAppConditions(
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
