package projectconfigs

import (
	"context"
	"errors"
	"time"

	"github.com/gin-gonic/gin"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database/projectconfigstore"
	"github.com/akuity/kargo/pkg/logging"
	natsrec "github.com/akuity/kargo/pkg/reconciler/nats"
	"github.com/akuity/kargo/pkg/server/auth/can"
	"github.com/akuity/kargo/pkg/server/middleware"
	"github.com/akuity/kargo/pkg/server/user"
)

// Handler serves the ProjectConfig endpoints of one API server.
type Handler struct {
	// store holds the ProjectConfigs.
	store projectconfigstore.Store
	// authorize checks that the caller may perform an operation. Access to
	// the database carries no authorization of its own, unlike access
	// through the Kubernetes client, so every operation runs it before it
	// reaches the store.
	authorize func(context.Context, can.Access) error
	// conn is the NATS connection watches subscribe through, to the topic the
	// store announces its writes on.
	conn natsrec.Subscriber
}

// watchKeepalive is how often a watch writes a keepalive, as every other
// watch of the API server does, and re-reads the ProjectConfig. It bounds
// how long a lost event delays a change reaching a watcher.
const watchKeepalive = 30 * time.Second

// New returns a Handler. Watches subscribe through conn.
func New(
	store projectconfigstore.Store,
	authorize func(context.Context, can.Access) error,
	conn natsrec.Subscriber,
) *Handler {
	return &Handler{store: store, authorize: authorize, conn: conn}
}

// paramProject is the route parameter that names the Project.
const paramProject = "project"

// Register adds the ProjectConfig routes to a project's route group, which
// must run middleware.RequireProject. Every route checks that the caller holds
// the verb the operation requires on the Project's ProjectConfig before it
// reaches the store: as route middleware, or, for the writes that are also
// served elsewhere, in Upsert and Delete.
func (h *Handler) Register(project *gin.RouterGroup) {
	config := project.Group("/config")
	config.GET("", h.require(can.Get().ProjectConfig), h.get)
	config.PUT("", h.put)
	config.DELETE("", h.delete)
	config.PUT("/status", h.require(can.Update().ProjectConfigStatus), h.status)
	config.POST("/refresh", h.require(can.Get().ProjectConfig), h.refresh)
}

// require returns middleware that checks the caller may perform the access
// the given function describes for the route's Project, and refuses the
// request if not, so that no handler reaches the store for a caller who may
// not.
func (h *Handler) require(access func(project string) can.Access) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := h.authorize(c.Request.Context(), access(c.Param(paramProject))); err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}
}

// errNoProject reports a route mounted without middleware.RequireProject.
var errNoProject = errors.New("the request carries no Project")

// projectOf returns the Project middleware.RequireProject fetched for the
// request. It reports false after recording an error on the context, in which
// case the handler must return.
func projectOf(c *gin.Context) (*kargoapi.Project, bool) {
	project, ok := middleware.ProjectFromContext(c.Request.Context())
	if !ok {
		_ = c.Error(errNoProject)
	}
	return project, ok
}

// logChange records a change to a ProjectConfig and who made it.
// ProjectConfigs are not Kubernetes resources, so the Kubernetes audit log
// never sees these.
func logChange(ctx context.Context, what, project string) {
	actor := "unknown"
	if id, ok := user.IdentityFromContext(ctx); ok {
		actor = id.Actor()
	}
	logging.LoggerFromContext(ctx).Info(
		what,
		"project", project,
		"projectConfig", project,
		"actor", actor,
	)
}
