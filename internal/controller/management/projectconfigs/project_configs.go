package projectconfigs

import (
	"context"
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/conditions"
	"github.com/akuity/kargo/internal/controller"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/internal/webhook/external"
)

type ReconcilerConfig struct {
	ExternalWebhookServerBaseURL string `envconfig:"EXTERNAL_WEBHOOK_SERVER_BASE_URL" required:"true"`
	MaxConcurrentReconciles      int    `envconfig:"MAX_CONCURRENT_PROJECT_CONFIG_RECONCILES" default:"4"`
}

func ReconcilerConfigFromEnv() ReconcilerConfig {
	cfg := ReconcilerConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

type reconciler struct {
	cfg    ReconcilerConfig
	client client.Client
}

func SetupReconcilerWithManager(
	ctx context.Context,
	kargoMgr manager.Manager,
	cfg ReconcilerConfig,
) error {
	_, err := ctrl.NewControllerManagedBy(kargoMgr).
		For(&kargoapi.ProjectConfig{}).
		WithEventFilter(
			predicate.Funcs{
				DeleteFunc: func(event.DeleteEvent) bool {
					return false
				},
			},
		).
		WithOptions(controller.CommonOptions(cfg.MaxConcurrentReconciles)).
		Build(newReconciler(kargoMgr.GetClient(), cfg))
	if err != nil {
		return fmt.Errorf("error creating ProjectConfig reconciler: %w", err)
	}

	logging.LoggerFromContext(ctx).Info(
		"Initialized ProjectConfig reconciler",
		"maxConcurrentReconciles", cfg.MaxConcurrentReconciles,
	)
	return nil
}

func newReconciler(kubeClient client.Client, cfg ReconcilerConfig) *reconciler {
	return &reconciler{
		cfg:    cfg,
		client: kubeClient,
	}
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *reconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"project-config", req.NamespacedName.Name,
	)
	ctx = logging.ContextWithLogger(ctx, logger)

	// Fetch the ProjectConfig
	projectConfig, err := api.GetProjectConfig(ctx, r.client, req.NamespacedName.Name)
	if err != nil {
		return ctrl.Result{}, err
	}
	if projectConfig == nil {
		// Ignore if not found. This can happen if the ProjectConfig was deleted
		// after the current reconciliation request was issued.
		return ctrl.Result{}, nil
	}

	if !projectConfig.DeletionTimestamp.IsZero() {
		logger.Debug("ProjectConfig is being deleted; nothing to do")
		return ctrl.Result{}, nil
	}

	logger.Debug("reconciling ProjectConfig")
	newStatus, reconcileErr := r.reconcile(ctx, projectConfig)
	logger.Debug("done reconciling ProjectConfig")

	// Patch the status of the ProjectConfig.
	if err := kubeclient.PatchStatus(
		ctx,
		r.client,
		projectConfig,
		func(status *kargoapi.ProjectConfigStatus) {
			*status = newStatus
		},
	); err != nil {
		// Prioritize the reconcile error if it exists.
		if reconcileErr != nil {
			logger.Error(
				err,
				"failed to update ProjectConfig status after reconciliation error",
			)
			return ctrl.Result{}, reconcileErr
		}
		return ctrl.Result{},
			fmt.Errorf("failed to update ProjectConfig status: %w", err)
	}

	// Return the reconcile error if it exists.
	if reconcileErr != nil {
		return ctrl.Result{}, reconcileErr
	}

	// TODO: Make the requeue delay configurable.
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *reconciler) reconcile(
	ctx context.Context,
	projectCfg *kargoapi.ProjectConfig,
) (
	kargoapi.ProjectConfigStatus,
	error,
) {
	logger := logging.LoggerFromContext(ctx)
	status := *projectCfg.Status.DeepCopy()

	subReconcilers := []struct {
		name      string
		reconcile func() (kargoapi.ProjectConfigStatus, error)
	}{{
		name: "syncing WebhookReceivers",
		reconcile: func() (kargoapi.ProjectConfigStatus, error) {
			return r.syncWebhookReceivers(ctx, projectCfg)
		},
	}}
	for _, subR := range subReconcilers {
		logger.Debug(subR.name)

		// Reconcile the ProjectConfig with the sub-reconciler.
		var err error
		status, err = subR.reconcile()

		// If an error occurred during the sub-reconciler, then we should return the
		// error which will cause the ProjectConfig to be requeued.
		if err != nil {
			return status, err
		}

		// Patch the status of the ProjectConfig after each sub-reconciler to show
		// progress.
		if err = kubeclient.PatchStatus(
			ctx,
			r.client,
			projectCfg,
			func(st *kargoapi.ProjectConfigStatus) { *st = status },
		); err != nil {
			logger.Error(
				err,
				fmt.Sprintf("failed to update Project status after %s", subR.name),
			)
		}
	}

	return status, nil
}

