// Package promotionrollouts serves PromotionRollouts, which live in the
// database rather than in Kubernetes, over the REST API.
package promotionrollouts

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/database/promotionrolloutstore"
	"github.com/akuity/kargo/pkg/kubernetes"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/auth/can"
	"github.com/akuity/kargo/pkg/server/middleware"
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
	// kube reads the Stage and Freight a rollout is created for, and the
	// Promotions that report what has become of a rollout's Targets, with
	// the caller's authorization.
	kube client.Client
	// creator creates rollouts on behalf of the promote endpoints and this
	// handler alike. It is nil when store is.
	creator *Creator
}

// New returns a Handler. The store may be nil; see the store field.
func New(
	store promotionrolloutstore.Store,
	authorize func(context.Context, can.Access) error,
	kube client.Client,
) *Handler {
	h := &Handler{store: store, authorize: authorize, kube: kube}
	if store != nil {
		h.creator = NewCreator(store)
	}
	return h
}

// paramProject and paramName are the route parameters the handlers read.
// paramName is not "promotion-rollout" because the generated clients could
// not hold a path parameter and a PromotionRollout schema of the same name.
const (
	paramProject = "project"
	paramName    = "promotion-rollout-name"
)

// Register adds the PromotionRollout routes to a project's route group.
// Every route first confirms that a database is configured. Reads then
// check that the caller holds the verb on the rollout the route names or,
// for the collection, on the Project's rollouts. A create is a promotion,
// so it is authorized by the promote verb on the Stage instead, which the
// handler checks once it knows the Stage.
func (h *Handler) Register(project *gin.RouterGroup) {
	rollouts := project.Group(
		"/promotion-rollouts",
		middleware.RequireFeature(h.store != nil, errDatabaseNotConfigured),
	)
	rollouts.GET("", h.require(can.List()), h.list)
	rollouts.POST("", h.create)
	rollouts.GET("/:promotion-rollout-name", h.require(can.Get()), h.get)
}

// require returns middleware that checks the caller may perform the verb on
// the rollout the route names, or on the Project's rollouts for the
// collection, and refuses the request if not, so that no handler reaches
// the store for a caller who may not.
func (h *Handler) require(verb can.Verb) gin.HandlerFunc {
	return func(c *gin.Context) {
		access := verb.PromotionRollout(c.Param(paramProject), c.Param(paramName))
		if err := h.authorize(c.Request.Context(), access); err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}
}

// fillTargetStatus sets, on each rollout, what has become of its Targets:
// the newest Promotion to each and its phase, and a tally of those. The
// database records none of that; the Promotions say so themselves. They are
// listed once, narrowed to the Stage when every rollout is the same Stage's,
// with the caller's authorization: a caller who may read rollouts but not
// Promotions is refused.
func (h *Handler) fillTargetStatus(
	ctx context.Context,
	project string,
	stage string,
	rollouts []*kargoapi.PromotionRollout,
) error {
	if len(rollouts) == 0 {
		return nil
	}
	opts := []client.ListOption{client.InNamespace(project)}
	if stage != "" {
		opts = append(opts, client.MatchingLabels{
			kargoapi.LabelKeyStage: kubernetes.ShortenLabelValue(stage),
		})
	}
	promos := &kargoapi.PromotionList{}
	if err := h.kube.List(ctx, promos, opts...); err != nil {
		return fmt.Errorf("error listing Promotions in Project %q: %w", project, err)
	}
	for _, rollout := range rollouts {
		rollout.Status.Targets, rollout.Status.Summary = api.PromotionRolloutTargetStatuses(
			rollout.Name,
			promos.Items,
		)
	}
	return nil
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
