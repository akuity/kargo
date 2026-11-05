// Package promotionrollouts serves PromotionRollouts, which live in the
// database rather than in Kubernetes, over the REST API.
package promotionrollouts

import (
	"context"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database/promotionrolloutstore"
	"github.com/akuity/kargo/pkg/server/auth/can"
	"github.com/akuity/kargo/pkg/server/middleware"
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
	// creator builds the rollouts the create endpoint asks for. It is used
	// only with a store, and is nil without one, as the store is.
	creator Creator
	// createEnabled is whether this server creates rollouts. When it does
	// not, the create endpoint responds 501; rollouts can still be read.
	createEnabled bool
}

// New returns a Handler. The store may be nil; see the store field. The
// creator builds the rollouts the create endpoint asks for; it may be nil
// without a store, and is shared with the promote endpoints, which create
// rollouts the same way. createEnabled is whether the create endpoint is
// served; see the createEnabled field.
func New(
	store promotionrolloutstore.Store,
	creator Creator,
	authorize func(context.Context, can.Access) error,
	kube client.Client,
	createEnabled bool,
) *Handler {
	return &Handler{
		store:         store,
		creator:       creator,
		authorize:     authorize,
		kube:          kube,
		createEnabled: createEnabled,
	}
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
// check that the caller holds the verb on the Project's rollouts, and may
// list its Promotions; see require. A create is a promotion, so it is
// authorized by the promote verb on the Stage instead, which the handler
// checks once it knows the Stage; before that, a server that does not
// create rollouts refuses it.
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

// require returns middleware that checks the caller may read the Project's
// rollouts, and refuses the request if not, so that no handler reaches the
// store for a caller who may not. That takes the verb on the Project's
// rollouts: a rollout read by its number has no name to check until it is
// read, so access to one is access to all of the Project's. It also takes
// list on the Project's Promotions. A rollout is served with what has become
// of its Targets, which present reads from the mirrored Promotions in the
// database, and the database carries no authorization of its own; reading
// those Promotions from Kubernetes would take list on them, so it is
// required here too. A caller who may read rollouts but not Promotions is
// refused, as the API server refused them when it read the Promotions
// themselves.
func (h *Handler) require(verb can.Verb) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		project := c.Param(paramProject)
		for _, access := range []can.Access{
			verb.PromotionRollout(project, ""),
			can.List().Promotion(project, ""),
		} {
			if err := h.authorize(ctx, access); err != nil {
				_ = c.Error(err)
				c.Abort()
				return
			}
		}
	}
}

// present returns the stored rollouts as the PromotionRollout resources the
// API serves, each with what has become of its Targets: the newest
// Promotion to each and its phase, and a tally of those. That comes from the
// mirrored Promotions the rollouts' ids link, read in one query for all of
// them, so it lags the cluster by the mirror's sync. That the caller may
// list the Project's Promotions has been checked by then; see require.
func (h *Handler) present(
	ctx context.Context,
	stored []promotionrolloutstore.Rollout,
) ([]*kargoapi.PromotionRollout, error) {
	statuses, err := h.store.ListTargetStatuses(ctx, stored)
	if err != nil {
		return nil, err
	}
	rollouts := make([]*kargoapi.PromotionRollout, len(stored))
	for i, s := range stored {
		rollouts[i] = promotionrolloutstore.ToAPI(s)
		rollouts[i].Status.Targets, rollouts[i].Status.Summary =
			promotionrolloutstore.TargetStatuses(statuses[s.ID])
	}
	return rollouts, nil
}
