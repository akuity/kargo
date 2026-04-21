package builtin

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/xeipuuv/gojsonschema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libargocd "github.com/akuity/kargo/pkg/argocd"
	argocd "github.com/akuity/kargo/pkg/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const (
	stepKindArgoCDWait = "argocd-wait"

	// healthStatusKey is the key used to store per-app health statuses in
	// step output. Used for degradation detection across re-invocations.
	healthStatusKey = "healthStatus"
)

// defaultWaitFor is the default set of conditions to wait for, matching the
// behavior of `argocd app wait` when no flags are specified.
var defaultWaitFor = []builtin.WaitFor{builtin.Health, builtin.Sync, builtin.Operation}

// healthErrorConditions are the ApplicationConditionType conditions that
// indicate an Argo CD Application has a configuration error.
var waitHealthErrorConditions = []argocd.ApplicationConditionType{
	argocd.ApplicationConditionComparisonError,
	argocd.ApplicationConditionInvalidSpecError,
}

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name: stepKindArgoCDWait,
			Metadata: promotion.StepRunnerMetadata{
				DefaultTimeout: 5 * time.Minute,
				RequiredCapabilities: []promotion.StepRunnerCapability{
					promotion.StepCapabilityAccessArgoCD,
				},
			},
			Value: newArgocdWaiter,
		},
	)
}

// argocdWaiter is an implementation of the promotion.StepRunner interface that
// waits for one or more Argo CD Applications to reach desired statuses.
type argocdWaiter struct {
	schemaLoader gojsonschema.JSONLoader
	argocdClient client.Client

	// These behaviors are overridable for testing purposes:

	getApplicationsFn func(
		ctx context.Context,
		argocdClient client.Client,
		name string,
		namespace string,
		selector *builtin.ArgoCDAppSelector,
	) ([]*argocd.Application, error)

	checkAppReadinessFn func(
		ctx context.Context,
		app *argocd.Application,
		waitFor []builtin.WaitFor,
		prevHealthStatus string,
	) (ready bool, healthStatus string, err error)
}

// newArgocdWaiter returns an implementation of the promotion.StepRunner
// interface that waits for Argo CD Applications to reach desired statuses.
func newArgocdWaiter(caps promotion.StepRunnerCapabilities) promotion.StepRunner {
	w := &argocdWaiter{argocdClient: caps.ArgoCDClient}
	w.schemaLoader = getConfigSchemaLoader(stepKindArgoCDWait)
	w.getApplicationsFn = getArgoCDApplications
	w.checkAppReadinessFn = w.checkAppReadiness
	return w
}

// Run implements the promotion.StepRunner interface.
func (w *argocdWaiter) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := w.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return w.run(ctx, stepCtx, cfg)
}

// convert validates argocdWaiter configuration against a JSON schema and
// converts it into a builtin.ArgoCDWaitConfig struct.
func (w *argocdWaiter) convert(cfg promotion.Config) (builtin.ArgoCDWaitConfig, error) {
	return validateAndConvert[builtin.ArgoCDWaitConfig](w.schemaLoader, cfg, stepKindArgoCDWait)
}

