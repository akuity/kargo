package analysis

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
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/logging"
)

// reconciler reconciles Argo Rollouts AnalysisRun resources.
type reconciler struct {
	kubeClient client.Client
}

// SetupReconcilerWithManager initializes a reconciler for Argo Rollouts
// AnalysisRun resources and registers it with the provided Manager.
func SetupReconcilerWithManager(
	ctx context.Context,
	kargoMgr manager.Manager,
	analysisMgr manager.Manager,
	shardName string,
) error {
	// Index Stages by AnalysisRun
	if err := kubeclient.IndexStagesByAnalysisRun(ctx, kargoMgr, shardName); err != nil {
		return errors.Wrap(err, "index Stages by Argo Rollouts AnalysisRun")
	}
	logger := logging.LoggerFromContext(ctx)
	return ctrl.NewControllerManagedBy(analysisMgr).
		For(&rollouts.AnalysisRun{}).
		WithEventFilter(analysisRunPhaseChangePredicate{logger: logger}).
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
		"analysisRunNamespace": req.NamespacedName.Namespace,
		"analysisRun":          req.NamespacedName.Name,
	})
	logger.Debug("reconciling Argo Rollouts AnalysisRun")

	// Find the Stage associated with this AnalysisRun
	stages := &kargoapi.StageList{}
	if err := r.kubeClient.List(
		ctx,
		stages,
		&client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(
				kubeclient.StagesByAnalysisRunIndexField,
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
			"error listing Stages for AnalysisRun %q in namespace %q",
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

type analysisRunPhaseChangePredicate struct {
	predicate.Funcs

	logger *log.Entry
}

// Update implements default UpdateEvent filter for checking if AnalysisRun
// changed phase.
func (a analysisRunPhaseChangePredicate) Update(e event.UpdateEvent) bool {
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
		a.logger.Errorf("Failed to convert new AnalysisRun: %v", e.ObjectNew)
		return false
	}
	oldUn, err := runtime.DefaultUnstructuredConverter.ToUnstructured(e.ObjectOld)
	if err != nil {
		a.logger.Errorf("Failed to convert old AnalysisRun: %v", e.ObjectOld)
		return false
	}
	oldPhase, _, _ := unstructured.NestedString(oldUn, "status", "phase")
	newPhase, _, _ := unstructured.NestedString(newUn, "status", "phase")
	return newPhase != oldPhase
}
