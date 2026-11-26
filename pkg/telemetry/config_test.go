package telemetry

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigFromEnv(t *testing.T) {
	testCases := []struct {
		name   string
		env    map[string]string
		assert func(*testing.T, Config)
	}{
		{
			name: "defaults",
			env:  map[string]string{"OTEL_SDK_DISABLED": ""},
			assert: func(t *testing.T, cfg Config) {
				require.True(t, cfg.Enabled)
				require.Equal(t, ProtocolHTTPProtobuf, cfg.Protocol)
				require.Empty(t, cfg.TracesProtocol)
				require.Empty(t, cfg.Categories)
			},
		},
		{
			name: "everything set",
			env: map[string]string{
				"OTEL_SDK_DISABLED":                  "false",
				"OTEL_EXPORTER_OTLP_PROTOCOL":        "grpc",
				"OTEL_EXPORTER_OTLP_TRACES_PROTOCOL": "http/protobuf",
				"TRACING_CATEGORIES":                 "Default, k8s",
			},
			assert: func(t *testing.T, cfg Config) {
				require.True(t, cfg.Enabled)
				require.Equal(t, ProtocolGRPC, cfg.Protocol)
				require.Equal(t, ProtocolHTTPProtobuf, cfg.TracesProtocol)
				require.Equal(t, []string{"Default", " k8s"}, cfg.Categories)
			},
		},
		{
			name: "OTEL_SDK_DISABLED true",
			env:  map[string]string{"OTEL_SDK_DISABLED": "TRUE"},
			assert: func(t *testing.T, cfg Config) {
				require.False(t, cfg.Enabled)
			},
		},
		{
			name: "OTEL_SDK_DISABLED mixed-case true",
			env:  map[string]string{"OTEL_SDK_DISABLED": "tRuE"},
			assert: func(t *testing.T, cfg Config) {
				require.False(t, cfg.Enabled)
			},
		},
		{
			name: "OTEL_SDK_DISABLED 1",
			env:  map[string]string{"OTEL_SDK_DISABLED": "1"},
			assert: func(t *testing.T, cfg Config) {
				require.True(t, cfg.Enabled)
			},
		},
		{
			name: "OTEL_SDK_DISABLED not a bool",
			env:  map[string]string{"OTEL_SDK_DISABLED": "yes"},
			assert: func(t *testing.T, cfg Config) {
				require.True(t, cfg.Enabled)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			for k, v := range testCase.env {
				t.Setenv(k, v)
			}
			testCase.assert(t, ConfigFromEnv())
		})
	}
}

func TestConfig_EffectiveTracesProtocol(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		cfg      Config
		expected Protocol
	}{
		{
			name:     "general protocol only",
			cfg:      Config{Protocol: ProtocolGRPC},
			expected: ProtocolGRPC,
		},
		{
			name: "traces-specific protocol overrides general protocol",
			cfg: Config{
				Protocol:       ProtocolGRPC,
				TracesProtocol: ProtocolHTTPProtobuf,
			},
			expected: ProtocolHTTPProtobuf,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.expected, testCase.cfg.EffectiveTracesProtocol())
		})
	}
}

func TestConfig_validate(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		cfg    Config
		assert func(*testing.T, error)
	}{
		{
			name: "grpc",
			cfg:  Config{Protocol: ProtocolGRPC},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "http/protobuf",
			cfg:  Config{Protocol: ProtocolHTTPProtobuf},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "unsupported protocol",
			cfg:  Config{Protocol: "http/json"},
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, `unsupported OTLP protocol "http/json"`)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			testCase.assert(t, testCase.cfg.validate())
		})
	}
}
