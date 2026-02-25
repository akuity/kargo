package replication

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
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
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/controller"
	"github.com/akuity/kargo/pkg/logging"
)

// lastAppliedConfigAnnotation is the kubectl annotation that stores a JSON
// copy of the last-applied configuration. It is excluded from the content hash
// and from replication because it is noisy, large, and source-object-specific.
const lastAppliedConfigAnnotation = "kubectl.kubernetes.io/last-applied-configuration"

// ReconcilerConfig holds configuration for the shared resource replication
// reconciler. It is populated manually in management_controller.go.
type ReconcilerConfig struct {
	SharedResourcesNamespace string
	MaxConcurrentReconciles  int
}

// resourceAdapter abstracts the type-specific operations needed to replicate a
// Kubernetes resource (Secret or ConfigMap).
type resourceAdapter interface {
	newObject() client.Object
	newList() client.ObjectList
	getItems(client.ObjectList) []client.Object
	computeHash(client.Object) string
	copyFields(dst, src client.Object)
}

// ---- Secret adapter ----

type secretAdapter struct{}

var _ resourceAdapter = secretAdapter{}

func (secretAdapter) newObject() client.Object   { return &corev1.Secret{} }
func (secretAdapter) newList() client.ObjectList { return &corev1.SecretList{} }

func (secretAdapter) getItems(l client.ObjectList) []client.Object {
	list, ok := l.(*corev1.SecretList)
	if !ok {
		return nil
	}
	items := make([]client.Object, len(list.Items))
	for i := range list.Items {
		items[i] = &list.Items[i]
	}
	return items
}

func (secretAdapter) computeHash(obj client.Object) string {
	s, ok := obj.(*corev1.Secret)
	if !ok {
		return ""
	}
	return computeSecretHash(s)
}

func (secretAdapter) copyFields(dst, src client.Object) {
	d, ok := dst.(*corev1.Secret)
	if !ok {
		return
	}
	s, ok := src.(*corev1.Secret)
	if !ok {
		return
	}
	d.Data = s.Data
	d.Type = s.Type
}

// ---- ConfigMap adapter ----

type configMapAdapter struct{}

var _ resourceAdapter = configMapAdapter{}

func (configMapAdapter) newObject() client.Object   { return &corev1.ConfigMap{} }
func (configMapAdapter) newList() client.ObjectList { return &corev1.ConfigMapList{} }

func (configMapAdapter) getItems(l client.ObjectList) []client.Object {
	list, ok := l.(*corev1.ConfigMapList)
	if !ok {
		return nil
	}
	items := make([]client.Object, len(list.Items))
	for i := range list.Items {
		items[i] = &list.Items[i]
	}
	return items
}

func (configMapAdapter) computeHash(obj client.Object) string {
	cm, ok := obj.(*corev1.ConfigMap)
	if !ok {
		return ""
	}
	return computeConfigMapHash(cm)
}

func (configMapAdapter) copyFields(dst, src client.Object) {
	d, ok := dst.(*corev1.ConfigMap)
	if !ok {
		return
	}
	s, ok := src.(*corev1.ConfigMap)
	if !ok {
		return
	}
	d.Data = s.Data
	d.BinaryData = s.BinaryData
}

// ---- Reconciler ----

// reconciler replicates Secrets or ConfigMaps annotated with
// kargo.akuity.io/replicate-to: "*" from the shared resources namespace into
// every Project namespace.
type reconciler struct {
	cfg       ReconcilerConfig
	client    client.Client // cached — source resources + project listing + writes
	apiReader client.Reader // uncached — reading replicated resources in project namespaces
	adapter   resourceAdapter
}

// SetupSecretReconcilerWithManager initializes the Secret replication
// reconciler and registers it with the provided Manager.
func SetupSecretReconcilerWithManager(
	ctx context.Context,
	kargoMgr manager.Manager,
	cfg ReconcilerConfig,
) error {
	return setupReconcilerWithManager(
		ctx, kargoMgr, cfg, secretAdapter{},
		"shared-secrets-replication-controller",
	)
}

// SetupConfigMapReconcilerWithManager initializes the ConfigMap replication
// reconciler and registers it with the provided Manager.
func SetupConfigMapReconcilerWithManager(
	ctx context.Context,
	kargoMgr manager.Manager,
	cfg ReconcilerConfig,
) error {
	return setupReconcilerWithManager(
		ctx, kargoMgr, cfg, configMapAdapter{},
		"shared-configmaps-replication-controller",
	)
}

