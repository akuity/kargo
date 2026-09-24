package event

import (
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewCloudEvent(t *testing.T) {
	type payload struct {
		Foo string `json:"foo"`
	}
	testCases := []struct {
		name      string
		eventType string
		data      any
		assert    func(*testing.T, cloudevents.Event, error)
	}{
		{
			name: "missing event type",
			data: payload{Foo: "bar"},
			assert: func(t *testing.T, _ cloudevents.Event, err error) {
				require.ErrorIs(t, err, ErrMissingEventType)
			},
		},
		{
			name:      "data cannot be encoded",
			eventType: "CustomType",
			data:      make(chan int),
			assert: func(t *testing.T, _ cloudevents.Event, err error) {
				require.Error(t, err)
			},
		},
		{
			name:      "success",
			eventType: "CustomType",
			data:      payload{Foo: "bar"},
			assert: func(t *testing.T, evt cloudevents.Event, err error) {
				require.NoError(t, err)
				require.Equal(t, cloudevents.CloudEventsVersionV1, evt.SpecVersion())
				require.Equal(t, "CustomType", evt.Type())
				require.Equal(t, cloudevents.ApplicationJSON, evt.DataContentType())
				require.JSONEq(t, `{"foo":"bar"}`, string(evt.Data()))
				require.NotContains(t, evt.Extensions(), KindExtension)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			evt, err := NewCloudEvent(testCase.eventType, testCase.data)
			testCase.assert(t, evt, err)
		})
	}
}

func Test_newUserEvent(t *testing.T) {
	testCases := []struct {
		name   string
		kind   string
		assert func(*testing.T, cloudevents.Event, error)
	}{
		{
			name: "with kind",
			kind: "Thing",
			assert: func(t *testing.T, evt cloudevents.Event, err error) {
				require.NoError(t, err)
				requireCloudEvent(t, evt, "CustomType", "Thing")
			},
		},
		{
			name: "without kind",
			assert: func(t *testing.T, evt cloudevents.Event, err error) {
				require.NoError(t, err)
				requireCloudEvent(t, evt, "CustomType", "")
				require.NotContains(t, evt.Extensions(), KindExtension)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			evt, err := newUserEvent("CustomType", testCase.kind, nil)
			testCase.assert(t, evt, err)
		})
	}
}

func TestComponentSource(t *testing.T) {
	require.Equal(t, "/kargo/stage-controller", ComponentSource("stage-controller"))
}

func TestKindOf(t *testing.T) {
	evt := cloudevents.New()
	require.Empty(t, KindOf(evt))
	evt.SetExtension(KindExtension, "Thing")
	require.Equal(t, "Thing", KindOf(evt))
}

// requireCloudEvent asserts that the given CloudEvent has the given type, is
// about an object of the given kind, and carries JSON data.
func requireCloudEvent(
	t *testing.T,
	evt cloudevents.Event,
	eventType kargoapi.EventType,
	kind string,
) {
	t.Helper()
	require.Equal(t, cloudevents.CloudEventsVersionV1, evt.SpecVersion())
	require.Equal(t, string(eventType), evt.Type())
	require.Equal(t, kind, KindOf(evt))
	require.Equal(t, cloudevents.ApplicationJSON, evt.DataContentType())
}

// dataAs decodes the data of the given CloudEvent as a T.
func dataAs[T any](t *testing.T, evt cloudevents.Event) *T {
	t.Helper()
	data := new(T)
	require.NoError(t, evt.DataAs(data))
	return data
}
