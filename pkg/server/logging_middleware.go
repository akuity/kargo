package server

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/user"
)

// LoggingMiddleware returns Gin middleware that records one line per request
// using Kargo's own logger, so that LOG_LEVEL governs the request log as it
// governs everything else.
//
// Each line names the actor the request was authenticated as, in the same
// form used to attribute Promotions and Events, so that a request log can
// answer who did something and not only where it came from. A request that
// never authenticated is attributed to EventActorUnknown.
//
// A server error is recorded at error level. A request that was refused is
// recorded at info level, because a refusal is never routine and is the first
// thing anyone looks for when a user reports being unable to do something.
// Everything else is operational detail and recorded at debug level.
//
// This must be the outermost middleware, so that the status and any reported
// error it records are the ones the client actually received.
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		status := c.Writer.Status()
		// The authentication middleware binds the actor to the request context
		// on the way in, so it is available here on the way out.
		actor := kargoapi.EventActorUnknown
		if u, ok := user.InfoFromContext(c.Request.Context()); ok {
			actor = api.FormatEventUserActor(u)
		}
		// Without stack traces, because this middleware sits above every handler
		// and every other middleware, so a trace from here describes only the path
		// through Gin and reveals nothing about what went wrong.
		logger := logging.LoggerFromContext(c.Request.Context()).
			WithoutStackTraces().
			WithValues(
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"actor", actor,
				"status", status,
				// Logged as a string because the encoder would otherwise render a
				// duration as a bare number of seconds.
				"duration", time.Since(start).String(),
			)
		reported := c.Errors.Last()

		if status < http.StatusInternalServerError {
			if reported != nil {
				logger = logger.WithValues("error", reported.Err.Error())
			}
			if status == http.StatusUnauthorized || status == http.StatusForbidden {
				logger.Info("refused request")
				return
			}
			logger.Debug("handled request")
			return
		}
		if reported == nil {
			// Nothing was reported, which means a handler answered with a server
			// error on its own rather than leaving that to the error-handling
			// middleware.
			logger.Error(errors.New("no error was reported"), "handled request")
			return
		}
		logger.Error(reported.Err, "handled request")
	}
}
