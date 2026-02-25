package sharedsecrets

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller"
	"github.com/akuity/kargo/pkg/logging"
)

// ReconcilerConfig is the configuration for the shared secret replication
// reconciler. It is populated manually in management_controller.go.
type ReconcilerConfig struct {
	SharedResourcesNamespace string
	MaxConcurrentReconciles  int
}

// reconciler replicates Secrets annotated with
// kargo.akuity.io/replicate-to: "*" from the shared resources namespace into
// every Project namespace.
type reconciler struct {
	cfg       ReconcilerConfig
	client    client.Client // cached — source secrets + project listing + writes
	apiReader client.Reader // uncached — reading replicated secrets in project namespaces
}

// SetupReconcilerWithManager initializes the shared-secrets reconciler and
// registers it with the provided Manager.
func SetupReconcilerWithManager(
	ctx context.Context,
	kargoMgr manager.Manager,
	cfg ReconcilerConfig,
) error {
	r := &reconciler{
		cfg:       cfg,
		client:    kargoMgr.GetClient(),
		apiReader: kargoMgr.GetAPIReader(),
	}

	c, err := ctrl.NewControllerManagedBy(kargoMgr).
		Named("shared-secrets-replication-controller").
		For(&corev1.Secret{}).
		WithEventFilter(
			predicate.And(
				// Only reconcile Secrets from the shared resources namespace.
				predicate.NewPredicateFuncs(func(obj client.Object) bool {
					return obj.GetNamespace() == cfg.SharedResourcesNamespace
				}),
				// Smart annotation predicate:
				// Create/Delete: object must have the replicate-to annotation.
				// Update: old OR new object must have the replicate-to annotation
				//         (to handle annotation removal).
				predicate.Funcs{
					CreateFunc: func(e event.CreateEvent) bool {
						return e.Object.GetAnnotations()[kargoapi.AnnotationKeyReplicateTo] ==
							kargoapi.AnnotationValueReplicateToAll
					},
					DeleteFunc: func(e event.DeleteEvent) bool {
						return e.Object.GetAnnotations()[kargoapi.AnnotationKeyReplicateTo] ==
							kargoapi.AnnotationValueReplicateToAll
					},
					UpdateFunc: func(e event.UpdateEvent) bool {
						oldHas := e.ObjectOld.GetAnnotations()[kargoapi.AnnotationKeyReplicateTo] ==
							kargoapi.AnnotationValueReplicateToAll
						newHas := e.ObjectNew.GetAnnotations()[kargoapi.AnnotationKeyReplicateTo] ==
							kargoapi.AnnotationValueReplicateToAll
						// Also trigger when the finalizer is present but annotation
						// was removed (finalizer-only case handled in Reconcile).
						hasFinalizer := controllerutil.ContainsFinalizer(e.ObjectNew, kargoapi.FinalizerNameReplicated)
						return oldHas || newHas || hasFinalizer
					},
					GenericFunc: func(event.GenericEvent) bool {
						return false
					},
				},
			),
		).
		WithOptions(controller.CommonOptions(cfg.MaxConcurrentReconciles)).
		Build(r)
	if err != nil {
		return fmt.Errorf("error building SharedSecrets reconciler: %w", err)
	}

	// Watch for new Projects and enqueue all annotated source secrets so they
	// are replicated to the new project namespace.
	if err = c.Watch(
		source.Kind(
			kargoMgr.GetCache(),
			&kargoapi.Project{},
			&projectCreatedEnqueuer{
				client:   kargoMgr.GetClient(),
				sourceNS: cfg.SharedResourcesNamespace,
			},
			predicate.TypedFuncs[*kargoapi.Project]{
				CreateFunc:  func(event.TypedCreateEvent[*kargoapi.Project]) bool { return true },
				UpdateFunc:  func(event.TypedUpdateEvent[*kargoapi.Project]) bool { return false },
				DeleteFunc:  func(event.TypedDeleteEvent[*kargoapi.Project]) bool { return false },
				GenericFunc: func(event.TypedGenericEvent[*kargoapi.Project]) bool { return false },
			},
		),
	); err != nil {
		return fmt.Errorf("error watching Projects for SharedSecrets reconciler: %w", err)
	}

	logging.LoggerFromContext(ctx).Info(
		"Initialized SharedSecrets replication reconciler",
		"sharedResourcesNamespace", cfg.SharedResourcesNamespace,
	)
	return nil
}

