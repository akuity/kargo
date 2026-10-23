// Package promotionrollouts contains the PromotionRollout controller.
//
// PromotionRollouts live in the database, so the controller is built on
// pkg/reconciler rather than controller-runtime: its one source relists the
// open rollouts from the store on an interval, and its reconcile function
// reads each one fresh and acts on what it finds.
//
// Every reconciler of rollouts first resolves a rollout's Targets: the
// Stage's selectors, evaluated once against the Project's Targets when the
// rollout is first seen, and recorded on it for good; see ResolveTargets. In Kargo OSS that is where
// the work ends: the reconciler reports on every open rollout that fanning
// Freight out to Targets is a Kargo Enterprise-only feature, so that a user
// whose Stage has stopped promoting can find out why from the rollout the
// Stage produced.
//
// There is deliberately no hook to swap this behavior out. A reconciler that
// fans a rollout out into Promotions must also watch the child Promotions
// that drive it forward, and a watch is not something a swappable reconcile
// body can contribute; such a reconciler is its own controller, built from
// the same store and the same engine.
package promotionrollouts

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/promotionrolloutstore"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/reconciler"
	"github.com/akuity/kargo/pkg/reconciler/kube"
	"github.com/akuity/kargo/pkg/reconciler/list"
	"github.com/akuity/kargo/pkg/telemetry"
)

const (
	controllerName = "promotion-rollouts"

	enterpriseOnlyReason  = "EnterpriseOnlyFeature"
	enterpriseOnlyMessage = "PromotionRollouts are a Kargo Enterprise-only feature"

	// reconcileTimeout bounds one reconcile, so a blocked query or an
	// unreachable database fails the key for retry with backoff instead of
	// holding a worker indefinitely.
	reconcileTimeout = 5 * time.Second
	// resyncJitter spreads the relists of replicas that started together.
	resyncJitter = 0.2
)

// tracer is the instrumentation scope under which this package's spans are
// recorded.
var tracer = otel.Tracer("github.com/akuity/kargo/pkg/controller/promotionrollouts")

// ReconcilerConfig represents configuration for the PromotionRollout
// reconciler.
type ReconcilerConfig struct {
	// ResyncInterval is how often every open PromotionRollout is relisted
	// from the database. There is no change feed, so this is also how soon
	// a new rollout is first reconciled.
	ResyncInterval time.Duration `envconfig:"PROMOTION_ROLLOUT_RESYNC_INTERVAL" default:"1m"`
	// MaxConcurrentReconciles is the number of PromotionRollouts reconciled
	// at once.
	MaxConcurrentReconciles int `envconfig:"MAX_CONCURRENT_PROMOTION_ROLLOUT_RECONCILES" default:"4"`
}

// ReconcilerConfigFromEnv returns a ReconcilerConfig populated from
// environment variables.
func ReconcilerConfigFromEnv() ReconcilerConfig {
	cfg := ReconcilerConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// Store is the slice of promotionrolloutstore.Store the controller uses.
type Store interface {
	Get(context.Context, string, string) (promotionrolloutstore.Rollout, error)
	ListOpen(context.Context) ([]promotionrolloutstore.Key, error)
	ResolveTargets(context.Context, promotionrolloutstore.Rollout, []uuid.UUID) (promotionrolloutstore.Rollout, error)
	UpdateStatus(context.Context, string, database.PromotionRolloutRow) (database.PromotionRolloutRow, error)
}

// TargetLister lists a Project's Targets, which a rollout's are resolved
// from. targetstore.Store is one.
type TargetLister interface {
	List(context.Context, string) ([]database.TargetRow, error)
}

// SetupWithManager builds the PromotionRollout controller and adds it to
// the manager, which starts it once it holds the leader lease. Stages are
// read through the manager's client; the caller owns the database behind
// the stores.
func SetupWithManager(
	ctx context.Context,
	mgr manager.Manager,
	cfg ReconcilerConfig,
	store Store,
	targets TargetLister,
) error {
	c, err := reconciler.New[promotionrolloutstore.Key](controllerName).
		Watch(list.New(store.ListOpen).Every(cfg.ResyncInterval).Jitter(resyncJitter)).
		Workers(cfg.MaxConcurrentReconciles).
		Func(newReconciler(mgr.GetClient(), store, targets))
	if err != nil {
		return fmt.Errorf("error building PromotionRollout reconciler: %w", err)
	}
	if err = mgr.Add(kube.Runnable(c)); err != nil {
		return fmt.Errorf("error adding PromotionRollout reconciler to manager: %w", err)
	}
	logging.LoggerFromContext(ctx).Info(
		"Initialized PromotionRollout reconciler",
		"maxConcurrentReconciles", cfg.MaxConcurrentReconciles,
		"resyncInterval", cfg.ResyncInterval,
	)
	return nil
}

// newReconciler returns the reconcile function. It is handed only a key,
// reads the rollout fresh, resolves its Targets if that has not been done
// and, unless it has already ended, ends it as an Enterprise-only feature.
func newReconciler(
	reader client.Reader,
	store Store,
	targets TargetLister,
) reconciler.Func[promotionrolloutstore.Key] {
	return func(ctx context.Context, key promotionrolloutstore.Key) (reconciler.Result, error) {
		ctx, cancel := context.WithTimeout(ctx, reconcileTimeout)
		defer cancel()
		ctx, span := tracer.Start(
			ctx,
			"Reconcile PromotionRollout",
			trace.WithAttributes(
				telemetry.ProjectKey.String(key.Project),
				telemetry.PromotionRolloutKey.String(key.Name),
			),
		)
		defer span.End()
		logger := logging.LoggerFromContext(ctx).WithValues(
			"project", key.Project,
			"promotionRollout", key.Name,
		)

		rollout, err := store.Get(ctx, key.Project, key.Name)
		if errors.Is(err, database.ErrNotFound) {
			// Gone between the push and now: nothing left to do.
			return reconciler.Result{}, nil
		}
		if err != nil {
			return reconciler.Result{}, fmt.Errorf("error getting PromotionRollout: %w", err)
		}
		if kargoapi.PromotionRolloutPhase(rollout.StatusPhase).IsTerminal() {
			// Level-triggered: a rollout that has ended needs nothing from
			// this controller however often it is pushed.
			return reconciler.Result{}, nil
		}

		if rollout, err = ResolveTargets(ctx, reader, store, targets, rollout); err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return reconciler.Result{}, nil
			}
			return reconciler.Result{}, err
		}

		row := rollout.PromotionRolloutRow
		markEnterpriseOnly(&row, time.Now())
		// row still carries the ID and UpdatedAt read above, so a write that
		// landed in between is refused with ErrConflict and the key retried
		// against a fresh read.
		if _, err = store.UpdateStatus(ctx, key.Project, row); err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return reconciler.Result{}, nil
			}
			return reconciler.Result{}, fmt.Errorf("error updating PromotionRollout status: %w", err)
		}
		logger.Debug("reported PromotionRollout as an enterprise-only feature")
		return reconciler.Result{}, nil
	}
}

