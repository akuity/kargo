package projectconfigs

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/client/apiserver"
	"github.com/akuity/kargo/pkg/conditions"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/projectconfigstore"
	"github.com/akuity/kargo/pkg/logging"
	engine "github.com/akuity/kargo/pkg/reconciler"
	"github.com/akuity/kargo/pkg/reconciler/kube"
	"github.com/akuity/kargo/pkg/reconciler/list"
	"github.com/akuity/kargo/pkg/reconciler/nats"
	"github.com/akuity/kargo/pkg/server/projectconfigs/wire"
	"github.com/akuity/kargo/pkg/webhook/external"
)

const (
	controllerName = "project-configs"

	// resyncInterval is how often every ProjectConfig is reconciled whether
	// or not an event said it changed. It recovers lost events and picks up
	// rotated Secrets, which change receiver paths without changing the
	// ProjectConfig.
	resyncInterval = 5 * time.Minute
	// resyncJitter spreads the resyncs of replicas that started together.
	resyncJitter = 0.2

	// conflictRequeueDelay is how long a reconcile whose status write lost a
	// race with a change to the ProjectConfig waits before running again.
	conflictRequeueDelay = time.Second
	// notFoundRequeueDelay is how long a reconcile waits before running again
	// when the API server reported the ProjectConfig not found but the store
	// still holds it, as when the API server's cache has not yet seen a new
	// Project, or an API server pod predating the status route answered.
	notFoundRequeueDelay = 5 * time.Second
	// adoptRequeueDelay is how long a reconcile that adopted a ProjectConfig
	// for a Project of the same name with a new UID waits before reconciling
	// it under that Project. The adoption changes neither the generation nor
	// the refresh token, so its event does not trigger that reconcile.
	adoptRequeueDelay = time.Second

	reasonWebhookReceiverPathClaimed = "WebhookReceiverPathClaimed"
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

// configStore is the slice of the ProjectConfig store the reconciler reads,
// cleans up and adopts through.
type configStore interface {
	GetUnfiltered(ctx context.Context, project string) (projectconfigstore.Config, error)
	List(ctx context.Context) ([]projectconfigstore.Config, error)
	DeleteForProject(ctx context.Context, project, projectUID string) error
	AdoptForProject(ctx context.Context, project, newUID string) (bool, error)
	GetByWebhookReceiverPath(ctx context.Context, path string) (projectconfigstore.Config, error)
}

// statusWriter records the outcome of a reconcile. The API server client is
// one.
type statusWriter interface {
	UpdateProjectConfigStatus(
		ctx context.Context,
		project string,
		req wire.StatusRequest,
	) ([]string, error)
}

type reconciler struct {
	cfg    ReconcilerConfig
	client client.Client
	// project scoped secrets are not cached so we need to query the api-server directly
	apiReader client.Reader
	store     configStore
	status    statusWriter
}

// SetupReconciler adds the ProjectConfig reconciler to the manager. It runs
// in the leader only, reconciling a ProjectConfig when an event on conn says
// its spec changed or a refresh was requested, and every ProjectConfig in the
// store on an interval. It reads ProjectConfigs from the store and writes
// their status through status.
func SetupReconciler(
	ctx context.Context,
	mgr manager.Manager,
	conn nats.Subscriber,
	store configStore,
	status statusWriter,
	cfg ReconcilerConfig,
) error {
	r := newReconciler(mgr.GetClient(), mgr.GetAPIReader(), store, status, cfg)
	c, err := newController(conn, store, r)
	if err != nil {
		return fmt.Errorf("error creating ProjectConfig reconciler: %w", err)
	}
	if err = mgr.Add(kube.Runnable(c)); err != nil {
		return fmt.Errorf("error adding ProjectConfig reconciler to manager: %w", err)
	}

	logging.LoggerFromContext(ctx).Info(
		"Initialized ProjectConfig reconciler",
		"maxConcurrentReconciles", cfg.MaxConcurrentReconciles,
	)
	return nil
}

// newController assembles the controller: events first, so they are
// subscribed while the initial list loads, then the resync.
func newController(
	conn nats.Subscriber,
	store configStore,
	r *reconciler,
) (*engine.Controller[projectconfigstore.Key], error) {
	return engine.New[projectconfigstore.Key](controllerName).
		Watch(
			nats.Subject(conn, projectconfigstore.Events).
				Ignore(nats.Deleted).
				OnUpdate(specChanged),
		).
		Watch(list.New(listKeys(store)).Every(resyncInterval).Jitter(resyncJitter)).
		Workers(r.cfg.MaxConcurrentReconciles).
		Func(r.Reconcile)
}

// specChanged reports whether an update needs a reconcile: a new generation
// or a new refresh token. Status writes, the reconciler's own among them,
// change neither.
func specChanged(old, updated database.ProjectConfigRow) bool {
	return old.Generation != updated.Generation ||
		!ptrEqual(old.RefreshToken, updated.RefreshToken)
}

func ptrEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// listKeys returns the resync's list function: the key of every
// ProjectConfig in the store.
func listKeys(
	store configStore,
) func(context.Context) ([]projectconfigstore.Key, error) {
	return func(ctx context.Context) ([]projectconfigstore.Key, error) {
		cfgs, err := store.List(ctx)
		if err != nil {
			return nil, err
		}
		keys := make([]projectconfigstore.Key, len(cfgs))
		for i, cfg := range cfgs {
			keys[i] = projectconfigstore.Key{Project: cfg.ProjectName}
		}
		return keys, nil
	}
}

func newReconciler(
	kubeClient client.Client,
	apiReader client.Reader,
	store configStore,
	status statusWriter,
	cfg ReconcilerConfig,
) *reconciler {
	return &reconciler{
		apiReader: apiReader,
		cfg:       cfg,
		client:    kubeClient,
		store:     store,
		status:    status,
	}
}

// Reconcile reads the ProjectConfig from the store, deletes it if its
// Project is gone, adopts it for a Project of the same name with another UID,
// and otherwise syncs its webhook receivers and writes the resulting status,
// unless that status is already stored.
func (r *reconciler) Reconcile(
	ctx context.Context,
	key projectconfigstore.Key,
) (engine.Result, error) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"project-config", key.Project,
	)
	ctx = logging.ContextWithLogger(ctx, logger)

	stored, err := r.store.GetUnfiltered(ctx, key.Project)
	if errors.Is(err, database.ErrNotFound) {
		// Deleted after the request was pushed.
		return engine.Result{}, nil
	}
	if err != nil {
		return engine.Result{}, fmt.Errorf("error getting ProjectConfig: %w", err)
	}

	liveUID, err := r.liveProjectUID(ctx, key.Project)
	if err != nil {
		return engine.Result{}, err
	}
	switch liveUID {
	case "":
		logger.Debug(
			"deleting ProjectConfig left behind by a deleted Project",
			"projectUID", stored.ProjectUID,
		)
		if err = r.store.DeleteForProject(ctx, key.Project, stored.ProjectUID); err != nil {
			return engine.Result{}, fmt.Errorf("error deleting leftover ProjectConfig: %w", err)
		}
		return engine.Result{}, nil
	case stored.ProjectUID:
	default:
		// The Project was replaced without its deletion being seen, as when
		// a cluster is restored from a backup while the database is kept.
		// The ProjectConfig follows the Project by name.
		logger.Info(
			"adopting ProjectConfig for a Project of the same name with a new UID",
			"oldProjectUID", stored.ProjectUID,
			"projectUID", liveUID,
		)
		if _, err = r.store.AdoptForProject(ctx, key.Project, liveUID); err != nil {
			return engine.Result{}, fmt.Errorf("error adopting ProjectConfig: %w", err)
		}
		return engine.Result{RequeueAfter: adoptRequeueDelay}, nil
	}

	projectCfg := projectconfigstore.ToAPI(stored)

	logger.Debug("reconciling ProjectConfig")
	newStatus, reconcileErr := r.syncWebhookReceivers(ctx, projectCfg)
	logger.Debug("done reconciling ProjectConfig")

	req := wire.StatusRequest{
		ObservedUID:        stored.ID.String(),
		ObservedGeneration: stored.Generation,
		// The store records the observed token as status.lastHandledRefresh.
		ObservedRefreshToken: deref(stored.RefreshToken),
		Conditions:           newStatus.Conditions,
		WebhookReceivers:     newStatus.WebhookReceivers,
	}
	// Receivers whose paths another ProjectConfig owns are reported before
	// the write, so that a status already reporting them is not rewritten.
	// They are applied to the computed conditions, before the merge, so
	// that Ready keeps its stored transition time.
	claimed, err := r.claimedElsewhere(ctx, key.Project, req.WebhookReceivers)
	if err != nil {
		return engine.Result{}, err
	}
	if len(claimed) != 0 {
		req = withSkippedReceivers(req, claimed)
	}
	req.Conditions = mergeConditions(projectCfg.Status.Conditions, req.Conditions)

	res, err := r.writeStatus(ctx, key.Project, projectCfg.Status, req)
	if err != nil {
		if reconcileErr != nil {
			logger.Error(
				err,
				"failed to update ProjectConfig status after reconciliation error",
			)
			return engine.Result{}, reconcileErr
		}
		return engine.Result{}, err
	}
	if res.RequeueAfter > 0 {
		// The ProjectConfig changed under the reconcile, which runs again.
		return res, nil
	}
	return engine.Result{}, reconcileErr
}