func (r *reconciler) syncWebhookReceivers(
	ctx context.Context,
	projectCfg *kargoapi.ProjectConfig,
) (kargoapi.ProjectConfigStatus, error) {
	logger := logging.LoggerFromContext(ctx)
	status := projectCfg.Status.DeepCopy()

	if len(projectCfg.Spec.WebhookReceivers) == 0 {
		logger.Debug("ProjectConfig does not define any webhook receiver configurations")
		conditions.Delete(status, kargoapi.ConditionTypeReconciling)
		conditions.Set(status, &metav1.Condition{
			Type:               kargoapi.ConditionTypeReady,
			Status:             metav1.ConditionTrue,
			Reason:             "Synced",
			Message:            "ProjectConfig is synced and ready for use",
			ObservedGeneration: projectCfg.GetGeneration(),
		})
		return *status, nil
	}

	conditions.Set(status, &metav1.Condition{
		Type:               kargoapi.ConditionTypeReconciling,
		Status:             metav1.ConditionTrue,
		Reason:             "SyncingWebhooks",
		Message:            "Syncing WebhookReceivers",
		ObservedGeneration: projectCfg.GetGeneration(),
	})
	conditions.Set(status, &metav1.Condition{
		Type:               kargoapi.ConditionTypeReady,
		Status:             metav1.ConditionFalse,
		Reason:             "SyncingWebhooks",
		Message:            "Syncing WebhookReceivers",
		ObservedGeneration: projectCfg.GetGeneration(),
	})

	logger.Debug("syncing WebhookReceivers")

	var errs []error
	status.WebhookReceivers = make(
		[]kargoapi.WebhookReceiverDetails,
		0,
		len(projectCfg.Spec.WebhookReceivers),
	)
	for _, receiverCfg := range projectCfg.Spec.WebhookReceivers {
		receiver, err := external.NewReceiver(
			ctx,
			r.client,
			r.cfg.ExternalWebhookServerBaseURL,
			projectCfg.Name,
			projectCfg.Name, // Secret namespace is the same as the Project name/namespace
			receiverCfg,
		)
		if err != nil {
			errs = append(
				errs,
				fmt.Errorf("error syncing WebhookReceiver %q: %w", receiverCfg.Name, err),
			)
			continue
		}
		status.WebhookReceivers = append(status.WebhookReceivers, receiver.GetDetails())
	}

	if len(errs) != 0 {
		flattenedErrs := kerrors.Flatten(kerrors.NewAggregate(errs))
		conditions.Set(status, &metav1.Condition{
			Type:               kargoapi.ConditionTypeReady,
			Status:             metav1.ConditionFalse,
			Reason:             "SyncWebhookReceiversFailed",
			Message:            "Failed to sync WebhookReceivers: " + flattenedErrs.Error(),
			ObservedGeneration: projectCfg.GetGeneration(),
		})
		return *status, flattenedErrs
	}

	conditions.Delete(status, kargoapi.ConditionTypeReconciling)
	conditions.Set(status, &metav1.Condition{
		Type:               kargoapi.ConditionTypeReady,
		Status:             metav1.ConditionTrue,
		Reason:             "Synced",
		Message:            "ProjectConfig is synced and ready for use",
		ObservedGeneration: projectCfg.GetGeneration(),
	})
	return *status, nil
}