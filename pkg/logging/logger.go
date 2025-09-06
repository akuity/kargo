package logging

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/klog/v2"
	runtimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

type (
	Level  uint32
	Format string
)

const (
	ErrorLevel = iota
	InfoLevel
	DebugLevel
	TraceLevel

	JSONFormat    Format = "json"
	ConsoleFormat Format = "console"
	DefaultFormat Format = ConsoleFormat

	LogLevelEnvVar  = "LOG_LEVEL"
	LogFormatEnvVar = "LOG_FORMAT"
)

type loggerContextKey struct{}

var globalLogger *Logger

func init() {
	levelStr := os.Getenv(LogLevelEnvVar)
	if levelStr == "" {
		levelStr = "INFO"
	}
	level, err := ParseLevel(levelStr)
	if err != nil {
		panic(err)
	}

	// Create a write syncer that we can also pass to klog
	writer, _, err := zap.Open("stderr")
	if err != nil {
		panic(err)
	}

	// The default for backwards compat is ConsoleFormat. We should probably change it to JSONFormat
	// in the future.
	format := DefaultFormat
	formatStr := os.Getenv(LogFormatEnvVar)
	if formatStr != "" {
		format, err = ParseFormat(formatStr)
		if err != nil {
			panic(err)
		}
	}

	// Create the global logger
	globalLogger, err = newLoggerInternal(level, format, writer)
	if err != nil {
		panic(err)
	}

	klog.InitFlags(nil)
	klog.SetOutput(writer)
	klogLevel := os.Getenv("KLOG_LEVEL")
	if klogLevel == "" {
		klogLevel = "0"
	}
	if err = flag.Set("v", klogLevel); err != nil {
		panic(err)
	}

	runtimelog.SetLogger(globalLogger.logger)
}

// ParseLevel parses a string representation of a log level and returns the
// corresponding Level value.
func ParseLevel(levelStr string) (Level, error) {
	switch strings.ToLower(levelStr) {
	case "error":
		return ErrorLevel, nil
	case "info":
		return InfoLevel, nil
	case "debug":
		return DebugLevel, nil
	case "trace":
		return TraceLevel, nil
	default:
		return InfoLevel, fmt.Errorf("invalid log level %q", levelStr)
	}
}

// toZapLevel converts a Level to a zapcore.Level.
func toZapLevel(level Level) zapcore.Level {
	switch level {
	case ErrorLevel:
		return zapcore.ErrorLevel
	case InfoLevel:
		return zapcore.InfoLevel
	case DebugLevel:
		return zapcore.DebugLevel
	case TraceLevel:
		// There is no TraceLevel in zap, so we use DebugLevel
		return zapcore.DebugLevel
	default:
		return zapcore.InfoLevel
	}
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
func NewLogger(level Level, format Format) (*Logger, error) {
	return newLoggerInternal(level, format, nil)
}

// NewLoggerOrDie returns a new *Logger with the provided log level or panics if there is an error
// configuring the logger
func NewLoggerOrDie(level Level, format Format) *Logger {
	logger, err := newLoggerInternal(level, format, nil)
	if err != nil {
		panic(err)
	}
	return logger
}

func newLoggerInternal(level Level, format Format, writer zapcore.WriteSyncer) (*Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Encoding = string(format)
	cfg.EncoderConfig.EncodeTime = func(time time.Time, encoder zapcore.PrimitiveArrayEncoder) {
		zapcore.RFC3339TimeEncoder(time.UTC(), encoder)
	}
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	cfg.DisableStacktrace = false
	cfg.Level = zap.NewAtomicLevelAt(toZapLevel(level))

	options := []zap.Option{}
	if writer != nil {
		var encoder zapcore.Encoder
		// As far as I can tell from looking at the source code, there aren't constants for this and
		// we can't pull the current encoder from the `core` passed to `WrapCore` So we have to do
		// this a bit hackily by checking the config
		switch cfg.Encoding {
		case "json":
			encoder = zapcore.NewJSONEncoder(cfg.EncoderConfig)
		case "console":
			encoder = zapcore.NewConsoleEncoder(cfg.EncoderConfig)
		default:
			return nil, fmt.Errorf("unknown encoding: %s", cfg.Encoding)
		}
		options = append(options, zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			// We create a new core so that we can use the provided writer. `core` is technically
			// wrapped to take advantage of its `LevelEnabler` functionality
			return zapcore.NewCore(encoder, writer, core)
		}))
	}
	zl, err := cfg.Build(options...)
	if err != nil {
		return nil, err
	}

	return Wrap(zapr.NewLoggerWithOptions(zl)), nil
}

// ParseFormat parses a string representation of a log format and returns the
// corresponding Format value or an error if it isn't recognized
func ParseFormat(f string) (Format, error) {
	switch Format(strings.TrimSpace(strings.ToLower(f))) {
	case JSONFormat:
		return JSONFormat, nil
	case ConsoleFormat:
		return ConsoleFormat, nil
	default:
		return "", fmt.Errorf("invalid log format %q", f)
	}
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
