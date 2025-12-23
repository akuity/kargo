package builtin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xeipuuv/gojsonschema"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libargocd "github.com/akuity/kargo/pkg/argocd"
	argocd "github.com/akuity/kargo/pkg/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/pkg/health"
	checkers "github.com/akuity/kargo/pkg/health/checker/builtin"
	"github.com/akuity/kargo/pkg/kubeclient"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/urls"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const (
	stepKindArgoCDUpdate = "argocd-update"

	applicationOperationInitiator = "kargo-controller"
	promotionInfoKey              = "kargo.akuity.io/promotion"
)

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name: stepKindArgoCDUpdate,
			Metadata: promotion.StepRunnerMetadata{
				DefaultTimeout: 5 * time.Minute,
				RequiredCapabilities: []promotion.StepRunnerCapability{
					promotion.StepCapabilityAccessArgoCD,
				},
			},
			Value: newArgocdUpdater,
		},
	)
}

// argocdUpdater is an implementation of the promotion.StepRunner interface that
// updates one or more Argo CD Application resources.
type argocdUpdater struct {
	schemaLoader gojsonschema.JSONLoader

	argocdClient client.Client

	// These behaviors are overridable for testing purposes:

	getAuthorizedApplicationsFn func(
		context.Context,
		*promotion.StepContext,
		*builtin.ArgoCDAppUpdate,
	) ([]*argocd.Application, error)

	buildLabelSelectorFn func(
		*builtin.ArgoCDAppSelector,
	) (labels.Selector, error)

	buildDesiredSourcesFn func(
		update *builtin.ArgoCDAppUpdate,
		desiredRevisions []string,
		app *argocd.Application,
	) (argocd.ApplicationSources, error)

	mustPerformUpdateFn func(
		context.Context,
		*promotion.StepContext,
		*builtin.ArgoCDAppUpdate,
		*argocd.Application,
	) (argocd.OperationPhase, bool, error)

	syncApplicationFn func(
		ctx context.Context,
		stepCtx *promotion.StepContext,
		app *argocd.Application,
		desiredSources argocd.ApplicationSources,
	) error

	applyArgoCDSourceUpdateFn func(
		update *builtin.ArgoCDAppSourceUpdate,
		desiredRevision string,
		src argocd.ApplicationSource,
	) (argocd.ApplicationSource, bool)

	argoCDAppPatchFn func(
		context.Context,
		kubeclient.ObjectWithKind,
		kubeclient.UnstructuredPatchFn,
	) error

	logAppEventFn func(
		ctx context.Context,
		app *argocd.Application,
		user string,
		reason string,
		message string,
	)
}

// newArgocdUpdater returns a implementation of the promotion.StepRunner
// interfaces that updates Argo CD Application resources.
func newArgocdUpdater(caps promotion.StepRunnerCapabilities) promotion.StepRunner {
	r := &argocdUpdater{argocdClient: caps.ArgoCDClient}
	r.schemaLoader = getConfigSchemaLoader(stepKindArgoCDUpdate)
	r.getAuthorizedApplicationsFn = r.getAuthorizedApplications
	r.buildLabelSelectorFn = r.buildLabelSelector
	r.buildDesiredSourcesFn = r.buildDesiredSources
	r.mustPerformUpdateFn = r.mustPerformUpdate
	r.syncApplicationFn = r.syncApplication
	r.applyArgoCDSourceUpdateFn = r.applyArgoCDSourceUpdate
	r.argoCDAppPatchFn = r.argoCDAppPatch
	r.logAppEventFn = r.logAppEvent
	return r
}

// Run implements the promotion.StepRunner interface.
func (a *argocdUpdater) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := a.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return a.run(ctx, stepCtx, cfg)
}

// convert validates argocdUpdater configuration against a JSON schema and
// converts it into a builtin.ArgoCDUpdateConfig struct.
func (a *argocdUpdater) convert(cfg promotion.Config) (builtin.ArgoCDUpdateConfig, error) {
	return validateAndConvert[builtin.ArgoCDUpdateConfig](a.schemaLoader, cfg, stepKindArgoCDUpdate)
}

