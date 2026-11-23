package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// projectKey is the key under which RequireProject stores the Project it
// fetched in the request's context.
type projectKey struct{}

// RequireProject returns Gin middleware that confirms the Project named by
// the request's "project" path parameter exists, looking it up with the given
// client, before any handler runs. A request for anything in a Project that
// does not exist is answered with the lookup's error, a 404, and no handler
// has to check for itself. The Project it fetched is stored in the request's
// context, where ProjectFromContext finds it.
func RequireProject(cl client.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		project := &kargoapi.Project{}
		if err := cl.Get(
			ctx,
			client.ObjectKey{Name: c.Param("project")},
			project,
		); err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}
		c.Request = c.Request.WithContext(contextWithProject(ctx, project))
		c.Next()
	}
}

func contextWithProject(ctx context.Context, project *kargoapi.Project) context.Context {
	return context.WithValue(ctx, projectKey{}, project)
}

// ProjectFromContext returns the Project RequireProject fetched for the
// request, and whether there is one.
func ProjectFromContext(ctx context.Context) (*kargoapi.Project, bool) {
	project, ok := ctx.Value(projectKey{}).(*kargoapi.Project)
	return project, ok && project != nil
}
