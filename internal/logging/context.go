package logging

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/akuity/kargo/internal/os"
)

type loggerContextKey struct{}

var globalLogger *log.Entry

func init() {
	globalLogger = log.New().WithFields(nil)
	level, err := log.ParseLevel(os.GetEnv("LOG_LEVEL", "PANIC"))
	if err != nil {
		panic(err)
	}
	globalLogger.Logger.SetLevel(level)
}

// ContextWithLogger returns a context.Context that has been augmented with
// the provided log.Entry.
func ContextWithLogger(ctx context.Context, logger *log.Entry) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

// LoggerFromContext extracts a *log.Entry from the provided context.Context and
// returns it. If no *log.Entry is found, a global, error-level *log.Entry is
// returned.
func LoggerFromContext(ctx context.Context) *log.Entry {
	if logger := ctx.Value(loggerContextKey{}); logger != nil {
		return ctx.Value(loggerContextKey{}).(*log.Entry) // nolint: forcetypeassert
	}
	return globalLogger
}
