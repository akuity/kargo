package logging

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/klog/v2"
	runtimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

type (
	Level  int8
	Format string
)

const (
	// Note(krancour): Numerically speaking, zap supports levels above or below
	// those for it has defined constants. This is how we implement our own
	// Discard and Trace levels.
	DiscardLevel Level = Level(zapcore.FatalLevel + 1)
	ErrorLevel   Level = Level(zapcore.ErrorLevel)
	InfoLevel    Level = Level(zapcore.InfoLevel)
	DebugLevel   Level = Level(zapcore.DebugLevel)
	TraceLevel   Level = DebugLevel - 1

	ConsoleFormat Format = "console"
	JSONFormat    Format = "json"
	DefaultFormat Format = ConsoleFormat

	LogLevelEnvVar  = "LOG_LEVEL"
	LogFormatEnvVar = "LOG_FORMAT"
)

var (
	writer       zapcore.WriteSyncer
	globalLogger *Logger
)

func init() {
	level := InfoLevel
	if l := os.Getenv(LogLevelEnvVar); l != "" {
		var err error
		if level, err = ParseLevel(l); err != nil {
			panic(err)
		}
	}

	// The default, for reasons of backwards compatibility, is ConsoleFormat. It
	// might be nice to change it to JSONFormat in the future.
	format := DefaultFormat
	if formatStr := os.Getenv(LogFormatEnvVar); formatStr != "" {
		format = Format(formatStr)
	}

	// Create a write syncer that all our underlying zap.Loggers will use and we
	// can also pass to klog. This ensures all logs are synchronized and written
	// to the same destination.
	var err error
	if writer, _, err = zap.Open("stderr"); err != nil {
		panic(err)
	}

	// Create the global logger
	if globalLogger, err = newLoggerInternal(level, format); err != nil {
		panic(err)
	}

	// Configure klog to use the same writer
	klog.InitFlags(nil)
	klog.SetOutput(writer)
	klogLevel := "0"
	if k := os.Getenv("KLOG_LEVEL"); k != "" {
		klogLevel = k
	}
	if err = flag.Set("v", klogLevel); err != nil {
		panic(err)
	}

	// Configure controller-runtime to use our globalLogger's underlying
	// zap.Logger wrapped as a logr.Logger.
	runtimelog.SetLogger(
		zapr.NewLoggerWithOptions(
			// Reverse the skip we added in Wrap()
			globalLogger.logger.Desugar().WithOptions(zap.AddCallerSkip(-1)),
		),
	)
}

// Logger is a simple wrapper around zap.Logger that provides a more ergonomic
// API.
type Logger struct {
	// Note(krancour): Carrying around a zap.SugaredLogger may possibly incur a
	// very small performance penalty, but it spares us from having to duplicate
	// some of that type's functionality, such as the conversion of context in the
	// form of []any to []zap.Field. I consider this a worthwhile trade-off.
	logger *zap.SugaredLogger
}

// NewDiscardLoggerOrDie returns a new *Logger that discards all log output or
// panics if there is an error configuring the logger. This is primarily useful
// for tests.
func NewDiscardLoggerOrDie() *Logger {
	return NewLoggerOrDie(DiscardLevel, ConsoleFormat)
}

// NewLoggerOrDie returns a new *Logger with the provided log level or panics if
// there is an error configuring the logger.
func NewLoggerOrDie(level Level, format Format) *Logger {
	logger, err := newLoggerInternal(level, format)
	if err != nil {
		panic(err)
	}
	return logger
}

// NewLogger returns a new *Logger with the provided log level.
func NewLogger(level Level, format Format) (*Logger, error) {
	return newLoggerInternal(level, format)
}

