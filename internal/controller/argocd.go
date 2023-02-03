package controller

import (
	"context"
	"fmt"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/akuityio/kargo/api/v1alpha1"
)

func (e *environmentReconciler) promoteWithArgoCD(
	ctx context.Context,
	env *api.Environment,
	newState api.EnvironmentState,
) error {
	// If any of the following is true, this function ought not to have been
	// invoked, but we don't take that on faith.
	if env == nil ||
		env.Spec.PromotionMechanisms == nil ||
		env.Spec.PromotionMechanisms.ArgoCD == nil ||
		len(env.Spec.PromotionMechanisms.ArgoCD.AppUpdates) == 0 {
		return nil
	}

	for _, appUpdate := range env.Spec.PromotionMechanisms.ArgoCD.AppUpdates {
		if err := e.updateArgoCDAppFn(
			ctx,
			env,
			newState,
			appUpdate,
		); err != nil {
			return errors.Wrapf(
				err,
				"error updating Argo CD Application %q in namespace %q",
				appUpdate.Name,
				env.Namespace,
			)
		}
	}

	return nil
}

func (e *environmentReconciler) checkHealth(
	ctx context.Context,
	env *api.Environment,
) *api.Health {
	if env.Spec.HealthChecks == nil ||
		len(env.Spec.HealthChecks.ArgoCDApps) == 0 ||
		len(env.Status.States) == 0 {
		return nil
	}
	for _, appName := range env.Spec.HealthChecks.ArgoCDApps {
		app, err := e.getArgoCDAppFn(ctx, env.Namespace, appName)
		if err != nil {
			return &api.Health{
				Status: api.HealthStateUnknown,
				StatusReason: fmt.Sprintf(
					"error finding Argo CD Application %q in namespace %q: %s",
					appName,
					env.Namespace,
					err,
				),
			}
		}
		if app == nil {
			return &api.Health{
				Status: api.HealthStateUnknown,
				StatusReason: fmt.Sprintf(
					"unable to find Argo CD Application %q in namespace %q",
					appName,
					env.Namespace,
				),
			}
		}

		if commit := env.Status.States[0].HealthCheckCommit; commit != "" {
			if synced := e.isArgoCDAppSynced(app, commit); !synced {
				return &api.Health{
					Status: api.HealthStateUnhealthy,
					StatusReason: fmt.Sprintf(
						"Argo CD Application %q in namespace %q is not synced to current "+
							"Environment state",
						appName,
						env.Namespace,
					),
				}
			}
		}

		if app.Status.Health.Status != health.HealthStatusHealthy {
			return &api.Health{
				Status: api.HealthStateUnhealthy,
				StatusReason: fmt.Sprintf(
					"Argo CD Application %q in namespace %q has health state %q",
					appName,
					env.Namespace,
					app.Status.Health.Status,
				),
			}
		}
	}

	return &api.Health{
		Status: api.HealthStateHealthy,
	}
}

// TODO: This probably has more things it needs to take into account, for
// instance, in the event that image substitutions are applied directly to the
// Argo CD App, we had better check that they match the current Environment
// state.
func (e *environmentReconciler) isArgoCDAppSynced(
	app *argocd.Application,
	commit string,
) bool {
	if app == nil || app.Status.Sync.Status != argocd.SyncStatusCodeSynced {
		return false
	}
	return app.Status.Sync.Revision == commit
}

// getArgoCDApp returns a pointer to the Argo CD Application resource specified
// by the namespacedName argument. If no such resource is found, nil is returned
// instead.
func (e *environmentReconciler) getArgoCDApp(
	ctx context.Context,
	namespace string,
	name string,
) (*argocd.Application, error) {
	// TODO: Logging can be improved in this function
	app := argocd.Application{}
	if err := e.client.Get(
		ctx,
		client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		},
		&app,
	); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			e.logger.WithFields(log.Fields{
				"namespace": namespace,
				"name":      name,
			}).Warn("Argo CD Application not found")
			return nil, nil
		}
		return nil, errors.Wrapf(
			err,
			"error getting Argo CD Application %q in namespace %q",
			name,
			namespace,
		)
	}
	return &app, nil
}

