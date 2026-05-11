package secrets

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/logging"
)

const (
	originNamespaceAnnotation = "kargo.akuity.io/origin-namespace"
	syncedDataHashAnnotation  = "kargo.akuity.io/synced-data-hash"
)

type ReconcilerConfig struct {
	ControllerName       string
	SourceNamespace      string
	DestinationNamespace string
}

// reconciler copies Secret resources from a source namespace to a destination
// namespace on a continuous basis. This is to ease the transition from:
//
//   - A conceptual "cluster secrets namespace" with a default value of
//     "kargo-cluster-secrets" to a more broadly purposed "system resources
//     namespace" with a default value of "kargo-system-resources".
//
//   - Conceptual "global credentials namespace(s)" to a more broadly purposed
//     "shared resources namespace" with a default value of
//     "kargo-shared-resources".
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
		Named(cfg.ControllerName).
		For(&corev1.Secret{}).
		WithEventFilter(
			predicate.And(
				predicate.Funcs{
					DeleteFunc: func(event.DeleteEvent) bool {
						// We're not interested in any deletes
						return false
					},
				},
				// Only reconcile Secrets from the source namespace
				predicate.NewPredicateFuncs(func(obj client.Object) bool {
					return obj.GetNamespace() == cfg.SourceNamespace
				}),
			),
		).
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
		// Ignore if not found. This can happen if the Secret was deleted after the
		// current reconciliation request was issued.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !srcSecret.DeletionTimestamp.IsZero() {
		logger.Debug("source Secret is being deleted; handling deletion")
		return ctrl.Result{}, r.handleDelete(ctx, srcSecret)
	}

	// Ensure the Secret has a finalizer and requeue if it was added.
	// The reason to requeue is to ensure that a possible deletion of the Secret
	// directly after the finalizer was added is handled without delay.
	if ok, err := api.EnsureFinalizer(ctx, r.client, srcSecret); ok || err != nil {
		logger.Debug("ensured finalizer on source Secret; requeuing")
		return ctrl.Result{RequeueAfter: 100 * time.Millisecond}, err
	}

	destSecret := srcSecret.DeepCopy()

	// Remove the finalizer that was just copied over so that deletes of new
	// secrets won't be blocked by anything.
	controllerutil.RemoveFinalizer(destSecret, kargoapi.FinalizerName)

	destSecret.Namespace = r.cfg.DestinationNamespace
	// Clear the ResourceVersion and UID from the copy so we can create/patch it in the
	// destination namespace.
	destSecret.ResourceVersion = ""
	destSecret.UID = ""
	destSecret.DeletionTimestamp = nil
	if destSecret.Annotations == nil {
		destSecret.Annotations = make(map[string]string, 2)
	}
	destSecret.Annotations[originNamespaceAnnotation] = r.cfg.SourceNamespace
	destSecret.Annotations[syncedDataHashAnnotation] = computeDataHash(srcSecret.Data)

	// Try to create the destination secret
	if err := r.client.Create(ctx, destSecret); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return ctrl.Result{}, fmt.Errorf(
				"error creating destination Secret %q in namespace %q: %w",
				req.Name, r.cfg.DestinationNamespace, err,
			)
		}
		// Secret already exists - check if we should update it
		existing := &corev1.Secret{}
		if err = r.client.Get(ctx, types.NamespacedName{
			Namespace: r.cfg.DestinationNamespace,
			Name:      req.Name,
		}, existing); err != nil {
			return ctrl.Result{}, fmt.Errorf(
				"error getting destination Secret %q in namespace %q: %w",
				req.Name, r.cfg.DestinationNamespace, err,
			)
		}

		// Only update if we previously synced this secret (has our hash annotation)
		// and it hasn't been modified externally since
		lastSyncedHash, hasAnnotation := existing.Annotations[syncedDataHashAnnotation]
		if !hasAnnotation {
			logger.Debug("destination Secret missing sync annotation; skipping update")
			return ctrl.Result{}, nil
		}
		if lastSyncedHash != computeDataHash(existing.Data) {
			logger.Info("destination Secret was modified externally; skipping update")
			return ctrl.Result{}, nil
		}

		// Safe to update - modify existing in place and use Update for optimistic
		// concurrency control. If the destination was modified between our Get and
		// this Update, the API server will reject with a conflict error and we'll
		// re-evaluate on the next reconciliation.
		existing.Labels = srcSecret.Labels
		existing.Data = srcSecret.Data
		existing.Type = srcSecret.Type
		if existing.Annotations == nil {
			existing.Annotations = make(map[string]string, 2)
		}
		existing.Annotations[originNamespaceAnnotation] = r.cfg.SourceNamespace
		existing.Annotations[syncedDataHashAnnotation] = computeDataHash(srcSecret.Data)
		if err = r.client.Update(ctx, existing); err != nil {
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

// handleDelete handles the deletion of a source Secret by deleting the
// corresponding destination Secret and removing the finalizer from the source
// Secret to unblock its deletion.
func (r *reconciler) handleDelete(ctx context.Context, srcSecret *corev1.Secret) error {
	// If the Secret does not have the finalizer, there is nothing to do.
	if !controllerutil.ContainsFinalizer(srcSecret, kargoapi.FinalizerName) {
		return nil
	}

	logger := logging.LoggerFromContext(ctx)

	// Check if the source namespace itself is being deleted. If so, this is a
	// bulk cleanup operation and we should preserve all destination secrets.
	srcNamespace := &corev1.Namespace{}
	if err := r.client.Get(
		ctx,
		types.NamespacedName{Name: r.cfg.SourceNamespace},
		srcNamespace,
	); err == nil && !srcNamespace.DeletionTimestamp.IsZero() {
		logger.Info("source namespace being deleted; preserving destination Secret")
		return r.removeFinalizer(ctx, srcSecret)
	}

	destSecret := &corev1.Secret{}
	err := r.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: r.cfg.DestinationNamespace,
			Name:      srcSecret.Name,
		},
		destSecret,
	)
	if client.IgnoreNotFound(err) != nil {
		return fmt.Errorf(
			"error getting destination Secret %q in namespace %q: %w",
			srcSecret.Name, r.cfg.DestinationNamespace, err,
		)
	}

	// If destination exists, only delete if we manage it and it hasn't been
	// modified externally
	if err == nil {
		lastSyncedHash, hasAnnotation := destSecret.Annotations[syncedDataHashAnnotation]
		if !hasAnnotation {
			logger.Debug("destination Secret missing sync annotation; skipping delete")
			return r.removeFinalizer(ctx, srcSecret)
		}
		if lastSyncedHash != computeDataHash(destSecret.Data) {
			logger.Info("destination Secret was modified externally; skipping delete")
			return r.removeFinalizer(ctx, srcSecret)
		}

		// Safe to delete
		if err = r.client.Delete(ctx, destSecret); err != nil {
			return fmt.Errorf(
				"error deleting destination Secret %q in namespace %q: %w",
				srcSecret.Name, r.cfg.DestinationNamespace, err,
			)
		}
		logger.Debug("deleted corresponding destination Secret")
	}

	return r.removeFinalizer(ctx, srcSecret)
}

// computeDataHash returns a deterministic hash of the secret data.
func computeDataHash(data map[string][]byte) string {
	h := sha256.New()
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write(data[k])
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func (r *reconciler) removeFinalizer(ctx context.Context, secret *corev1.Secret) error {
	if err := api.RemoveFinalizer(ctx, r.client, secret); err != nil {
		return fmt.Errorf(
			"error removing finalizer from source Secret %q in namespace %q: %w",
			secret.Name, secret.Namespace, err,
		)
	}
	logging.LoggerFromContext(ctx).Debug("removed finalizer from source Secret")
	return nil
}
