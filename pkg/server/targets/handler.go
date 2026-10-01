// Package targets serves Targets, which live in the database rather than in
// Kubernetes, over the REST API.
package targets

import (
	"context"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/rest"
	"github.com/akuity/kargo/pkg/server/user"
)

// Store is the slice of database.Store the handlers use.
type Store interface {
	CreateTarget(context.Context, string, *kargoapi.Target) (*kargoapi.Target, error)
	GetTarget(context.Context, string, string) (*kargoapi.Target, error)
	ListTargets(context.Context, string) ([]kargoapi.Target, error)
	UpdateTarget(context.Context, string, *kargoapi.Target) (*kargoapi.Target, error)
	DeleteTarget(context.Context, string, string) error
}

var _ Store = database.Store(nil)

// Handler serves the Target endpoints of one API server.
type Handler struct {
	// store holds the Targets. It is nil when the server runs without a
	// database, in which case every endpoint responds 501.
	store Store
	// authorizer runs the SubjectAccessReview an operation requires. Access
	// to the database carries no authorization of its own, unlike access
	// through the Kubernetes client, so every route runs it before its
	// handler; see Register.
	authorizer kubernetes.Authorizer
	// kube reads the Stage a list is filtered by, with the caller's
	// authorization.
	kube client.Reader
}

// New returns a Handler. The store may be nil; see the store field.
func New(store database.Store, authorizer kubernetes.Authorizer, kube client.Reader) *Handler {
	h := &Handler{authorizer: authorizer, kube: kube}
	// Assign only a non-nil store: a nil database.Store held in an interface
	// of another type would not compare equal to nil.
	if store != nil {
		h.store = store
	}
	return h
}

// resource is the Target resource as RBAC rules name it. The rules are just
// strings, so those granting access to targets keep working although no
// Target custom resource is read or written anymore.
const resource = "targets"

// paramProject and paramName are the route parameters the handlers read.
// paramName is not "target" because the generated clients could not hold a
// path parameter and a Target body of the same name.
const (
	paramProject = "project"
	paramName    = "target-name"
)

// Register adds the Target routes to a project's route group. Every route
// first confirms that a database is configured and that the caller holds the
// verb the operation requires on the Target resource, named by the route or,
// for the collection, by the project alone.
func (h *Handler) Register(project *gin.RouterGroup) {
	guard := rest.NewGuard(h.authorizer, kargoapi.GroupVersion.WithResource(resource)).
		Namespace(paramProject).
		Name(paramName)
	targets := project.Group("/targets", h.requireStore)
	targets.GET("", guard.Require("list"), h.list)
	targets.POST("", guard.Require("create"), rest.Bind(bindTarget, h.create))
	targets.GET("/:target-name", guard.Require("get"), h.get)
	targets.PUT("/:target-name", guard.Require("update"), rest.Bind(bindTarget, h.update))
	targets.DELETE("/:target-name", guard.Require("delete"), h.delete)
}

// requireStore refuses every request when the server has no database.
func (h *Handler) requireStore(c *gin.Context) {
	if h.store == nil {
		_ = c.Error(errDatabaseNotConfigured)
		c.Abort()
		return
	}
	c.Next()
}

// logChange records a change to a Target and who made it. Targets are not
// Kubernetes resources, so the Kubernetes audit log never sees these.
func logChange(ctx context.Context, what, project, name string) {
	actor := "unknown"
	if u, ok := user.InfoFromContext(ctx); ok {
		actor = api.FormatEventUserActor(u)
	}
	logging.LoggerFromContext(ctx).Info(
		what,
		"project", project,
		"target", name,
		"actor", actor,
	)
}