func (e *environmentReconciler) updateArgoCDApp(
	ctx context.Context,
	env *api.Environment,
	newState api.EnvironmentState,
	appUpdate api.ArgoCDAppUpdate,
) error {
	app, err := e.getArgoCDAppFn(ctx, env.Namespace, appUpdate.Name)
	if err != nil {
		return errors.Wrapf(
			err,
			"error finding Argo CD Application %q in namespace %q",
			appUpdate.Name,
			env.Namespace,
		)
	}
	if app == nil {
		return errors.Errorf(
			"unable to find Argo CD Application %q in namespace %q",
			appUpdate.Name,
			env.Namespace,
		)
	}

	patch := client.MergeFrom(app.DeepCopy())

	if appUpdate.UpdateTargetRevision && newState.GitCommit != nil {
		app.Spec.Source.TargetRevision = newState.GitCommit.ID
	}

	if appUpdate.Kustomize != nil {
		if app.Spec.Source.Kustomize == nil {
			app.Spec.Source.Kustomize = &argocd.ApplicationSourceKustomize{}
		}
		app.Spec.Source.Kustomize.Images = buildKustomizeImagesForArgoCDApp(
			newState.Images,
			appUpdate.Kustomize.Images,
		)
	} else if appUpdate.Helm != nil {
		if app.Spec.Source.Helm == nil {
			app.Spec.Source.Helm = &argocd.ApplicationSourceHelm{}
		}
		if app.Spec.Source.Helm.Parameters == nil {
			app.Spec.Source.Helm.Parameters = []argocd.HelmParameter{}
		}
		changes :=
			buildHelmParamChangesForArgoCDApp(newState.Images, appUpdate.Helm.Images)
	imageUpdateLoop:
		for k, v := range changes {
			newParam := argocd.HelmParameter{
				Name:  k,
				Value: v,
			}
			for i, param := range app.Spec.Source.Helm.Parameters {
				if param.Name == k {
					app.Spec.Source.Helm.Parameters[i] = newParam
					continue imageUpdateLoop
				}
			}
			app.Spec.Source.Helm.Parameters =
				append(app.Spec.Source.Helm.Parameters, newParam)
		}
	}

	if appUpdate.RefreshAndSync ||
		(appUpdate.UpdateTargetRevision && newState.GitCommit != nil) ||
		appUpdate.Kustomize != nil ||
		appUpdate.Helm != nil {
		app.ObjectMeta.Annotations[argocd.AnnotationKeyRefresh] =
			string(argocd.RefreshTypeHard)
		app.Operation = &argocd.Operation{
			Sync: &argocd.SyncOperation{
				Revision: app.Spec.Source.TargetRevision,
			},
		}
	}

	if err = e.client.Patch(ctx, app, patch, &client.PatchOptions{}); err != nil {
		return errors.Wrapf(err, "error patching Argo CD Application %q", app.Name)
	}
	e.logger.WithFields(log.Fields{
		"namespace": env.Namespace,
		"env":       env.Name,
		"app":       app.Name,
	}).Debug("patched Argo CD Application")

	return nil
}

func buildKustomizeImagesForArgoCDApp(
	images []api.Image,
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

func buildHelmParamChangesForArgoCDApp(
	images []api.Image,
	imageUpdates []api.ArgoCDHelmImageUpdate,
) map[string]string {
	tagsByImage := map[string]string{}
	for _, image := range images {
		tagsByImage[image.RepoURL] = image.Tag
	}
	changes := map[string]string{}
	for _, imageUpdate := range imageUpdates {
		if imageUpdate.Value != api.ImageUpdateValueTypeImage &&
			imageUpdate.Value != api.ImageUpdateValueTypeTag {
			// This really shouldn't happen, so we'll ignore it.
			continue
		}
		tag, found := tagsByImage[imageUpdate.Image]
		if !found {
			// There's no change to make in this case.
			continue
		}
		if imageUpdate.Value == api.ImageUpdateValueTypeImage {
			changes[imageUpdate.Key] = fmt.Sprintf("%s:%s", imageUpdate.Image, tag)
		} else {
			changes[imageUpdate.Key] = tag
		}
	}
	return changes
}
