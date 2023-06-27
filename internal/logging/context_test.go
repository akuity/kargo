package logging

import (
	"context"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestContextWithLogger(t *testing.T) {
	testLogger := log.New().WithFields(nil)
	ctx := ContextWithLogger(context.Background(), testLogger)
	require.Same(t, testLogger, ctx.Value(loggerContextKey{}))
}

func TestLoggerFromContext(t *testing.T) {
	logger := LoggerFromContext(context.Background())
	// This should give us the global logger if one was never explicitly added to
	// the context.
	require.NotNil(t, logger)
	require.IsType(t, &log.Entry{}, logger)
	require.Equal(t, log.InfoLevel, logger.Logger.Level)

	testLogger := log.New().WithFields(nil)
	ctx := context.WithValue(context.Background(), loggerContextKey{}, testLogger)
	require.Same(t, testLogger, LoggerFromContext(ctx))
}
