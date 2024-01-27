package projects

import (
	"context"

	log "github.com/sirupsen/logrus"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/logging"
)

// reconciler reconciles Project resources.
type reconciler struct {
	client client.Client

	// The following behaviors are overridable for testing purposes:

	getProjectFn func(
		context.Context,
		client.Client,
		string,
	) (*kargoapi.Project, error)

	syncProjectFn func(
		context.Context,
		*kargoapi.Project,
	) (kargoapi.ProjectStatus, error)

	patchProjectStatusFn func(
		context.Context,
		*kargoapi.Project,
		kargoapi.ProjectStatus,
	) error
}

// SetupReconcilerWithManager initializes a reconciler for Project resources and
// registers it with the provided Manager.
func SetupReconcilerWithManager(kargoMgr manager.Manager) error {
	return ctrl.NewControllerManagedBy(kargoMgr).
		For(&kargoapi.Project{}).
		WithEventFilter(
			predicate.Funcs{
				DeleteFunc: func(event.DeleteEvent) bool {
					// We're not interested in any deletes
					return false
				},
			},
		).
		WithOptions(controller.CommonOptions()).
		Complete(newReconciler(kargoMgr.GetClient()))
}

func newReconciler(kubeClient client.Client) *reconciler {
	r := &reconciler{
		client: kubeClient,
	}
	r.getProjectFn = kargoapi.GetProject
	r.syncProjectFn = r.syncProject
	r.patchProjectStatusFn = r.patchProjectStatus
	return r
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *reconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	result := ctrl.Result{
		// Note: If there is a failure, controller runtime ignores this and uses
		// progressive backoff instead. So this value only prevents requeueing
		// a Project if THIS reconciliation succeeds.
		RequeueAfter: 0,
	}

	logger := logging.LoggerFromContext(ctx).WithFields(log.Fields{
		"project": req.NamespacedName.Name,
	})
	ctx = logging.ContextWithLogger(ctx, logger)
	logger.Debug("reconciling Project")

	// Find the Project
	project, err := r.getProjectFn(ctx, r.client, req.NamespacedName.Name)
	if err != nil {
		return result, err
	}
	if project == nil {
		// Ignore if not found. This can happen if the Project was deleted after the
		// current reconciliation request was issued.
		return result, nil
	}

	if project.DeletionTimestamp != nil {
		logger.Debug("Project is being deleted; nothing to do")
		return result, nil
	}

	if project.Status.Phase.IsTerminal() {
		logger.Debugf("Project is %s; nothing to do", project.Status.Phase)
		return result, nil
	}

	newStatus, err := r.syncProjectFn(ctx, project)
	if err != nil {
		newStatus.Message = err.Error()
		logger.Errorf("error syncing Project: %s", err)
	} else {
		// Be sure to blank this out in case there's an error in this field from
		// the previous reconciliation
		newStatus.Message = ""
	}

	patchErr := r.patchProjectStatusFn(ctx, project, newStatus)
	if patchErr != nil {
		logger.Errorf("error updating Project status: %s", patchErr)
	}

	// If we had no error, but couldn't patch, then we DO have an error. But we
	// do it this way so that a failure to patch is never counted as THE failure
	// when something else more serious occurred first.
	if err == nil {
		err = patchErr
	}
	logger.Debug("done reconciling Project")

	// Controller runtime automatically gives us a progressive backoff if err is
	// not nil
	return result, err
}

func (r *reconciler) syncProject(
	_ context.Context,
	project *kargoapi.Project,
) (kargoapi.ProjectStatus, error) {
	status := *project.Status.DeepCopy()
	// TODO: This used to create the Project's associated namespace, but the
	// webhook now does that. This remains because it is where we will add
	// creation of other Project-owned resources in the future.
	status.Phase = kargoapi.ProjectPhaseReady
	return status, nil
}

func (r *reconciler) patchProjectStatus(
	ctx context.Context,
	project *kargoapi.Project,
	status kargoapi.ProjectStatus,
) error {
	return kubeclient.PatchStatus(
		ctx,
		r.client,
		project,
		func(s *kargoapi.ProjectStatus) {
			*s = status
		},
	)
}
