package telemetry

import (
	"errors"
	"fmt"
	"reflect"
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
		t := typeName(err)
		next := errors.Unwrap(err)
		if t != fmtWrapErrorType || next == nil {
			return t
		}
		err = next
	}
}

// typeName returns err's type as the OTel SDK's RecordError names it.
func typeName(err error) string {
	t := reflect.TypeOf(err)
	if t.PkgPath() == "" && t.Name() == "" {
		return t.String()
	}
	return t.PkgPath() + "." + t.Name()
}

// fmtWrapErrorType is the type of an error fmt.Errorf returns for a single %w.
var fmtWrapErrorType = typeName(fmt.Errorf("%w", errors.New("")))
