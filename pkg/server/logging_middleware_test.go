package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/logging"
)

func TestLoggingMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name string
		// level is the level the logger is configured with, so that what a given
		// LOG_LEVEL suppresses is observable.
		level          zapcore.Level
		handler        gin.HandlerFunc
		expectedStatus int
		assertions     func(t *testing.T, entries []observer.LoggedEntry)
	}{
		{
			name:           "successful request is recorded at debug level",
			level:          zapcore.DebugLevel,
			handler:        func(c *gin.Context) { c.Status(http.StatusOK) },
			expectedStatus: http.StatusOK,
			assertions: func(t *testing.T, entries []observer.LoggedEntry) {
				require.Len(t, entries, 1)
				require.Equal(t, zapcore.DebugLevel, entries[0].Level)
				require.Equal(t, "handled request", entries[0].Message)
				fields := entries[0].ContextMap()
				require.Equal(t, http.MethodGet, fields["method"])
				require.Equal(t, "/", fields["path"])
				require.EqualValues(t, http.StatusOK, fields["status"])
				require.Contains(t, fields, "duration")
			},
		},
		{
			// Which is the whole point: gin's own logger writes regardless of the
			// configured level.
			name:           "successful request is suppressed at error level",
			level:          zapcore.ErrorLevel,
			handler:        func(c *gin.Context) { c.Status(http.StatusOK) },
			expectedStatus: http.StatusOK,
			assertions: func(t *testing.T, entries []observer.LoggedEntry) {
				require.Empty(t, entries)
			},
		},
		{
			name:  "refused request is recorded at info level, with its reason",
			level: zapcore.InfoLevel,
			handler: func(c *gin.Context) {
				_ = c.Error(libhttp.ErrorStr("invalid token", http.StatusUnauthorized))
			},
			expectedStatus: http.StatusUnauthorized,
			assertions: func(t *testing.T, entries []observer.LoggedEntry) {
				require.Len(t, entries, 1)
				require.Equal(t, zapcore.InfoLevel, entries[0].Level)
				require.Equal(t, "refused request", entries[0].Message)
				fields := entries[0].ContextMap()
				require.EqualValues(t, http.StatusUnauthorized, fields["status"])
				require.Equal(t, "invalid token", fields["error"])
			},
		},
		{
			name:  "forbidden request is recorded at info level",
			level: zapcore.InfoLevel,
			handler: func(c *gin.Context) {
				_ = c.Error(libhttp.ErrorStr("not permitted", http.StatusForbidden))
			},
			expectedStatus: http.StatusForbidden,
			assertions: func(t *testing.T, entries []observer.LoggedEntry) {
				require.Len(t, entries, 1)
				require.Equal(t, zapcore.InfoLevel, entries[0].Level)
				require.EqualValues(t, http.StatusForbidden, entries[0].ContextMap()["status"])
			},
		},
		{
			name:  "request for something absent is recorded at debug level",
			level: zapcore.InfoLevel,
			handler: func(c *gin.Context) {
				_ = c.Error(libhttp.ErrorStr("not found", http.StatusNotFound))
			},
			expectedStatus: http.StatusNotFound,
			assertions: func(t *testing.T, entries []observer.LoggedEntry) {
				require.Empty(t, entries)
			},
		},
		{
			// A server error is worth recording even when nothing else is.
			name:  "failed request is recorded at error level",
			level: zapcore.ErrorLevel,
			handler: func(c *gin.Context) {
				_ = c.Error(errors.New("something we did not anticipate"))
			},
			expectedStatus: http.StatusInternalServerError,
			assertions: func(t *testing.T, entries []observer.LoggedEntry) {
				// Exactly one record: the error-handling middleware responds, but
				// leaves the recording of what happened to this middleware.
				require.Len(t, entries, 1)
				last := entries[0]
				require.Equal(t, zapcore.ErrorLevel, last.Level)
				// Logger.Error folds the error into the message.
				require.Equal(
					t,
					"handled request: something we did not anticipate",
					last.Message,
				)
				fields := last.ContextMap()
				require.EqualValues(t, http.StatusInternalServerError, fields["status"])
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			core, recorded := observer.New(testCase.level)
			logger := logging.Wrap(zap.New(core))

			s := &server{}
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Request = c.Request.WithContext(
					logging.ContextWithLogger(c.Request.Context(), logger),
				)
				c.Next()
			})
			router.Use(loggingMiddleware())
			router.Use(s.handleError)
			router.GET("/", testCase.handler)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

			require.Equal(t, testCase.expectedStatus, w.Code)
			testCase.assertions(t, recorded.All())
		})
	}
}
