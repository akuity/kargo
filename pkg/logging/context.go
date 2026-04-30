package logging

import "context"

type loggerContextKey struct{}

// ContextWithLogger returns a context.Context that has been augmented with
// the provided *Logger.
func ContextWithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

// LoggerFromContext extracts a *Logger from the provided context.Context and
// returns it. If no *Logger is found, a global *Logger is returned.
func LoggerFromContext(ctx context.Context) *Logger {
	if logger := ctx.Value(loggerContextKey{}); logger != nil {
		return ctx.Value(loggerContextKey{}).(*Logger) // nolint: forcetypeassert
	}
	return globalLogger
}
