package promotion

import (
	"context"
	"fmt"
	"strings"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/gobwas/glob"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libArgoCD "github.com/akuity/kargo/internal/argocd"
	"github.com/akuity/kargo/internal/logging"
)

const authorizedStageAnnotationKey = "kargo.akuity.io/authorized-stage"

// argoCDMechanism is an implementation of the Mechanism interface that updates
// Argo CD Application resources.
type argoCDMechanism struct {
	// These behaviors are overridable for testing purposes:
	doSingleUpdateFn func(
		ctx context.Context,
		stageMeta metav1.ObjectMeta,
		update kargoapi.ArgoCDAppUpdate,
		newFreight kargoapi.Freight,
	) error
	getArgoCDAppFn func(
		ctx context.Context,
		namespace string,
		name string,
	) (*argocd.Application, error)
	applyArgoCDSourceUpdateFn func(
		argocd.ApplicationSource,
		kargoapi.Freight,
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
func newArgoCDMechanism(
	argoClient client.Client,
) Mechanism {
	a := &argoCDMechanism{}
	a.doSingleUpdateFn = a.doSingleUpdate
	a.getArgoCDAppFn = getApplicationFn(argoClient)
	a.applyArgoCDSourceUpdateFn = applyArgoCDSourceUpdate
	a.argoCDAppPatchFn = argoClient.Patch
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
	newFreight kargoapi.Freight,
) (kargoapi.Freight, error) {
	updates := stage.Spec.PromotionMechanisms.ArgoCDAppUpdates

	if len(updates) == 0 {
		return newFreight, nil
	}

	logger := logging.LoggerFromContext(ctx)
	logger.Debug("executing Argo CD-based promotion mechanisms")

	for _, update := range updates {
		if err := a.doSingleUpdateFn(
			ctx,
			stage.ObjectMeta,
			update,
			newFreight,
		); err != nil {
			return newFreight, err
		}
	}

	logger.Debug("done executing Argo CD-based promotion mechanisms")

	return newFreight, nil
}

func (a *argoCDMechanism) doSingleUpdate(
	ctx context.Context,
	stageMeta metav1.ObjectMeta,
	update kargoapi.ArgoCDAppUpdate,
	newFreight kargoapi.Freight,
) error {
	app, err :=
		a.getArgoCDAppFn(ctx, update.AppNamespace, update.AppName)
	if err != nil {
		return errors.Wrapf(
			err,
			"error finding Argo CD Application %q in namespace %q",
			update.AppName,
			update.AppNamespace,
		)
	}
	if app == nil {
		return errors.Errorf(
			"unable to find Argo CD Application %q in namespace %q",
			update.AppName,
			update.AppNamespace,
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
				return errors.Wrapf(
					err,
					"error updating source of Argo CD Application %q in namespace %q",
					update.AppName,
					update.AppNamespace,
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
				return errors.Wrapf(
					err,
					"error updating source(s) of Argo CD Application %q in namespace %q",
					update.AppName,
					update.AppNamespace,
				)
			}
			app.Spec.Sources[i] = source
		}
	}
	app.ObjectMeta.Annotations[argocd.AnnotationKeyRefresh] =
		string(argocd.RefreshTypeHard)
	app.Operation = &argocd.Operation{
		Sync: &argocd.SyncOperation{
			Revisions: []string{},
		},
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
		&client.PatchOptions{},
	); err != nil {
		return errors.Wrapf(err, "error patching Argo CD Application %q", app.Name)
	}
	logging.LoggerFromContext(ctx).WithField("app", app.Name).
		Debug("patched Argo CD Application")
	return nil
}

func getApplicationFn(
	argoClient client.Client,
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
		return libArgoCD.GetApplication(ctx, argoClient, namespace, name)
	}
}

// authorizeArgoCDAppUpdate returns an error if the Argo CD Application
// represented by appMeta does not explicitly permit mutation by the Kargo Stage
// represented by stageMeta.
func authorizeArgoCDAppUpdate(
	stageMeta metav1.ObjectMeta,
	appMeta metav1.ObjectMeta,
) error {
	permErr := errors.Errorf(
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
		return errors.Errorf(
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
		return errors.Errorf(
			"Argo CD Application %q in namespace %q has invalid glob expression: %q",
			appMeta.Name,
			appMeta.Namespace,
			tokens[0],
		)
	}
	allowedNameGlob, err := glob.Compile(tokens[1])
	if err != nil {
		return errors.Errorf(
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
	newFreight kargoapi.Freight,
	update kargoapi.ArgoCDSourceUpdate,
) (argocd.ApplicationSource, error) {
	if source.RepoURL != update.RepoURL || source.Chart != update.Chart {
		return source, nil
	}

	if update.UpdateTargetRevision {
		var done bool
		for _, commit := range newFreight.Commits {
			if commit.RepoURL == source.RepoURL {
				source.TargetRevision = commit.ID
				done = true
				break
			}
		}
		if !done {
			for _, chart := range newFreight.Charts {
				if chart.RegistryURL == source.RepoURL && chart.Name == source.Chart {
					source.TargetRevision = chart.Version
					break
				}
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
	imageUpdates []string,
) argocd.KustomizeImages {
	tagsByImage := map[string]string{}
	for _, image := range images {
		tagsByImage[image.RepoURL] = image.Tag
	}
	kustomizeImages := argocd.KustomizeImages{}
	for _, imageUpdate := range imageUpdates {
		tag, found := tagsByImage[imageUpdate]
		if !found {
			// There's no change to make in this case.
			continue
		}
		kustomizeImages = append(
			kustomizeImages,
			argocd.KustomizeImage(
				fmt.Sprintf("%s=%s:%s", imageUpdate, imageUpdate, tag),
			),
		)
	}
	return kustomizeImages
}

func buildHelmParamChangesForArgoCDAppSource(
	images []kargoapi.Image,
	imageUpdates []kargoapi.ArgoCDHelmImageUpdate,
) map[string]string {
	tagsByImage := map[string]string{}
	for _, image := range images {
		tagsByImage[image.RepoURL] = image.Tag
	}
	changes := map[string]string{}
	for _, imageUpdate := range imageUpdates {
		if imageUpdate.Value != kargoapi.ImageUpdateValueTypeImage &&
			imageUpdate.Value != kargoapi.ImageUpdateValueTypeTag {
			// This really shouldn't happen, so we'll ignore it.
			continue
		}
		tag, found := tagsByImage[imageUpdate.Image]
		if !found {
			// There's no change to make in this case.
			continue
		}
		if imageUpdate.Value == kargoapi.ImageUpdateValueTypeImage {
			changes[imageUpdate.Key] = fmt.Sprintf("%s:%s", imageUpdate.Image, tag)
		} else {
			changes[imageUpdate.Key] = tag
		}
	}
	return changes
}