func (w *argocdWaiter) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	stepCfg builtin.ArgoCDWaitConfig,
) (promotion.StepResult, error) {
	if w.argocdClient == nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			errors.New( // nolint:staticcheck
				"Argo CD integration is disabled on this controller; cannot " +
					"wait for Argo CD Application resources",
			)
	}

	logger := logging.LoggerFromContext(ctx)
	logger.Info("executing argocd-wait promotion step")

	// Load previous health statuses from shared state (for degradation
	// detection across re-invocations).
	prevHealthStatuses := w.loadPreviousHealthStatuses(stepCtx)
	// Seed with previous statuses so unchecked apps retain their last-known
	// health if we bail early (e.g. on a TerminalError for another app).
	newHealthStatuses := make(map[string]any, len(prevHealthStatuses))
	for k, v := range prevHealthStatuses {
		newHealthStatuses[k] = v
	}

	allReady := true
	for i := range stepCfg.Apps {
		appSpec := &stepCfg.Apps[i]

		// Apply defaults.
		waitFor := appSpec.WaitFor
		if len(waitFor) == 0 {
			waitFor = defaultWaitFor
		}

		// Resolve matching applications.
		apps, err := w.getApplicationsFn(
			ctx,
			w.argocdClient,
			appSpec.Name,
			appSpec.Namespace,
			appSpec.Selector,
		)
		if err != nil {
			return promotion.StepResult{
				Status: kargoapi.PromotionStepStatusErrored,
			}, err
		}

		for _, app := range apps {
			appKey := fmt.Sprintf("%s/%s", app.Namespace, app.Name)
			appLogger := logger.WithValues(
				"app", app.Name, "namespace", app.Namespace,
			)

			ready, healthStatus, err :=
				w.checkAppReadinessFn(ctx, app, waitFor, prevHealthStatuses[appKey])
			if healthStatus != "" {
				newHealthStatuses[appKey] = healthStatus
			}
			if err != nil {
				return promotion.StepResult{
					Status: kargoapi.PromotionStepStatusErrored,
					Output: map[string]any{
						healthStatusKey: newHealthStatuses,
					},
				}, err
			}
			if !ready {
				appLogger.Info("application is not ready")
				allReady = false
			} else {
				appLogger.Info("application is ready")
			}
		}
	}

	if allReady {
		logger.Info("all applications are ready", "status", "Succeeded")
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusSucceeded,
			Output: map[string]any{
				healthStatusKey: newHealthStatuses,
			},
		}, nil
	}

	logger.Info("waiting for applications to become ready")
	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusRunning,
		Output: map[string]any{
			healthStatusKey: newHealthStatuses,
		},
	}, nil
}

// checkAppReadiness checks whether a single Argo CD Application meets the
// desired conditions specified by waitFor. It returns:
//   - ready: whether all conditions are met
//   - healthStatus: the current health status string (for tracking)
//   - err: a TerminalError if a non-recoverable condition is detected
func (w *argocdWaiter) checkAppReadiness(
	ctx context.Context,
	app *argocd.Application,
	waitFor []builtin.WaitFor,
	prevHealthStatus string,
) (ready bool, healthStatus string, err error) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"app", app.Name, "namespace", app.Namespace,
	)
	healthStatus = string(app.Status.Health.Status)

	// Check for error conditions on the Application.
	if errConditions := filterArgoCDAppConditions(
		app, waitHealthErrorConditions...,
	); len(errConditions) > 0 {
		issues := make([]error, len(errConditions))
		for i, condition := range errConditions {
			issues[i] = fmt.Errorf( // nolint:staticcheck
				"Argo CD Application %q in namespace %q has %q condition: %s",
				app.Name, app.Namespace, condition.Type, condition.Message,
			)
		}
		return false, healthStatus, &promotion.TerminalError{
			Err: errors.Join(issues...),
		}
	}

	// Operation check.
	if slices.Contains(waitFor, "operation") {
		if operationInProgress(app) {
			logger.Info("operation is in progress")
			return false, healthStatus, nil
		}
	}

	healthBeingChecked := slices.Contains(waitFor, "health") ||
		slices.Contains(waitFor, "suspended") ||
		slices.Contains(waitFor, "degraded")

	// Argo CD executes operations and assesses health in separate reconciliation
	// loops. Immediately after an operation completes, the health status may
	// reflect state from prior to the operation. If app.Status.ReconciledAt is
	// at or after the operation's FinishedAt, health was assessed after the
	// operation completed and can be trusted. Otherwise, request a hard refresh
	// so Argo CD will reconcile and update ReconciledAt; return not-ready until
	// that occurs.
	if healthBeingChecked {
		if app.Status.OperationState != nil &&
			app.Status.OperationState.FinishedAt != nil &&
			(app.Status.ReconciledAt == nil || app.Status.ReconciledAt.Before(app.Status.OperationState.FinishedAt)) {
			libargocd.RequestAppRefresh(ctx, w.argocdClient, app)
			logger.Info("application not yet reconciled after last operation, " +
				"health status not trusted")
			return false, healthStatus, nil
		}
	}

	// Health check: health, suspended, degraded are OR'd.
	if healthBeingChecked {
		healthCheckPassed := false
		if slices.Contains(waitFor, "health") {
			healthCheckPassed = healthCheckPassed ||
				app.Status.Health.Status == argocd.HealthStatusHealthy
		}
		if slices.Contains(waitFor, "suspended") {
			healthCheckPassed = healthCheckPassed ||
				app.Status.Health.Status == argocd.HealthStatusSuspended
		}
		if slices.Contains(waitFor, "degraded") {
			healthCheckPassed = healthCheckPassed ||
				app.Status.Health.Status == argocd.HealthStatusDegraded
		}

		// Degradation detection: if waiting for health and the app transitions
		// TO Degraded from a non-Degraded/non-Unknown state, fail immediately.
		if slices.Contains(waitFor, "health") &&
			app.Status.Health.Status == argocd.HealthStatusDegraded &&
			prevHealthStatus != "" &&
			prevHealthStatus != string(argocd.HealthStatusDegraded) &&
			prevHealthStatus != string(argocd.HealthStatusUnknown) {
			return false, healthStatus, &promotion.TerminalError{
				Err: fmt.Errorf( // nolint:staticcheck
					"Argo CD Application %q in namespace %q health has "+
						"regressed from %s to %s",
					app.Name, app.Namespace,
					prevHealthStatus, app.Status.Health.Status,
				),
			}
		}

		if !healthCheckPassed {
			logger.Info(
				"health check not passed",
				"currentHealth", app.Status.Health.Status,
			)
			return false, healthStatus, nil
		}
	}

	// Sync check.
	if slices.Contains(waitFor, "sync") {
		if app.Status.Sync.Status != argocd.SyncStatusCodeSynced {
			logger.Info(
				"sync check not passed",
				"currentSync", app.Status.Sync.Status,
			)
			return false, healthStatus, nil
		}
	}

	return true, healthStatus, nil
}

