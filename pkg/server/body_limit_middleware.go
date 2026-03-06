package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// bodyLimitMiddleware returns Gin middleware that limits request body size.
// It wraps the request body with http.MaxBytesReader which returns an error
// when the limit is exceeded.
func bodyLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil && c.Request.ContentLength != 0 {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