// liveProjectUID returns the UID of the Project with the given name, or ""
// if there is none. The Project is read uncached, so that a Project the
// cache has not seen yet is not mistaken for a missing one.
func (r *reconciler) liveProjectUID(ctx context.Context, name string) (string, error) {
	project := &kargoapi.Project{}
	err := r.apiReader.Get(ctx, client.ObjectKey{Name: name}, project)
	if apierrors.IsNotFound(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("error getting Project %q: %w", name, err)
	}
	return string(project.UID), nil
}

// claimedElsewhere returns the names of the receivers whose paths the
// ProjectConfig of another Project owns. A path no ProjectConfig owns, or
// that this one owns, is not claimed elsewhere.
func (r *reconciler) claimedElsewhere(
	ctx context.Context,
	project string,
	receivers []kargoapi.WebhookReceiverDetails,
) ([]string, error) {
	var claimed []string
	for _, receiver := range receivers {
		owner, err := r.store.GetByWebhookReceiverPath(ctx, receiver.Path)
		if errors.Is(err, database.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf(
				"error looking up the owner of webhook receiver %q: %w",
				receiver.Name, err,
			)
		}
		if owner.ProjectName != project {
			claimed = append(claimed, receiver.Name)
		}
	}
	return claimed, nil
}

// writeStatus writes the status unless it equals the stored one. When the
// API server skips receivers whose paths another ProjectConfig claimed since
// claimedElsewhere looked, it writes the status again with Ready False
// naming them. A status write that lost a race with a change to the
// ProjectConfig requeues; one for a ProjectConfig that is gone is done (see
// writeErrorResult).
func (r *reconciler) writeStatus(
	ctx context.Context,
	project string,
	stored kargoapi.ProjectConfigStatus,
	req wire.StatusRequest,
) (engine.Result, error) {
	logger := logging.LoggerFromContext(ctx)
	if statusEqual(stored, req) {
		logger.Debug("ProjectConfig status is unchanged")
		return engine.Result{}, nil
	}
	skipped, err := r.status.UpdateProjectConfigStatus(ctx, project, req)
	if err != nil {
		return r.writeErrorResult(ctx, project, req, err)
	}
	if len(skipped) == 0 {
		return engine.Result{}, nil
	}

	logger.Info("webhook receivers skipped: paths owned elsewhere", "receivers", skipped)
	req = withSkippedReceivers(req, skipped)
	if skipped, err = r.status.UpdateProjectConfigStatus(ctx, project, req); err != nil {
		return r.writeErrorResult(ctx, project, req, err)
	}
	if len(skipped) != 0 {
		return engine.Result{}, fmt.Errorf(
			"webhook receivers %q were skipped writing a status without them",
			skipped,
		)
	}
	return engine.Result{}, nil
}

