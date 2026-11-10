package telemetry

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// recordingTracer returns a tracer whose ended spans recorder holds, without
// touching global state.
func recordingTracer() (trace.Tracer, *tracetest.SpanRecorder) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	return tp.Tracer("test"), recorder
}

func TestRecordError_redactsURLs(t *testing.T) {
	t.Parallel()
	tracer, recorder := recordingTracer()
	_, span := tracer.Start(context.Background(), "test")
	RecordError(span, errors.New(`Get "https://user:token@host/x?sig=abc": EOF`))
	RecordError(span, nil)
	span.End()
	spans := recorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, `Get "https://host/x": EOF`, spans[0].Status().Description)
	require.Len(t, spans[0].Events(), 1)
	require.Contains(
		t,
		spans[0].Events()[0].Attributes,
		semconv.ExceptionMessage(`Get "https://host/x": EOF`),
	)
}

func TestRecordError_redactsSecretNames(t *testing.T) {
	t.Parallel()
	secrets := corev1.Resource("secrets")
	forbidden := apierrors.NewForbidden(
		secrets,
		"repo-creds",
		errors.New(`User "system:serviceaccount:kargo:kargo-controller" cannot get `+
			`resource "secrets" in API group "" in the namespace "kargo-demo"`),
	)
	testCases := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "Kubernetes NotFound",
			err:  apierrors.NewNotFound(secrets, "repo-creds"),
			want: `secrets "{redacted}" not found`,
		},
		{
			name: "Kubernetes Forbidden",
			err:  forbidden,
			want: `secrets "{redacted}" is forbidden: User "system:serviceaccount:kargo:kargo-controller" ` +
				`cannot get resource "secrets" in API group "" in the namespace "kargo-demo"`,
		},
		{
			name: "Kubernetes AlreadyExists",
			err:  apierrors.NewAlreadyExists(secrets, "repo-creds"),
			want: `secrets "{redacted}" already exists`,
		},
		{
			name: "Kubernetes Conflict",
			err: apierrors.NewConflict(
				secrets,
				"repo-creds",
				errors.New("the object has been modified; please apply your changes to the latest version and try again"),
			),
			want: `Operation cannot be fulfilled on secrets "{redacted}": the object has been modified; ` +
				`please apply your changes to the latest version and try again`,
		},
		{
			name: "Kubernetes Invalid",
			err: apierrors.NewInvalid(
				schema.GroupKind{Kind: "Secret"},
				"repo-creds",
				field.ErrorList{field.Required(field.NewPath("data"), "")},
			),
			want: `Secret "{redacted}" is invalid: data: Required value`,
		},
		{
			name: "secret() expression function",
			err: fmt.Errorf(
				"failed to get secret %q: %w",
				"repo-creds",
				forbidden,
			),
			want: `failed to get secret "{redacted}": secrets "{redacted}" is forbidden: ` +
				`User "system:serviceaccount:kargo:kargo-controller" cannot get resource "secrets" ` +
				`in API group "" in the namespace "kargo-demo"`,
		},
		{
			name: "Kargo Secret %q",
			err: fmt.Errorf(
				"error getting signing key Secret %q: %w",
				"gpg-key",
				apierrors.NewNotFound(secrets, "gpg-key"),
			),
			want: `error getting signing key Secret "{redacted}": secrets "{redacted}" not found`,
		},
		{
			name: "Kargo secret %q, with a key",
			err:  fmt.Errorf("secret %q does not contain expected key %q", "gpg-key", "signingKey"),
			want: `secret "{redacted}" does not contain expected key "signingKey"`,
		},
		{
			name: "a URL and a Secret",
			err: fmt.Errorf(
				`error cloning "https://user:token@github.com/org/repo.git": %w`,
				forbidden,
			),
			want: `error cloning "https://github.com/org/repo.git": secrets "{redacted}" is forbidden: ` +
				`User "system:serviceaccount:kargo:kargo-controller" cannot get resource "secrets" ` +
				`in API group "" in the namespace "kargo-demo"`,
		},
		{
			name: "unrelated quoted words",
			err:  errors.New(`Stage "secret" not found; secret data has no key "token"`),
			want: `Stage "secret" not found; secret data has no key "token"`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			tracer, recorder := recordingTracer()
			_, span := tracer.Start(context.Background(), "test")
			RecordError(span, testCase.err)
			span.End()
			got := recorder.Ended()[0]
			require.Equal(t, testCase.want, got.Status().Description)
			require.Contains(
				t,
				got.Events()[0].Attributes,
				semconv.ExceptionMessage(testCase.want),
			)
		})
	}
}

func TestRecordError_truncates(t *testing.T) {
	t.Parallel()
	tracer, recorder := recordingTracer()
	_, span := tracer.Start(context.Background(), "test")
	// Multi-byte characters: the limit counts characters, not bytes.
	RecordError(span, errors.New(strings.Repeat("é", maxErrorMessageLength+100)))
	span.End()
	want := strings.Repeat("é", maxErrorMessageLength)
	require.Equal(t, want, recorder.Ended()[0].Status().Description)
	require.Contains(
		t,
		recorder.Ended()[0].Events()[0].Attributes,
		semconv.ExceptionMessage(want),
	)
}

// testError is an error type for asserting exception.type.
type testError struct{}

func (testError) Error() string { return "test" }

func TestRecordError_exceptionType(t *testing.T) {
	t.Parallel()
	tracer, recorder := recordingTracer()
	_, span := tracer.Start(context.Background(), "test")
	// The type of what's wrapped, not of the wrapping.
	RecordError(span, fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", testError{})))
	span.End()
	require.Contains(
		t,
		recorder.Ended()[0].Events()[0].Attributes,
		semconv.ExceptionType("telemetry.testError"),
	)
}

func TestErrorType(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "unwrapped",
			err:  testError{},
			want: "telemetry.testError",
		},
		{
			name: "through fmt.Errorf wrapping",
			err:  fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", testError{})),
			want: "telemetry.testError",
		},
		{
			name: "stops at a typed wrapper",
			err: fmt.Errorf(
				"error cloning: %w",
				&url.Error{Op: "Get", URL: "https://host", Err: testError{}},
			),
			want: "*url.Error",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.want, errorType(testCase.err))
		})
	}
}
