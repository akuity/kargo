package server

import (
	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// projectExistsMiddleware returns Gin middleware that validates the project
// specified in the URL path exists. If the project does not exist, an error
// is added to the gin context and the request is aborted.
func (s *server) projectExistsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		project := c.Param("project")
		if project == "" {
			c.Next()
			return
		}
		p := &kargoapi.Project{}
		if err := s.client.Get(
			c.Request.Context(),
			client.ObjectKey{Name: project},
			p,
		); err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}
		c.Next()
	}
}
