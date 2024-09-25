package directives

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gobwas/glob"
	"github.com/xeipuuv/gojsonschema"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	libargocd "github.com/akuity/kargo/internal/argocd"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/freight"
	"github.com/akuity/kargo/internal/git"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/logging"
)

const (
	authorizedStageAnnotationKey = "kargo.akuity.io/authorized-stage"

	applicationOperationInitiator = "kargo-controller"
	freightCollectionInfoKey      = "kargo.akuity.io/freight-collection"
)

func init() {
	builtins.RegisterPromotionStepRunner(
		newArgocdUpdater(),
		&StepRunnerPermissions{
			AllowKargoClient:  true,
			AllowArgoCDClient: true,
		},
	)
}

// argocdUpdater is an implementation of the PromotionStepRunner interface that
// updates one or more Argo CD Application resources.
type argocdUpdater struct {
	schemaLoader gojsonschema.JSONLoader

	// These behaviors are overridable for testing purposes:

	getAuthorizedApplicationFn func(
		context.Context,
		*PromotionStepContext,
		client.ObjectKey,
	) (*argocd.Application, error)

	buildDesiredSourcesFn func(
		context.Context,
		*PromotionStepContext,
		*ArgoCDUpdateConfig,
		*ArgoCDAppUpdate,
		*argocd.Application,
	) (argocd.ApplicationSources, error)

	mustPerformUpdateFn func(
		context.Context,
		*PromotionStepContext,
		*ArgoCDUpdateConfig,
		*ArgoCDAppUpdate,
		*argocd.Application,
		argocd.ApplicationSources,
	) (argocd.OperationPhase, bool, error)

	syncApplicationFn func(
		ctx context.Context,
		stepCtx *PromotionStepContext,
		app *argocd.Application,
		desiredSources argocd.ApplicationSources,
	) error

	applyArgoCDSourceUpdateFn func(
		context.Context,
		*PromotionStepContext,
		*ArgoCDUpdateConfig,
		*ArgoCDAppSourceUpdate,
		argocd.ApplicationSource,
	) (argocd.ApplicationSource, error)

	argoCDAppPatchFn func(
		context.Context,
		*PromotionStepContext,
		kubeclient.ObjectWithKind,
		kubeclient.UnstructuredPatchFn,
	) error

	logAppEventFn func(
		ctx context.Context,
		stepCtx *PromotionStepContext,
		app *argocd.Application,
		user string,
		reason string,
		message string,
	)
}

// newArgocdUpdater returns a implementation of the PromotionStepRunner
// interface that updates one or more Argo CD Application resources.
func newArgocdUpdater() PromotionStepRunner {
	r := &argocdUpdater{}
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

// Name implements the PromotionStepRunner interface.
func (a *argocdUpdater) Name() string {
	return "argocd-update"
}

// RunPromotionStep implements the PromotionStepRunner interface.
func (a *argocdUpdater) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	if err := a.validate(stepCtx.Config); err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure}, err
	}
	cfg, err := configToStruct[ArgoCDUpdateConfig](stepCtx.Config)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("could not convert config into %s config: %w", a.Name(), err)
	}
	return a.runPromotionStep(ctx, stepCtx, cfg)
}

// validate validates argocdUpdatePromotionStepRunner configuration against a
// JSON schema.
func (a *argocdUpdater) validate(cfg Config) error {
	return validate(a.schemaLoader, gojsonschema.NewGoLoader(cfg), a.Name())
}