func setupReconcilerWithManager(
	ctx context.Context,
	kargoMgr manager.Manager,
	cfg ReconcilerConfig,
	adapter resourceAdapter,
	controllerName string,
) error {
	r := &reconciler{
		cfg:       cfg,
		client:    kargoMgr.GetClient(),
		apiReader: kargoMgr.GetAPIReader(),
		adapter:   adapter,
	}

	c, err := ctrl.NewControllerManagedBy(kargoMgr).
		Named(controllerName).
		For(adapter.newObject()).
		WithEventFilter(sharedEventFilter(cfg)).
		WithOptions(controller.CommonOptions(cfg.MaxConcurrentReconciles)).
		Build(r)
	if err != nil {
		return fmt.Errorf("error building %s: %w", controllerName, err)
	}

	if err = c.Watch(
		source.Kind(
			kargoMgr.GetCache(),
			&kargoapi.Project{},
			&projectCreatedEnqueuer{
				client:   kargoMgr.GetClient(),
				sourceNS: cfg.SharedResourcesNamespace,
				adapter:  adapter,
			},
			predicate.TypedFuncs[*kargoapi.Project]{
				CreateFunc:  func(event.TypedCreateEvent[*kargoapi.Project]) bool { return true },
				UpdateFunc:  func(event.TypedUpdateEvent[*kargoapi.Project]) bool { return false },
				DeleteFunc:  func(event.TypedDeleteEvent[*kargoapi.Project]) bool { return false },
				GenericFunc: func(event.TypedGenericEvent[*kargoapi.Project]) bool { return false },
			},
		),
	); err != nil {
		return fmt.Errorf("error watching Projects for %s: %w", controllerName, err)
	}

	logging.LoggerFromContext(ctx).Info(
		"Initialized replication reconciler",
		"controller", controllerName,
		"sharedResourcesNamespace", cfg.SharedResourcesNamespace,
	)
	return nil
}

// sharedEventFilter builds the predicate shared by both the Secret and
// ConfigMap replication controllers.
func sharedEventFilter(cfg ReconcilerConfig) predicate.Predicate {
	return predicate.And(
		// Only reconcile resources from the shared resources namespace.
		predicate.NewPredicateFuncs(func(obj client.Object) bool {
			return obj.GetNamespace() == cfg.SharedResourcesNamespace
		}),
		// Smart annotation predicate:
		// Create/Delete: object must have the replicate-to annotation.
		// Update: old OR new object must have it (to catch annotation removal).
		predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				// Also allow through objects that have our finalizer but lost the
				// annotation while the controller was down, so the cleanup path
				// runs on startup.
				return e.Object.GetAnnotations()[kargoapi.AnnotationKeyReplicateTo] ==
					kargoapi.AnnotationValueReplicateToAll ||
					controllerutil.ContainsFinalizer(e.Object, kargoapi.FinalizerName)
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
				hasFinalizer := controllerutil.ContainsFinalizer(e.ObjectNew, kargoapi.FinalizerName)
				return oldHas || newHas || hasFinalizer
			},
			GenericFunc: func(event.GenericEvent) bool { return false },
		},
	)
}

// Reconcile is part of the main Kubernetes reconciliation loop.
func (r *reconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"name", req.Name,
		"namespace", req.Namespace,
	)
	ctx = logging.ContextWithLogger(ctx, logger)

	logger.Debug("reconciling shared resource")

	srcObj := r.adapter.newObject()
	if err := r.client.Get(ctx, req.NamespacedName, srcObj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	isBeingDeleted := !srcObj.GetDeletionTimestamp().IsZero()
	hasAnnotation := srcObj.GetAnnotations()[kargoapi.AnnotationKeyReplicateTo] == kargoapi.AnnotationValueReplicateToAll
	hasFinalizer := controllerutil.ContainsFinalizer(srcObj, kargoapi.FinalizerName)

	// Cleanup branch: being deleted OR annotation removed but finalizer still present.
	if isBeingDeleted || (!hasAnnotation && hasFinalizer) {
		logger.Debug("entering cleanup path for shared resource")
		return ctrl.Result{}, r.cleanup(ctx, srcObj)
	}

	// No-op branch: no annotation and no finalizer.
	if !hasAnnotation {
		return ctrl.Result{}, nil
	}

	// Replication branch.

	// Ensure the Secret/ConfigMap has a finalizer and requeue if it was added.
	// The reason to requeue is to ensure that a possible deletion of the resource
	// directly after the finalizer was added is handled without delay.
	if ok, err := api.EnsureFinalizer(ctx, r.client, srcObj); ok || err != nil {
		logger.Debug("ensured finalizer on source resource; requeuing")
		return ctrl.Result{RequeueAfter: 100 * time.Millisecond}, err
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

	sourceHash := r.adapter.computeHash(srcObj)

	// List replicated resources that are out of date (SHA does not match
	// sourceHash). Up-to-date replicas are excluded to avoid unnecessary
	// network transfer; syncToProjectNamespace handles the AlreadyExists case
	// gracefully when existing is nil.
	replicatedFromReq, err := labels.NewRequirement(
		kargoapi.LabelKeyReplicatedFrom, selection.Equals, []string{srcObj.GetName()})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error building replicated-from label requirement: %w", err)
	}
	shaOutOfDateReq, err := labels.NewRequirement(
		kargoapi.LabelKeyReplicatedSHA, selection.NotIn, []string{sourceHash})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error building replicated-sha label requirement: %w", err)
	}
	existingList := r.adapter.newList()
	if err = r.apiReader.List(ctx, existingList, client.MatchingLabelsSelector{
		Selector: labels.NewSelector().Add(*replicatedFromReq, *shaOutOfDateReq),
	}); err != nil {
		return ctrl.Result{}, fmt.Errorf("error listing replicated resources: %w", err)
	}
	existingByNamespace := make(map[string]client.Object)
	for _, item := range r.adapter.getItems(existingList) {
		existingByNamespace[item.GetNamespace()] = item
	}

	// Sync to each project namespace.
	for ns := range projectNamespaces {
		if err := r.syncToProjectNamespace(ctx, srcObj, ns, sourceHash, existingByNamespace[ns]); err != nil {
			return ctrl.Result{}, err
		}
	}

	logger.Debug("done reconciling shared resource")
	return ctrl.Result{}, nil
}

