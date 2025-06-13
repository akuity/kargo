package clusterconfigs

import (
	"context"
	"errors"
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
	ClusterSecretsNamespace      string `envconfig:"CLUSTER_SECRETS_NAMESPACE"`
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
	if _, err := ctrl.NewControllerManagedBy(kargoMgr).
		For(&kargoapi.ClusterConfig{}).
		WithEventFilter(
			predicate.Funcs{
				DeleteFunc: func(event.DeleteEvent) bool {
					return false
				},
			},
		).
		WithOptions(controller.CommonOptions(1)). // There's only ever one ClusterConfig resource
		Build(newReconciler(kargoMgr.GetClient(), cfg)); err != nil {
		return fmt.Errorf("error creating ClusterConfig reconciler: %w", err)
	}
	logging.LoggerFromContext(ctx).Info(
		"Initialized ClusterConfig reconciler",
		"maxConcurrentReconciles", 1,
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
		"clusterConfig", req.NamespacedName.Name,
	)
	ctx = logging.ContextWithLogger(ctx, logger)

	// Find the ClusterConfig
	clusterCfg, err := api.GetClusterConfig(ctx, r.client)
	if err != nil {
		return ctrl.Result{}, err
	}
	if clusterCfg == nil {
		// Ignore if not found. This can happen if the ClusterConfig was deleted
		// after the current reconciliation request was issued.
		return ctrl.Result{}, nil
	}

	if !clusterCfg.DeletionTimestamp.IsZero() {
		logger.Debug("ClusterConfig is being deleted; nothing to do")
		return ctrl.Result{}, nil
	}

	logger.Debug("reconciling ClusterConfig")
	newStatus, reconcileErr := r.reconcile(ctx, clusterCfg)
	logger.Debug("done reconciling ClusterConfig")

	// Patch the status of the ClusterConfig.
	if err := kubeclient.PatchStatus(
		ctx,
		r.client,
		clusterCfg,
		func(status *kargoapi.ClusterConfigStatus) {
			*status = newStatus
		},
	); err != nil {
		// Prioritize the reconcile error if it exists.
		if reconcileErr != nil {
			logger.Error(
				err,
				"failed to update ClusterConfig status after reconciliation error",
			)
			return ctrl.Result{}, reconcileErr
		}
		return ctrl.Result{},
			fmt.Errorf("failed to update ClusterConfig status: %w", err)
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
	clusterCfg *kargoapi.ClusterConfig,
) (kargoapi.ClusterConfigStatus, error) {
	logger := logging.LoggerFromContext(ctx)
	status := *clusterCfg.Status.DeepCopy()

	subReconcilers := []struct {
		name      string
		reconcile func() (kargoapi.ClusterConfigStatus, error)
	}{{
		name: "syncing WebhookReceivers",
		reconcile: func() (kargoapi.ClusterConfigStatus, error) {
			return r.syncWebhookReceivers(ctx, clusterCfg)
		},
	}}
	for _, subR := range subReconcilers {
		logger.Debug(subR.name)

		// Reconcile the ClusterConfig with the sub-reconciler.
		var err error
		status, err = subR.reconcile()

		// If an error occurred during the sub-reconciler, then we should return the
		// error which will cause the ClusterConfig to be requeued.
		if err != nil {
			return status, err
		}

		// Patch the status of the ClusterConfig after each sub-reconciler to show
		// progress.
		if err = kubeclient.PatchStatus(
			ctx,
			r.client,
			clusterCfg,
			func(st *kargoapi.ClusterConfigStatus) { *st = status },
		); err != nil {
			logger.Error(
				err,
				fmt.Sprintf("failed to update ClusterConfig status after %s", subR.name),
			)
		}
	}

	// At this point, we have successfully reconciled the ClusterConfig and
	// can set the observed generation.
	status.ObservedGeneration = clusterCfg.GetGeneration()

	return status, nil
}

func (r *reconciler) syncWebhookReceivers(
	ctx context.Context,
	clusterCfg *kargoapi.ClusterConfig,
) (kargoapi.ClusterConfigStatus, error) {
	logger := logging.LoggerFromContext(ctx)
	status := clusterCfg.Status.DeepCopy()

	if len(clusterCfg.Spec.WebhookReceivers) == 0 {
		logger.Debug("ClusterConfig does not define any webhook receiver configurations")
		conditions.Delete(status, kargoapi.ConditionTypeReconciling)
		conditions.Set(status, &metav1.Condition{
			Type:               kargoapi.ConditionTypeReady,
			Status:             metav1.ConditionTrue,
			Reason:             "Synced",
			Message:            "ClusterConfig is synced and ready for use",
			ObservedGeneration: clusterCfg.GetGeneration(),
		})
		return *status, nil
	}

	conditions.Set(status, &metav1.Condition{
		Type:               kargoapi.ConditionTypeReconciling,
		Status:             metav1.ConditionTrue,
		Reason:             "SyncingWebhooks",
		Message:            "Syncing WebhookReceivers",
		ObservedGeneration: clusterCfg.GetGeneration(),
	})
	conditions.Set(status, &metav1.Condition{
		Type:               kargoapi.ConditionTypeReady,
		Status:             metav1.ConditionFalse,
		Reason:             "SyncingWebhooks",
		Message:            "Syncing WebhookReceivers",
		ObservedGeneration: clusterCfg.GetGeneration(),
	})

	logger.Debug("syncing WebhookReceivers")

	var errs []error
	status.WebhookReceivers = make(
		[]kargoapi.WebhookReceiverDetails,
		0,
		len(clusterCfg.Spec.WebhookReceivers),
	)
	if len(clusterCfg.Spec.WebhookReceivers) > 0 && r.cfg.ClusterSecretsNamespace == "" {
		err := errors.New(
			"no namespace is designated for storing Secrets referenced by " +
				"cluster-level resources; please ensure environment variable " +
				`"CLUSTER_SECRETS_NAMESPACE" is set`,
		)
		conditions.Set(status, &metav1.Condition{
			Type:               kargoapi.ConditionTypeReady,
			Status:             metav1.ConditionFalse,
			Reason:             "SyncWebhookReceiversFailed",
			Message:            "Failed to sync webhook receivers: " + err.Error(),
			ObservedGeneration: clusterCfg.GetGeneration(),
		})
		return *status, err
	}
	for _, receiverCfg := range clusterCfg.Spec.WebhookReceivers {
		receiver, err := external.NewReceiver(
			ctx,
			r.client,
			r.cfg.ExternalWebhookServerBaseURL,
			"",                            // No Project name for cluster-level receivers
			r.cfg.ClusterSecretsNamespace, // Secret namespace is one designated for cluster-level Secrets
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
			Message:            "Failed to sync webhook receivers: " + flattenedErrs.Error(),
			ObservedGeneration: clusterCfg.GetGeneration(),
		})
		return *status, flattenedErrs
	}

	conditions.Delete(status, kargoapi.ConditionTypeReconciling)
	conditions.Set(status, &metav1.Condition{
		Type:               kargoapi.ConditionTypeReady,
		Status:             metav1.ConditionTrue,
		Reason:             "Synced",
		Message:            "ClusterConfig is synced and ready for use",
		ObservedGeneration: clusterCfg.GetGeneration(),
	})
	return *status, nil
}
