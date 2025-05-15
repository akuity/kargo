package projectconfigs

import (
	"context"
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/conditions"
	"github.com/akuity/kargo/internal/controller"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/internal/webhook/external"
)

type ReconcilerConfig struct {
	MaxConcurrentReconciles int `envconfig:"MAX_CONCURRENT_PROJECT_RECONCILES" default:"4"`
}

func ReconcilerConfigFromEnv() ReconcilerConfig {
	cfg := ReconcilerConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

type reconciler struct {
	cfg    ReconcilerConfig
	client client.Client

	ensureWebhookReceiversFn func(
		context.Context,
		*kargoapi.ProjectConfig,
	) error
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
		return fmt.Errorf("error creating Project reconciler: %w", err)
	}

	logging.LoggerFromContext(ctx).Info(
		"Initialized ProjectConfig reconciler",
	)
	return nil
}

func newReconciler(kubeClient client.Client, cfg ReconcilerConfig) *reconciler {
	r := &reconciler{
		cfg:    cfg,
		client: kubeClient,
	}
	r.ensureWebhookReceiversFn = r.ensureWebhookReceivers
	return r
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

	// Fetch the ProjectConfig instance
	projectConfig := &kargoapi.ProjectConfig{}
	if err := r.client.Get(ctx, req.NamespacedName, projectConfig); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if projectConfig.DeletionTimestamp != nil {
		logger.Debug("ProjectConfig is being deleted; nothing to do")
		return ctrl.Result{}, nil
	}

	logger.Debug("reconciling ProjectConfig")
	newStatus, needsRequeue, reconcileErr := r.syncProjectConfig(ctx, projectConfig)
	logger.Debug("done reconciling ProjectConfig")

	// Patch the status of the ProjectConfig.
	if err := kubeclient.PatchStatus(ctx, r.client, projectConfig, func(status *kargoapi.ProjectConfigStatus) {
		*status = newStatus
	}); err != nil {
		// Prioritize the reconcile error if it exists.
		if reconcileErr != nil {
			logger.Error(err, "failed to update ProjectConfig status after reconciliation error")
			return ctrl.Result{}, reconcileErr
		}
		return ctrl.Result{}, fmt.Errorf("failed to update ProjectConfig status: %w", err)
	}

	// Return the reconcile error if it exists.
	if reconcileErr != nil {
		return ctrl.Result{}, reconcileErr
	}
	// Immediate requeue if needed.
	if needsRequeue {
		return ctrl.Result{Requeue: true}, nil
	}
	// Otherwise, requeue after a delay.
	// TODO: Make the requeue delay configurable.
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *reconciler) syncProjectConfig(
	ctx context.Context,
	projectConfig *kargoapi.ProjectConfig,
) (kargoapi.ProjectConfigStatus, bool, error) {
	logger := logging.LoggerFromContext(ctx)
	status := projectConfig.Status.DeepCopy()

	conditions.Set(status, &metav1.Condition{
		Type:               kargoapi.ConditionTypeReconciling,
		Status:             metav1.ConditionTrue,
		Reason:             "Syncing",
		Message:            "Ensuring project config webhook receivers",
		ObservedGeneration: projectConfig.GetGeneration(),
	})
	conditions.Set(status, &metav1.Condition{
		Type:               kargoapi.ConditionTypeReady,
		Status:             metav1.ConditionFalse,
		Reason:             "Syncing",
		Message:            "Ensuring project config webhook receivers",
		ObservedGeneration: projectConfig.GetGeneration(),
	})

	if err := r.ensureWebhookReceiversFn(ctx, projectConfig); err != nil {
		logger.Error(err, "error ensuring webhook receivers")
		conditions.Set(status, &metav1.Condition{
			Type:               kargoapi.ConditionTypeReady,
			Status:             metav1.ConditionFalse,
			Reason:             "Syncing",
			Message:            "Failed to ensure project config webhook receivers: " + err.Error(),
			ObservedGeneration: projectConfig.GetGeneration(),
		})
		return *status, true, fmt.Errorf("error ensuring webhook receivers: %w", err)
	}
	status.WebhookReceivers = projectConfig.Status.WebhookReceivers

	conditions.Delete(status, kargoapi.ConditionTypeReconciling)
	conditions.Set(status, &metav1.Condition{
		Type:               kargoapi.ConditionTypeReady,
		Status:             metav1.ConditionTrue,
		Reason:             "Synced",
		Message:            "ProjectConfig is synced and ready for use",
		ObservedGeneration: projectConfig.GetGeneration(),
	})
	return *status, false, nil
}

func (r *reconciler) ensureWebhookReceivers(
	ctx context.Context,
	pc *kargoapi.ProjectConfig,
) error {
	logger := logging.LoggerFromContext(ctx)

	if pc.Spec.WebhookReceiverConfigs == nil {
		logger.Debug("ProjectConfig does not have any receiver configurations")
		return nil
	}
	logger.Debug("ensuring receivers",
		"receiver-configs", len(pc.Spec.WebhookReceiverConfigs),
	)
	var whReceivers []kargoapi.WebhookReceiver
	for _, rc := range pc.Spec.WebhookReceiverConfigs {
		var secret corev1.Secret
		err := r.client.Get(
			ctx,
			types.NamespacedName{
				Namespace: pc.Namespace,
				Name:      rc.SecretRef,
			},
			&secret,
		)
		if err != nil {
			logger.Error(err, "secret not found",
				"secret", rc.SecretRef,
			)
			if kubeerr.IsNotFound(err) {
				return fmt.Errorf(
					"secret-reference %q in namespace %q not found",
					rc.SecretRef, pc.Namespace,
				)
			}
			return fmt.Errorf(
				"error getting webhook receiver secret-reference %q in project namespace %q: %w",
				rc.SecretRef, pc.Name, err,
			)
		}
		logger.Debug("secret found", "secret", secret.Name)

		seed, ok := secret.Data["seed"]
		if !ok {
			logger.Error(err, "secret data not found")
			return fmt.Errorf(
				"error getting receiver secret %q in project namespace %q: %w",
				rc.SecretRef, pc.Name, err,
			)
		}
		pc.Status.WebhookReceivers = append(whReceivers, kargoapi.WebhookReceiver{
			Path: external.GenerateWebhookPath(
				pc.Name,
				rc.Type,
				string(seed),
			),
		})
	}
	return nil
}
