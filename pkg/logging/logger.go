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
	// Note(krancour): We can actually initialize zap.Loggers with levels above
	// or below those they officially support and except for the absence of
	// conveniently named methods like Trace(), they'll work as expected.
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

type loggerContextKey struct{}

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
	runtimelog.SetLogger(zapr.NewLoggerWithOptions(globalLogger.logger))
}

// Logger is a simple wrapper around zap.Logger that provides a more ergonomic
// API.
type Logger struct {
	logger  *zap.Logger
	skipped *zap.Logger // logger with caller skip applied
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
	if level < TraceLevel || level > DiscardLevel {
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
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	cfg.DisableStacktrace = false
	cfg.Level = zap.NewAtomicLevelAt(zapcore.Level(level))

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

// Wrap returns a new *Logger that wraps the provided zap.Logger.
func Wrap(zapLogger *zap.Logger) *Logger {
	return &Logger{
		logger:  zapLogger,
		skipped: zapLogger.WithOptions(zap.AddCallerSkip(1)),
	}
}

// WithValues adds key-value pairs to a logger's context.
func (l *Logger) WithValues(keysAndValues ...any) *Logger {
	logger := l.logger.With(toZapFields(keysAndValues...)...)
	return &Logger{
		logger:  logger,
		skipped: logger.WithOptions(zap.AddCallerSkip(1)),
	}
}

// Error logs a message at the error level.
func (l *Logger) Error(err error, msg string, keysAndValues ...any) {
	msg = fmt.Sprintf("%s: %v", msg, err)
	l.skipped.Error(msg, toZapFields(keysAndValues...)...)
}

// Info logs a message at the info level.
func (l *Logger) Info(msg string, keysAndValues ...any) {
	l.skipped.Info(msg, toZapFields(keysAndValues...)...)
}

// Debug logs a message at the debug level.
func (l *Logger) Debug(msg string, keysAndValues ...any) {
	l.skipped.Debug(msg, toZapFields(keysAndValues...)...)
}

// Trace logs a message at the trace level.
func (l *Logger) Trace(msg string, keysAndValues ...any) {
	// Note(krancour): Zap doesn't actually have a trace level, but WE have a
	// trace level that is numerically below debug. A zap.Logger that was
	// initialized with that numeric level WILL answer true to
	// Enabled(zapcore.Level(TraceLevel)). Neat, right? So if this pseudo-trace
	// level is enabled, we log at debug level with a "TRACE: " prefix.
	if l.skipped.Core().Enabled(zapcore.Level(TraceLevel)) {
		l.skipped.Debug("TRACE: "+msg, toZapFields(keysAndValues...)...)
	}
}

// Logr returns a logr.Logger wrapped in this Logger's underlying zap.Logger for
// cases where one needs to to pass the a logr.Logger to another library that
// obviously doesn't know how to work with our custom one.
func (l *Logger) Logr() logr.Logger {
	return zapr.NewLoggerWithOptions(l.logger)
}

func toZapFields(keysAndValues ...any) []zap.Field {
	fields := make([]zap.Field, 0, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues); i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			key = fmt.Sprintf("non_string_key_%d", i)
		}
		var value any
		if i+1 < len(keysAndValues) {
			value = keysAndValues[i+1]
		} else {
			value = "<missing>"
		}
		fields = append(fields, zap.Any(key, value))
	}
	return fields
}
