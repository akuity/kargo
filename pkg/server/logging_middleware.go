package server

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/config"
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
// Everything else is operational detail and recorded at debug level, unless
// cfg.RequestLogAllEnabled raises it to info level.
//
// The IP address a request came from is recorded only when
// cfg.RequestLogSourceIPEnabled is set, because it is personal data and is
// usually better recorded by whatever proxy sits in front of the server. It is
// resolved by the engine, which ConfigureEngine must have configured.
//
// This must be the outermost middleware, so that the status and any reported
// error it records are the ones the client actually received.
func LoggingMiddleware(cfg config.ServerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Taken on the way in and used for the line recorded below, so that line
		// does not depend on what anything within does to the request's context.
		ctx := c.Request.Context()
		logger := logging.LoggerFromContext(ctx)
		if cfg.RequestLogSourceIPEnabled {
			// Bound to the request's logger, so that everything logged on the
			// request's behalf says where it came from.
			logger = logger.WithValues("sourceIP", c.ClientIP())
			// As received, and so only as trustworthy as whatever sent it.
			if forwarded := c.Request.Header.Values("X-Forwarded-For"); len(forwarded) > 0 {
				logger = logger.WithValues("forwardedFor", strings.Join(forwarded, ", "))
			}
			c.Request = c.Request.WithContext(logging.ContextWithLogger(ctx, logger))
		}

		c.Next()

		status := c.Writer.Status()
		// The authentication middleware binds the actor to the request context
		// on the way in, so it is available here on the way out.
		actor := kargoapi.EventActorUnknown
		if u, ok := user.IdentityFromContext(c.Request.Context()); ok {
			actor = u.Actor()
		}
		// Without stack traces, because this middleware sits above every handler
		// and every other middleware, so a trace from here describes only the path
		// through Gin and reveals nothing about what went wrong.
		logger = logger.
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
			if cfg.RequestLogAllEnabled {
				logger.Info("handled request")
			} else {
				logger.Debug("handled request")
			}
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
