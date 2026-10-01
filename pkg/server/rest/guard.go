// Package rest holds what the handlers for resources that live outside
// Kubernetes, such as those in the database, share: authorizing a request,
// binding its body, and reporting a store's errors. Handlers for Kubernetes
// resources get all of that from the authorizing client instead.
package rest

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/kubernetes"
)

// Func adapts a function to kubernetes.Authorizer, so that the server's
// overridable authorization hook can be handed to anything that takes the
// interface.
type Func func(
	ctx context.Context,
	verb string,
	gvr schema.GroupVersionResource,
	subresource string,
	key client.ObjectKey,
) error

// Authorize implements kubernetes.Authorizer.
func (f Func) Authorize(
	ctx context.Context,
	verb string,
	gvr schema.GroupVersionResource,
	subresource string,
	key client.ObjectKey,
) error {
	return f(ctx, verb, gvr, subresource, key)
}

var errNoAuthorizer = errors.New("authorizer is not configured")

// Guard authorizes requests against one resource. It is declared once per
// resource, with the route parameters that name an object, and then each
// route states only the verb it requires:
//
//	guard := rest.NewGuard(authorizer, gvr).Namespace("project").Name("target-name")
//	targets.GET("", guard.Require("list"), h.list)
//	targets.GET("/:target-name", guard.Require("get"), h.get)
//
// A route without the name parameter, such as the collection's, yields a
// key with no name, which is what a review of list or create expects.
type Guard struct {
	authorizer     kubernetes.Authorizer
	gvr            schema.GroupVersionResource
	namespaceParam string
	nameParam      string
}

// NewGuard returns a Guard for the resource. Set the route parameters that
// name an object with Namespace and Name.
func NewGuard(authorizer kubernetes.Authorizer, gvr schema.GroupVersionResource) *Guard {
	return &Guard{authorizer: authorizer, gvr: gvr}
}

// Namespace names the route parameter that holds an object's namespace.
func (g *Guard) Namespace(param string) *Guard {
	g.namespaceParam = param
	return g
}

// Name names the route parameter that holds an object's name.
func (g *Guard) Name(param string) *Guard {
	g.nameParam = param
	return g
}

// Require returns middleware that runs the SubjectAccessReview the verb
// requires on the object the route names, and refuses the request if it
// fails. Handlers behind it can assume the caller is authorized.
//
// A missing authorizer fails closed: that is a server bug, and reporting it
// as an unexpected failure is better than serving unauthorized requests.
func (g *Guard) Require(verb string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if g.authorizer == nil {
			_ = c.Error(errNoAuthorizer)
			c.Abort()
			return
		}
		key := client.ObjectKey{
			Namespace: c.Param(g.namespaceParam),
			Name:      c.Param(g.nameParam),
		}
		if err := g.authorizer.Authorize(c.Request.Context(), verb, g.gvr, "", key); err != nil {
			// A refusal carries its own status; keep it rather than reporting
			// it as an unexpected failure.
			var statusErr *apierrors.StatusError
			if errors.As(err, &statusErr) {
				err = libhttp.Error(err, int(statusErr.Status().Code))
			}
			_ = c.Error(err)
			c.Abort()
			return
		}
		c.Next()
	}
}
