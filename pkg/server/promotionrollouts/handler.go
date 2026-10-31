// Package promotionrollouts serves PromotionRollouts, which live in the
// database rather than in Kubernetes, over the REST API.
package promotionrollouts

import (
	"context"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database/promotionrolloutstore"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/auth/can"
	"github.com/akuity/kargo/pkg/server/middleware"
	"github.com/akuity/kargo/pkg/server/targets"
	"github.com/akuity/kargo/pkg/server/user"
)

// Handler serves the PromotionRollout endpoints of one API server.
type Handler struct {
	// store holds the rollouts. It is nil when the server runs without a
	// database, in which case every endpoint responds 501.
	store promotionrolloutstore.Store
	// authorize checks that the caller may perform an operation. Access to
	// the database carries no authorization of its own, unlike access
	// through the Kubernetes client, so every route runs it before its
	// handler; see require.
	authorize func(context.Context, can.Access) error
	// kube reads the Stage and Freight a rollout is created for, with the
	// caller's authorization.
	kube client.Client
	// creator creates rollouts on behalf of the promote endpoints and this
	// handler alike. It is nil when store is.
	creator *Creator
	// createEnabled is whether this server creates rollouts. When it does
	// not, the create endpoint responds 501; rollouts can still be read.
	createEnabled bool
}

// New returns a Handler. The store may be nil; see the store field.
// targetLister is where the Targets a created rollout promotes to are read
// from; it is used only with a store. createEnabled is whether the create
// endpoint is served; see the createEnabled field.
func New(
	store promotionrolloutstore.Store,
	targetLister targets.Lister,
	authorize func(context.Context, can.Access) error,
	kube client.Client,
	createEnabled bool,
) *Handler {
	h := &Handler{
		store:         store,
		authorize:     authorize,
		kube:          kube,
		createEnabled: createEnabled,
	}
	if store != nil {
		h.creator = NewCreator(store, targetLister)
	}
	return h
}

// paramProject, paramStage and paramNumber are the route parameters the
// handlers read.
const (
	paramProject = "project"
	paramStage   = "stage"
	paramNumber  = "number"
)

// Register adds the PromotionRollout routes to a project's route group.
// Every route first confirms that a database is configured. Reads then
// check that the caller holds the verb on the Project's rollouts. A create
// is a promotion, so it is authorized by the promote verb on the Stage
// instead, which the handler checks once it knows the Stage; before that, a
// server that does not create rollouts refuses it.
//
// A rollout belongs to one Stage and is numbered among its rollouts, so it
// is created and read under the Stage, as a Promotion is created; the
// Project's rollouts are listed together.
func (h *Handler) Register(project *gin.RouterGroup) {
	requireDatabase := middleware.RequireFeature(h.store != nil, errDatabaseNotConfigured)
	project.GET("/promotion-rollouts", requireDatabase, h.require(can.List()), h.list)
	stage := project.Group("/stages/:stage/promotion-rollouts", requireDatabase)
	stage.POST("", middleware.RequireFeature(h.createEnabled, errCreateDisabled), h.create)
	stage.GET("/:number", h.require(can.Get()), h.get)
}

// require returns middleware that checks the caller may perform the verb on
// the Project's rollouts, and refuses the request if not, so that no handler
// reaches the store for a caller who may not. A rollout read by its number
// has no name to check until it is read, so access to one is access to all
// of the Project's.
func (h *Handler) require(verb can.Verb) gin.HandlerFunc {
	return func(c *gin.Context) {
		access := verb.PromotionRollout(c.Param(paramProject), "")
		if err := h.authorize(c.Request.Context(), access); err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}
}

// present returns the stored rollouts as the PromotionRollout resources the
// API serves, each with what has become of its Targets: the newest
// Promotion to each and its phase, and a tally of those. That comes from the
// mirrored Promotions the rollout's id links, so it lags the cluster by the
// mirror's sync.
//
// Those are Promotions, read from the database, which carries no
// authorization of its own. Reading them from Kubernetes would take list on
// the Project's Promotions, so it is required here too: a caller who may read
// rollouts but not Promotions is refused, as the API server refused them when
// it read the Promotions themselves.
func (h *Handler) present(
	ctx context.Context,
	project string,
	stored []promotionrolloutstore.Rollout,
) ([]*kargoapi.PromotionRollout, error) {
	rollouts := make([]*kargoapi.PromotionRollout, len(stored))
	if len(stored) == 0 {
		return rollouts, nil
	}
	if err := h.authorize(ctx, can.List().Promotion(project, "")); err != nil {
		return nil, err
	}
	for i, s := range stored {
		rows, err := h.store.ListTargetStatuses(ctx, s)
		if err != nil {
			return nil, err
		}
		rollouts[i] = promotionrolloutstore.ToAPI(s)
		rollouts[i].Status.Targets, rollouts[i].Status.Summary = promotionrolloutstore.TargetStatuses(rows)
	}
	return rollouts, nil
}

// logChange records a change to a PromotionRollout and who made it.
// PromotionRollouts are not Kubernetes resources, so the Kubernetes audit
// log never sees these.
func logChange(ctx context.Context, what, project, name string) {
	actor := "unknown"
	if id, ok := user.IdentityFromContext(ctx); ok {
		actor = id.Actor()
	}
	logging.LoggerFromContext(ctx).Info(
		what,
		"project", project,
		"promotionRollout", name,
		"actor", actor,
	)
}