// operationInProgress returns true if the Application has an operation that is
// still running.
func operationInProgress(app *argocd.Application) bool {
	// Active operation request.
	if app.Operation != nil {
		return true
	}
	if app.Status.OperationState == nil {
		return false
	}
	// Operation not yet finished.
	return app.Status.OperationState.FinishedAt == nil
}

// loadPreviousHealthStatuses loads the previous health statuses from the step's
// shared state output. Returns an empty map if not available.
func (w *argocdWaiter) loadPreviousHealthStatuses(
	stepCtx *promotion.StepContext,
) map[string]string {
	result := make(map[string]string)
	prevOutput, ok := stepCtx.SharedState[stepCtx.Alias]
	if !ok {
		return result
	}
	outputMap, ok := prevOutput.(map[string]any)
	if !ok {
		return result
	}
	statuses, ok := outputMap[healthStatusKey]
	if !ok {
		return result
	}
	statusMap, ok := statuses.(map[string]any)
	if !ok {
		return result
	}
	for k, v := range statusMap {
		if s, ok := v.(string); ok {
			result[k] = s
		}
	}
	return result
}

// getArgoCDApplications resolves Argo CD Applications by name or label
// selector.
func getArgoCDApplications(
	ctx context.Context,
	argocdClient client.Client,
	name string,
	namespace string,
	selector *builtin.ArgoCDAppSelector,
) ([]*argocd.Application, error) {
	if namespace == "" {
		namespace = libargocd.Namespace()
	}

	if selector != nil {
		labelSelector, err := buildArgoCDAppLabelSelector(selector)
		if err != nil {
			return nil, fmt.Errorf("error building label selector: %w", err)
		}

		appList := &argocd.ApplicationList{}
		if err = argocdClient.List(ctx, appList,
			client.InNamespace(namespace),
			client.MatchingLabelsSelector{Selector: labelSelector},
		); err != nil {
			return nil, fmt.Errorf(
				"error listing Argo CD Applications matching selector: %w", err,
			)
		}
		if len(appList.Items) == 0 {
			return nil, fmt.Errorf(
				"no Argo CD Applications found matching selector in namespace %q",
				namespace,
			)
		}
		apps := make([]*argocd.Application, len(appList.Items))
		for i := range appList.Items {
			apps[i] = &appList.Items[i]
		}
		return apps, nil
	}

	app, err := argocd.GetApplication(ctx, argocdClient, namespace, name)
	if err != nil {
		return nil, fmt.Errorf(
			"error finding Argo CD Application %q in namespace %q: %w",
			name, namespace, err,
		)
	}
	if app == nil {
		return nil, fmt.Errorf(
			"unable to find Argo CD Application %q in namespace %q",
			name, namespace,
		)
	}
	return []*argocd.Application{app}, nil
}

// filterArgoCDAppConditions returns a slice of ApplicationCondition that match
// the provided types.
func filterArgoCDAppConditions(
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