// syncToProjectNamespace ensures the source resource is replicated correctly
// into the given project namespace. existing is the current out-of-date
// replicated resource in that namespace, or nil if the replica is up to date
// or does not exist.
func (r *reconciler) syncToProjectNamespace(
	ctx context.Context,
	src client.Object,
	namespace string,
	sourceHash string,
	existing client.Object,
) error {
	logger := logging.LoggerFromContext(ctx).WithValues("namespace", namespace)

	if existing == nil {
		// Either no replica exists yet, or it already has the current SHA
		// (filtered out by the List). Try to create; handle AlreadyExists.
		destObj := r.adapter.newObject()
		destObj.SetName(src.GetName())
		destObj.SetNamespace(namespace)
		destObj.SetLabels(replicaLabels(src, sourceHash))
		destObj.SetAnnotations(replicaAnnotations(src))
		r.adapter.copyFields(destObj, src)
		if err := r.client.Create(ctx, destObj); err != nil {
			if apierrors.IsAlreadyExists(err) {
				logger.Debug("resource already exists in namespace; will re-evaluate on next reconcile")
				return nil
			}
			return fmt.Errorf(
				"error creating replicated resource %q in namespace %q: %w",
				src.GetName(), namespace, err,
			)
		}
		logger.Debug("created replicated resource")
		return nil
	}

	// Replica exists and is out of date — check we own it.
	if _, hasLabel := existing.GetLabels()[kargoapi.LabelKeyReplicatedFrom]; !hasLabel {
		logger.Debug("resource exists but has no replicated-from label (conflict); skipping")
		return nil
	}

	// This technically shouldn't happen because we filtered the List to only
	// include out-of-date replicas, but check defensively before updating.
	if existing.GetLabels()[kargoapi.LabelKeyReplicatedSHA] == sourceHash {
		logger.Debug("replicated resource is already up to date; skipping")
		return nil
	}

	// Source changed — update the replica.
	r.adapter.copyFields(existing, src)
	existing.SetLabels(replicaLabels(src, sourceHash))
	existing.SetAnnotations(replicaAnnotations(src))
	if err := r.client.Update(ctx, existing); err != nil {
		return fmt.Errorf(
			"error updating replicated resource %q in namespace %q: %w",
			src.GetName(), namespace, err,
		)
	}
	logger.Debug("updated replicated resource")
	return nil
}

// cleanup deletes all managed replicated resources and removes the replication
// finalizer from the source resource. It is called when the source resource is
// being deleted or when its replicate-to annotation has been removed.
func (r *reconciler) cleanup(ctx context.Context, src client.Object) error {
	if !controllerutil.ContainsFinalizer(src, kargoapi.FinalizerName) {
		return nil
	}

	logger := logging.LoggerFromContext(ctx)

	existingList := r.adapter.newList()
	if err := r.apiReader.List(ctx, existingList, client.MatchingLabels{
		kargoapi.LabelKeyReplicatedFrom: src.GetName(),
	}); err != nil {
		return fmt.Errorf("error listing replicated resources for cleanup: %w", err)
	}

	for _, dest := range r.adapter.getItems(existingList) {
		logger.Debug("deleting replicated resource", "namespace", dest.GetNamespace())
		if err := r.client.Delete(ctx, dest); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf(
				"error deleting replicated resource %q in namespace %q: %w",
				dest.GetName(), dest.GetNamespace(), err,
			)
		}
	}

	return api.RemoveFinalizer(ctx, r.client, src)
}

