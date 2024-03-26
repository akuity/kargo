package upgrade

import (
	"context"

	log "github.com/sirupsen/logrus"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

// stageReconciler reconciles Stage resources to upgrade them from
// v0.4.0-compatible to v0.5.0-compatible.
type stageReconciler struct {
	client client.Client
}

// SetupStageReconcilerWithManager initializes a stageReconciler and registers
// it with the provided Manager.
func SetupStageReconcilerWithManager(mgr manager.Manager) error {
	notV050CompatiblePredicate, err := getNotV050CompatiblePredicate()
	if err != nil {
		return err
	}
	_, err = ctrl.NewControllerManagedBy(mgr).
		For(&kargoapi.Stage{}).
		WithEventFilter(ignoreDeletesPredicate()).
		WithEventFilter(notV050CompatiblePredicate).
		Build(&stageReconciler{
			client: mgr.GetClient(),
		})
	return err
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (s *stageReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	logger := logging.LoggerFromContext(ctx).WithFields(log.Fields{
		"namespace": req.NamespacedName.Namespace,
		"stage":     req.NamespacedName.Name,
	})
	logger.Debug("reconciling Stage")

	// Find the Stage
	stage, err := kargoapi.GetStage(ctx, s.client, req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, err
	}
	if stage == nil {
		// Ignore if not found. This can happen if the Stage was deleted after the
		// current reconciliation request was issued.
		return ctrl.Result{}, nil // Do not requeue
	}

	// Update the Stage to be v0.5.0-compatible
	if stage.Status.CurrentFreight != nil {
		stage.Status.CurrentFreight.Name = stage.Status.CurrentFreight.ID
	}
	for i := range stage.Status.History {
		stage.Status.History[i].Name = stage.Status.History[i].ID
	}
	if err := s.client.Status().Update(ctx, stage); err != nil {
		return ctrl.Result{}, nil
	}

	// If we get to here, patch the Stage with the v0.5.0 compatibility label
	// so that we won't ever have to reconcile it again.
	if err := patchLabel(
		ctx,
		s.client,
		stage,
		v050CompatibilityLabelKey,
		kargoapi.LabelTrueValue,
	); err != nil {
		return ctrl.Result{}, err
	}

	logger.Debug("updated Stage for v0.5.0 compatibility")

	return ctrl.Result{
		Requeue: false,
	}, nil
}
