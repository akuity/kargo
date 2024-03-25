package logging

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2"
	runtimelog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/akuity/kargo/internal/os"
)

type loggerContextKey struct{}

var (
	globalLogger logr.Logger

	// programLevel allows for setting the logging level dynamically
	// via SetLevel().
	programLevel = new(slog.LevelVar)
)

const (
	LevelTrace = slog.Level(-2)
	LevelDebug = slog.Level(-1)
	LevelInfo  = slog.Level(0)
	LevelError = slog.Level(8)

	// These constants contain logging level strings,
	// purely for the performance benefit.
	TRACE = "TRACE"
	DEBUG = "DEBUG"
	INFO  = "INFO"
	ERROR = "ERROR"
)

func init() {
	level, err := parseLevel(os.GetEnv("LOG_LEVEL", "INFO"))
	if err != nil {
		panic(err)
	}
	programLevel.Set(level)

	opts := &slog.HandlerOptions{
		Level: programLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize the name of the level key and the output string,
			// including custom level values. Meaning:
			// log.V(1).Info() becomes level=DEBUG instead of level=DEBUG+3
			// log.V(2).Info() becomes level=TRACE instead of level=DEBUG+2
			if a.Key == slog.LevelKey {
				// Handle custom level values.
				level := a.Value.Any().(slog.Level)

				switch {
				case level < LevelDebug:
					a.Value = slog.StringValue(TRACE)
				case level < LevelInfo:
					a.Value = slog.StringValue(DEBUG)
				case level < LevelError:
					a.Value = slog.StringValue(INFO)
				default:
					a.Value = slog.StringValue(ERROR)
				}
			}
			return a
		},
	}

	globalLogger = logr.FromSlogHandler(slog.NewTextHandler(os.Stderr, opts))

	SetKLogLevel(os.GetEnvInt("KLOG_LEVEL", 0))

	runtimelog.SetLogger(globalLogger)
}

// ContextWithLogger returns a context.Context that has been augmented with
// the provided logr.Logger.
func ContextWithLogger(ctx context.Context, logger logr.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

// LoggerFromContext extracts a logr.Logger from the provided context.Context and
// returns it. If no logr.Logger is found, a global logr.Logger is returned.
func LoggerFromContext(ctx context.Context) logr.Logger {
	if logger := ctx.Value(loggerContextKey{}); logger != nil {
		return ctx.Value(loggerContextKey{}).(logr.Logger) // nolint: forcetypeassert
	}
	return globalLogger
}

// SetKLogLevel set the klog level for the k8s go-client
func SetKLogLevel(klogLevel int) {
	klog.InitFlags(nil)
	klog.SetOutput(os.Stderr)
	_ = flag.Set("v", strconv.Itoa(klogLevel))
}

// SetLevel dynamically modifies the level of the globbal logr.Logger.
func SetLevel(l slog.Level) {
	programLevel.Set(l)
}

func parseLevel(lvl string) (slog.Level, error) {
	switch strings.ToLower(lvl) {
	case "error":
		return LevelError, nil
	case "info":
		return LevelInfo, nil
	case "debug":
		return LevelDebug, nil
	case "trace":
		return LevelTrace, nil
	}

	var l slog.Level
	return l, fmt.Errorf("not a valid log level: %q.", lvl)
}