func newLoggerInternal(level Level, format Format) (*Logger, error) {
	if level == DiscardLevel {
		// Note(krancour): Building a leveled logger with the level set higher than
		// LevelFatal may actually work, but comments within zap's source code
		// suggest that was intended to be invalid. Rather than relying on the
		// observed behavior to never change, we take zap at its word and just
		// return a no-op logger here when the caller has requested a logger of
		// DiscardLevel.
		return &Logger{
			logger: zap.NewNop().Sugar(),
		}, nil
	}
	if level < TraceLevel || level > ErrorLevel {
		return nil, fmt.Errorf("invalid log level: %d", level)
	}
	// Re-parsing the format we were given has the side effects of validating and
	// normalizing it.
	format, err := ParseFormat(string(format))
	if err != nil {
		return nil, err
	}

	cfg := zap.NewProductionConfig()
	cfg.Encoding = string(format)
	cfg.EncoderConfig.EncodeTime = func(
		time time.Time,
		encoder zapcore.PrimitiveArrayEncoder,
	) {
		zapcore.RFC3339TimeEncoder(time.UTC(), encoder)
	}
	cfg.DisableStacktrace = false
	cfg.Level = zap.NewAtomicLevelAt(zapcore.Level(level))

	// Add custom encoding for our Trace level
	cfg.EncoderConfig.EncodeLevel = traceEncoder

	var encoder zapcore.Encoder
	switch format { // format was already validated above
	case ConsoleFormat:
		encoder = zapcore.NewConsoleEncoder(cfg.EncoderConfig)
	case JSONFormat:
		encoder = zapcore.NewJSONEncoder(cfg.EncoderConfig)
	}
	logger, err := cfg.Build(
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			// Create a new core with our global writer plugged in.
			return zapcore.NewCore(encoder, writer, core)
		}),
	)
	if err != nil {
		return nil, err
	}

	return Wrap(logger), nil
}

func traceEncoder(
	level zapcore.Level,
	enc zapcore.PrimitiveArrayEncoder,
) {
	if level == zapcore.Level(TraceLevel) {
		enc.AppendString("TRACE")
	} else {
		zapcore.CapitalLevelEncoder(level, enc)
	}
}

// Wrap returns a new *Logger that wraps the provided zap.Logger.
func Wrap(zapLogger *zap.Logger) *Logger {
	return &Logger{
		logger: zapLogger.Sugar().WithOptions(zap.AddCallerSkip(1)),
	}
}

// WithValues adds key-value pairs to a logger's context.
func (l *Logger) WithValues(keysAndValues ...any) *Logger {
	return &Logger{
		logger: l.logger.With(keysAndValues...),
	}
}

// Error logs a message at the error level.
func (l *Logger) Error(err error, msg string, keysAndValues ...any) {
	l.logger.Errorw(fmt.Sprintf("%s: %v", msg, err), keysAndValues...,
	)
}

// Info logs a message at the info level.
func (l *Logger) Info(msg string, keysAndValues ...any) {
	l.logger.Infow(msg, keysAndValues...)
}

// Debug logs a message at the debug level.
func (l *Logger) Debug(msg string, keysAndValues ...any) {
	l.logger.Debugw(msg, keysAndValues...)
}

// Trace logs a message at the trace level.
func (l *Logger) Trace(msg string, keysAndValues ...any) {
	// Note(krancour): Zap doesn't have a Trace method, but numerically speaking,
	// does support arbitrary levels. We've defined TraceLevel as one less than
	// DebugLevel and, assuming the logger is one whose core was created using
	// newLoggerInternal() and not a wrapped user-supplied zap.Logger, a log entry
	// written as follows will be logged as `TRACE`. Rad, huh?
	l.logger.With(keysAndValues...).Log(zapcore.Level(TraceLevel), msg)
}

// Logr returns a logr.Logger wrapped in this Logger's underlying zap.Logger for
// cases where one needs to to pass the a logr.Logger to another library that
// obviously doesn't know how to work with our custom one.
func (l *Logger) Logr() logr.Logger {
	return zapr.NewLoggerWithOptions(l.logger.Desugar().WithOptions(zap.AddCallerSkip(-1)))
}
