package server

import (
	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// projectExistsMiddleware returns Gin middleware that confirms the Project
// named in the request path exists before any handler runs, so that a
// request for anything in a Project that does not exist is answered with a
// 404 and no handler has to check for itself.
//
// The lookup uses the API server's own client rather than the authorizing
// one on purpose. Through the authorizing client, every project-scoped
// request would also require permission to get Projects, which a user who is
// granted access to a Project's Stages or Freight alone does not have. The
// lookup discloses nothing but whether the Project exists, and only its
// existence; the request is then authorized as usual.
func (s *server) projectExistsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := s.client.InternalClient().Get(
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
