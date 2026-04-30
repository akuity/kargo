package logging

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContextWithLogger(t *testing.T) {
	testLogger := &Logger{}
	ctx := ContextWithLogger(t.Context(), testLogger)
	require.Same(t, testLogger, ctx.Value(loggerContextKey{}))
}

func TestLoggerFromContext(t *testing.T) {
	logger := LoggerFromContext(t.Context())
	// This should give us the global logger if one was never explicitly added to
	// the context.
	require.NotNil(t, logger)
	require.Same(t, globalLogger, logger)

	testLogger := &Logger{}
	ctx := context.WithValue(t.Context(), loggerContextKey{}, testLogger)
	require.Same(t, testLogger, LoggerFromContext(ctx))
}
