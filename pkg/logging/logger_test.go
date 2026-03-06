package logging

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestGlobalLogger(t *testing.T) {
	require.NotNil(t, globalLogger)
}

func TestNewDiscardLoggerOrDie(t *testing.T) {
	logger := NewDiscardLoggerOrDie()
	require.NotNil(t, logger)
	require.Equal(
		t,
		"zapcore.nopCore",
		reflect.TypeOf(logger.logger.Desugar().Core()).String(),
	)
}

func TestNewLoggerOrDie(t *testing.T) {
	testCases := []struct {
		name        string
		level       Level
		format      Format
		shouldPanic bool
		assertions  func(*testing.T, *Logger)
	}{
		{
			name:        "invalid level",
			level:       Level(100),
			format:      ConsoleFormat,
			shouldPanic: true,
		},
		{
			name:        "invalid format",
			level:       InfoLevel,
			format:      "invalid-format",
			shouldPanic: true,
		},
		{
			name:        "valid parameters",
			level:       DebugLevel,
			format:      JSONFormat,
			shouldPanic: false,
			assertions: func(t *testing.T, l *Logger) {
				require.NotNil(t, l)
				require.Equal(t, zap.DebugLevel, l.logger.Level())
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.shouldPanic {
				require.Panics(t, func() {
					NewLoggerOrDie(testCase.level, testCase.format)
				})
				return
			}
			testCase.assertions(t, NewLoggerOrDie(testCase.level, testCase.format))
		})
	}
}

func TestNewLogger(t *testing.T) {
	testCases := []struct {
		name       string
		level      Level
		format     Format
		assertions func(*testing.T, *Logger, error)
	}{
		{
			name:   "invalid level",
			level:  Level(100),
			format: ConsoleFormat,
			assertions: func(t *testing.T, _ *Logger, err error) {
				require.ErrorContains(t, err, "invalid log level")
			},
		},
		{
			name:   "invalid format",
			level:  InfoLevel,
			format: "invalid-format",
			assertions: func(t *testing.T, _ *Logger, err error) {
				require.ErrorContains(t, err, "invalid log format")
			},
		},
		{
			name:   "valid parameters",
			level:  DebugLevel,
			format: ConsoleFormat,
			assertions: func(t *testing.T, l *Logger, err error) {
				require.NoError(t, err)
				require.NotNil(t, l)
				require.Equal(t, zap.DebugLevel, l.logger.Level())
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			logger, err := NewLogger(testCase.level, testCase.format)
			if testCase.assertions != nil {
				testCase.assertions(t, logger, err)
			}
		})
	}
}