// writeErrorResult classifies a failed status write for the ProjectConfig
// whose row ID req observed. A conflict means the ProjectConfig changed since
// it was read and is requeued, quietly, since it is expected. Not found is
// checked against the store: if the ProjectConfig, or the Project it belonged
// to, is gone or replaced, there is nothing left to do. If the same
// ProjectConfig is still there, the API server answered from a view that
// lags the store, and the reconcile is requeued.
func (r *reconciler) writeErrorResult(
	ctx context.Context,
	project string,
	req wire.StatusRequest,
	err error,
) (engine.Result, error) {
	logger := logging.LoggerFromContext(ctx)
	var apiErr *apiserver.APIError
	if !errors.As(err, &apiErr) {
		return engine.Result{}, err
	}
	switch apiErr.StatusCode {
	case http.StatusConflict:
		logger.Debug(
			"ProjectConfig changed during reconcile; requeueing",
			"reason", apiErr.Message,
		)
		return engine.Result{RequeueAfter: conflictRequeueDelay}, nil
	case http.StatusNotFound:
		current, getErr := r.store.GetUnfiltered(ctx, project)
		if errors.Is(getErr, database.ErrNotFound) ||
			(getErr == nil && current.ID.String() != req.ObservedUID) {
			logger.Debug("ProjectConfig is gone; dropping its status")
			return engine.Result{}, nil
		}
		if getErr != nil {
			return engine.Result{}, fmt.Errorf(
				"error getting ProjectConfig after the API server reported it not found: %w",
				getErr,
			)
		}
		logger.Debug(
			"API server reported a stored ProjectConfig not found; requeueing",
			"reason", apiErr.Message,
		)
		return engine.Result{RequeueAfter: notFoundRequeueDelay}, nil
	}
	return engine.Result{}, err
}