func (a *argocdUpdater) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	stepCfg builtin.ArgoCDUpdateConfig,
) (promotion.StepResult, error) {
	if a.argocdClient == nil {
		// nolint:staticcheck
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, errors.New(
			"Argo CD integration is disabled on this controller; cannot update " +
				"Argo CD Application resources",
		)
	}

	logger := logging.LoggerFromContext(ctx)
	logger.Info("executing argocd-update promotion step")

	updateResults := make([]argocd.OperationPhase, 0, len(stepCfg.Apps))
	var appHealthChecks []checkers.ArgoCDAppHealthCheck
	for i := range stepCfg.Apps {
		update := &stepCfg.Apps[i]

		// Retrieve the Argo CD Application(s) matching the update specification.
		apps, err := a.getAuthorizedApplicationsFn(ctx, stepCtx, update)
		if err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
		}

		// Log the number of applications matched when using selectors.
		if update.Selector != nil && len(apps) > 0 {
			logger.Info(
				"found Applications matching selector",
				"count", len(apps),
				"namespace", update.Namespace,
			)
		}

		// If we found multiple Applications, and we are instructed to update
		// their sources, ensure the updates are applicable to all.
		if len(update.Sources) > 0 {
			logger.Info(
				"validating source updates are applicable to all Applications",
				"count", len(apps),
			)
			if err := a.validateSourceUpdatesApplicable(update, apps); err != nil {
				return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
			}
			logger.Info("validation successful for all Applications")
		}

		// Process each matched application.
		for _, app := range apps {
			desiredRevisions := a.getDesiredRevisions(update, app)
			appHealthChecks = append(appHealthChecks, checkers.ArgoCDAppHealthCheck{
				Name:             app.Name,
				Namespace:        app.Namespace,
				DesiredRevisions: desiredRevisions,
			})

			// Process the application and get its phase.
			phase, err := a.processApplication(ctx, stepCtx, update, app)
			if err != nil {
				return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
			}

			// If the phase indicates failure, fail-fast.
			if phase.Failed() {
				return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, nil
			}

			// If we have a phase, append it to the results.
			if phase != "" {
				updateResults = append(updateResults, phase)
			}
		}
	}

	aggregatedStatus := a.operationPhaseToPromotionStepStatus(updateResults...)
	if aggregatedStatus == "" {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, fmt.Errorf(
			"could not determine promotion step status from operation phases: %v",
			updateResults,
		)
	}

	logger.Info(
		"done executing argocd-update promotion step",
		"status", aggregatedStatus,
	)

	// TODO(krancour): This enables more aggressive polling while waiting to
	// observe the Application has successfully synced. This is a workaround for
	// an as-yet-unexplained, but rare phenomenon where Application status change
	// events do not seem to be promptly triggering re-reconciliation of the
	// Promotion resources.
	var retryAfter *time.Duration
	if aggregatedStatus == kargoapi.PromotionStepStatusRunning {
		retryAfter = ptr.To(30 * time.Second)
		logger.Info("step to be retried", "interval", retryAfter)
	}

	return promotion.StepResult{
		Status: aggregatedStatus,
		HealthCheck: &health.Criteria{
			Kind: stepKindArgoCDUpdate,
			Input: health.Input{
				"apps": appHealthChecks,
			},
		},
		RetryAfter: retryAfter,
	}, nil
}

// validateSourceUpdatesApplicable validates that all source updates can be
// applied to all selected Applications before any updates are performed. This
// prevents partial updates due to validation failures (e.g., if only some apps
// have sources matching the update).
func (a *argocdUpdater) validateSourceUpdatesApplicable(
	update *builtin.ArgoCDAppUpdate,
	apps []*argocd.Application,
) error {
	if len(apps) <= 1 {
		return nil
	}

	const maxValidationErrorsToReport = 3
	var validationErrors []error

	for _, app := range apps {
		desiredRevisions := a.getDesiredRevisions(update, app)
		if _, err := a.buildDesiredSourcesFn(update, desiredRevisions, app); err != nil {
			// nolint:staticcheck
			validationErrors = append(validationErrors, fmt.Errorf(
				"Application %q in namespace %q: %w",
				app.Name, app.Namespace, err,
			))
		}
	}

	if len(validationErrors) > 0 {
		reportedErrors := validationErrors
		errorMessage := "selected Applications must have compatible sources; " +
			"%d incompatible. No Applications were updated:\n%w"

		if len(validationErrors) > maxValidationErrorsToReport {
			reportedErrors = validationErrors[:maxValidationErrorsToReport]
			errorMessage = "selected Applications must have compatible sources; " +
				"%d incompatible (showing first %d). No Applications were updated:\n%w"
			return fmt.Errorf(
				errorMessage,
				len(validationErrors),
				maxValidationErrorsToReport,
				errors.Join(reportedErrors...),
			)
		}

		return fmt.Errorf(
			errorMessage,
			len(validationErrors),
			errors.Join(reportedErrors...),
		)
	}

	return nil
}