func (a *argocdUpdater) runPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	stepCfg ArgoCDUpdateConfig,
) (PromotionStepResult, error) {
	if stepCtx.ArgoCDClient == nil {
		return PromotionStepResult{Status: PromotionStatusFailure}, errors.New(
			"Argo CD integration is disabled on this controller; cannot update " +
				"Argo CD Application resources",
		)
	}

	logger := logging.LoggerFromContext(ctx)
	logger.Debug("executing argocd-update promotion step")

	var updateResults = make([]argocd.OperationPhase, 0, len(stepCfg.Apps))
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
			return PromotionStepResult{Status: PromotionStatusFailure}, fmt.Errorf(
				"error getting Argo CD Application %q in namespace %q: %w",
				appKey.Name, appKey.Namespace, err,
			)
		}

		// Build the desired source(s) for the Argo CD Application.
		desiredSources, err := a.buildDesiredSourcesFn(
			ctx,
			stepCtx,
			&stepCfg,
			update,
			app,
		)
		if err != nil {
			return PromotionStepResult{Status: PromotionStatusFailure}, fmt.Errorf(
				"error building desired sources for Argo CD Application %q in namespace %q: %w",
				app.Name, app.Namespace, err,
			)
		}

		// Check if the update needs to be performed and retrieve its phase.
		phase, mustUpdate, err := a.mustPerformUpdateFn(
			ctx,
			stepCtx,
			&stepCfg,
			update,
			app,
			desiredSources,
		)

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
					return PromotionStepResult{Status: PromotionStatusFailure}, err
				}
				// Log the error as a warning, but continue to the next update.
				logger.Info(err.Error())
			}
			if phase.Failed() {
				// Record the reason for the failure if available.
				if app.Status.OperationState != nil {
					return PromotionStepResult{Status: PromotionStatusFailure}, fmt.Errorf(
						"Argo CD Application %q in namespace %q failed with: %s",
						app.Name,
						app.Namespace,
						app.Status.OperationState.Message,
					)
				}
				// If the update failed, we can short-circuit. This is
				// effectively "fail fast" behavior.
				return PromotionStepResult{Status: PromotionStatusFailure}, nil
			}
			// If we get here, we can continue to the next update.
			continue
		}

		// Perform the update.
		if err = a.syncApplicationFn(
			ctx,
			stepCtx,
			app,
			desiredSources,
		); err != nil {
			return PromotionStepResult{Status: PromotionStatusFailure}, fmt.Errorf(
				"error syncing Argo CD Application %q in namespace %q: %w",
				app.Name, app.Namespace, err,
			)
		}
		// As we have initiated an update, we should wait for it to complete.
		updateResults = append(updateResults, argocd.OperationRunning)
	}

	aggregatedStatus := a.operationPhaseToPromotionStatus(updateResults...)
	if aggregatedStatus == "" {
		return PromotionStepResult{Status: PromotionStatusFailure}, fmt.Errorf(
			"could not determine promotion step status from operation phases: %v",
			updateResults,
		)
	}

	logger.Debug("done executing argocd-update promotion step")
	return PromotionStepResult{Status: aggregatedStatus}, nil
}

// buildDesiredSources returns the desired source(s) for an Argo CD Application,
// by updating the current source(s) with the given source updates.
func (a *argocdUpdater) buildDesiredSources(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	stepCfg *ArgoCDUpdateConfig,
	update *ArgoCDAppUpdate,
	app *argocd.Application,
) (argocd.ApplicationSources, error) {
	desiredSources := app.Spec.Sources.DeepCopy()
	if len(desiredSources) == 0 && app.Spec.Source != nil {
		desiredSources = []argocd.ApplicationSource{*app.Spec.Source.DeepCopy()}
	}
	for i := range desiredSources {
		for j := range update.Sources {
			srcUpdate := &update.Sources[j]
			var err error
			if desiredSources[i], err = a.applyArgoCDSourceUpdateFn(
				ctx,
				stepCtx,
				stepCfg,
				srcUpdate,
				desiredSources[i],
			); err != nil {
				return nil, fmt.Errorf(
					"error applying source update to Argo CD Application %q in namespace %q: %w",
					update.Name,
					app.Namespace,
					err,
				)
			}
		}
	}
	return desiredSources, nil
}

