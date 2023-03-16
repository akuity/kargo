package logging

import (
	"context"

	log "github.com/sirupsen/logrus"
)

type loggerContextKey struct{}

var globalLogger *log.Entry

func init() {
	logger := log.New()
	logger.SetLevel(log.PanicLevel)
	globalLogger = logger.WithFields(nil)
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
		return ctx.Value(loggerContextKey{}).(*log.Entry)
	}
	return globalLogger
}