// processApplication handles the update logic for a single Argo CD Application.
// It returns the operation phase (if any) and an error if processing failed.
func (a *argocdUpdater) processApplication(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	update *builtin.ArgoCDAppUpdate,
	app *argocd.Application,
) (argocd.OperationPhase, error) {
	logger := logging.LoggerFromContext(ctx).WithValues("app", app.Name, "namespace", app.Namespace)

	// Check if the update needs to be performed and retrieve its phase.
	phase, mustUpdate, err := a.mustPerformUpdateFn(ctx, stepCtx, update, app)
	if mustUpdate {
		logger.Info("Argo CD Application requires update")
	} else {
		logger.Info("Argo CD Application does not require update")
	}

	// If we don't need to perform an update, further processing depends on
	// the phase and whether an error occurred.
	if !mustUpdate {
		if err != nil {
			if phase == "" {
				// If we do not have a phase, we cannot continue processing
				// this update by waiting.
				return "", err
			}
			// Log the error for observability but continue processing other
			// updates.
			logger.Info("Argo CD Application update cannot be performed", "reason", err.Error())
		}
		if phase.Failed() {
			// Record the reason for the failure if available.
			if app.Status.OperationState != nil {
				// nolint:staticcheck
				return "", fmt.Errorf(
					"Argo CD Application %q in namespace %q failed with: %s",
					app.Name,
					app.Namespace,
					app.Status.OperationState.Message,
				)
			}
			// If the update failed, we can short-circuit. This is
			// effectively "fail fast" behavior.
			return phase, nil
		}
		// Return the phase without error to indicate we should continue
		return phase, nil
	}

	// Log the error, as it contains information about why we need to
	// perform an update.
	if err != nil {
		logger.Info("performing update of Argo CD Application", "reason", err.Error())
	}

	// Build the desired source(s) for the Argo CD Application.
	desiredRevisions := a.getDesiredRevisions(update, app)
	desiredSources, err := a.buildDesiredSourcesFn(
		update,
		desiredRevisions,
		app,
	)
	if err != nil {
		return "", fmt.Errorf(
			"error building desired sources for Argo CD Application %q in namespace %q: %w",
			app.Name, app.Namespace, err,
		)
	}

	// Perform the update.
	if err = a.syncApplicationFn(
		ctx,
		stepCtx,
		app,
		desiredSources,
	); err != nil {
		return "", fmt.Errorf(
			"error syncing Argo CD Application %q in namespace %q: %w",
			app.Name, app.Namespace, err,
		)
	}

	// As we have initiated an update, we should wait for it to complete.
	return argocd.OperationRunning, nil
}

// buildDesiredSources returns the desired source(s) for an Argo CD Application,
// by updating the current source(s) with the given source updates.
func (a *argocdUpdater) buildDesiredSources(
	update *builtin.ArgoCDAppUpdate,
	desiredRevisions []string,
	app *argocd.Application,
) (argocd.ApplicationSources, error) {
	desiredSources := app.Spec.Sources.DeepCopy()
	if len(desiredSources) == 0 && app.Spec.Source != nil {
		desiredSources = []argocd.ApplicationSource{*app.Spec.Source.DeepCopy()}
	}
	if len(desiredSources) != len(desiredRevisions) {
		// This really shouldn't happen.
		// nolint:staticcheck
		return nil, fmt.Errorf(
			"Argo CD Application %q in namespace %q has %d sources but %d desired revisions",
			app.Name, app.Namespace, len(desiredSources), len(desiredRevisions),
		)
	}
updateLoop:
	for i := range update.Sources {
		srcUpdate := &update.Sources[i]
		var updateUsed bool
		for j := range desiredSources {
			desiredSources[j], updateUsed = a.applyArgoCDSourceUpdateFn(
				srcUpdate,
				desiredRevisions[j],
				desiredSources[j],
			)
			if updateUsed {
				continue updateLoop
			}
		}

		// If we get here, the update was not applied to any source
		if srcUpdate.Chart == "" {
			return nil, fmt.Errorf(
				"no source of Argo CD Application %q in namespace %q matched update "+
					"for source with repoURL %s",
				app.Name, app.Namespace, srcUpdate.RepoURL,
			)
		}
		return nil, fmt.Errorf(
			"no source of Argo CD Application %q in namespace %q matched update "+
				"for source with repoURL %s and chart %q",
			app.Name, app.Namespace, srcUpdate.RepoURL, srcUpdate.Chart,
		)
	}
	return desiredSources, nil
}

