package middleware

import (
	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// RequireProject returns Gin middleware that confirms the Project named by
// the request's "project" path parameter exists, looking it up with the given
// client, before any handler runs. A request for anything in a Project that
// does not exist is answered with the lookup's error, a 404, and no handler
// has to check for itself.
func RequireProject(cl client.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := cl.Get(
			c.Request.Context(),
			client.ObjectKey{Name: c.Param("project")},
			&kargoapi.Project{},
		); err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}
		c.Next()
	}
}
