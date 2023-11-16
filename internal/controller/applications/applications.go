package applications

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/logging"
)

// reconciler reconciles Argo CD Application resources.
type reconciler struct {
	kubeClient client.Client
}

// SetupReconcilerWithManager initializes a reconciler for Argo CD Application
// resources and registers it with the provided Manager.
func SetupReconcilerWithManager(
	ctx context.Context,
	kargoMgr manager.Manager,
	argoMgr manager.Manager,
	shardName string,
) error {
	// Index Stages by Argo CD Applications
	if err := kubeclient.IndexStagesByArgoCDApplications(ctx, kargoMgr, shardName); err != nil {
		return errors.Wrap(err, "index Stages by Argo CD Applications")
	}
	logger := logging.LoggerFromContext(ctx)
	return ctrl.NewControllerManagedBy(argoMgr).
		For(&argocd.Application{}).
		WithEventFilter(AppHealthSyncStatusChangePredicate{logger: logger}).
		WithOptions(controller.CommonOptions()).
		Complete(newReconciler(kargoMgr.GetClient()))
}

func newReconciler(kubeClient client.Client) *reconciler {
	return &reconciler{
		kubeClient: kubeClient,
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

	// Find all Stages associated with this Application
	stages := &kargoapi.StageList{}
	if err := r.kubeClient.List(
		ctx,
		stages,
		&client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(
				kubeclient.StagesByArgoCDApplicationsIndexField,
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
			"error listing Stages for Application %q in namespace %q",
			req.NamespacedName.Name,
			req.NamespacedName.Namespace,
		)
	}

	// Force associated Stages to reconcile by patching an annotation
	errs := make([]error, 0, len(stages.Items))
	for _, e := range stages.Items {
		stage := e // This is to sidestep implicit memory aliasing in this for loop
		objKey := client.ObjectKey{
			Namespace: stage.Namespace,
			Name:      stage.Name,
		}
		_, err := kargoapi.RefreshStage(ctx, r.kubeClient, objKey)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		logger.WithFields(log.Fields{
			"stageNamespace": stage.Namespace,
			"stage":          stage.Name,
		}).Debug("successfully patched Stage to force reconciliation")
	}
	if len(errs) > 0 {
		return result, errs[0]
	}

	return result, nil
}

type AppHealthSyncStatusChangePredicate struct {
	predicate.Funcs

	logger *log.Entry
}

// Update implements default UpdateEvent filter for checking if application changed
// health or sync status. What we detect here should agree with what we examine in
// stages/reconciler.checkHealth()
func (a AppHealthSyncStatusChangePredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		a.logger.Errorf("Update event has no old object to update: %v", e)
		return false
	}
	if e.ObjectNew == nil {
		a.logger.Errorf("Update event has no new object for update: %v", e)
		return false
	}
	newUn, err := runtime.DefaultUnstructuredConverter.ToUnstructured(e.ObjectNew)
	if err != nil {
		a.logger.Errorf("Failed to convert new app: %v", e.ObjectNew)
		return false
	}
	oldUn, err := runtime.DefaultUnstructuredConverter.ToUnstructured(e.ObjectOld)
	if err != nil {
		a.logger.Errorf("Failed to convert old app: %v", e.ObjectOld)
		return false
	}
	oldHealth, _, _ := unstructured.NestedString(oldUn, "status", "health", "status")
	newHealth, _, _ := unstructured.NestedString(newUn, "status", "health", "status")

	// TODO: switch from checking sync status to whether or not operation is complete
	oldSync, _, _ := unstructured.NestedString(oldUn, "status", "sync", "status")
	newSync, _, _ := unstructured.NestedString(newUn, "status", "sync", "status")
	//_, oldOp := oldUn["operation"]
	//_, newOp := newUn["operation"]

	oldRev, _, _ := unstructured.NestedString(oldUn, "status", "sync", "revision")
	newRev, _, _ := unstructured.NestedString(newUn, "status", "sync", "revision")

	return newHealth != oldHealth ||
		oldSync != newSync ||
		oldRev != newRev
}