// withSkippedReceivers returns the request without the skipped receivers and
// with Ready False, naming each of them.
func withSkippedReceivers(
	req wire.StatusRequest,
	skipped []string,
) wire.StatusRequest {
	req.WebhookReceivers = slices.DeleteFunc(
		slices.Clone(req.WebhookReceivers),
		func(d kargoapi.WebhookReceiverDetails) bool {
			return slices.Contains(skipped, d.Name)
		},
	)
	msgs := make([]string, len(skipped))
	for i, name := range skipped {
		msgs[i] = fmt.Sprintf(
			"webhook receiver %q: path is already claimed by another ProjectConfig",
			name,
		)
	}
	req.Conditions = slices.Clone(req.Conditions)
	meta.SetStatusCondition(&req.Conditions, metav1.Condition{
		Type:               kargoapi.ConditionTypeReady,
		Status:             metav1.ConditionFalse,
		Reason:             reasonWebhookReceiverPathClaimed,
		Message:            strings.Join(msgs, "; "),
		ObservedGeneration: req.ObservedGeneration,
	})
	return req
}

// mergeConditions applies the computed conditions to a copy of the stored
// ones and drops the stored ones not computed. A condition whose status is
// unchanged keeps its stored transition time; one whose status changed, or
// that is new, is stamped now.
func mergeConditions(
	stored []metav1.Condition,
	computed []metav1.Condition,
) []metav1.Condition {
	merged := slices.Clone(stored)
	for _, cond := range computed {
		cond.LastTransitionTime = metav1.Time{}
		meta.SetStatusCondition(&merged, cond)
	}
	return slices.DeleteFunc(merged, func(cond metav1.Condition) bool {
		return meta.FindStatusCondition(computed, cond.Type) == nil
	})
}

// statusEqual reports whether writing the request would leave the stored
// status as it is. Conditions are compared by type and receivers by name,
// since neither order is meaningful.
func statusEqual(stored kargoapi.ProjectConfigStatus, req wire.StatusRequest) bool {
	if stored.ObservedGeneration != req.ObservedGeneration ||
		stored.LastHandledRefresh != req.ObservedRefreshToken {
		return false
	}
	if len(stored.Conditions) != len(req.Conditions) {
		return false
	}
	for _, want := range req.Conditions {
		got := meta.FindStatusCondition(stored.Conditions, want.Type)
		if got == nil || !conditionEqual(*got, want) {
			return false
		}
	}
	return slices.Equal(sortedByName(stored.WebhookReceivers), sortedByName(req.WebhookReceivers))
}

func conditionEqual(a, b metav1.Condition) bool {
	return conditions.Equal(a, b) &&
		a.ObservedGeneration == b.ObservedGeneration &&
		a.LastTransitionTime.Equal(&b.LastTransitionTime)
}

func sortedByName(
	receivers []kargoapi.WebhookReceiverDetails,
) []kargoapi.WebhookReceiverDetails {
	sorted := slices.Clone(receivers)
	slices.SortFunc(sorted, func(a, b kargoapi.WebhookReceiverDetails) int {
		return strings.Compare(a.Name, b.Name)
	})
	return sorted
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (r *reconciler) syncWebhookReceivers(
	ctx context.Context,
	projectCfg *kargoapi.ProjectConfig,
) (kargoapi.ProjectConfigStatus, error) {
	logger := logging.LoggerFromContext(ctx)
	status := projectCfg.Status.DeepCopy()

	if len(projectCfg.Spec.WebhookReceivers) == 0 {
		logger.Debug("ProjectConfig does not define any webhook receiver configurations")
		status.WebhookReceivers = nil
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
			r.apiReader, // project scoped secrets are not cached so we need to query the api-server directly
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