func (a *argocdUpdater) mustPerformUpdate(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	update *builtin.ArgoCDAppUpdate,
	app *argocd.Application,
) (phase argocd.OperationPhase, mustUpdate bool, err error) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"app", app.Name, "namespace", app.Namespace,
	)

	status := app.Status.OperationState
	if status == nil {
		// The application has no operation.
		logger.Info("no current operation found")
		return "", true, nil
	}

	// Deal with the possibility that the operation was not initiated by Kargo
	if !isKargoInitiatedOperation(status.Operation) {
		logger.Info(
			"current operation was not initiated by Kargo",
			"initiatedBy", status.Operation.InitiatedBy.Username,
		)
		if !status.Phase.Completed() {
			logger.Info("waiting for current operation to complete")
			// We should wait for the operation to complete before attempting to
			// apply an update ourselves.
			// NB: We return the current phase here because we want the caller
			//     to know that an operation is still running.
			return status.Phase, false, fmt.Errorf(
				"current operation was initiated by %q: waiting for operation to complete",
				status.Operation.InitiatedBy.Username,
			)
		}
		// Initiate our own operation.
		logger.Info("current operation is complete; can start a new one")
		return "", true, nil
	}

	logger.Info("current operation was initiated by Kargo")

	// Deal with the possibility that the operation was not initiated for the
	// current freight collection. i.e. Not related to the current promotion.
	var correctPromotionIDFound bool
	for _, info := range status.Operation.Info {
		if info.Name == promotionInfoKey {
			correctPromotionIDFound = info.Value == stepCtx.Promotion
			break
		}
	}
	if !correctPromotionIDFound {
		logger.Info("current operation was not initiated for this promotion")
		// The operation was not initiated for the current Promotion.
		if !status.Phase.Completed() {
			logger.Info("waiting for current operation to complete")
			// We should wait for the operation to complete before attempting to
			// apply an update ourselves.
			// NB: We return the current phase here because we want the caller
			//     to know that an operation is still running.
			return status.Phase, false, fmt.Errorf(
				"current operation was not initiated for Promotion %s: waiting for operation to complete",
				stepCtx.Promotion,
			)
		}
		// Initiate our own operation.
		logger.Info("current operation is complete; can start a new one")
		return "", true, nil
	}

	if !status.Phase.Completed() {
		// The operation is still running.
		logger.Info("waiting for current operation to complete")
		return status.Phase, false, nil
	}

	logger.Info("current operation is complete")

	if status.SyncResult == nil {
		logger.Info("no sync result found")
		// We do not have a sync result, so we cannot determine if the operation
		// was successful. The best recourse is to retry the operation.
		return "", true, errors.New("operation completed without a sync result")
	}

	// Check if the desired revisions were applied.
	desiredRevisions := a.getDesiredRevisions(update, app)
	if len(desiredRevisions) == 0 {
		// We do not have any desired revisions, so we cannot determine if the
		// operation was successful.
		logger.Info("no desired revisions specified")
		return status.Phase, false, nil
	}

	logger.Info("desired revisions were specified for some sources")

	observedRevisions := status.SyncResult.Revisions
	if len(observedRevisions) == 0 {
		observedRevisions = []string{status.SyncResult.Revision}
	}
	for i, observedRevision := range observedRevisions {
		desiredRevision := desiredRevisions[i]
		if desiredRevision == "" {
			continue
		}
		if observedRevision != desiredRevision {
			logger.Info(
				"sync result revision does not match desired revision",
				"sourceIndex", i,
			)
			return "", true, fmt.Errorf(
				"sync result revisions %v do not match desired revisions %v",
				observedRevisions, desiredRevisions,
			)
		}
	}

	logger.Info("desired revisions were observably applied")

	// The operation has completed.
	return status.Phase, false, nil
}

// isKargoInitiatedOperation returns true if the given operation was initiated
// by Kargo, based on the info key we set on all kargo initiated operations.
func isKargoInitiatedOperation(op argocd.Operation) bool {
	for _, info := range op.Info {
		if info != nil && info.Name == promotionInfoKey {
			return true
		}
	}
	return false
}

