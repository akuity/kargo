package telemetry

import (
	"errors"
	"fmt"
	"unicode/utf8"

	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

// maxErrorMessageLength bounds the message RecordError records, in
// characters. The SDK's attribute limit covers neither events nor status.
const maxErrorMessageLength = defaultAttributeValueLengthLimit

// RecordError records err, if any, on span and sets its status to Error, with
// URLs and Secret names in the message redacted and the message cut to
// maxErrorMessageLength.
func RecordError(span trace.Span, err error) {
	if err == nil || !span.IsRecording() {
		return
	}
	msg := redactSecretNames(RedactURLs(err.Error()))
	if utf8.RuneCountInString(msg) > maxErrorMessageLength {
		msg = string([]rune(msg)[:maxErrorMessageLength])
	}
	span.AddEvent(
		semconv.ExceptionEventName,
		trace.WithAttributes(
			semconv.ExceptionType(errorType(err)),
			semconv.ExceptionMessage(msg),
		),
	)
	span.SetStatus(codes.Error, msg)
}

// errorType returns err's type, skipping fmt.Errorf("...: %w") layers. It
// stops at the first other type, such as *url.Error.
func errorType(err error) string {
	for {
		t := fmt.Sprintf("%T", err)
		next := errors.Unwrap(err)
		if t != fmtWrapErrorType || next == nil {
			return t
		}
		err = next
	}
}

// fmtWrapErrorType is the type of an error fmt.Errorf returns for a single %w.
var fmtWrapErrorType = fmt.Sprintf("%T", fmt.Errorf("%w", errors.New("")))
