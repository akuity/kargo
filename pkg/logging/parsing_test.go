package logging

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseLevel(t *testing.T) {
	testCases := []struct {
		name    string
		input   string
		want    Level
		wantErr bool
	}{
		{
			name:    "discard level",
			input:   "discard",
			want:    DiscardLevel,
			wantErr: false,
		},
		{
			name:    "discard level uppercase",
			input:   "DISCARD",
			want:    DiscardLevel,
			wantErr: false,
		},
		{
			name:    "error level",
			input:   "error",
			want:    ErrorLevel,
			wantErr: false,
		},
		{
			name:    "error level uppercase",
			input:   "ERROR",
			want:    ErrorLevel,
			wantErr: false,
		},
		{
			name:    "info level",
			input:   "info",
			want:    InfoLevel,
			wantErr: false,
		},
		{
			name:    "info level uppercase",
			input:   "INFO",
			want:    InfoLevel,
			wantErr: false,
		},
		{
			name:    "debug level",
			input:   "debug",
			want:    DebugLevel,
			wantErr: false,
		},
		{
			name:    "debug level mixed case",
			input:   "DeBuG",
			want:    DebugLevel,
			wantErr: false,
		},
		{
			name:    "trace level",
			input:   "trace",
			want:    TraceLevel,
			wantErr: false,
		},
		{
			name:    "trace level uppercase",
			input:   "TRACE",
			want:    TraceLevel,
			wantErr: false,
		},
		{
			name:    "invalid level",
			input:   "invalid",
			want:    InfoLevel,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    InfoLevel,
			wantErr: true,
		},
		{
			name:    "numeric string",
			input:   "123",
			want:    InfoLevel,
			wantErr: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := ParseLevel(testCase.input)
			if testCase.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid log level")
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, testCase.want, result)
		})
	}
}

func TestParseFormat(t *testing.T) {
	testCases := []struct {
		name    string
		input   string
		want    Format
		wantErr bool
	}{
		{
			name:    "json format",
			input:   "json",
			want:    JSONFormat,
			wantErr: false,
		},
		{
			name:    "json format uppercase",
			input:   "JSON",
			want:    JSONFormat,
			wantErr: false,
		},
		{
			name:    "json format mixed case",
			input:   "JsOn",
			want:    JSONFormat,
			wantErr: false,
		},
		{
			name:    "console format",
			input:   "console",
			want:    ConsoleFormat,
			wantErr: false,
		},
		{
			name:    "console format uppercase",
			input:   "CONSOLE",
			want:    ConsoleFormat,
			wantErr: false,
		},
		{
			name:    "console format mixed case",
			input:   "CoNsOlE",
			want:    ConsoleFormat,
			wantErr: false,
		},
		{
			name:    "whitespace",
			input:   " json ",
			want:    JSONFormat,
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "invalid",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "numeric string",
			input:   "123",
			want:    "",
			wantErr: true,
		},
		{
			name:    "special characters",
			input:   "json!@#",
			want:    "",
			wantErr: true,
		},
		{
			name:    "partial match",
			input:   "jso",
			want:    "",
			wantErr: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := ParseFormat(testCase.input)
			if testCase.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid log format")
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, testCase.want, result)
		})
	}
}
