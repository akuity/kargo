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
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libargocd "github.com/akuity/kargo/internal/argocd"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/git"
	checkers "github.com/akuity/kargo/internal/health/checker/builtin"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/pkg/health"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const (
	applicationOperationInitiator = "kargo-controller"
	promotionInfoKey              = "kargo.akuity.io/promotion"
)

// argocdUpdater is an implementation of the promotion.StepRunner interface that
// updates one or more Argo CD Application resources.
type argocdUpdater struct {
	schemaLoader gojsonschema.JSONLoader

	argocdClient client.Client

	// These behaviors are overridable for testing purposes:

	getAuthorizedApplicationFn func(
		context.Context,
		*promotion.StepContext,
		client.ObjectKey,
	) (*argocd.Application, error)

	buildDesiredSourcesFn func(
		update *builtin.ArgoCDAppUpdate,
		desiredRevisions []string,
		app *argocd.Application,
	) (argocd.ApplicationSources, error)

	mustPerformUpdateFn func(
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
func newArgocdUpdater(argocdClient client.Client) *argocdUpdater {
	r := &argocdUpdater{
		argocdClient: argocdClient,
	}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	r.getAuthorizedApplicationFn = r.getAuthorizedApplication
	r.buildDesiredSourcesFn = r.buildDesiredSources
	r.mustPerformUpdateFn = r.mustPerformUpdate
	r.syncApplicationFn = r.syncApplication
	r.applyArgoCDSourceUpdateFn = r.applyArgoCDSourceUpdate
	r.argoCDAppPatchFn = r.argoCDAppPatch
	r.logAppEventFn = r.logAppEvent
	return r
}

// Name implements the promotion.StepRunner interface.
func (a *argocdUpdater) Name() string {
	return "argocd-update"
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
	return validateAndConvert[builtin.ArgoCDUpdateConfig](a.schemaLoader, cfg, a.Name())
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
	logger.Debug("executing argocd-update promotion step")

	var updateResults = make([]argocd.OperationPhase, 0, len(stepCfg.Apps))
	appHealthChecks := make([]checkers.ArgoCDAppHealthCheck, len(stepCfg.Apps))
	for i := range stepCfg.Apps {
		update := &stepCfg.Apps[i]
		// Retrieve the Argo CD Application.
		appKey := client.ObjectKey{
			Namespace: update.Namespace,
			Name:      update.Name,
		}
		if appKey.Namespace == "" {
			appKey.Namespace = libargocd.Namespace()
		}
		app, err := a.getAuthorizedApplicationFn(ctx, stepCtx, appKey)
		if err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, fmt.Errorf(
				"error getting Argo CD Application %q in namespace %q: %w",
				appKey.Name, appKey.Namespace, err,
			)
		}

		desiredRevisions := a.getDesiredRevisions(update, app)
		appHealthChecks[i] = checkers.ArgoCDAppHealthCheck{
			Name:             app.Name,
			Namespace:        app.Namespace,
			DesiredRevisions: desiredRevisions,
		}

		// Check if the update needs to be performed and retrieve its phase.
		phase, mustUpdate, err := a.mustPerformUpdateFn(stepCtx, update, app)

		// If we have a phase, append it to the results.
		if phase != "" {
			updateResults = append(updateResults, phase)
		}

		// If we don't need to perform an update, further processing depends on
		// the phase and whether an error occurred.
		if !mustUpdate {
			if err != nil {
				if phase == "" {
					// If we do not have a phase, we cannot continue processing
					// this update by waiting.
					return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
				}
				// Log the error as a warning, but continue to the next update.
				logger.Info(err.Error())
			}
			if phase.Failed() {
				// Record the reason for the failure if available.
				if app.Status.OperationState != nil {
					// nolint:staticcheck
					return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, fmt.Errorf(
						"Argo CD Application %q in namespace %q failed with: %s",
						app.Name,
						app.Namespace,
						app.Status.OperationState.Message,
					)
				}
				// If the update failed, we can short-circuit. This is
				// effectively "fail fast" behavior.
				return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, nil
			}
			// If we get here, we can continue to the next update.
			continue
		}

		// Log the error, as it contains information about why we need to
		// perform an update.
		if err != nil {
			logger.Debug(err.Error())
		}

		// Build the desired source(s) for the Argo CD Application.
		desiredSources, err := a.buildDesiredSourcesFn(
			update,
			desiredRevisions,
			app,
		)
		if err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, fmt.Errorf(
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
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, fmt.Errorf(
				"error syncing Argo CD Application %q in namespace %q: %w",
				app.Name, app.Namespace, err,
			)
		}
		// As we have initiated an update, we should wait for it to complete.
		updateResults = append(updateResults, argocd.OperationRunning)
	}

	aggregatedStatus := a.operationPhaseToPromotionStepStatus(updateResults...)
	if aggregatedStatus == "" {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, fmt.Errorf(
			"could not determine promotion step status from operation phases: %v",
			updateResults,
		)
	}

	logger.Debug("done executing argocd-update promotion step")

	return promotion.StepResult{
		Status: aggregatedStatus,
		HealthCheck: &health.Criteria{
			Kind: a.Name(),
			Input: health.Input{
				"apps": appHealthChecks,
			},
		},
	}, nil
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
		if !updateUsed {
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
	}
	return desiredSources, nil
}

func (a *argocdUpdater) mustPerformUpdate(
	stepCtx *promotion.StepContext,
	update *builtin.ArgoCDAppUpdate,
	app *argocd.Application,
) (phase argocd.OperationPhase, mustUpdate bool, err error) {
	status := app.Status.OperationState
	if status == nil {
		// The application has no operation.
		return "", true, nil
	}

	// Deal with the possibility that the operation was not initiated by Kargo
	if !isKargoInitiatedOperation(status.Operation) {
		if !status.Phase.Completed() {
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
		return "", true, nil
	}

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
		// The operation was not initiated for the current Promotion.
		if !status.Phase.Completed() {
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
		return "", true, nil
	}

	if !status.Phase.Completed() {
		// The operation is still running.
		return status.Phase, false, nil
	}

	if status.SyncResult == nil {
		// We do not have a sync result, so we cannot determine if the operation
		// was successful. The best recourse is to retry the operation.
		return "", true, errors.New("operation completed without a sync result")
	}

	// Check if the desired revisions were applied.
	desiredRevisions := a.getDesiredRevisions(update, app)
	if len(desiredRevisions) == 0 {
		// We do not have any desired revisions, so we cannot determine if the
		// operation was successful.
		return status.Phase, false, nil
	}

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
			return "", true, fmt.Errorf(
				"sync result revisions %v do not match desired revisions %v",
				observedRevisions, desiredRevisions,
			)
		}
	}

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
	if app.Spec.Source != nil {
		app.Spec.Source = desiredSources[0].DeepCopy()
	} else {
		app.Spec.Sources = desiredSources.DeepCopy()
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
	if app.Spec.Source != nil {
		app.Operation.Sync.Revisions = []string{app.Spec.Source.TargetRevision}
	}
	for _, source := range app.Spec.Sources {
		app.Operation.Sync.Revisions = append(app.Operation.Sync.Revisions, source.TargetRevision)
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
			dst.Object["status"].(map[string]any)["operationState"] =
				src.Object["status"].(map[string]any)["operationState"]
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
	//
	// TODO(hidde): It is not clear what we should do if we have a list of
	// sources.
	message := "initiated sync"
	if app.Spec.Source != nil {
		message += " to " + app.Spec.Source.TargetRevision
	}
	a.logAppEventFn(
		ctx,
		app,
		applicationOperationInitiator,
		argocd.EventReasonOperationStarted,
		message,
	)
	return nil
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

// getAuthorizedApplication returns an Argo CD Application in the given namespace
// with the given name, if it is authorized for mutation by the Kargo Stage
// represented by stageMeta.
func (a *argocdUpdater) getAuthorizedApplication(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	appKey client.ObjectKey,
) (*argocd.Application, error) {
	app, err := argocd.GetApplication(
		ctx,
		a.argocdClient,
		appKey.Namespace,
		appKey.Name,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"error finding Argo CD Application %q in namespace %q: %w",
			appKey.Name, appKey.Namespace, err,
		)
	}
	if app == nil {
		return nil, fmt.Errorf(
			"unable to find Argo CD Application %q in namespace %q",
			appKey.Name, appKey.Namespace,
		)
	}

	if err = a.authorizeArgoCDAppUpdate(stepCtx, app.ObjectMeta); err != nil {
		return nil, err
	}

	return app, nil
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
		sourceRepoURL := git.NormalizeURL(source.RepoURL)
		if sourceRepoURL != git.NormalizeURL(update.RepoURL) {
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