func Test_newLoggerInternal(t *testing.T) {
	testCases := []struct {
		name       string
		level      Level
		format     Format
		assertions func(*testing.T, *Logger, error)
	}{
		{
			name:   "invalid level",
			level:  Level(100),
			format: ConsoleFormat,
			assertions: func(t *testing.T, _ *Logger, err error) {
				require.ErrorContains(t, err, "invalid log level")
			},
		},
		{
			name:   "invalid format",
			level:  InfoLevel,
			format: "invalid-format",
			assertions: func(t *testing.T, _ *Logger, err error) {
				require.ErrorContains(t, err, "invalid log format")
			},
		},
		{
			name:   "valid parameters",
			level:  DebugLevel,
			format: ConsoleFormat,
			assertions: func(t *testing.T, l *Logger, err error) {
				require.NoError(t, err)
				require.NotNil(t, l)
				require.Equal(t, zap.DebugLevel, l.logger.Level())
				// TODO: Figure out how to verify that the write syncer is actually
				// being used.
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			logger, err := newLoggerInternal(testCase.level, testCase.format)
			if testCase.assertions != nil {
				testCase.assertions(t, logger, err)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	Wrap(zap.New(core)).WithValues("component", "test").Info("test message")
	// If the message made it through to the wrapped zap.Logger, the Wrap()
	// worked.
	require.Len(t, logs.All(), 1)
	entry := logs.All()[0]
	require.Equal(t, zapcore.InfoLevel, entry.Level)
	require.Equal(t, "test message", entry.Message)
	require.Equal(t, []zapcore.Field{zap.Any("component", "test")}, entry.Context)
}

func TestLogger_WithValues(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	// Wrap() has its own tests, so here we assume it works.
	Wrap(zap.New(core)).WithValues("component", "test").Info("test message")
	// If the message sent to the underlying zap.Logger has the expected key/value
	// pair, then WithValues() worked.
	require.Len(t, logs.All(), 1)
	entry := logs.All()[0]
	require.Equal(t, zapcore.InfoLevel, entry.Level)
	require.Equal(t, "test message", entry.Message)
	require.Equal(t, []zapcore.Field{zap.Any("component", "test")}, entry.Context)
}

func TestLogger_Error(t *testing.T) {
	core, logs := observer.New(zapcore.ErrorLevel)
	// Wrap() has its own tests, so here we assume it works.
	Wrap(zap.New(core)).Error(
		errors.New("something went wrong"),
		"an error occurred",
		"component", "test",
	)
	// Examine captured logs...
	require.Len(t, logs.All(), 1)
	entry := logs.All()[0]
	require.Equal(t, zapcore.ErrorLevel, entry.Level)
	require.Equal(t, "an error occurred: something went wrong", entry.Message)
	require.Equal(t, []zapcore.Field{zap.Any("component", "test")}, entry.Context)
}

func TestLogger_Info(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	// Wrap() has its own tests, so here we assume it works.
	Wrap(zap.New(core)).Info("test message", "component", "test")
	// Examine captured logs...
	require.Len(t, logs.All(), 1)
	entry := logs.All()[0]
	require.Equal(t, zapcore.InfoLevel, entry.Level)
	require.Equal(t, "test message", entry.Message)
	require.Equal(t, []zapcore.Field{zap.Any("component", "test")}, entry.Context)
}

func TestLogger_Debug(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	// Wrap() has its own tests, so here we assume it works.
	Wrap(zap.New(core)).Debug("test message", "component", "test")
	// Examine captured logs...
	require.Len(t, logs.All(), 1)
	entry := logs.All()[0]
	require.Equal(t, zapcore.DebugLevel, entry.Level)
	require.Equal(t, "test message", entry.Message)
	require.Equal(t, []zapcore.Field{zap.Any("component", "test")}, entry.Context)
}

func TestLogger_Trace(t *testing.T) {
	// Build a logger with an observable core that ALSO uses outr custom
	// traceEncoder.
	observableCore, logs := observer.New(zapcore.Level(TraceLevel))
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeLevel = traceEncoder
	logger, err := cfg.Build(
		zap.WrapCore(func(zapcore.Core) zapcore.Core {
			return observableCore
		}),
	)
	require.NoError(t, err)
	// Wrap() has its own tests, so here we assume it works.
	Wrap(logger).Trace("test message", "component", "test")
	require.Len(t, logs.All(), 1)
	// Examine captured logs...
	entry := logs.All()[0]
	require.Equal(t, zapcore.Level(TraceLevel), entry.Level)
	require.Equal(t, "test message", entry.Message)
	require.Equal(t, []zapcore.Field{zap.Any("component", "test")}, entry.Context)
}

func TestLogger_Logr(t *testing.T) {
	core, logs := observer.New(zapcore.Level(TraceLevel))
	// Wrap() has its own tests, so here we assume it works.
	logger := Wrap(zap.New(core))
	logrLogger := logger.Logr()
	require.NotNil(t, logrLogger)
	// Write a message using the logr.Logger...
	logrLogger.Info("test message", "component", "test")
	// If the message sent through the logr.Logger made it to the underlying
	// zap.Logger, then the Logr() method worked.
	require.Len(t, logs.All(), 1)
	entry := logs.All()[0]
	require.Equal(t, zapcore.InfoLevel, entry.Level)
	require.Equal(t, "test message", entry.Message)
	require.Equal(t, []zapcore.Field{zap.Any("component", "test")}, entry.Context)

}
