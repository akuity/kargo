package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

func Test_errorFromWatchStartError(t *testing.T) {
	t.Parallel()

	t.Run("passes through ordinary errors", func(t *testing.T) {
		t.Parallel()
		err := errors.New("network unavailable")

		require.ErrorIs(t, errorFromWatchStartError(err), err)
	})

	t.Run("maps expired resource versions to out of range", func(t *testing.T) {
		t.Parallel()
		err := apierrors.NewResourceExpired("too old resource version")

		mapped := errorFromWatchStartError(err)

		require.Error(t, mapped)
		require.Equal(t, connect.CodeOutOfRange, connect.CodeOf(mapped))
		require.ErrorContains(t, mapped, "watch resource version expired")
	})

	t.Run("returns nil for nil", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, errorFromWatchStartError(nil))
	})
}

func Test_errorFromWatchStartError_withStatusError(t *testing.T) {
	t.Parallel()

	err := apierrors.NewNotFound(
		schema.GroupResource{Group: "kargo.akuity.io", Resource: "promotions"},
		"promotion-1",
	)

	mapped := errorFromWatchStartError(err)

	require.ErrorIs(t, mapped, err)
	require.Equal(t, connect.CodeUnknown, connect.CodeOf(mapped))
}

func Test_sendSSEWatchError(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	err := connect.NewError(connect.CodeOutOfRange, errors.New("resource version expired"))

	sendSSEWatchError(c, err)

	require.Contains(t, recorder.Body.String(), "event: error")
	require.Contains(t, recorder.Body.String(), `"code":"out_of_range"`)
	require.Contains(t, recorder.Body.String(), `"message":"resource version expired"`)
	require.NotContains(t, recorder.Body.String(), "out_of_range: resource version expired")
}

func Test_convertAndSendWatchEvent_errorEvent(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	keepGoing := convertAndSendWatchEvent(c, watch.Event{
		Type: watch.Error,
		Object: &metav1.Status{
			Status:  metav1.StatusFailure,
			Message: "too old resource version",
			Reason:  metav1.StatusReasonExpired,
			Code:    http.StatusGone,
		},
	}, (*metav1.PartialObjectMetadata)(nil))

	require.False(t, keepGoing)
	require.Contains(t, recorder.Body.String(), "event: error")
	require.Contains(t, recorder.Body.String(), `"code":"out_of_range"`)
}

func Test_filteredWatchEventType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		eventType watch.EventType
		matches   bool
		wantType  watch.EventType
		wantSend  bool
	}{
		{
			name:      "matching modified passes through",
			eventType: watch.Modified,
			matches:   true,
			wantType:  watch.Modified,
			wantSend:  true,
		},
		{
			name:      "matching added passes through",
			eventType: watch.Added,
			matches:   true,
			wantType:  watch.Added,
			wantSend:  true,
		},
		{
			name:      "matching deleted passes through",
			eventType: watch.Deleted,
			matches:   true,
			wantType:  watch.Deleted,
			wantSend:  true,
		},
		{
			name:      "non-matching modified becomes delete",
			eventType: watch.Modified,
			wantType:  watch.Deleted,
			wantSend:  true,
		},
		{
			name:      "non-matching added is skipped",
			eventType: watch.Added,
		},
		{
			name:      "non-matching deleted is skipped",
			eventType: watch.Deleted,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			eventType, send := filteredWatchEventType(testCase.eventType, testCase.matches)

			require.Equal(t, testCase.wantSend, send)
			require.Equal(t, testCase.wantType, eventType)
		})
	}
}