// ---- Project-created enqueuer ----

// projectCreatedEnqueuer is a TypedEventHandler that enqueues all annotated
// source resources when a new Project is created, ensuring they are replicated
// into the new project namespace.
type projectCreatedEnqueuer struct {
	client   client.Client
	sourceNS string
	adapter  resourceAdapter
}

func (e *projectCreatedEnqueuer) Create(
	ctx context.Context,
	_ event.TypedCreateEvent[*kargoapi.Project],
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	list := e.adapter.newList()
	if err := e.client.List(ctx, list, client.InNamespace(e.sourceNS)); err != nil {
		logging.LoggerFromContext(ctx).Error(err, "error listing shared resources for new Project")
		return
	}
	for _, obj := range e.adapter.getItems(list) {
		if obj.GetAnnotations()[kargoapi.AnnotationKeyReplicateTo] == kargoapi.AnnotationValueReplicateToAll {
			wq.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: e.sourceNS,
					Name:      obj.GetName(),
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

// ---- Helpers ----

// replicaLabels builds the label map for a replicated resource: all source
// labels plus the two replication-managed labels.
func replicaLabels(src client.Object, sourceHash string) map[string]string {
	srcLabels := src.GetLabels()
	result := make(map[string]string, len(srcLabels)+2)
	for k, v := range srcLabels {
		result[k] = v
	}
	result[kargoapi.LabelKeyReplicatedFrom] = src.GetName()
	result[kargoapi.LabelKeyReplicatedSHA] = sourceHash
	return result
}

// replicaAnnotations builds the annotation map for a replicated resource: all
// source annotations except those that must not be propagated.
func replicaAnnotations(src client.Object) map[string]string {
	var result map[string]string
	for k, v := range src.GetAnnotations() {
		if k == kargoapi.AnnotationKeyReplicateTo || k == lastAppliedConfigAnnotation {
			continue
		}
		if result == nil {
			result = make(map[string]string, len(src.GetAnnotations()))
		}
		result[k] = v
	}
	return result
}

// hashMetadata writes the labels and annotations of a resource into h, using
// the same exclusions applied by both hash functions.
func hashMetadata(h hash.Hash, lbls, annotations map[string]string) {
	h.Write([]byte("labels"))
	labelKeys := make([]string, 0, len(lbls))
	for k := range lbls {
		if k != kargoapi.LabelKeyReplicatedFrom && k != kargoapi.LabelKeyReplicatedSHA {
			labelKeys = append(labelKeys, k)
		}
	}
	sort.Strings(labelKeys)
	for _, k := range labelKeys {
		h.Write([]byte(k))
		h.Write([]byte(lbls[k]))
	}

	h.Write([]byte("annotations"))
	annotationKeys := make([]string, 0, len(annotations))
	for k := range annotations {
		if k != kargoapi.AnnotationKeyReplicateTo && k != lastAppliedConfigAnnotation {
			annotationKeys = append(annotationKeys, k)
		}
	}
	sort.Strings(annotationKeys)
	for _, k := range annotationKeys {
		h.Write([]byte(k))
		h.Write([]byte(annotations[k]))
	}
}

// computeSecretHash returns a deterministic 16-character truncated hex SHA-256
// hash of a Secret's labels, annotations, and data.
func computeSecretHash(secret *corev1.Secret) string {
	h := sha256.New()
	hashMetadata(h, secret.Labels, secret.Annotations)
	h.Write([]byte("data"))
	dataKeys := make([]string, 0, len(secret.Data))
	for k := range secret.Data {
		dataKeys = append(dataKeys, k)
	}
	sort.Strings(dataKeys)
	for _, k := range dataKeys {
		h.Write([]byte(k))
		h.Write(secret.Data[k])
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// computeConfigMapHash returns a deterministic 16-character truncated hex
// SHA-256 hash of a ConfigMap's labels, annotations, data, and binaryData.
func computeConfigMapHash(cm *corev1.ConfigMap) string {
	h := sha256.New()
	hashMetadata(h, cm.Labels, cm.Annotations)
	h.Write([]byte("data"))
	dataKeys := make([]string, 0, len(cm.Data))
	for k := range cm.Data {
		dataKeys = append(dataKeys, k)
	}
	sort.Strings(dataKeys)
	for _, k := range dataKeys {
		h.Write([]byte(k))
		h.Write([]byte(cm.Data[k]))
	}
	h.Write([]byte("binaryData"))
	binaryDataKeys := make([]string, 0, len(cm.BinaryData))
	for k := range cm.BinaryData {
		binaryDataKeys = append(binaryDataKeys, k)
	}
	sort.Strings(binaryDataKeys)
	for _, k := range binaryDataKeys {
		h.Write([]byte(k))
		h.Write(cm.BinaryData[k])
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}
