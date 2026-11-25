package telemetry

import (
	"fmt"
	"os"
	"strings"

	"github.com/kelseyhightower/envconfig"
)

// Protocol identifies the transport used to export telemetry over OTLP.
type Protocol string

const (
	// ProtocolGRPC exports telemetry over OTLP/gRPC.
	ProtocolGRPC Protocol = "grpc"
	// ProtocolHTTPProtobuf exports telemetry over OTLP/HTTP with a protobuf
	// payload.
	ProtocolHTTPProtobuf Protocol = "http/protobuf"
)

// Config represents configuration for a component's telemetry. Apart from
// OTEL_SDK_DISABLED, which the Go SDK doesn't read, it is deliberately
// minimal: the OpenTelemetry SDK and exporters configure themselves from the standard OTEL_* environment
// variables (OTEL_EXPORTER_OTLP_ENDPOINT, OTEL_EXPORTER_OTLP_HEADERS,
// OTEL_TRACES_SAMPLER, OTEL_RESOURCE_ATTRIBUTES, and so on), and this package
// intentionally does not duplicate them.
type Config struct {
	// Enabled indicates whether traces are exported at all: true unless
	// OTEL_SDK_DISABLED is true. When false, the OpenTelemetry API's no-op
	// implementations remain in place and the instrumentation throughout Kargo
	// costs next to nothing.
	Enabled bool `ignored:"true"`
	// Protocol is the OTLP transport used for all signals unless overridden for
	// a specific signal. The Go exporters do not read this environment variable
	// themselves, so it is handled here.
	Protocol Protocol `envconfig:"OTEL_EXPORTER_OTLP_PROTOCOL" default:"http/protobuf"`
	// TracesProtocol, when set, overrides Protocol for traces specifically.
	TracesProtocol Protocol `envconfig:"OTEL_EXPORTER_OTLP_TRACES_PROTOCOL"`
	// Categories selects the span categories to record.
	Categories []string `envconfig:"TRACING_CATEGORIES"`
}

// ConfigFromEnv returns a Config populated from environment variables.
func ConfigFromEnv() Config {
	cfg := Config{}
	envconfig.MustProcess("", &cfg)
	// The Go SDK doesn't read OTEL_SDK_DISABLED; per spec, only a
	// case-insensitive "true" disables it.
	cfg.Enabled = !strings.EqualFold(os.Getenv("OTEL_SDK_DISABLED"), "true")
	return cfg
}

// EffectiveTracesProtocol returns the Protocol to use for exporting traces:
// TracesProtocol if set, otherwise Protocol.
func (c Config) EffectiveTracesProtocol() Protocol {
	if c.TracesProtocol != "" {
		return c.TracesProtocol
	}
	return c.Protocol
}

// validate returns an error if the Config would not produce a usable
// exporter.
func (c Config) validate() error {
	switch p := c.EffectiveTracesProtocol(); p {
	case ProtocolGRPC, ProtocolHTTPProtobuf:
		return nil
	default:
		return fmt.Errorf(
			"unsupported OTLP protocol %q; supported protocols are %q and %q",
			p, ProtocolGRPC, ProtocolHTTPProtobuf,
		)
	}
}
