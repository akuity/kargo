package middleware

import "github.com/gin-gonic/gin"

// RequireFeature returns Gin middleware that refuses every request with the
// given error when the feature it guards is not enabled, so that handlers for
// optional features need not check for themselves. The error should carry the
// status the client is to see; the API server's error-handling middleware
// reports an error without one as an internal server error.
func RequireFeature(enabled bool, disabledErr error) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !enabled {
			_ = c.Error(disabledErr)
			c.Abort()
			return
		}
		c.Next()
	}
}
