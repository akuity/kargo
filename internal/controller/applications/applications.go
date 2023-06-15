package applications

import (
	"context"
	"fmt"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/fields"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

const (
	envsByAppIndexField         = "applications"
	forceReconcileAnnotationKey = "kargo.akuity.io/force-reconcile"
)

// reconciler reconciles Argo CD Application resources.
type reconciler struct {
	client client.Client
}

// SetupReconcilerWithManager initializes a reconciler for Argo CD Application
// resources and registers it with the provided Manager.
func SetupReconcilerWithManager(kargoMgr, argoMgr manager.Manager) error {
	return ctrl.NewControllerManagedBy(argoMgr).
		For(&argocd.Application{}).
		Complete(newReconciler(kargoMgr.GetClient()))
}

func newReconciler(client client.Client) *reconciler {
	return &reconciler{
		client: client,
	}
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *reconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	result := ctrl.Result{}

	logger := logging.LoggerFromContext(ctx).WithFields(log.Fields{
		"applicationNamespace": req.NamespacedName.Namespace,
		"application":          req.NamespacedName.Name,
	})
	logger.Debug("reconciling Argo CD Application")

	// Find all Environments associated with this Application
	envs := &api.EnvironmentList{}
	if err := r.client.List(
		ctx,
		envs,
		&client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(
				envsByAppIndexField,
				fmt.Sprintf(
					"%s:%s",
					req.NamespacedName.Namespace,
					req.NamespacedName.Name,
				),
			),
		},
	); err != nil {
		return result, errors.Wrapf(
			err,
			"error listing Environments for Application %q in namespace %q",
			req.NamespacedName.Name,
			req.NamespacedName.Namespace,
		)
	}

	// Force associated Environments to reconcile by patching an annotation
	for _, e := range envs.Items {
		env := e // This is to sidestep implicit memory aliasing in this for loop
		patch := client.MergeFrom(env.DeepCopy())
		env.Annotations[forceReconcileAnnotationKey] = uuid.NewV4().String()
		if err := r.client.Patch(ctx, &env, patch); err != nil {
			logger.Error(err)
			return result, errors.Wrapf(
				err,
				"error patching Environment %q in namespace %q",
				env.Name,
				env.Namespace,
			)
		}
		logger.WithFields(log.Fields{
			"environmentNamespace": env.Namespace,
			"environment":          env.Name,
		}).Debug("successfully patched Environment to force reconciliation")
	}

	return result, nil
}