func (a *argocdUpdater) syncApplication(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	app *argocd.Application,
	desiredSources argocd.ApplicationSources,
) error {
	// Initiate a "hard" refresh.
	if app.Annotations == nil {
		app.Annotations = make(map[string]string, 1)
	}
	app.Annotations[argocd.AnnotationKeyRefresh] = string(argocd.RefreshTypeHard)

	// Update the desired source(s) in the Argo CD Application.
	if len(app.Spec.Sources) > 0 {
		app.Spec.Sources = desiredSources.DeepCopy()
	} else if app.Spec.Source != nil {
		app.Spec.Source = desiredSources[0].DeepCopy()
	}

	// Initiate a new operation.
	actor := applicationOperationInitiator
	automated := true
	if stepCtx.PromotionActor != "" {
		// PromotionActor is extracted from the `create-actor` annotation of the Promotion
		// object. If set, it implies it was a manually triggered promotion, which we
		// carry over to the Argo CD operation. This is important so that Argo CD sync
		// windows will be respected
		actor = stepCtx.PromotionActor
		automated = false
	}
	app.Operation = &argocd.Operation{
		InitiatedBy: argocd.OperationInitiator{
			Username:  actor,
			Automated: automated,
		},
		Info: []*argocd.Info{
			{
				Name:  "Reason",
				Value: "Promotion triggered a sync of this Application resource.",
			},
			{
				Name:  promotionInfoKey,
				Value: stepCtx.Promotion,
			},
		},
		Sync: &argocd.SyncOperation{
			Revisions: []string{},
		},
	}
	if app.Spec.SyncPolicy != nil {
		if app.Spec.SyncPolicy.Retry != nil {
			app.Operation.Retry = *app.Spec.SyncPolicy.Retry
		}
		if app.Spec.SyncPolicy.SyncOptions != nil {
			app.Operation.Sync.SyncOptions = app.Spec.SyncPolicy.SyncOptions
		}
	}
	// TODO(krancour): This is a workaround for the Argo CD Application controller
	// not handling this correctly itself. It is Argo CD's API server that usually
	// handles this, but we are bypassing the API server here.
	//
	// See issue: https://github.com/argoproj/argo-cd/issues/20875
	//
	// We can remove this hack once the issue is resolved and all Argo CD versions
	// without the fix have reached their EOL.
	app.Status.OperationState = nil

	// Patch the Argo CD Application.
	if err := a.argoCDAppPatchFn(ctx, app, func(src, dst unstructured.Unstructured) error {
		dst.SetAnnotations(src.GetAnnotations())
		dst.Object["spec"] = a.recursiveMerge(src.Object["spec"], dst.Object["spec"])
		dst.Object["operation"] = src.Object["operation"]
		// TODO(krancour): This is a workaround for the Argo CD Application
		// controller not handling this correctly itself. It is Argo CD's API server
		// that usually handles this, but we are bypassing the API server here.
		//
		// See issue: https://github.com/argoproj/argo-cd/issues/20875
		//
		// We can remove this hack once the issue is resolved and all Argo CD
		// versions without the fix have reached their EOL.
		//
		// We've once encountered an occasion where the unstructured representation
		// of the destination App was missing the status field because it had never
		// yet been reconciled (Application controller was not yet running), so we
		// are completely bailing on this hack if we find this to be the case. We
		// check the source object too for good measure, although that should not be
		// prone to the same problem.
		_, dstHasStatus := dst.Object["status"]
		_, srcHasStatus := src.Object["status"]
		if dstHasStatus && srcHasStatus {
			// nolint: forcetypeassert
			dst.Object["status"].(map[string]any)["operationState"] = src.Object["status"].(map[string]any)["operationState"]
		}
		return nil
	}); err != nil {
		return fmt.Errorf("error patching Argo CD Application %q: %w", app.Name, err)
	}
	logging.LoggerFromContext(ctx).Debug("patched Argo CD Application", "app", app.Name)

	// NB: This attempts to mimic the behavior of the Argo CD API server,
	// which logs an event when a sync is initiated. However, we do not
	// have access to the same enriched event data the Argo CD API server
	// has, so we are limited to logging an event with the best
	// information we have at hand.
	// xref: https://github.com/argoproj/argo-cd/blob/44894e9e438bca5adccf58d2f904adc63365805c/server/application/application.go#L1887-L1895
	// nolint:lll
	message := a.formatSyncMessage(app)
	a.logAppEventFn(
		ctx,
		app,
		applicationOperationInitiator,
		argocd.EventTypeOperationStarted,
		message,
	)
	return nil
}

// formatSyncMessage generates a concise, human-friendly message describing
// the sync target of the given Application. This message is intended only
// for logging and event recording within argocdUpdater.
//
// Behavior:
//   - If the Application has exactly one entry in .spec.sources, the message
//     includes its TargetRevision.
//   - If there are multiple sources, the message reports only the count to
//     avoid overly noisy logs.
//   - If only the legacy .spec.source field is set, the message includes its
//     TargetRevision.
//
// Full sync details remain available directly on the Application object.
func (a *argocdUpdater) formatSyncMessage(app *argocd.Application) string {
	message := "initiated sync"
	switch {
	case len(app.Spec.Sources) == 1:
		message += " to " + app.Spec.Sources[0].TargetRevision
	case len(app.Spec.Sources) > 1:
		message += fmt.Sprintf(" to %d sources", len(app.Spec.Sources))
	case app.Spec.Source != nil:
		message += " to " + app.Spec.Source.TargetRevision
	}

	return message
}

