package projectconfigs

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	ExternalWebhookServerBaseURL string `envconfig:"EXTERNAL_WEBHOOK_SERVER_BASE_URL"`
	MaxConcurrentReconciles      int    `envconfig:"MAX_CONCURRENT_PROJECT_RECONCILES" default:"4"`
}

func ReconcilerConfigFromEnv() ReconcilerConfig {
	cfg := ReconcilerConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

type reconciler struct {
	cfg    ReconcilerConfig
	client client.Client

	syncWebhookReceiversFn func(
		context.Context,
		*kargoapi.ProjectConfig,
	) ([]kargoapi.WebhookReceiver, error)
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
	r.syncWebhookReceiversFn = r.syncWebhookReceivers
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

	if !projectConfig.DeletionTimestamp.IsZero() {
		logger.Debug("ProjectConfig is being deleted; nothing to do")
		return ctrl.Result{}, nil
	}

	// NOTE: If we start requiring cleanup operations or other management of state. Add a finalizer here

	logger.Debug("reconciling ProjectConfig")
	newStatus, reconcileErr := r.syncProjectConfig(ctx, projectConfig)
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

	// TODO: Make the requeue delay configurable.
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *reconciler) syncProjectConfig(
	ctx context.Context,
	pc *kargoapi.ProjectConfig,
) (
	kargoapi.ProjectConfigStatus,
	error,
) {
	logger := logging.LoggerFromContext(ctx)
	status := pc.Status.DeepCopy()

	conditions.Set(status, &metav1.Condition{
		Type:               kargoapi.ConditionTypeReconciling,
		Status:             metav1.ConditionTrue,
		Reason:             "SyncingWebhooks",
		ObservedGeneration: pc.GetGeneration(),
	})
	conditions.Set(status, &metav1.Condition{
		Type:               kargoapi.ConditionTypeReady,
		Status:             metav1.ConditionFalse,
		Reason:             "SyncingWebhooks",
		ObservedGeneration: pc.GetGeneration(),
	})

	whReceivers, err := r.syncWebhookReceiversFn(ctx, pc)
	status.WebhookReceivers = whReceivers
	if err != nil {
		logger.Error(err, "error syncing webhook receivers",
			"project-config", pc.Name,
		)
		conditions.Set(status, &metav1.Condition{
			Type:               kargoapi.ConditionTypeReady,
			Status:             metav1.ConditionFalse,
			Reason:             "SecretMisconfiguration",
			Message:            "Failed to sync webhook receivers: " + err.Error(),
			ObservedGeneration: pc.GetGeneration(),
		})
		return *status, err
	}

	conditions.Set(status, &metav1.Condition{
		Type:               kargoapi.ConditionTypeReady,
		Status:             metav1.ConditionTrue,
		Reason:             "Synced",
		Message:            "ProjectConfig is synced and ready for use",
		ObservedGeneration: pc.GetGeneration(),
	})
	conditions.Delete(status, kargoapi.ConditionTypeReconciling)
	return *status, nil
}

func (r *reconciler) syncWebhookReceivers(
	ctx context.Context,
	pc *kargoapi.ProjectConfig,
) ([]kargoapi.WebhookReceiver, error) {
	logger := logging.LoggerFromContext(ctx)
	if pc.Spec.WebhookReceivers == nil {
		logger.Debug("ProjectConfig does not have any receiver configurations")
		return nil, nil
	}

	logger.Debug("syncing webhook receivers",
		"webhook-receiver-configs", len(pc.Spec.WebhookReceivers),
	)

	var errs []error
	webhookReceivers := make([]kargoapi.WebhookReceiver, 0, len(pc.Spec.WebhookReceivers))
	for _, rc := range pc.Spec.WebhookReceivers {
		whr, err := r.newWebhookReceiver(ctx, pc, rc)
		if err != nil {
			logger.Error(err, "error initializing new webhook receiver",
				"receiver-config", rc,
			)
			errs = append(errs, fmt.Errorf(
				"error initializing webhook receiver %q: %w",
				rc.Name, err,
			),
			)
			continue
		}
		webhookReceivers = append(webhookReceivers, *whr)
	}
	return webhookReceivers, kerrors.Flatten(kerrors.NewAggregate(errs))
}

func (r *reconciler) newWebhookReceiver(
	ctx context.Context,
	pc *kargoapi.ProjectConfig,
	rc kargoapi.WebhookReceiverConfig,
) (*kargoapi.WebhookReceiver, error) {
	logger := logging.LoggerFromContext(ctx)
	cfg, err := getProviderConfig(rc)
	if err != nil {
		logger.Error(err, "error getting provider config",
			"receiver-config", rc,
			"project-config", pc,
		)
		return nil, fmt.Errorf("error getting secret name: %w", err)
	}

	var s corev1.Secret
	err = r.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: pc.Namespace,
			Name:      cfg.secretName,
		},
		&s,
	)
	if err != nil {
		logger.Error(err, "secret not found",
			"secret-name", cfg.secretName,
		)
		if kubeerr.IsNotFound(err) {
			return nil, fmt.Errorf(
				"secret %q in namespace %q not found",
				cfg.secretName, pc.Namespace,
			)
		}
		return nil, fmt.Errorf(
			"error getting webhook receiver secret %q in project namespace %q: %w",
			cfg.secretName, pc.Namespace, err,
		)
	}
	logger.Debug("secret found", "secret", cfg.secretName)

	secret, ok := s.Data[cfg.targetKey]
	if !ok {
		logger.Error(err, "target key not found in secret data",
			"target-key", cfg.targetKey,
		)
		return nil, fmt.Errorf(
			"key %q not found in secret %q for project config %q",
			cfg.targetKey, cfg.secretName, pc.Name,
		)
	}

	wr := &kargoapi.WebhookReceiver{
		Name: rc.Name,
		Path: external.GenerateWebhookPath(
			rc.Name,
			pc.Name,
			cfg.receiverType,
			string(secret),
		),
	}
	wr.URL, err = url.JoinPath(r.cfg.ExternalWebhookServerBaseURL + wr.Path)
	if err != nil {
		logger.Error(err, "error generating webhook URL",
			"base-url", r.cfg.ExternalWebhookServerBaseURL,
			"path", wr.Path,
		)
		return nil, fmt.Errorf("error generating webhook URL: %w", err)
	}

	logger.Debug("webhook receiver initialized",
		"webhook-receiver", wr,
	)
	return wr, nil
}

type providerConfig struct {
	secretName   string
	targetKey    string
	receiverType kargoapi.WebhookReceiverType
}

func getProviderConfig(rc kargoapi.WebhookReceiverConfig) (*providerConfig, error) {
	switch {
	case rc.GitHub != nil:
		if rc.GitHub.SecretRef.Name == "" {
			return nil, errors.New("receiver config does not have a secret reference name")
		}
		return &providerConfig{
			secretName:   rc.GitHub.SecretRef.Name,
			targetKey:    api.WebhookReceiverSecretKeyGithub,
			receiverType: kargoapi.WebhookReceiverTypeGitHub,
		}, nil
	default:
		return nil, errors.New("webhook receiver config has no valid entry")
	}
}