// Reconcile is part of the main Kubernetes reconciliation loop.
func (r *reconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"secret.name", req.Name,
		"secret.namespace", req.Namespace,
	)
	ctx = logging.ContextWithLogger(ctx, logger)

	logger.Debug("reconciling shared Secret")

	// 1. Get source secret.
	srcSecret := &corev1.Secret{}
	if err := r.client.Get(ctx, req.NamespacedName, srcSecret); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	isBeingDeleted := !srcSecret.DeletionTimestamp.IsZero()
	hasAnnotation := srcSecret.Annotations[kargoapi.AnnotationKeyReplicateTo] == kargoapi.AnnotationValueReplicateToAll
	hasFinalizer := controllerutil.ContainsFinalizer(srcSecret, kargoapi.FinalizerNameReplicated)

	// 2. Cleanup branch: being deleted OR annotation removed but finalizer present.
	if isBeingDeleted || (!hasAnnotation && hasFinalizer) {
		logger.Debug("entering cleanup path for shared Secret")
		return ctrl.Result{}, r.cleanup(ctx, srcSecret)
	}

	// 3. No-op branch: no annotation and no finalizer.
	if !hasAnnotation {
		return ctrl.Result{}, nil
	}

	// 4. Replication branch.

	// Ensure the finalizer is present and requeue if just added.
	if controllerutil.AddFinalizer(srcSecret, kargoapi.FinalizerNameReplicated) {
		if err := patchFinalizers(ctx, r.client, srcSecret); err != nil {
			return ctrl.Result{}, fmt.Errorf("error adding finalizer to source Secret: %w", err)
		}
		logger.Debug("added finalizer to source Secret; requeuing")
		return ctrl.Result{RequeueAfter: 100 * time.Millisecond}, nil
	}

	// List all Projects to build the set of target namespaces.
	projectList := &kargoapi.ProjectList{}
	if err := r.client.List(ctx, projectList); err != nil {
		return ctrl.Result{}, fmt.Errorf("error listing Projects: %w", err)
	}
	projectNamespaces := make(map[string]struct{}, len(projectList.Items))
	for _, p := range projectList.Items {
		projectNamespaces[p.Name] = struct{}{}
	}

	sourceHash := computeDataHash(srcSecret.Data)

	// List all existing replicated secrets cluster-wide.
	existingList := &corev1.SecretList{}
	if err := r.apiReader.List(ctx, existingList, client.MatchingLabels{
		kargoapi.LabelKeyReplicatedFrom: srcSecret.Name,
	}); err != nil {
		return ctrl.Result{}, fmt.Errorf("error listing replicated Secrets: %w", err)
	}
	existingByNamespace := make(map[string]*corev1.Secret, len(existingList.Items))
	for i := range existingList.Items {
		existingByNamespace[existingList.Items[i].Namespace] = &existingList.Items[i]
	}

	// Sync to each project namespace.
	for ns := range projectNamespaces {
		existing := existingByNamespace[ns] // may be nil
		if err := r.syncToProjectNamespace(ctx, srcSecret, ns, sourceHash, existing); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Clean up replicated secrets in orphaned namespaces (project was deleted).
	for ns, dest := range existingByNamespace {
		if _, isProjectNS := projectNamespaces[ns]; !isProjectNS {
			logger.Debug("deleting orphaned replicated Secret", "namespace", ns)
			if err := r.client.Delete(ctx, dest); err != nil && !apierrors.IsNotFound(err) {
				return ctrl.Result{}, fmt.Errorf(
					"error deleting orphaned replicated Secret %q in namespace %q: %w",
					dest.Name, ns, err,
				)
			}
		}
	}

	logger.Debug("done reconciling shared Secret")
	return ctrl.Result{}, nil
}

// syncToProjectNamespace ensures the source Secret is replicated correctly into
// the given project namespace. existing is the current replicated Secret in
// that namespace, or nil if none exists.
func (r *reconciler) syncToProjectNamespace(
	ctx context.Context,
	srcSecret *corev1.Secret,
	namespace string,
	sourceHash string,
	existing *corev1.Secret,
) error {
	logger := logging.LoggerFromContext(ctx).WithValues("namespace", namespace)

	if existing == nil {
		// Create a new replicated Secret.
		destSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      srcSecret.Name,
				Labels: map[string]string{
					kargoapi.LabelKeyReplicatedFrom: srcSecret.Name,
					kargoapi.LabelKeyReplicatedSHA:  sourceHash,
				},
			},
			Type: srcSecret.Type,
			Data: srcSecret.Data,
		}
		if err := r.client.Create(ctx, destSecret); err != nil {
			if apierrors.IsAlreadyExists(err) {
				// A Secret was created between our List and this Create. Requeue
				// so we can handle it on the next pass.
				logger.Debug("Secret already exists in namespace; will re-evaluate on next reconcile")
				return nil
			}
			return fmt.Errorf(
				"error creating replicated Secret %q in namespace %q: %w",
				srcSecret.Name, namespace, err,
			)
		}
		logger.Debug("created replicated Secret")
		return nil
	}

	// Secret exists — check if we manage it.
	if _, hasLabel := existing.Labels[kargoapi.LabelKeyReplicatedFrom]; !hasLabel {
		logger.Debug("Secret exists but has no replicated-from label (conflict); skipping")
		return nil
	}

	// Detect external modification: if the stored SHA differs from the actual
	// data hash, the Secret was modified outside of Kargo's control.
	replicatedSHA := existing.Labels[kargoapi.LabelKeyReplicatedSHA]
	currentDataSHA := computeDataHash(existing.Data)
	if replicatedSHA != currentDataSHA {
		logger.Info("replicated Secret was modified externally; skipping update")
		return nil
	}

	// Already up to date.
	if replicatedSHA == sourceHash {
		logger.Debug("replicated Secret is already up to date; skipping")
		return nil
	}

	// Source changed — update the replicated Secret.
	existing.Data = srcSecret.Data
	existing.Type = srcSecret.Type
	if existing.Labels == nil {
		existing.Labels = make(map[string]string)
	}
	existing.Labels[kargoapi.LabelKeyReplicatedFrom] = srcSecret.Name
	existing.Labels[kargoapi.LabelKeyReplicatedSHA] = sourceHash
	if err := r.client.Update(ctx, existing); err != nil {
		return fmt.Errorf(
			"error updating replicated Secret %q in namespace %q: %w",
			srcSecret.Name, namespace, err,
		)
	}
	logger.Debug("updated replicated Secret")
	return nil
}