func (a *argocdUpdater) argoCDAppPatch(
	ctx context.Context,
	app kubeclient.ObjectWithKind,
	modify kubeclient.UnstructuredPatchFn,
) error {
	return kubeclient.PatchUnstructured(ctx, a.argocdClient, app, modify)
}

func (a *argocdUpdater) logAppEvent(
	ctx context.Context,
	app *argocd.Application,
	user string,
	reason string,
	message string,
) {
	logger := logging.LoggerFromContext(ctx).WithValues("app", app.Name)

	// xref: https://github.com/argoproj/argo-cd/blob/44894e9e438bca5adccf58d2f904adc63365805c/server/application/application.go#L2145-L2147
	// nolint:lll
	if user == "" {
		user = "Unknown user"
	}

	t := metav1.Time{Time: time.Now()}
	event := corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v.%x", app.Name, t.UnixNano()),
			Namespace: app.Namespace,
			// xref: https://github.com/argoproj/argo-cd/blob/44894e9e438bca5adccf58d2f904adc63365805c/util/argo/audit_logger.go#L118-L124
			// nolint:lll
			Annotations: map[string]string{
				"user": user,
			},
		},
		Source: corev1.EventSource{
			Component: user,
		},
		InvolvedObject: corev1.ObjectReference{
			APIVersion:      argocd.GroupVersion.String(),
			Kind:            app.Kind,
			Namespace:       app.Namespace,
			Name:            app.Name,
			UID:             app.UID,
			ResourceVersion: app.ResourceVersion,
		},
		FirstTimestamp: t,
		LastTimestamp:  t,
		Count:          1,
		// xref: https://github.com/argoproj/argo-cd/blob/44894e9e438bca5adccf58d2f904adc63365805c/server/application/application.go#L2148
		// nolint:lll
		Message: user + " " + message,
		Type:    corev1.EventTypeNormal,
		Reason:  reason,
	}
	if err := a.argocdClient.Create(context.Background(), &event); err != nil {
		logger.Error(
			err, "unable to create event for Argo CD Application",
			"reason", reason,
		)
	}
}

// getAuthorizedApplications returns a slice of Argo CD Applications that match
// the given update specification (either by name or by label selector) and are
// authorized for mutation by the Kargo Stage.
func (a *argocdUpdater) getAuthorizedApplications(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	update *builtin.ArgoCDAppUpdate,
) ([]*argocd.Application, error) {
	namespace := update.Namespace
	if namespace == "" {
		namespace = libargocd.Namespace()
	}

	var apps []*argocd.Application

	if update.Selector != nil {
		// List Applications by label selector
		labelSelector, err := a.buildLabelSelectorFn(update.Selector)
		if err != nil {
			return nil, fmt.Errorf("error building label selector: %w", err)
		}

		appList := &argocd.ApplicationList{}
		listOpts := []client.ListOption{
			client.InNamespace(namespace),
			client.MatchingLabelsSelector{Selector: labelSelector},
		}

		if err = a.argocdClient.List(ctx, appList, listOpts...); err != nil {
			return nil, fmt.Errorf("error listing Argo CD Applications matching selector: %w", err)
		}

		// Convert to pointer slice
		for i := range appList.Items {
			apps = append(apps, &appList.Items[i])
		}
	} else {
		// Get single Application by name
		app, err := argocd.GetApplication(ctx, a.argocdClient, namespace, update.Name)
		if err != nil {
			return nil, fmt.Errorf(
				"error finding Argo CD Application %q in namespace %q: %w",
				update.Name, namespace, err,
			)
		}
		if app == nil {
			return nil, fmt.Errorf(
				"unable to find Argo CD Application %q in namespace %q",
				update.Name, namespace,
			)
		}
		apps = append(apps, app)
	}

	// Filter by authorization
	logger := logging.LoggerFromContext(ctx)
	authorizedApps := make([]*argocd.Application, 0, len(apps))
	for _, app := range apps {
		if err := a.authorizeArgoCDAppUpdate(stepCtx, app.ObjectMeta); err != nil {
			// Log warning but continue with other apps
			logger.Info(
				"skipping unauthorized Application",
				"app", app.Name,
				"namespace", app.Namespace,
				"reason", err.Error(),
			)
			continue
		}
		authorizedApps = append(authorizedApps, app)
	}

	if len(authorizedApps) == 0 {
		if update.Selector != nil {
			totalAppsFound := len(apps)
			if totalAppsFound == 0 {
				return nil, fmt.Errorf(
					"no Argo CD Applications found matching selector in namespace %q",
					namespace,
				)
			}
			return nil, fmt.Errorf(
				"found %d Application(s) matching selector in namespace %q, but none are authorized for Stage %s:%s",
				totalAppsFound, namespace, stepCtx.Project, stepCtx.Stage,
			)
		}
		// nolint:staticcheck
		return nil, fmt.Errorf(
			"Argo CD Application %q in namespace %q is not authorized",
			update.Name, namespace,
		)
	}

	return authorizedApps, nil
}

