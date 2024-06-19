package promotion

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/gobwas/glob"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libargocd "github.com/akuity/kargo/internal/argocd"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/git"
	"github.com/akuity/kargo/internal/logging"
)

const (
	authorizedStageAnnotationKey = "kargo.akuity.io/authorized-stage"

	applicationOperationInitiator = "kargo-controller"
)

// argoCDMechanism is an implementation of the Mechanism interface that updates
// Argo CD Application resources.
type argoCDMechanism struct {
	argocdClient client.Client
	// These behaviors are overridable for testing purposes:
	buildDesiredSourcesFn func(
		app *argocd.Application,
		update kargoapi.ArgoCDAppUpdate,
		newFreight kargoapi.FreightReference,
	) (*argocd.ApplicationSource, argocd.ApplicationSources, error)
	mustPerformUpdateFn func(
		app *argocd.Application,
		update kargoapi.ArgoCDAppUpdate,
		newFreight kargoapi.FreightReference,
		desiredSource *argocd.ApplicationSource,
		desiredSources argocd.ApplicationSources,
	) (argocd.OperationPhase, bool, error)
	updateApplicationSourcesFn func(
		ctx context.Context,
		app *argocd.Application,
		desiredSource *argocd.ApplicationSource,
		desiredSources argocd.ApplicationSources,
	) error
	getAuthorizedApplicationFn func(
		ctx context.Context,
		namespace string,
		name string,
		stageMeta metav1.ObjectMeta,
	) (*argocd.Application, error)
	applyArgoCDSourceUpdateFn func(
		argocd.ApplicationSource,
		kargoapi.FreightReference,
		kargoapi.ArgoCDSourceUpdate,
	) (argocd.ApplicationSource, error)
	argoCDAppPatchFn func(
		ctx context.Context,
		obj client.Object,
		patch client.Patch,
		opts ...client.PatchOption,
	) error
	logAppEventFn func(ctx context.Context, app *argocd.Application, user, reason, message string)
}

// newArgoCDMechanism returns an implementation of the Mechanism interface that
// updates Argo CD Application resources.
func newArgoCDMechanism(argocdClient client.Client) Mechanism {
	a := &argoCDMechanism{
		argocdClient: argocdClient,
	}
	a.buildDesiredSourcesFn = a.buildDesiredSources
	a.mustPerformUpdateFn = a.mustPerformUpdate
	a.updateApplicationSourcesFn = a.updateApplicationSources
	a.getAuthorizedApplicationFn = a.getAuthorizedApplication
	a.applyArgoCDSourceUpdateFn = applyArgoCDSourceUpdate
	if argocdClient != nil {
		a.argoCDAppPatchFn = argocdClient.Patch
		a.logAppEventFn = a.logAppEvent
	}
	return a
}

// GetName implements the Mechanism interface.
func (*argoCDMechanism) GetName() string {
	return "Argo CD promotion mechanism"
}

// Promote implements the Mechanism interface.
func (a *argoCDMechanism) Promote(
	ctx context.Context,
	stage *kargoapi.Stage,
	promo *kargoapi.Promotion,
	newFreight kargoapi.FreightReference,
) (*kargoapi.PromotionStatus, kargoapi.FreightReference, error) {
	updates := stage.Spec.PromotionMechanisms.ArgoCDAppUpdates

	if len(updates) == 0 {
		return promo.Status.WithPhase(kargoapi.PromotionPhaseSucceeded), newFreight, nil
	}

	if a.argocdClient == nil {
		return promo.Status.WithPhase(kargoapi.PromotionPhaseFailed), newFreight,
			errors.New(
				"Argo CD integration is disabled on this controller; cannot perform " +
					"promotion",
			)
	}

	logger := logging.LoggerFromContext(ctx)
	logger.Debug("executing Argo CD-based promotion mechanisms")

	var updateResults = make([]argocd.OperationPhase, 0, len(updates))
	var newStatus = promo.Status.DeepCopy()
	for _, update := range updates {
		// Retrieve the Argo CD Application.
		app, err := a.getAuthorizedApplicationFn(ctx, update.AppNamespace, update.AppName, stage.ObjectMeta)
		if err != nil {
			return nil, newFreight, err
		}

		// Build the desired source(s) for the Argo CD Application.
		desiredSource, desiredSources, err := a.buildDesiredSourcesFn(
			app,
			update,
			newFreight,
		)
		if err != nil {
			return nil, newFreight, err
		}

		// Check if the update needs to be performed and retrieve its phase.
		phase, mustUpdate, err := a.mustPerformUpdateFn(app, update, newFreight, desiredSource, desiredSources)

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
					return nil, newFreight, err
				}
				// Log the error as a warning, but continue to the next update.
				logger.Info(err.Error())
			}
			if phase.Failed() {
				// Record the reason for the failure if available.
				if app.Status.OperationState != nil {
					newStatus.Message = fmt.Sprintf(
						"Argo CD Application %q in namespace %q failed with: %s",
						app.Name,
						app.Namespace,
						app.Status.OperationState.Message,
					)
				}

				// If the update failed, we can short-circuit. This is
				// effectively "fail fast" behavior.
				break
			}
			// If we get here, we can continue to the next update.
			continue
		}

		// Perform the update.
		if err := a.updateApplicationSourcesFn(ctx, app, desiredSource, desiredSources); err != nil {
			return nil, newFreight, err
		}
		// As we have initiated an update, we should wait for it to complete.
		updateResults = append(updateResults, argocd.OperationRunning)
	}

	aggregatedPhase := operationPhaseToPromotionPhase(updateResults...)
	if aggregatedPhase == "" {
		return nil, newFreight, fmt.Errorf(
			"could not determine promotion phase from operation phases: %v",
			updateResults,
		)
	}

	logger.Debug("done executing Argo CD-based promotion mechanisms")
	newStatus.Phase = aggregatedPhase
	return newStatus, newFreight, nil
}

