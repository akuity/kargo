package promotion

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/gobwas/glob"
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
	mustPerformUpdateFn func(
		ctx context.Context,
		update kargoapi.ArgoCDAppUpdate,
		newFreight kargoapi.FreightReference,
	) (argocd.OperationPhase, bool, error)
	doSingleUpdateFn func(
		ctx context.Context,
		stageMeta metav1.ObjectMeta,
		update kargoapi.ArgoCDAppUpdate,
		newFreight kargoapi.FreightReference,
	) error
	getArgoCDAppFn func(
		ctx context.Context,
		namespace string,
		name string,
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
}

// newArgoCDMechanism returns an implementation of the Mechanism interface that
// updates Argo CD Application resources.
func newArgoCDMechanism(argocdClient client.Client) Mechanism {
	a := &argoCDMechanism{
		argocdClient: argocdClient,
	}
	a.mustPerformUpdateFn = a.mustPerformUpdate
	a.doSingleUpdateFn = a.doSingleUpdate
	a.getArgoCDAppFn = getApplicationFn(argocdClient)
	a.applyArgoCDSourceUpdateFn = applyArgoCDSourceUpdate
	if argocdClient != nil {
		a.argoCDAppPatchFn = argocdClient.Patch
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
	for _, update := range updates {
		// Check if the update needs to be performed and retrieve its phase.
		phase, mustUpdate, err := a.mustPerformUpdateFn(ctx, update, newFreight)

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
				logger.Warn(err)
			}
			if phase.Failed() {
				// If the update failed, we can short-circuit. This is
				// effectively "fail fast" behavior.
				break
			}
			// If we get here, we can continue to the next update.
			continue
		}

		// Perform the update.
		if err := a.doSingleUpdateFn(
			ctx,
			stage.ObjectMeta,
			update,
			newFreight,
		); err != nil {
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
	return promo.Status.WithPhase(aggregatedPhase), newFreight, nil
}

func (a *argoCDMechanism) mustPerformUpdate(
	ctx context.Context,
	update kargoapi.ArgoCDAppUpdate,
	newFreight kargoapi.FreightReference,
) (phase argocd.OperationPhase, mustUpdate bool, err error) {
	namespace := update.AppNamespace
	if namespace == "" {
		namespace = libargocd.Namespace()
	}
	app, err := a.getArgoCDAppFn(ctx, namespace, update.AppName)
	if err != nil {
		return "", false, fmt.Errorf(
			"error finding Argo CD Application %q in namespace %q: %w",
			update.AppName,
			namespace,
			err,
		)
	}
	if app == nil {
		return "", false, fmt.Errorf(
			"unable to find Argo CD Application %q in namespace %q: %w",
			update.AppName,
			namespace,
			err,
		)
	}

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

	// The operation has completed. Check if the desired revision was applied.
	desiredRevision := libargocd.GetDesiredRevision(app, newFreight)
	if status.SyncResult == nil {
		// We do not have a sync result, so we cannot determine if the operation
		// was successful. The best recourse is to retry the operation.
		return "", true, errors.New("operation completed without a sync result")
	}
	if desiredRevision != "" && status.SyncResult.Revision != desiredRevision {
		// The operation did not result in the desired revision being applied.
		// We should attempt to retry the operation.
		return "", true, fmt.Errorf(
			"operation result revision %q does not match desired revision %q",
			status.SyncResult.Revision, desiredRevision,
		)
	}

	// The operation has completed.
	return status.Phase, false, nil
}

func (a *argoCDMechanism) doSingleUpdate(
	ctx context.Context,
	stageMeta metav1.ObjectMeta,
	update kargoapi.ArgoCDAppUpdate,
	newFreight kargoapi.FreightReference,
) error {
	namespace := update.AppNamespace
	if namespace == "" {
		namespace = libargocd.Namespace()
	}
	app, err := a.getArgoCDAppFn(ctx, namespace, update.AppName)
	if err != nil {
		return fmt.Errorf(
			"error finding Argo CD Application %q in namespace %q: %w",
			update.AppName,
			namespace,
			err,
		)
	}
	if app == nil {
		return fmt.Errorf(
			"unable to find Argo CD Application %q in namespace %q: %w",
			update.AppName,
			namespace,
			err,
		)
	}
	// Make sure this is allowed!
	if err = authorizeArgoCDAppUpdate(stageMeta, app.ObjectMeta); err != nil {
		return err
	}
	patch := client.MergeFrom(app.DeepCopy())
	for _, srcUpdate := range update.SourceUpdates {
		if app.Spec.Source != nil {
			var source argocd.ApplicationSource
			if source, err = a.applyArgoCDSourceUpdateFn(
				*app.Spec.Source,
				newFreight,
				srcUpdate,
			); err != nil {
				return fmt.Errorf(
					"error updating source of Argo CD Application %q in namespace %q: %w",
					update.AppName,
					namespace,
					err,
				)
			}
			app.Spec.Source = &source
		}
		for i, source := range app.Spec.Sources {
			if source, err = a.applyArgoCDSourceUpdateFn(
				source,
				newFreight,
				srcUpdate,
			); err != nil {
				return fmt.Errorf(
					"error updating source(s) of Argo CD Application %q in namespace %q: %w",
					update.AppName,
					namespace,
					err,
				)
			}
			app.Spec.Sources[i] = source
		}
	}
	app.ObjectMeta.Annotations[argocd.AnnotationKeyRefresh] =
		string(argocd.RefreshTypeHard)
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
		app.Operation.Sync.Revisions =
			append(app.Operation.Sync.Revisions, source.TargetRevision)
	}
	if err = a.argoCDAppPatchFn(
		ctx,
		app,
		patch,
	); err != nil {
		return fmt.Errorf("error patching Argo CD Application %q: %w", app.Name, err)
	}
	logging.LoggerFromContext(ctx).WithField("app", app.Name).
		Debug("patched Argo CD Application")
	return nil
}

func getApplicationFn(
	argocdClient client.Client,
) func(
	ctx context.Context,
	namespace string,
	name string,
) (*argocd.Application, error) {
	return func(
		ctx context.Context,
		namespace string,
		name string,
	) (*argocd.Application, error) {
		return argocd.GetApplication(ctx, argocdClient, namespace, name)
	}
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
