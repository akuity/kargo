package secrets

import (
	"context"
	"fmt"

	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/akuity/kargo/pkg/logging"
)

type ReconcilerConfig struct {
	SourceNamespace      string `envconfig:"CLUSTER_SECRETS_NAMESPACE" default:"kargo-cluster-secrets"`
	DestinationNamespace string `envconfig:"CLUSTER_RESOURCES_NAMESPACE" default:"kargo-cluster-resources"`
}

func ReconcilerConfigFromEnv() ReconcilerConfig {
	cfg := ReconcilerConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// reconciler copies Secret resources from a source namespace to a destination
// namespace on a continuous basis. This is to ease the transition from a
// conceptual "cluster secrets namespace" with a default value of
// "kargo-cluster-secrets" to a more broadly purposed "cluster resources
// namespace" with a default value of "kargo-cluster-resources".
//
// TODO(krancour): Remove this reconciler in v1.12.0. By that time, affected
// users are expected to have made any necessary configuration to obviate the
// need for this reconciler.
type reconciler struct {
	cfg    ReconcilerConfig
	client client.Client
}

// SetupReconcilerWithManager initializes a reconciler for Secret resources
// and registers it with the provided Manager.
func SetupReconcilerWithManager(
	ctx context.Context,
	kargoMgr manager.Manager,
	cfg ReconcilerConfig,
) error {
	err := ctrl.NewControllerManagedBy(kargoMgr).
		For(&corev1.Secret{}).
		Complete(newReconciler(kargoMgr.GetClient(), cfg))
	if err == nil {
		logging.LoggerFromContext(ctx).Info(
			"Initialized Secrets reconciler",
			"sourceNamespace", cfg.SourceNamespace,
			"destinationNamespace", cfg.DestinationNamespace,
		)
	}
	return err
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
		"secret.name", req.Name,
		"secret.namespace", req.Namespace,
	)

	// Validate that the Secret is from the source namespace. If it's from a
	// different namespace, we should not reconcile it.
	if req.Namespace != r.cfg.SourceNamespace {
		logger.Debug(
			"ignoring secret from unexpected namespace",
			"expectedNamespace", r.cfg.SourceNamespace,
			"actualNamespace", req.Namespace,
		)
		return ctrl.Result{}, nil
	}

	logger.Debug("reconciling Secret")

	// Get the Secret from the source namespace
	srcSecret := &corev1.Secret{}
	if err := r.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: r.cfg.SourceNamespace,
			Name:      req.Name,
		},
		srcSecret,
	); err != nil {
		if apierrors.IsNotFound(err) {
			// The source Secret no longer exists. We should delete the copy in
			// the destination namespace if it exists.
			logger.Debug(
				"source Secret not found, deleting destination copy if it exists",
			)
			if err = r.client.Delete(
				ctx,
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: r.cfg.DestinationNamespace,
						Name:      req.Name,
					},
				},
			); err != nil && !apierrors.IsNotFound(err) {
				return ctrl.Result{}, fmt.Errorf(
					"error deleting copied secret %q from namespace %q: %w",
					req.Name, r.cfg.DestinationNamespace, err,
				)
			}
			logger.Debug("done reconciling Secret")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf(
			"error getting source secret %q in namespace %q: %w",
			req.Name, r.cfg.SourceNamespace, err,
		)
	}

	destSecret := srcSecret.DeepCopy()
	destSecret.Namespace = r.cfg.DestinationNamespace
	// Clear the ResourceVersion and UID from the copy so we can create/patch it in the
	// destination namespace.
	destSecret.ResourceVersion = ""
	destSecret.UID = ""
	if destSecret.Annotations == nil {
		destSecret.Annotations = make(map[string]string, 1)
	}
	destSecret.Annotations["kargo.akuity.io/origin-namespace"] = r.cfg.SourceNamespace

	// Try to create the destination secret
	if err := r.client.Create(ctx, destSecret); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return ctrl.Result{}, fmt.Errorf(
				"error creating destination Secret %q in namespace %q: %w",
				req.Name, r.cfg.DestinationNamespace, err,
			)
		}
		// Secret already exists, patch it instead.
		if err = r.client.Patch(ctx, destSecret, client.Merge); err != nil {
			return ctrl.Result{}, fmt.Errorf(
				"error updating destination Secret %q in namespace %q: %w",
				req.Name, r.cfg.DestinationNamespace, err,
			)
		}
		logger.Debug("updated destination Secret")
	} else {
		logger.Debug("created destination Secret")
	}

	logger.Debug("done reconciling Secret")

	return ctrl.Result{}, nil
}