// buildDesiredSources returns the desired source(s) for an Argo CD Application,
// by updating the current source(s) with the given source updates.
func (a *argoCDMechanism) buildDesiredSources(
	app *argocd.Application,
	update kargoapi.ArgoCDAppUpdate,
	newFreight kargoapi.FreightReference,
) (*argocd.ApplicationSource, argocd.ApplicationSources, error) {
	desiredSource, desiredSources := app.Spec.Source.DeepCopy(), app.Spec.Sources.DeepCopy()

	for _, srcUpdate := range update.SourceUpdates {
		if desiredSource != nil {
			newSrc, err := a.applyArgoCDSourceUpdateFn(*desiredSource, newFreight, srcUpdate)
			if err != nil {
				return nil, nil, fmt.Errorf(
					"error applying source update to Argo CD Application %q in namespace %q: %w",
					update.AppName,
					app.Namespace,
					err,
				)
			}
			desiredSource = &newSrc
		}

		for i, curSrc := range desiredSources {
			newSrc, err := a.applyArgoCDSourceUpdateFn(curSrc, newFreight, srcUpdate)
			if err != nil {
				return nil, nil, fmt.Errorf(
					"error applying source update to Argo CD Application %q in namespace %q: %w",
					update.AppName,
					app.Namespace,
					err,
				)
			}
			desiredSources[i] = newSrc
		}
	}

	return desiredSource, desiredSources, nil
}

func (a *argoCDMechanism) mustPerformUpdate(
	app *argocd.Application,
	update kargoapi.ArgoCDAppUpdate,
	newFreight kargoapi.FreightReference,
	desiredSource *argocd.ApplicationSource,
	desiredSources argocd.ApplicationSources,
) (phase argocd.OperationPhase, mustUpdate bool, err error) {
	status := app.Status.OperationState
	if status == nil {
		// The application has no operation.
		return "", true, nil
	}

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

	if !status.Phase.Completed() {
		// The operation is still running.
		return status.Phase, false, nil
	}

	if status.SyncResult == nil {
		// We do not have a sync result, so we cannot determine if the operation
		// was successful. The best recourse is to retry the operation.
		return "", true, errors.New("operation completed without a sync result")
	}

	// Check if the desired revision was applied.
	desiredRevision := libargocd.GetDesiredRevision(app, newFreight)
	if desiredRevision != "" && status.SyncResult.Revision != desiredRevision {
		// The operation did not result in the desired revision being applied.
		// We should attempt to retry the operation.
		return "", true, fmt.Errorf(
			"operation result revision %q does not match desired revision %q",
			status.SyncResult.Revision, desiredRevision,
		)
	}

	// Check if the desired source(s) were applied.
	if len(update.SourceUpdates) > 0 {
		if (desiredSource != nil && !desiredSource.Equals(&status.SyncResult.Source)) ||
			!desiredSources.Equals(status.SyncResult.Sources) {
			// The operation did not result in the desired source(s) being applied.
			// We should attempt to retry the operation.
			return "", true, fmt.Errorf(
				"operation result source does not match desired source",
			)
		}
	}

	// The operation has completed.
	return status.Phase, false, nil
}

