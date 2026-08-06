package server

import (
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"syscall"

	"github.com/gin-gonic/gin"

	"github.com/akuity/kargo/pkg/logging"
)

// recoveryMiddleware returns Gin middleware that recovers from a panic in any
// handler downstream of it and reports it as an error rather than answering the
// request itself. The error-handling middleware then responds as it does to any
// other unanticipated error.
//
// This exists in preference to gin.Recovery(), which answers a panic with
// c.AbortWithStatus and therefore sends a 500 with an empty body, leaving a
// client nothing to parse and bypassing the error-handling middleware
// altogether. Reporting the panic instead keeps every error response the same
// shape and keeps response writing in one place.
//
// This must be registered inside the error-handling middleware, so that the
// error it reports is still there to be found when that middleware inspects the
// request on its way back out.
func recoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			// A client that has gone away is not worth a stack trace, and there is
			// nothing left to respond to. gin.Recovery() singles out these same
			// three conditions for the same reason.
			if err, ok := rec.(error); ok &&
				(errors.Is(err, syscall.EPIPE) ||
					errors.Is(err, syscall.ECONNRESET) ||
					errors.Is(err, http.ErrAbortHandler)) {
				logging.LoggerFromContext(c.Request.Context()).Debug(
					"client closed the connection",
					"error", err.Error(),
				)
				c.Abort()
				return
			}
			// The stack accompanies the error rather than being logged separately,
			// so that a panic produces one record containing everything known about
			// it.
			_ = c.Error(fmt.Errorf("recovered from panic: %v\n%s", rec, debug.Stack()))
			c.Abort()
		}()

		c.Next()
	}
}
