package telemetry

import (
	"os"
	"strconv"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Span limits applied unless the standard environment variables set them.
const (
	// defaultAttributeValueLengthLimit bounds string attributes. The SDK's
	// default is no limit.
	defaultAttributeValueLengthLimit = 4096
	// defaultLinkCountLimit bounds a span's links. The SDK's default is 128.
	defaultLinkCountLimit = 32
)

// spanLimits returns the SDK's span limits from the environment, with our
// defaults for any unset or invalid.
func spanLimits() sdktrace.SpanLimits {
	limits := sdktrace.NewSpanLimits()
	if !limitInEnv(
		"OTEL_SPAN_ATTRIBUTE_VALUE_LENGTH_LIMIT",
		"OTEL_ATTRIBUTE_VALUE_LENGTH_LIMIT",
	) {
		limits.AttributeValueLengthLimit = defaultAttributeValueLengthLimit
	}
	if !limitInEnv("OTEL_SPAN_LINK_COUNT_LIMIT") {
		limits.LinkCountLimit = defaultLinkCountLimit
	}
	return limits
}

// limitInEnv returns whether the first of keys that is set holds a valid
// limit, as the SDK reads it.
func limitInEnv(keys ...string) bool {
	for _, key := range keys {
		if v := os.Getenv(key); v != "" {
			_, err := strconv.Atoi(v)
			return err == nil
		}
	}
	return false
}
