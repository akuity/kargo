// Package targets serves Targets, which live in the database rather than in
// Kubernetes, over the REST API.
package targets

import (
	"context"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/pkg/database/targetstore"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/auth/can"
	"github.com/akuity/kargo/pkg/server/middleware"
	"github.com/akuity/kargo/pkg/server/user"
)

// Handler serves the Target endpoints of one API server.
type Handler struct {
	// store holds the Targets. It is nil when the server runs without a
	// database, in which case every endpoint responds 501.
	store targetstore.Store
	// authorize checks that the caller may perform an operation. Access to
	// the database carries no authorization of its own, unlike access
	// through the Kubernetes client, so every route runs it before its
	// handler; see require.
	authorize func(context.Context, can.Access) error
	// kube reads the Stage a list is filtered by, with the caller's
	// authorization.
	kube client.Reader
}

// New returns a Handler. The store may be nil; see the store field.
func New(
	store targetstore.Store,
	authorize func(context.Context, can.Access) error,
	kube client.Reader,
) *Handler {
	return &Handler{store: store, authorize: authorize, kube: kube}
}

// paramProject and paramName are the route parameters the handlers read.
// paramName is not "target" because the generated clients could not hold a
// path parameter and a Target body of the same name.
const (
	paramProject = "project"
	paramName    = "target-name"
)

// Register adds the Target routes to a project's route group. Every route
// first confirms that a database is configured and that the caller holds the
// verb the operation requires on the Target the route names or, for the
// collection, on the Project's Targets.
func (h *Handler) Register(project *gin.RouterGroup) {
	targets := project.Group(
		"/targets",
		middleware.RequireFeature(h.store != nil, errDatabaseNotConfigured),
	)
	targets.GET("", h.require(can.List()), h.list)
	targets.POST("", h.require(can.Create()), h.create)
	targets.GET("/:target-name", h.require(can.Get()), h.get)
	targets.PUT("/:target-name", h.require(can.Update()), h.update)
	targets.DELETE("/:target-name", h.require(can.Delete()), h.delete)
}

// require returns middleware that checks the caller may perform the verb on
// the Target the route names, or on the Project's Targets for the collection,
// and refuses the request if not, so that no handler reaches the store for a
// caller who may not.
func (h *Handler) require(verb can.Verb) gin.HandlerFunc {
	return func(c *gin.Context) {
		access := verb.Target(c.Param(paramProject), c.Param(paramName))
		if err := h.authorize(c.Request.Context(), access); err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}
}

// logChange records a change to a Target and who made it. Targets are not
// Kubernetes resources, so the Kubernetes audit log never sees these.
func logChange(ctx context.Context, what, project, name string) {
	actor := "unknown"
	if id, ok := user.IdentityFromContext(ctx); ok {
		actor = id.Actor()
	}
	logging.LoggerFromContext(ctx).Info(
		what,
		"project", project,
		"target", name,
		"actor", actor,
	)
}
