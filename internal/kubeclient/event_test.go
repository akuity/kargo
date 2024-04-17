package kubeclient

import (
	"context"
	"errors"
	"net/http"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/akuity/kargo/internal/logging"
)

func Test_newEventRecorder(t *testing.T) {
	ctx := context.TODO()
	client := fake.NewClientBuilder().Build()
	logger := logging.LoggerFromContext(ctx)
	r := newEventRecorder(ctx, client, logger)

	require.NotNil(t, r.backoff)
	require.NotNil(t, r.sink)
	require.NotNil(t, r.logger)
	require.NotNil(t, r.newEventHandlerFn)
}

func Test_retryDecider(t *testing.T) {
	eventGR := schema.GroupResource{
		Group:    corev1.GroupName,
		Resource: "Event",
	}
	testCases := map[string]struct {
		input       error
		shouldRetry bool
	}{
		"event already exists": {
			input:       apierrors.NewAlreadyExists(eventGR, "fake-event"),
			shouldRetry: false,
		},
		"namespace is terminating": {
			input: &apierrors.StatusError{
				ErrStatus: metav1.Status{
					Code: http.StatusForbidden,
					Details: &metav1.StatusDetails{
						Causes: []metav1.StatusCause{
							{
								Type: corev1.NamespaceTerminatingCause,
							},
						},
					},
				},
			},
			shouldRetry: false,
		},
		"unknown error": {
			input:       errors.New("unknown error"),
			shouldRetry: true,
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			logger, _ := testlog.NewNullLogger()
			r := &eventRecorder{
				logger: logger.WithFields(nil),
			}
			require.Equal(t, tc.shouldRetry, r.newRetryDecider(&corev1.Event{})(tc.input))
		})
	}
}

func Test_newEventSink(t *testing.T) {
	s := newEventSink(
		context.TODO(),
		fake.NewClientBuilder().Build(),
	)
	require.NotNil(t, s.client)
	require.NotNil(t, s.ctx)
}
