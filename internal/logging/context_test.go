package logging

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
)

func TestContextWithLogger(t *testing.T) {
	testLogger := logr.New(nil)
	ctx := ContextWithLogger(context.Background(), testLogger)
	require.Equal(t, testLogger, ctx.Value(loggerContextKey{}))
}

func TestLoggerFromContext(t *testing.T) {
	logger := LoggerFromContext(context.Background())
	// This should give us the global logger if one was never explicitly added to
	// the context.
	require.NotNil(t, logger)
	require.IsType(t, logr.Logger{}, logger)
	require.Equal(t, true, logger.V(0).Enabled())  // INFO level enabled
	require.Equal(t, false, logger.V(1).Enabled()) // DEBUG level disabled

	testLogger := logr.New(nil)
	ctx := context.WithValue(context.Background(), loggerContextKey{}, testLogger)
	require.Equal(t, testLogger, LoggerFromContext(ctx))
}
