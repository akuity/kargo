package logging

import (
	"context"
	"flag"
	"fmt"

	"github.com/bombsimon/logrusr/v4"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"k8s.io/klog/v2"
	runtimelog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/akuity/kargo/internal/os"
)

type Level uint32

const (
	ErrorLevel = Level(logrus.ErrorLevel)
	InfoLevel  = Level(logrus.InfoLevel)
	DebugLevel = Level(logrus.DebugLevel)
	TraceLevel = Level(logrus.TraceLevel)
)

type loggerContextKey struct{}

var globalLogger *Logger

func init() {
	// TODO: Transition off of logrus?
	logrusLogger := logrus.New()
	levelStr := os.GetEnv("LOG_LEVEL", "INFO")
	level, err := logrus.ParseLevel(levelStr)
	if err != nil {
		panic(err)
	}
	// Some levels supported by logrus are not supported by logr
	switch level {
	case logrus.ErrorLevel, logrus.InfoLevel, logrus.DebugLevel, logrus.TraceLevel:
	default:
		panic(fmt.Errorf("invalid log level %q", levelStr))
	}
	logrusLogger.SetLevel(level)

	logrLogger := logrusr.New(logrusLogger)
	globalLogger = &Logger{}
	globalLogger.callStackHelper, globalLogger.logger = logrLogger.WithCallStackHelper()

	klog.InitFlags(nil)
	klog.SetOutput(logrusLogger.Writer())
	if err = flag.Set("v", os.GetEnv("KLOG_LEVEL", "0")); err != nil {
		panic(err)
	}

	runtimelog.SetLogger(globalLogger.logger)
}

// Logger is a wrapper around logr.Logger that provides a more ergonomic API.
// This is heavily inspired by a similar wrapper from
// https://github.com/kubernetes-sigs/cluster-api-provider-aws
type Logger struct {
	callStackHelper func()
	logger          logr.Logger
}

// Wrap returns a new *Logger that wraps the provided logr.Logger.
func Wrap(logrLogger logr.Logger) *Logger {
	logger := &Logger{}
	logger.callStackHelper, logger.logger = logrLogger.WithCallStackHelper()
	return logger
}

// NewLogger returns a new *Logger with the provided log level.
func NewLogger(level Level) *Logger {
	logrusLogger := logrus.New()
	logrusLogger.SetLevel(logrus.Level(level))
	return Wrap(logrusr.New(logrusLogger))
}

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

// Error logs a message at the error level.
func (l *Logger) Error(err error, msg string, keysAndValues ...any) {
	l.callStackHelper()
	l.logger.Error(err, msg, keysAndValues...)
}

// Info logs a message at the info level.
func (l *Logger) Info(msg string, keysAndValues ...any) {
	l.callStackHelper()
	l.logger.Info(msg, keysAndValues...)
}

// Debug logs a message at the debug level.
func (l *Logger) Debug(msg string, keysAndValues ...any) {
	l.callStackHelper()
	l.logger.V(1).Info(msg, keysAndValues...)
}

// Trace logs a message at the trace level.
func (l *Logger) Trace(msg string, keysAndValues ...any) {
	l.callStackHelper()
	l.logger.V(2).Info(msg, keysAndValues...)
}

// GetLogger returns the underlying logr.Logger for cases where one needs to
// interact with the logr API directly.
func (l *Logger) GetLogger() logr.Logger {
	return l.logger
}

// WithValues adds key-value pairs to a logger's context.
func (l *Logger) WithValues(keysAndValues ...any) *Logger {
	return &Logger{
		callStackHelper: l.callStackHelper,
		logger:          l.logger.WithValues(keysAndValues...),
	}
}
