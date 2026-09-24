// Package promotionrequests contains the PromotionRequest reconciler.
//
// PromotionRequests live in the database, so the reconciler is a dbreconcile
// controller rather than a controller-runtime one. It reconciles a request
// whenever an event about it arrives on the kargo.promotionrequests subjects,
// and resyncs every open request on an interval in case an event was lost.
// Each open request is handed to a StatusHandler, which decides the status it
// should now have; the reconciler persists whatever changed and announces the
// change.
package promotionrequests

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
	"github.com/nats-io/nats.go"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/dbreconcile"
	"github.com/akuity/kargo/pkg/logging"
)

const (
	// Subject matches every subject PromotionRequest events are published to.
	Subject = "kargo.promotionrequests.>"
	// SubjectCreated is where the creation of a PromotionRequest is announced.
	SubjectCreated = "kargo.promotionrequests.created"
	// SubjectUpdated is where a change to a PromotionRequest is announced.
	SubjectUpdated = "kargo.promotionrequests.updated"

	controllerName = "promotion-requests"

	enterpriseOnlyMessage = "PromotionRequests are a Kargo Enterprise-only feature"
)

// Event is an event about a PromotionRequest, keyed by its id. It carries the
// request as the database holds it, which is what its publisher has just
// written and what the reconciler reads.
type Event = dbreconcile.Event[uuid.UUID, database.PromotionRequestSnapshot]

// ReconcilerConfig represents configuration for the PromotionRequest
// reconciler.
type ReconcilerConfig struct {
	// Interval is the time between resyncs of every open PromotionRequest.
	// Requests are reconciled as soon as events about them arrive, so this
	// only bounds how long a request can wait when an event is lost.
	Interval time.Duration `envconfig:"PROMOTION_REQUEST_RECONCILE_INTERVAL" default:"2m"`
	// ResyncPageSize is the number of open PromotionRequest ids a resync reads
	// at a time.
	ResyncPageSize int `envconfig:"PROMOTION_REQUEST_RESYNC_PAGE_SIZE" default:"100"`
	// MaxConcurrentReconciles is the number of PromotionRequests reconciled
	// at once.
	MaxConcurrentReconciles int `envconfig:"MAX_CONCURRENT_PROMOTION_REQUEST_RECONCILES" default:"4"`
}

