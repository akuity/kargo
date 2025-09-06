package logging

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestContextWithLogger(t *testing.T) {
	testLogger := &Logger{}
	ctx := ContextWithLogger(context.Background(), testLogger)
	require.Same(t, testLogger, ctx.Value(loggerContextKey{}))
}

func TestLoggerFromContext(t *testing.T) {
	logger := LoggerFromContext(context.Background())
	// This should give us the global logger if one was never explicitly added to
	// the context.
	require.NotNil(t, logger)
	require.Same(t, globalLogger, logger)

	testLogger := &Logger{}
	ctx := context.WithValue(context.Background(), loggerContextKey{}, testLogger)
	require.Same(t, testLogger, LoggerFromContext(ctx))
}

func TestNewLogger(t *testing.T) {
	// First test the normal case and then test with a custom writer so we make sure the zap core
	// wrapper logic works
	logger, err := NewLogger(DebugLevel, ConsoleFormat)
	require.NoError(t, err)
	require.NotNil(t, logger)

	logger, err = newLoggerInternal(DebugLevel, JSONFormat, os.Stderr)
	require.NoError(t, err)
	require.NotNil(t, logger)

	// Pass an invalid format and ensure we get an error
	_, err = NewLogger(DebugLevel, "invalid-format")
	require.Error(t, err)
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Level
		wantErr  bool
	}{
		{
			name:     "error level",
			input:    "error",
			expected: ErrorLevel,
			wantErr:  false,
		},
		{
			name:     "error level uppercase",
			input:    "ERROR",
			expected: ErrorLevel,
			wantErr:  false,
		},
		{
			name:     "info level",
			input:    "info",
			expected: InfoLevel,
			wantErr:  false,
		},
		{
			name:     "info level uppercase",
			input:    "INFO",
			expected: InfoLevel,
			wantErr:  false,
		},
		{
			name:     "debug level",
			input:    "debug",
			expected: DebugLevel,
			wantErr:  false,
		},
		{
			name:     "debug level mixed case",
			input:    "DeBuG",
			expected: DebugLevel,
			wantErr:  false,
		},
		{
			name:     "trace level",
			input:    "trace",
			expected: TraceLevel,
			wantErr:  false,
		},
		{
			name:     "trace level uppercase",
			input:    "TRACE",
			expected: TraceLevel,
			wantErr:  false,
		},
		{
			name:     "invalid level",
			input:    "invalid",
			expected: InfoLevel,
			wantErr:  true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: InfoLevel,
			wantErr:  true,
		},
		{
			name:     "numeric string",
			input:    "123",
			expected: InfoLevel,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseLevel(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid log level")
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestToZapLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    Level
		expected zapcore.Level
	}{
		{
			name:     "error level",
			input:    ErrorLevel,
			expected: zapcore.ErrorLevel,
		},
		{
			name:     "info level",
			input:    InfoLevel,
			expected: zapcore.InfoLevel,
		},
		{
			name:     "debug level",
			input:    DebugLevel,
			expected: zapcore.DebugLevel,
		},
		{
			name:     "trace level",
			input:    TraceLevel,
			expected: zapcore.DebugLevel, // TraceLevel maps to DebugLevel in zap
		},
		{
			name:     "invalid level (default case)",
			input:    Level(999), // Invalid level to test default case
			expected: zapcore.InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toZapLevel(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Format
		wantErr  bool
	}{
		{
			name:     "json format",
			input:    "json",
			expected: JSONFormat,
			wantErr:  false,
		},
		{
			name:     "json format uppercase",
			input:    "JSON",
			expected: JSONFormat,
			wantErr:  false,
		},
		{
			name:     "json format mixed case",
			input:    "JsOn",
			expected: JSONFormat,
			wantErr:  false,
		},
		{
			name:     "console format",
			input:    "console",
			expected: ConsoleFormat,
			wantErr:  false,
		},
		{
			name:     "console format uppercase",
			input:    "CONSOLE",
			expected: ConsoleFormat,
			wantErr:  false,
		},
		{
			name:     "console format mixed case",
			input:    "CoNsOlE",
			expected: ConsoleFormat,
			wantErr:  false,
		},
		{
			name:     "whitespace",
			input:    " json ",
			expected: JSONFormat,
			wantErr:  false,
		},
		{
			name:     "invalid format",
			input:    "invalid",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "numeric string",
			input:    "123",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "special characters",
			input:    "json!@#",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "partial match",
			input:    "jso",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseFormat(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid log format")
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.expected, result)
		})
	}
}
