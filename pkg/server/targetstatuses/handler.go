// Package targetstatuses serves where each Target stands with the Stages
// that govern it: read-only views for pages that show a whole fleet, which
// would otherwise load every Target and every Promotion to work it out.
//
// Nothing records a Target's status. Each request derives it from the
// mirrored Promotions and the PromotionRollouts already in the database, for
// the Targets each Stage governs as the Stage's selectors say at the time of
// the request. What is served is therefore a view, in a shape of its own
// rather than a resource's, and where it is read from can change without the
// API changing.
package targetstatuses

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/pkg/database/targetstatusstore"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/auth/can"
	"github.com/akuity/kargo/pkg/server/middleware"
)

var errDatabaseNotConfigured = libhttp.ErrorStr(
	"database is not configured",
	http.StatusNotImplemented,
)

// Handler serves the Target status endpoints of one API server.
type Handler struct {
	// store reads the statuses. It is nil when the server runs without a
	// database, in which case every endpoint responds 501.
	store targetstatusstore.Store
	// authorize checks that the caller may perform an operation. Access to
	// the database carries no authorization of its own, unlike access
	// through the Kubernetes client, so every route runs it before its
	// handler; see requireFleet and requireTarget.
	authorize func(context.Context, can.Access) error
	// kube reads the Project's Stages, for their selectors and their current
	// Freight, with the caller's authorization.
	kube client.Reader
}

// New returns a Handler. The store may be nil; see the store field.
func New(
	store targetstatusstore.Store,
	authorize func(context.Context, can.Access) error,
	kube client.Reader,
) *Handler {
	return &Handler{store: store, authorize: authorize, kube: kube}
}

// paramProject and paramTarget are the route parameters the handlers read.
// paramTarget is the name the Target routes give theirs, since the history
// of a Target's Promotions is served beneath them.
const (
	paramProject = "project"
	paramTarget  = "target-name"
)

// The query parameters the endpoints share.
const (
	queryStage    = "stage"
	queryState    = "state"
	queryContinue = "continue"
	queryLimit    = "limit"
)

// Register adds the Target status routes to a project's route group. Every
// route first confirms that a database is configured and that the caller may
// read what the response is derived from.
//
// The statuses are a collection of their own rather than a path beneath the
// Project's Targets, where "summary" would shadow a Target of that name.
func (h *Handler) Register(project *gin.RouterGroup) {
	requireDatabase := middleware.RequireFeature(h.store != nil, errDatabaseNotConfigured)
	statuses := project.Group("/target-statuses", requireDatabase, h.requireFleet())
	statuses.GET("", h.list)
	statuses.GET("/summary", h.summarize)
	project.GET(
		"/targets/:"+paramTarget+"/promotions",
		requireDatabase,
		h.requireTarget(),
		h.listPromotions,
	)
}

// requireFleet returns middleware that checks the caller may read what the
// statuses of a Project's Targets are derived from, and refuses the request
// if not, so that no handler reaches the store for a caller who may not.
// The rows are the database's, which carries no authorization of its own:
// reading the same things from Kubernetes would take list on the Project's
// Targets, on its Promotions, whose records say how each promotion went, and
// on its PromotionRollouts, whose records say which Targets one has yet to
// reach or never did. So all three are required here. The Stages themselves
// are read through the Kubernetes client, which authorizes that read itself.
func (h *Handler) requireFleet() gin.HandlerFunc {
	return func(c *gin.Context) {
		project := c.Param(paramProject)
		h.require(
			c,
			can.List().Target(project, ""),
			can.List().Promotion(project, ""),
			can.List().PromotionRollout(project, ""),
		)
	}
}

// requireTarget returns middleware that checks the caller may read the
// Promotions to the Target the route names: get on the Target, and list on
// the Project's Promotions, since the history is their mirrored records and
// includes those of every Stage.
func (h *Handler) requireTarget() gin.HandlerFunc {
	return func(c *gin.Context) {
		project := c.Param(paramProject)
		h.require(
			c,
			can.Get().Target(project, c.Param(paramTarget)),
			can.List().Promotion(project, ""),
		)
	}
}

// require refuses the request unless the caller holds every access.
func (h *Handler) require(c *gin.Context, accesses ...can.Access) {
	for _, access := range accesses {
		if err := h.authorize(c.Request.Context(), access); err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}
	}
}