// ReconcilerConfigFromEnv returns a ReconcilerConfig populated from
// environment variables.
func ReconcilerConfigFromEnv() ReconcilerConfig {
	cfg := ReconcilerConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// StatusHandler decides the status an open PromotionRequest should now have.
// It is handed the request as it stands and returns the status to persist; a
// status equal to the current one is not written. Kargo Enterprise supplies
// the StatusHandler that fans a request out into Promotions.
type StatusHandler interface {
	Handle(context.Context, *kargoapi.PromotionRequest) (kargoapi.PromotionRequestStatus, error)
}

// enterpriseOnlyHandler reports on every open PromotionRequest that fanning
// Freight out to Targets is a Kargo Enterprise-only feature, so that a user
// whose Stage has stopped promoting can find out why from the request it
// produced.
type enterpriseOnlyHandler struct{}

// NewEnterpriseOnlyHandler returns the StatusHandler that ends every open
// PromotionRequest with an explanation that fan-out is Enterprise-only.
func NewEnterpriseOnlyHandler() StatusHandler {
	return enterpriseOnlyHandler{}
}

func (enterpriseOnlyHandler) Handle(
	_ context.Context,
	request *kargoapi.PromotionRequest,
) (kargoapi.PromotionRequestStatus, error) {
	status := *request.Status.DeepCopy()
	status.Phase = kargoapi.PromotionRequestPhaseErrored
	status.Message = enterpriseOnlyMessage
	if status.FinishedAt == nil {
		now := metav1.Now()
		status.FinishedAt = &now
	}
	return status, nil
}

// PublishCreated announces a PromotionRequest that has just been created, so
// that the reconciler takes it up without waiting for a resync. The resync is
// the safety net, so callers treat a failure to publish as something to log,
// not as a failure to create the request. It is a no-op when conn is nil.
func PublishCreated(conn *nats.Conn, created database.PromotionRequestSnapshot) error {
	return dbreconcile.Publish(conn, SubjectCreated, Event{
		Kind: dbreconcile.Created,
		Key:  created.ID,
		New:  &created,
	})
}

// publishUpdated announces a change to a PromotionRequest, carrying it as it
// was before and after.
func publishUpdated(conn *nats.Conn, before, after database.PromotionRequestSnapshot) error {
	return dbreconcile.Publish(conn, SubjectUpdated, Event{
		Kind: dbreconcile.Updated,
		Key:  after.ID,
		Old:  &before,
		New:  &after,
	})
}

// requestPredicate decides which events about PromotionRequests warrant a
// reconcile. A request's Stage and Freight never change, so the only change
// that asks for new work is to the Targets it fans out to. Everything else
// the reconciler writes itself -- phase, message, timestamps and each Target's
// promotion and phase -- and it announces every write, so an update that
// changed only those is ignored; otherwise every reconcile would cause the
// next. A deleted request has nothing left to reconcile.
var requestPredicate = dbreconcile.Predicate[uuid.UUID, database.PromotionRequestSnapshot]{
	Update: func(e Event) bool {
		return !slices.Equal(targetNames(e.Old), targetNames(e.New))
	},
	Delete: func(Event) bool { return false },
}

// targetNames returns the names of the request's Targets, in order.
func targetNames(snapshot *database.PromotionRequestSnapshot) []string {
	names := make([]string, len(snapshot.Targets))
	for i, target := range snapshot.Targets {
		names[i] = target.Name
	}
	return names
}

// promotionRequestStore is the slice of database.Store the reconciler uses.
type promotionRequestStore interface {
	GetPromotionRequestByID(context.Context, uuid.UUID) (database.PromotionRequestSnapshot, error)
	ListOpenPromotionRequestIDs(ctx context.Context, after uuid.UUID, limit int) ([]uuid.UUID, error)
	UpdatePromotionRequestStatus(
		context.Context,
		uuid.UUID,
		kargoapi.PromotionRequestStatus,
	) (before database.PromotionRequestSnapshot, after database.PromotionRequestSnapshot, err error)
}

// SetupWithManager registers the PromotionRequest reconciler with the
// manager. It watches the PromotionRequest subjects on conn and resyncs the
// open requests every cfg.Interval, handing each to the StatusHandler.
func SetupWithManager(
	mgr dbreconcile.Manager,
	store database.Store,
	handler StatusHandler,
	conn *nats.Conn,
	cfg ReconcilerConfig,
) error {
	if store == nil {
		return errors.New("PromotionRequest reconciler requires a store")
	}
	return setup(mgr, store, handler, conn, cfg)
}

func setup(
	mgr dbreconcile.Manager,
	store promotionRequestStore,
	handler StatusHandler,
	conn *nats.Conn,
	cfg ReconcilerConfig,
) error {
	if handler == nil {
		return errors.New("PromotionRequest reconciler requires a status handler")
	}
	if conn == nil {
		return errors.New("PromotionRequest reconciler requires a NATS connection")
	}
	if err := dbreconcile.NewControllerManagedBy[uuid.UUID](mgr).
		Named(controllerName).
		Watches(dbreconcile.NewWatch(
			dbreconcile.Subject[uuid.UUID, database.PromotionRequestSnapshot](conn, Subject),
			dbreconcile.EnqueueKey[uuid.UUID, database.PromotionRequestSnapshot](),
			requestPredicate,
		)).
		Resync(dbreconcile.ListerFunc[uuid.UUID](store.ListOpenPromotionRequestIDs), cfg.Interval).
		WithResyncPageSize(cfg.ResyncPageSize).
		WithWorkers(cfg.MaxConcurrentReconciles).
		Complete(&reconciler{store: store, handler: handler, conn: conn}); err != nil {
		return fmt.Errorf("error setting up PromotionRequest reconciler: %w", err)
	}
	return nil
}

// reconciler hands an open PromotionRequest to the StatusHandler and persists
// the status it returns.
type reconciler struct {
	store   promotionRequestStore
	handler StatusHandler
	conn    *nats.Conn
}

func (r *reconciler) Reconcile(
	ctx context.Context,
	req dbreconcile.Request[uuid.UUID],
) (dbreconcile.Result, error) {
	logger := logging.LoggerFromContext(ctx).WithValues("promotionRequestID", req.Key)

	snapshot, err := r.store.GetPromotionRequestByID(ctx, req.Key)
	if errors.Is(err, database.ErrNotFound) {
		logger.Debug("PromotionRequest no longer exists")
		return dbreconcile.Result{}, nil
	}
	if err != nil {
		return dbreconcile.Result{}, fmt.Errorf("error getting PromotionRequest %s: %w", req.Key, err)
	}
	request := database.PromotionRequestFromSnapshot(snapshot)
	logger = logger.WithValues("project", request.Namespace, "promotionRequest", request.Name)
	if request.Status.Phase.IsTerminal() {
		logger.Debug("PromotionRequest has already finished")
		return dbreconcile.Result{}, nil
	}

	status, err := r.handler.Handle(ctx, &request)
	if err != nil {
		return dbreconcile.Result{}, fmt.Errorf(
			"error handling PromotionRequest %q in Project %q: %w",
			request.Name, request.Namespace, err,
		)
	}
	if equality.Semantic.DeepEqual(status, request.Status) {
		logger.Debug("PromotionRequest status is unchanged")
		return dbreconcile.Result{}, nil
	}

	before, after, err := r.store.UpdatePromotionRequestStatus(ctx, req.Key, status)
	if errors.Is(err, database.ErrNotFound) {
		// Deleted since it was read; there is nothing left to update.
		logger.Debug("PromotionRequest no longer exists")
		return dbreconcile.Result{}, nil
	}
	if err != nil {
		return dbreconcile.Result{}, fmt.Errorf(
			"error updating status of PromotionRequest %q in Project %q: %w",
			request.Name, request.Namespace, err,
		)
	}
	logger.Debug("updated PromotionRequest status", "phase", status.Phase)

	// The write has committed, so it is announced whatever happens next. The
	// announcement is for other components; requestPredicate keeps it from
	// causing another reconcile here.
	if err = publishUpdated(r.conn, before, after); err != nil {
		logger.Error(err, "error announcing PromotionRequest update")
	}
	return dbreconcile.Result{}, nil
}