// ResolveTargets records, once, the Targets a rollout promotes to: those
// of its Project that its Stage's selectors match when this runs. A rollout
// already resolved is returned as it is. A rollout whose Stage is gone, or
// no longer selects Targets, resolves to none: its Promotions would have
// nowhere to go, and the reconciler that runs it will say so in its status.
// Selectors that cannot be parsed resolve to none for the same reason.
// Every rollout reconciler calls this before acting on a rollout.
func ResolveTargets(
	ctx context.Context,
	reader client.Reader,
	store Store,
	targets TargetLister,
	rollout promotionrolloutstore.Rollout,
) (promotionrolloutstore.Rollout, error) {
	if rollout.TargetsResolvedAt != nil {
		return rollout, nil
	}
	logger := logging.LoggerFromContext(ctx)

	ids, err := governedTargetIDs(ctx, reader, targets, rollout)
	if err != nil {
		return rollout, err
	}
	resolved, err := store.ResolveTargets(ctx, rollout, ids)
	if errors.Is(err, database.ErrConflict) {
		// Another reconciler got there first, or a Target was deleted as
		// we resolved. Read it afresh and go with what is there; a second
		// miss means the Target really is gone and the key is retried.
		if resolved, err = store.Get(ctx, rollout.ProjectName, rollout.Name); err == nil &&
			resolved.TargetsResolvedAt == nil {
			err = fmt.Errorf("error resolving Targets of PromotionRollout: %w", database.ErrConflict)
		}
	}
	if err != nil {
		return rollout, fmt.Errorf("error resolving Targets of PromotionRollout: %w", err)
	}
	logger.Debug("resolved PromotionRollout Targets", "targets", len(resolved.Targets))
	return resolved, nil
}

// governedTargetIDs returns the ids of the Project's Targets that the
// rollout's Stage governs, or none when the Stage is gone, no longer
// selects Targets, or has selectors that cannot be parsed.
func governedTargetIDs(
	ctx context.Context,
	reader client.Reader,
	targets TargetLister,
	rollout promotionrolloutstore.Rollout,
) ([]uuid.UUID, error) {
	logger := logging.LoggerFromContext(ctx)
	stage := &kargoapi.Stage{}
	if err := reader.Get(
		ctx,
		client.ObjectKey{Namespace: rollout.ProjectName, Name: rollout.StageName},
		stage,
	); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("error getting Stage %q: %w", rollout.StageName, err)
		}
		logger.Info("Stage of PromotionRollout not found; it governs no Targets", "stage", rollout.StageName)
		return nil, nil
	}
	selectors, err := api.TargetSelectorsForStage(stage)
	if err != nil {
		logger.Info(
			"Stage of PromotionRollout has invalid target selectors; it governs no Targets",
			"stage", rollout.StageName,
			"error", err.Error(),
		)
		return nil, nil
	}
	if len(selectors) == 0 {
		return nil, nil
	}
	rows, err := targets.List(ctx, rollout.ProjectName)
	if err != nil {
		return nil, fmt.Errorf("error listing Targets: %w", err)
	}
	var ids []uuid.UUID
	for _, row := range rows {
		if api.AnySelectorMatches(selectors, row.Labels) {
			ids = append(ids, row.ID)
		}
	}
	return ids, nil
}

// markEnterpriseOnly ends a rollout as Errored with the Enterprise-only
// message. The reason is what the API serves as the Ready condition's,
// which is where the UI reads it from.
func markEnterpriseOnly(row *database.PromotionRolloutRow, now time.Time) {
	row.StatusPhase = string(kargoapi.PromotionRolloutPhaseErrored)
	row.StatusReason = enterpriseOnlyReason
	row.StatusMessage = enterpriseOnlyMessage
	if row.StatusFinishedAt == nil {
		row.StatusFinishedAt = &now
	}
}