func (a *argoCDMechanism) updateApplicationSources(
	ctx context.Context,
	app *argocd.Application,
	desiredSource *argocd.ApplicationSource,
	desiredSources argocd.ApplicationSources,
) error {
	// Create a patch for the Application.
	patch := client.MergeFrom(app.DeepCopy())

	// Initiate a "hard" refresh.
	app.ObjectMeta.Annotations[argocd.AnnotationKeyRefresh] = string(argocd.RefreshTypeHard)

	// Update the desired source(s) in the Argo CD Application.
	app.Spec.Source = desiredSource.DeepCopy()
	app.Spec.Sources = desiredSources.DeepCopy()

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

	// Patch the Application with the changes from above.
	if err := a.argoCDAppPatchFn(
		ctx,
		app,
		patch,
	); err != nil {
		return fmt.Errorf("error patching Argo CD Application %q: %w", app.Name, err)
	}
	logging.LoggerFromContext(ctx).Debug(
		"patched Argo CD Application",
		"app", app.Name,
	)

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
	a.logAppEventFn(ctx, app, "kargo-controller", argocd.EventReasonOperationStarted, message)

	return nil
}

func (a *argoCDMechanism) logAppEvent(ctx context.Context, app *argocd.Application, user, reason, message string) {
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
func (a *argoCDMechanism) getAuthorizedApplication(
	ctx context.Context,
	namespace string,
	name string,
	stageMeta metav1.ObjectMeta,
) (*argocd.Application, error) {
	if namespace == "" {
		namespace = libargocd.Namespace()
	}

	app, err := argocd.GetApplication(ctx, a.argocdClient, namespace, name)
	if err != nil {
		return nil, fmt.Errorf("error finding Argo CD Application %q in namespace %q: %w", name, namespace, err)
	}
	if app == nil {
		return nil, fmt.Errorf("unable to find Argo CD Application %q in namespace %q", name, namespace)
	}

	if err = authorizeArgoCDAppUpdate(stageMeta, app.ObjectMeta); err != nil {
		return nil, err
	}

	return app, nil
}

// authorizeArgoCDAppUpdate returns an error if the Argo CD Application
// represented by appMeta does not explicitly permit mutation by the Kargo Stage
// represented by stageMeta.
func authorizeArgoCDAppUpdate(
	stageMeta metav1.ObjectMeta,
	appMeta metav1.ObjectMeta,
) error {
	permErr := fmt.Errorf(
		"Argo CD Application %q in namespace %q does not permit mutation by "+
			"Kargo Stage %s in namespace %s",
		appMeta.Name,
		appMeta.Namespace,
		stageMeta.Name,
		stageMeta.Namespace,
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
	if !allowedNamespaceGlob.Match(stageMeta.Namespace) ||
		!allowedNameGlob.Match(stageMeta.Name) {
		return permErr
	}
	return nil
}

// applyArgoCDSourceUpdate updates a single Argo CD ApplicationSource.
func applyArgoCDSourceUpdate(
	source argocd.ApplicationSource,
	newFreight kargoapi.FreightReference,
	update kargoapi.ArgoCDSourceUpdate,
) (argocd.ApplicationSource, error) {
	if source.Chart != "" || update.Chart != "" {
		// Infer that we're dealing with a chart repo. No need to normalize the
		// repo URL here.

		// Kargo uses the "oci://" prefix, but Argo CD does not.
		if source.RepoURL != strings.TrimPrefix(update.RepoURL, "oci://") || source.Chart != update.Chart {
			return source, nil
		}
		// If we get to here, we have confirmed that this update is applicable to
		// this source.
		//
		// Now find the chart in the new freight that corresponds to this
		// source.
		for _, chart := range newFreight.Charts {
			// path.Join accounts for the possibility that chart.Name is empty
			//
			// Kargo uses the "oci://" prefix, but Argo CD does not.
			if path.Join(strings.TrimPrefix(chart.RepoURL, "oci://"), chart.Name) == path.Join(source.RepoURL, source.Chart) {
				source.TargetRevision = chart.Version
				break
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
		//
		// Now find the commit in the new freight that corresponds to this source.
		for _, commit := range newFreight.Commits {
			if git.NormalizeURL(commit.RepoURL) == sourceRepoURL {
				if commit.Tag != "" {
					source.TargetRevision = commit.Tag
				} else {
					source.TargetRevision = commit.ID
				}
				break
			}
		}
	}

	if update.Kustomize != nil && len(update.Kustomize.Images) > 0 {
		if source.Kustomize == nil {
			source.Kustomize = &argocd.ApplicationSourceKustomize{}
		}
		source.Kustomize.Images = buildKustomizeImagesForArgoCDAppSource(
			newFreight.Images,
			update.Kustomize.Images,
		)
	}

	if update.Helm != nil && len(update.Helm.Images) > 0 {
		if source.Helm == nil {
			source.Helm = &argocd.ApplicationSourceHelm{}
		}
		if source.Helm.Parameters == nil {
			source.Helm.Parameters = []argocd.HelmParameter{}
		}
		changes := buildHelmParamChangesForArgoCDAppSource(
			newFreight.Images,
			update.Helm.Images,
		)
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

func buildKustomizeImagesForArgoCDAppSource(
	images []kargoapi.Image,
	imageUpdates []kargoapi.ArgoCDKustomizeImageUpdate,
) argocd.KustomizeImages {
	tagsByImage := make(map[string]string, len(images))
	digestsByImage := make(map[string]string, len(images))
	for _, image := range images {
		tagsByImage[image.RepoURL] = image.Tag
		digestsByImage[image.RepoURL] = image.Digest
	}
	kustomizeImages := make(argocd.KustomizeImages, 0, len(imageUpdates))
	for _, imageUpdate := range imageUpdates {
		tag, tagFound := tagsByImage[imageUpdate.Image]
		digest, digestFound := digestsByImage[imageUpdate.Image]
		if !tagFound && !digestFound {
			// There's no change to make in this case.
			continue
		}
		var kustomizeImageStr string
		if imageUpdate.UseDigest {
			kustomizeImageStr =
				fmt.Sprintf("%s=%s@%s", imageUpdate.Image, imageUpdate.Image, digest)
		} else {
			kustomizeImageStr =
				fmt.Sprintf("%s=%s:%s", imageUpdate.Image, imageUpdate.Image, tag)
		}
		kustomizeImages = append(
			kustomizeImages,
			argocd.KustomizeImage(kustomizeImageStr),
		)
	}
	return kustomizeImages
}

// buildHelmParamChangesForArgoCDAppSource takes a list of images and a list of
// instructions about changes that should be made to various Helm parameters and
// distills them into a map of new values indexed by parameter name.
func buildHelmParamChangesForArgoCDAppSource(
	images []kargoapi.Image,
	imageUpdates []kargoapi.ArgoCDHelmImageUpdate,
) map[string]string {
	tagsByImage := make(map[string]string, len(images))
	digestsByImage := make(map[string]string, len(images))
	for _, image := range images {
		tagsByImage[image.RepoURL] = image.Tag
		digestsByImage[image.RepoURL] = image.Digest
	}
	changes := map[string]string{}
	for _, imageUpdate := range imageUpdates {
		switch imageUpdate.Value {
		case kargoapi.ImageUpdateValueTypeImageAndTag,
			kargoapi.ImageUpdateValueTypeTag,
			kargoapi.ImageUpdateValueTypeImageAndDigest,
			kargoapi.ImageUpdateValueTypeDigest:
		default:
			// This really shouldn't happen, so we'll ignore it.
			continue
		}
		tag, tagFound := tagsByImage[imageUpdate.Image]
		digest, digestFound := digestsByImage[imageUpdate.Image]
		if !tagFound && !digestFound {
			// There's no change to make in this case.
			continue
		}
		switch imageUpdate.Value {
		case kargoapi.ImageUpdateValueTypeImageAndTag:
			changes[imageUpdate.Key] = fmt.Sprintf("%s:%s", imageUpdate.Image, tag)
		case kargoapi.ImageUpdateValueTypeTag:
			changes[imageUpdate.Key] = tag
		case kargoapi.ImageUpdateValueTypeImageAndDigest:
			changes[imageUpdate.Key] = fmt.Sprintf("%s@%s", imageUpdate.Image, digest)
		case kargoapi.ImageUpdateValueTypeDigest:
			changes[imageUpdate.Key] = digest
		}
	}
	return changes
}

func operationPhaseToPromotionPhase(phases ...argocd.OperationPhase) kargoapi.PromotionPhase {
	if len(phases) == 0 {
		return ""
	}

	sort.Sort(libargocd.ByOperationPhase(phases))

	switch phases[0] {
	case argocd.OperationRunning, argocd.OperationTerminating:
		return kargoapi.PromotionPhaseRunning
	case argocd.OperationFailed, argocd.OperationError:
		return kargoapi.PromotionPhaseFailed
	case argocd.OperationSucceeded:
		return kargoapi.PromotionPhaseSucceeded
	default:
		return ""
	}
}
