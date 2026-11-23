// Package targetstatuses serves read-only views of promotion to a Project's
// Targets, read from the database's record of Promotions rather than from
// Kubernetes, for pages that would otherwise load every Promotion in the
// Project to show what has been promoted to one Target.
//
// What is served is a view, in a shape of its own rather than a resource's,
// and where it is read from can change without the API changing.
package targetstatuses

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/akuity/kargo/pkg/database/targetstatusstore"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/auth/can"
	"github.com/akuity/kargo/pkg/server/middleware"
)

var errDatabaseNotConfigured = libhttp.ErrorStr(
	"database is not configured",
	http.StatusNotImplemented,
)

// Handler serves these views for one API server.
type Handler struct {
	// store reads what is on record. It is nil when the server runs without
	// a database, in which case every endpoint responds 501.
	store targetstatusstore.Store
	// authorize checks that the caller may perform an operation. Access to
	// the database carries no authorization of its own, unlike access
	// through the Kubernetes client, so every route runs it before its
	// handler; see requireTarget.
	authorize func(context.Context, can.Access) error
}

// New returns a Handler. The store may be nil; see the store field.
func New(
	store targetstatusstore.Store,
	authorize func(context.Context, can.Access) error,
) *Handler {
	return &Handler{store: store, authorize: authorize}
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
	queryContinue = "continue"
	queryLimit    = "limit"
)

// Register adds the routes to a project's route group. Every route first
// confirms that a database is configured and that the caller may read what
// the response is derived from.
func (h *Handler) Register(project *gin.RouterGroup) {
	requireDatabase := middleware.RequireFeature(h.store != nil, errDatabaseNotConfigured)
	project.GET(
		"/targets/:"+paramTarget+"/promotions",
		requireDatabase,
		h.requireTarget(),
		h.listPromotions,
	)
}

// requireTarget returns middleware that checks the caller may read the
// Promotions to the Target the route names, and refuses the request if not,
// so that no handler reaches the store for a caller who may not. The rows
// are the database's, which carries no authorization of its own: reading the
// same Promotions from Kubernetes would take list on the Project's
// Promotions, since the history includes those of every Stage, so that is
// required here, with get on the Target itself.
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