// cleanup deletes all managed replicated Secrets and removes the replication
// finalizer from the source Secret. It is called when the source Secret is
// being deleted or when its replicate-to annotation has been removed.
func (r *reconciler) cleanup(ctx context.Context, srcSecret *corev1.Secret) error {
	if !controllerutil.ContainsFinalizer(srcSecret, kargoapi.FinalizerNameReplicated) {
		return nil
	}

	logger := logging.LoggerFromContext(ctx)

	existingList := &corev1.SecretList{}
	if err := r.apiReader.List(ctx, existingList, client.MatchingLabels{
		kargoapi.LabelKeyReplicatedFrom: srcSecret.Name,
	}); err != nil {
		return fmt.Errorf("error listing replicated Secrets for cleanup: %w", err)
	}

	for i := range existingList.Items {
		dest := &existingList.Items[i]
		replicatedSHA := dest.Labels[kargoapi.LabelKeyReplicatedSHA]
		currentDataSHA := computeDataHash(dest.Data)
		if replicatedSHA != currentDataSHA {
			logger.Info(
				"replicated Secret was modified externally; skipping delete",
				"namespace", dest.Namespace,
			)
			continue
		}
		logger.Debug("deleting replicated Secret", "namespace", dest.Namespace)
		if err := r.client.Delete(ctx, dest); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf(
				"error deleting replicated Secret %q in namespace %q: %w",
				dest.Name, dest.Namespace, err,
			)
		}
	}

	return removeReplicatedFinalizer(ctx, r.client, srcSecret)
}

// projectCreatedEnqueuer is a TypedEventHandler that enqueues all annotated
// source Secrets when a new Project is created, ensuring they are replicated
// into the new project namespace.
type projectCreatedEnqueuer struct {
	client   client.Client
	sourceNS string
}

func (e *projectCreatedEnqueuer) Create(
	ctx context.Context,
	_ event.TypedCreateEvent[*kargoapi.Project],
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	secretList := &corev1.SecretList{}
	if err := e.client.List(ctx, secretList, client.InNamespace(e.sourceNS)); err != nil {
		logging.LoggerFromContext(ctx).Error(err, "error listing shared Secrets for new Project")
		return
	}
	for _, s := range secretList.Items {
		if s.Annotations[kargoapi.AnnotationKeyReplicateTo] == kargoapi.AnnotationValueReplicateToAll {
			wq.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: e.sourceNS,
					Name:      s.Name,
				},
			})
		}
	}
}

func (e *projectCreatedEnqueuer) Update(
	context.Context,
	event.TypedUpdateEvent[*kargoapi.Project],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
}

func (e *projectCreatedEnqueuer) Delete(
	context.Context,
	event.TypedDeleteEvent[*kargoapi.Project],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
}

func (e *projectCreatedEnqueuer) Generic(
	context.Context,
	event.TypedGenericEvent[*kargoapi.Project],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
}

// computeDataHash returns a deterministic 16-character truncated hex SHA-256
// hash of the secret data map. The hash is stable regardless of key ordering.
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

// patchFinalizers patches only the finalizers field of the given object.
func patchFinalizers(ctx context.Context, c client.Client, obj client.Object) error {
	type objectMeta struct {
		Finalizers []string `json:"finalizers"`
	}
	type patch struct {
		ObjectMeta objectMeta `json:"metadata"`
	}
	data, err := json.Marshal(patch{
		ObjectMeta: objectMeta{
			Finalizers: obj.GetFinalizers(),
		},
	})
	if err != nil {
		return fmt.Errorf("error marshaling finalizer patch: %w", err)
	}
	if err := c.Patch(ctx, obj, client.RawPatch(types.MergePatchType, data)); err != nil {
		return fmt.Errorf("error patching finalizers: %w", err)
	}
	return nil
}

// removeReplicatedFinalizer removes FinalizerNameReplicated from the object.
func removeReplicatedFinalizer(ctx context.Context, c client.Client, obj client.Object) error {
	if controllerutil.RemoveFinalizer(obj, kargoapi.FinalizerNameReplicated) {
		if err := patchFinalizers(ctx, c, obj); err != nil {
			return fmt.Errorf(
				"error removing replication finalizer from Secret %q: %w",
				obj.GetName(), err,
			)
		}
		logging.LoggerFromContext(ctx).Debug("removed replication finalizer from source Secret")
	}
	return nil
}