// authorizeArgoCDAppUpdate returns an error if the Argo CD Application
// represented by appMeta does not explicitly permit mutation by the Kargo Stage
// represented by stageMeta.
func (a *argocdUpdater) authorizeArgoCDAppUpdate(
	stepCtx *promotion.StepContext,
	appMeta metav1.ObjectMeta,
) error {
	// nolint:staticcheck
	permErr := fmt.Errorf(
		"Argo CD Application %q in namespace %q does not permit mutation by "+
			"Kargo Stage %s in namespace %s",
		appMeta.Name,
		appMeta.Namespace,
		stepCtx.Stage,
		stepCtx.Project,
	)

	allowedStage, ok := appMeta.Annotations[kargoapi.AnnotationKeyAuthorizedStage]
	if !ok {
		return permErr
	}

	tokens := strings.SplitN(allowedStage, ":", 2)
	if len(tokens) != 2 {
		return fmt.Errorf(
			"unable to parse value of annotation %q (%q) on Argo CD Application %q in namespace %q",
			kargoapi.AnnotationKeyAuthorizedStage,
			allowedStage,
			appMeta.Name,
			appMeta.Namespace,
		)
	}

	projectName, stageName := tokens[0], tokens[1]
	if strings.Contains(projectName, "*") || strings.Contains(stageName, "*") {
		// nolint:staticcheck
		return fmt.Errorf(
			"Argo CD Application %q in namespace %q has deprecated glob expression in annotation %q (%q)",
			appMeta.Name,
			appMeta.Namespace,
			kargoapi.AnnotationKeyAuthorizedStage,
			allowedStage,
		)
	}
	if projectName != stepCtx.Project || stageName != stepCtx.Stage {
		return permErr
	}
	return nil
}

// buildLabelSelector converts an ArgoCDAppSelector into a Kubernetes labels.Selector.
func (a *argocdUpdater) buildLabelSelector(
	selector *builtin.ArgoCDAppSelector,
) (labels.Selector, error) {
	if len(selector.MatchLabels) == 0 && len(selector.MatchExpressions) == 0 {
		return nil, fmt.Errorf("selector must have at least one match criterion")
	}

	labelSelector := labels.NewSelector()

	for key, value := range selector.MatchLabels {
		req, err := labels.NewRequirement(key, selection.Equals, []string{value})
		if err != nil {
			return nil, fmt.Errorf("invalid matchLabel %s=%s: %w", key, value, err)
		}
		labelSelector = labelSelector.Add(*req)
	}

	for _, expr := range selector.MatchExpressions {
		var op selection.Operator
		switch expr.Operator {
		case builtin.In:
			op = selection.In
		case builtin.NotIn:
			op = selection.NotIn
		case builtin.Exists:
			op = selection.Exists
		case builtin.DoesNotExist:
			op = selection.DoesNotExist
		default:
			return nil, fmt.Errorf("invalid operator: %s", expr.Operator)
		}

		req, err := labels.NewRequirement(expr.Key, op, expr.Values)
		if err != nil {
			return nil, fmt.Errorf("invalid matchExpression: %w", err)
		}
		labelSelector = labelSelector.Add(*req)
	}

	return labelSelector, nil
}