func (a *argocdUpdater) mustPerformUpdate(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	stepCfg *ArgoCDUpdateConfig,
	update *ArgoCDAppUpdate,
	app *argocd.Application,
	desiredSources argocd.ApplicationSources,
) (phase argocd.OperationPhase, mustUpdate bool, err error) {
	status := app.Status.OperationState
	if status == nil {
		// The application has no operation.
		return "", true, nil
	}

	// Deal with the possibility that the operation was not initiated by the
	// expected user.
	if status.Operation.InitiatedBy.Username != applicationOperationInitiator {
		// The operation was not initiated by the expected user.
		if !status.Phase.Completed() {
			// We should wait for the operation to complete before attempting to
			// apply an update ourselves.
			// NB: We return the current phase here because we want the caller
			//     to know that an operation is still running.
			return status.Phase, false, fmt.Errorf(
				"current operation was not initiated by %q and not by %q: waiting for operation to complete",
				applicationOperationInitiator, status.Operation.InitiatedBy.Username,
			)
		}
		// Initiate our own operation.
		return "", true, nil
	}

	// Deal with the possibility that the operation was not initiated for the
	// current freight collection. i.e. Not related to the current promotion.
	var correctFreightColIDFound bool
	for _, info := range status.Operation.Info {
		if info.Name == freightCollectionInfoKey {
			correctFreightColIDFound = info.Value == stepCtx.Freight.ID
			break
		}
	}
	if !correctFreightColIDFound {
		// The operation was not initiated for the current freight collection.
		if !status.Phase.Completed() {
			// We should wait for the operation to complete before attempting to
			// apply an update ourselves.
			// NB: We return the current phase here because we want the caller
			//     to know that an operation is still running.
			return status.Phase, false, fmt.Errorf(
				"current operation was not initiated for freight collection %q: waiting for operation to complete",
				stepCtx.Freight.ID,
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

	// Check if the desired sources were applied.
	if len(update.Sources) > 0 {
		if (status.SyncResult.Source.RepoURL != "" && !status.SyncResult.Source.Equals(&desiredSources[0])) ||
			(status.SyncResult.Source.RepoURL == "" && !status.SyncResult.Sources.Equals(desiredSources)) {
			// The operation did not result in the desired sources being applied. We
			// should attempt to retry the operation.
			return "", true, fmt.Errorf(
				"operation result source does not match desired source",
			)
		}
	}

	// Check if the desired revisions were applied.
	desiredRevisions, err := a.getDesiredRevisions(
		ctx,
		stepCtx,
		stepCfg,
		update,
		app,
	)
	if err != nil {
		return "", true, fmt.Errorf("error determining desired revision: %w", err)
	}

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

func (a *argocdUpdater) syncApplication(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	app *argocd.Application,
	desiredSources argocd.ApplicationSources,
) error {
	// Initiate a "hard" refresh.
	if app.ObjectMeta.Annotations == nil {
		app.ObjectMeta.Annotations = make(map[string]string, 1)
	}
	app.ObjectMeta.Annotations[argocd.AnnotationKeyRefresh] = string(argocd.RefreshTypeHard)

	// Update the desired source(s) in the Argo CD Application.
	if app.Spec.Source != nil {
		app.Spec.Source = desiredSources[0].DeepCopy()
	} else {
		app.Spec.Sources = desiredSources.DeepCopy()
	}

	// Initiate a new operation.
	app.Operation = &argocd.Operation{
		InitiatedBy: argocd.OperationInitiator{
			Username:  applicationOperationInitiator,
			Automated: true,
		},
		Info: []*argocd.Info{
			{
				Name:  "Reason",
				Value: "Promotion triggered a sync of this Application resource.",
			},
			{
				Name:  freightCollectionInfoKey,
				Value: stepCtx.Freight.ID,
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

	// Patch the Argo CD Application.
	if err := a.argoCDAppPatchFn(ctx, stepCtx, app, func(src, dst unstructured.Unstructured) error {
		// If the resource has been modified since we fetched it, an update
		// can result in unexpected merge results. Detect this, and return an
		// error if it occurs.
		if src.GetGeneration() != dst.GetGeneration() {
			return fmt.Errorf("unable to update sources to desired revisions: resource has been modified")
		}

		dst.SetAnnotations(src.GetAnnotations())
		dst.Object["spec"] = a.recursiveMerge(src.Object["spec"], dst.Object["spec"])
		dst.Object["operation"] = src.Object["operation"]
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
		stepCtx,
		app,
		applicationOperationInitiator,
		argocd.EventReasonOperationStarted,
		message,
	)
	return nil
}

func (a *argocdUpdater) argoCDAppPatch(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	app kubeclient.ObjectWithKind,
	modify kubeclient.UnstructuredPatchFn,
) error {
	return kubeclient.PatchUnstructured(ctx, stepCtx.ArgoCDClient, app, modify)
}

func (a *argocdUpdater) logAppEvent(
	ctx context.Context,
	stepCtx *PromotionStepContext,
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
			Kind:            app.TypeMeta.Kind,
			Namespace:       app.ObjectMeta.Namespace,
			Name:            app.ObjectMeta.Name,
			UID:             app.ObjectMeta.UID,
			ResourceVersion: app.ObjectMeta.ResourceVersion,
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
	if err := stepCtx.ArgoCDClient.Create(context.Background(), &event); err != nil {
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
	stepCtx *PromotionStepContext,
	appKey client.ObjectKey,
) (*argocd.Application, error) {
	app, err := argocd.GetApplication(
		ctx,
		stepCtx.ArgoCDClient,
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
	stepCtx *PromotionStepContext,
	appMeta metav1.ObjectMeta,
) error {
	permErr := fmt.Errorf(
		"Argo CD Application %q in namespace %q does not permit mutation by "+
			"Kargo Stage %s in namespace %s",
		appMeta.Name,
		appMeta.Namespace,
		stepCtx.Stage,
		stepCtx.Project,
	)
	if appMeta.Annotations == nil {
		return permErr
	}
	allowedStage, ok := appMeta.Annotations[authorizedStageAnnotationKey]
	if !ok {
		return permErr
	}
	tokens := strings.SplitN(allowedStage, ":", 2)
	if len(tokens) != 2 {
		return fmt.Errorf(
			"unable to parse value of annotation %q (%q) on Argo CD Application "+
				"%q in namespace %q",
			authorizedStageAnnotationKey,
			allowedStage,
			appMeta.Name,
			appMeta.Namespace,
		)
	}
	allowedNamespaceGlob, err := glob.Compile(tokens[0])
	if err != nil {
		return fmt.Errorf(
			"Argo CD Application %q in namespace %q has invalid glob expression: %q",
			appMeta.Name,
			appMeta.Namespace,
			tokens[0],
		)
	}
	allowedNameGlob, err := glob.Compile(tokens[1])
	if err != nil {
		return fmt.Errorf(
			"Argo CD Application %q in namespace %q has invalid glob expression: %q",
			appMeta.Name,
			appMeta.Namespace,
			tokens[1],
		)
	}
	if !allowedNamespaceGlob.Match(stepCtx.Project) ||
		!allowedNameGlob.Match(stepCtx.Stage) {
		return permErr
	}
	return nil
}

// applyArgoCDSourceUpdate updates a single Argo CD ApplicationSource.
func (a *argocdUpdater) applyArgoCDSourceUpdate(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	stepCfg *ArgoCDUpdateConfig,
	update *ArgoCDAppSourceUpdate,
	source argocd.ApplicationSource,
) (argocd.ApplicationSource, error) {
	if source.Chart != "" || update.Chart != "" {

		if source.RepoURL != update.RepoURL || source.Chart != update.Chart {
			// There's no change to make in this case.
			return source, nil
		}

		// If we get to here, we have confirmed that this update is applicable to
		// this source.

		if update.UpdateTargetRevision {
			desiredOrigin := getDesiredOrigin(stepCfg, update)
			repoURL := update.RepoURL
			chartName := update.Chart
			if !strings.Contains(repoURL, "://") {
				// Where OCI is concerned, ArgoCDSourceUpdates play by Argo CD rules. i.e.
				// No leading oci://, and the repository URL is really a registry URL, and
				// the chart name is a repository within that registry. Warehouses and
				// Freight, however, do lead with oci:// and handle things more correctly
				// where a repoURL points directly to a repository and chart name is
				// irrelevant / blank. We need to account for this when we search our
				// Freight for the chart.
				repoURL = fmt.Sprintf(
					"oci://%s/%s",
					strings.TrimSuffix(repoURL, "/"),
					chartName,
				)
				chartName = ""
			}
			chart, err := freight.FindChart(
				ctx,
				stepCtx.KargoClient,
				stepCtx.Project,
				stepCtx.FreightRequests,
				desiredOrigin,
				stepCtx.Freight.References(),
				repoURL,
				chartName,
			)
			if err != nil {
				return source,
					fmt.Errorf("error chart from repo %q: %w", update.RepoURL, err)
			}
			if chart != nil {
				source.TargetRevision = chart.Version
			}
		}
	} else {
		// We're dealing with a git repo, so we should normalize the repo URLs
		// before comparing them.
		sourceRepoURL := git.NormalizeURL(source.RepoURL)
		if sourceRepoURL != git.NormalizeURL(update.RepoURL) {
			return source, nil
		}

		// If we get to here, we have confirmed that this update is applicable to
		// this source.

		if update.UpdateTargetRevision {
			desiredOrigin := getDesiredOrigin(stepCfg, update)
			commit, err := freight.FindCommit(
				ctx,
				stepCtx.KargoClient,
				stepCtx.Project,
				stepCtx.FreightRequests,
				desiredOrigin,
				stepCtx.Freight.References(),
				update.RepoURL,
			)
			if err != nil {
				return source,
					fmt.Errorf("error finding commit from repo %q: %w", update.RepoURL, err)
			}
			if commit != nil {
				if commit.Tag != "" {
					source.TargetRevision = commit.Tag
				} else {
					source.TargetRevision = commit.ID
				}
			}
		}
	}

	if update.Kustomize != nil && len(update.Kustomize.Images) > 0 {
		if source.Kustomize == nil {
			source.Kustomize = &argocd.ApplicationSourceKustomize{}
		}
		var err error
		if source.Kustomize.Images, err = a.buildKustomizeImagesForAppSource(
			ctx,
			stepCtx,
			stepCfg,
			update.Kustomize,
		); err != nil {
			return source, err
		}
	}

	if update.Helm != nil && len(update.Helm.Images) > 0 {
		if source.Helm == nil {
			source.Helm = &argocd.ApplicationSourceHelm{}
		}
		if source.Helm.Parameters == nil {
			source.Helm.Parameters = []argocd.HelmParameter{}
		}
		changes, err := a.buildHelmParamChangesForAppSource(
			ctx,
			stepCtx,
			stepCfg,
			update.Helm,
		)
		if err != nil {
			return source,
				fmt.Errorf("error building Helm parameter changes: %w", err)
		}
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

	return source, nil
}

func (a *argocdUpdater) buildKustomizeImagesForAppSource(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	stepCfg *ArgoCDUpdateConfig,
	update *ArgoCDKustomizeImageUpdates,
) (argocd.KustomizeImages, error) {
	kustomizeImages := make(argocd.KustomizeImages, 0, len(update.Images))
	for i := range update.Images {
		imageUpdate := &update.Images[i]
		desiredOrigin := getDesiredOrigin(stepCfg, imageUpdate)
		image, err := freight.FindImage(
			ctx,
			stepCtx.KargoClient,
			stepCtx.Project,
			stepCtx.FreightRequests,
			desiredOrigin,
			stepCtx.Freight.References(),
			imageUpdate.RepoURL,
		)
		if err != nil {
			return nil,
				fmt.Errorf("error finding image from repo %q: %w", imageUpdate.RepoURL, err)
		}
		if image == nil {
			// There's no change to make in this case.
			continue
		}
		kustomizeImageStr := imageUpdate.RepoURL
		if imageUpdate.NewName != "" {
			kustomizeImageStr = fmt.Sprintf("%s=%s", kustomizeImageStr, imageUpdate.NewName)
		}
		if imageUpdate.UseDigest {
			kustomizeImageStr = fmt.Sprintf("%s@%s", kustomizeImageStr, image.Digest)
		} else {
			kustomizeImageStr = fmt.Sprintf("%s:%s", kustomizeImageStr, image.Tag)
		}
		kustomizeImages = append(
			kustomizeImages,
			argocd.KustomizeImage(kustomizeImageStr),
		)
	}
	return kustomizeImages, nil
}

func (a *argocdUpdater) buildHelmParamChangesForAppSource(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	stepCfg *ArgoCDUpdateConfig,
	update *ArgoCDHelmParameterUpdates,
) (map[string]string, error) {
	changes := map[string]string{}
	for i := range update.Images {
		imageUpdate := &update.Images[i]
		switch imageUpdate.Value {
		case ImageAndTag, Tag, ImageAndDigest, Digest:
		default:
			// This really shouldn't happen, so we'll ignore it.
			continue
		}
		desiredOrigin := getDesiredOrigin(stepCfg, imageUpdate)
		image, err := freight.FindImage(
			ctx,
			stepCtx.KargoClient,
			stepCtx.Project,
			stepCtx.FreightRequests,
			desiredOrigin,
			stepCtx.Freight.References(),
			imageUpdate.RepoURL,
		)
		if err != nil {
			return nil,
				fmt.Errorf("error finding image from repo %q: %w", imageUpdate.RepoURL, err)
		}
		if image == nil {
			continue
		}
		switch imageUpdate.Value {
		case ImageAndTag:
			changes[imageUpdate.Key] = fmt.Sprintf("%s:%s", imageUpdate.RepoURL, image.Tag)
		case Tag:
			changes[imageUpdate.Key] = image.Tag
		case ImageAndDigest:
			changes[imageUpdate.Key] = fmt.Sprintf("%s@%s", imageUpdate.RepoURL, image.Digest)
		case Digest:
			changes[imageUpdate.Key] = image.Digest
		}
	}
	return changes, nil
}

func (a *argocdUpdater) operationPhaseToPromotionStatus(
	phases ...argocd.OperationPhase,
) PromotionStatus {
	if len(phases) == 0 {
		return ""
	}

	libargocd.ByOperationPhase(phases).Sort()

	switch phases[0] {
	case argocd.OperationRunning, argocd.OperationTerminating:
		return PromotionStatusPending
	case argocd.OperationFailed, argocd.OperationError:
		return PromotionStatusFailure
	case argocd.OperationSucceeded:
		return PromotionStatusSuccess
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
