package event

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/require"
)

// recordingSender is a Sender that records what it is asked to send. The fake
// package's Sender can't be used here because it imports this package.
type recordingSender struct {
	err           error
	subjectPrefix string
	sent          []cloudevents.Event
	shutdown      bool
}

func (r *recordingSender) Send(
	_ context.Context,
	subjectPrefix string,
	evt cloudevents.Event,
) error {
	r.subjectPrefix = subjectPrefix
	r.sent = append(r.sent, evt)
	return r.err
}

func (r *recordingSender) Shutdown() {
	r.shutdown = true
}

func TestDefaultingSender_Send(t *testing.T) {
	testCases := []struct {
		name   string
		err    error
		assert func(*testing.T, *recordingSender, error)
	}{
		{
			name: "underlying sender fails",
			err:  errors.New("something went wrong"),
			assert: func(t *testing.T, _ *recordingSender, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success",
			assert: func(t *testing.T, r *recordingSender, err error) {
				require.NoError(t, err)
				require.Equal(t, "akuity.kargo.things", r.subjectPrefix)
				require.Len(t, r.sent, 2)
				for _, evt := range r.sent {
					require.NoError(t, evt.Validate())
					require.Equal(t, "/kargo/test-component", evt.Source())
					require.WithinDuration(t, time.Now(), evt.Time(), time.Minute)
					id, err := uuid.Parse(evt.ID())
					require.NoError(t, err)
					// The version is the high nibble of the seventh byte
					require.Equal(t, byte(7), id[6]>>4)
				}
				// UUIDv7s sort in creation order
				require.Less(t, r.sent[0].ID(), r.sent[1].ID())
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			underlying := &recordingSender{err: testCase.err}
			sender := NewDefaultingSender(underlying, "test-component")
			var err error
			for range 2 {
				evt, newErr := NewCloudEvent("CustomType", nil)
				require.NoError(t, newErr)
				if err = sender.Send(t.Context(), "akuity.kargo.things", evt); err != nil {
					break
				}
			}
			testCase.assert(t, underlying, err)
		})
	}
}

func TestDefaultingSender_Shutdown(t *testing.T) {
	underlying := &recordingSender{}
	NewDefaultingSender(underlying, "test-component").Shutdown()
	require.True(t, underlying.shutdown)
}