// applyArgoCDSourceUpdate updates a single Argo CD ApplicationSource.
func (a *argocdUpdater) applyArgoCDSourceUpdate(
	update *builtin.ArgoCDAppSourceUpdate,
	desiredRevision string,
	source argocd.ApplicationSource,
) (argocd.ApplicationSource, bool) {
	if source.Chart != "" || update.Chart != "" {
		if source.RepoURL != update.RepoURL || source.Chart != update.Chart {
			// The update is not applicable to this source.
			return source, false
		}
		// If we get to here, we have confirmed that this update is applicable to
		// this source.
		if update.UpdateTargetRevision && desiredRevision != "" {
			source.TargetRevision = desiredRevision
		}
	} else {
		// We're dealing with a git repo, so we should normalize the repo URLs
		// before comparing them.
		sourceRepoURL := urls.NormalizeGit(source.RepoURL)
		if sourceRepoURL != urls.NormalizeGit(update.RepoURL) {
			// The update is not applicable to this source.
			return source, false
		}
		// If we get to here, we have confirmed that this update is applicable to
		// this source.
		if update.UpdateTargetRevision && desiredRevision != "" {
			source.TargetRevision = desiredRevision
		}
	}

	if update.Kustomize != nil && len(update.Kustomize.Images) > 0 {
		if source.Kustomize == nil {
			source.Kustomize = &argocd.ApplicationSourceKustomize{}
		}
		source.Kustomize.Images = a.buildKustomizeImagesForAppSource(update.Kustomize)
	}

	if update.Helm != nil && len(update.Helm.Images) > 0 {
		if source.Helm == nil {
			source.Helm = &argocd.ApplicationSourceHelm{}
		}
		if source.Helm.Parameters == nil {
			source.Helm.Parameters = []argocd.HelmParameter{}
		}
		changes := a.buildHelmParamChangesForAppSource(update.Helm)
	imageUpdateLoop:
		for k, v := range changes {
			newParam := argocd.HelmParameter{
				Name:  k,
				Value: v,
			}
			for i, param := range source.Helm.Parameters {
				if param.Name == k {
					source.Helm.Parameters[i] = newParam
					continue imageUpdateLoop
				}
			}
			source.Helm.Parameters = append(source.Helm.Parameters, newParam)
		}
	}

	return source, true
}

func (a *argocdUpdater) buildKustomizeImagesForAppSource(
	update *builtin.ArgoCDKustomizeImageUpdates,
) argocd.KustomizeImages {
	kustomizeImages := make(argocd.KustomizeImages, 0, len(update.Images))
	for i := range update.Images {
		imageUpdate := &update.Images[i]
		var digest, tag string
		switch {
		case imageUpdate.Digest != "":
			digest = imageUpdate.Digest
		case imageUpdate.Tag != "":
			tag = imageUpdate.Tag
		}
		kustomizeImageStr := imageUpdate.RepoURL
		if imageUpdate.NewName != "" {
			kustomizeImageStr = fmt.Sprintf("%s=%s", kustomizeImageStr, imageUpdate.NewName)
		}
		if digest != "" {
			kustomizeImageStr = fmt.Sprintf("%s@%s", kustomizeImageStr, digest)
		} else {
			kustomizeImageStr = fmt.Sprintf("%s:%s", kustomizeImageStr, tag)
		}
		kustomizeImages = append(
			kustomizeImages,
			argocd.KustomizeImage(kustomizeImageStr),
		)
	}
	return kustomizeImages
}

func (a *argocdUpdater) buildHelmParamChangesForAppSource(
	update *builtin.ArgoCDHelmParameterUpdates,
) map[string]string {
	changes := map[string]string{}
	for i := range update.Images {
		imageUpdate := &update.Images[i]
		changes[imageUpdate.Key] = imageUpdate.Value
	}
	return changes
}

func (a *argocdUpdater) operationPhaseToPromotionStepStatus(
	phases ...argocd.OperationPhase,
) kargoapi.PromotionStepStatus {
	if len(phases) == 0 {
		return ""
	}

	libargocd.ByOperationPhase(phases).Sort()

	switch phases[0] {
	case argocd.OperationRunning, argocd.OperationTerminating:
		return kargoapi.PromotionStepStatusRunning
	case argocd.OperationFailed, argocd.OperationError:
		return kargoapi.PromotionStepStatusErrored
	case argocd.OperationSucceeded:
		return kargoapi.PromotionStepStatusSucceeded
	default:
		return ""
	}
}

func (a *argocdUpdater) recursiveMerge(src, dst any) any {
	switch src := src.(type) {
	case map[string]any:
		dst, ok := dst.(map[string]any)
		if !ok {
			return src
		}
		for srcK, srcV := range src {
			if dstV, ok := dst[srcK]; ok {
				dst[srcK] = a.recursiveMerge(srcV, dstV)
			} else {
				dst[srcK] = srcV
			}
		}
	case []any:
		dst, ok := dst.([]any)
		if !ok {
			return src
		}
		result := make([]any, len(src))
		for i, srcV := range src {
			if i < len(dst) {
				result[i] = a.recursiveMerge(srcV, dst[i])
			} else {
				result[i] = srcV
			}
		}
		return result
	default:
		return src
	}
	return dst
}
