package nats

import (
	"encoding/json"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	natstest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/event"
)

func TestEventSender_Send(t *testing.T) {
	srv := natstest.RunRandClientPortServer()
	t.Cleanup(srv.Shutdown)
	conn, err := nats.Connect(srv.ClientURL())
	require.NoError(t, err)
	t.Cleanup(conn.Close)

	evt, err := event.NewCloudEvent(
		string(kargoapi.EventTypeFreightCreated),
		map[string]string{"foo": "bar"},
	)
	require.NoError(t, err)
	evt.SetID("fake-id")
	evt.SetSource("/kargo/fake-component")

	sub, err := conn.SubscribeSync(event.EventsSubjectPrefix + ".>")
	require.NoError(t, err)
	t.Cleanup(func() { _ = sub.Unsubscribe() })
	require.NoError(t, conn.Flush())

	sender := NewEventSender(conn)
	require.NoError(
		t,
		sender.Send(t.Context(), event.NewEventsSubjectPrefix("Freight"), evt),
	)
	sender.Shutdown()

	msg, err := sub.NextMsg(time.Second)
	require.NoError(t, err)
	require.Equal(
		t,
		"akuity.kargo.events.freight.FreightCreated",
		msg.Subject,
	)
	require.Equal(
		t,
		cloudevents.ApplicationCloudEventsJSON,
		msg.Header.Get(contentTypeHeader),
	)
	var received cloudevents.Event
	require.NoError(t, json.Unmarshal(msg.Data, &received))
	require.Equal(t, "fake-id", received.ID())
	require.Equal(t, evt.Type(), received.Type())
	require.Equal(t, "/kargo/fake-component", received.Source())
	// The subject attribute is published along with the event
	require.Equal(t, msg.Subject, received.Subject())
	require.JSONEq(t, `{"foo":"bar"}`, string(received.Data()))
}

func TestNewDefaultingEventSender(t *testing.T) {
	srv := natstest.RunRandClientPortServer()
	t.Cleanup(srv.Shutdown)
	conn, err := nats.Connect(srv.ClientURL())
	require.NoError(t, err)
	t.Cleanup(conn.Close)

	evt, err := event.NewCloudEvent("KindlessEvent", nil)
	require.NoError(t, err)

	sub, err := conn.SubscribeSync(event.EventsSubjectPrefix + ".>")
	require.NoError(t, err)
	t.Cleanup(func() { _ = sub.Unsubscribe() })
	require.NoError(t, conn.Flush())

	sender := NewDefaultingEventSender(conn, "test-component")
	require.NoError(
		t,
		sender.Send(t.Context(), event.NewEventsSubjectPrefix(""), evt),
	)
	sender.Shutdown()

	msg, err := sub.NextMsg(time.Second)
	require.NoError(t, err)
	require.Equal(t, "akuity.kargo.events.GLOBAL.KindlessEvent", msg.Subject)
	var received cloudevents.Event
	require.NoError(t, json.Unmarshal(msg.Data, &received))
	// The defaulting wrapper fills in everything the event was missing
	require.NoError(t, received.Validate())
	require.NotEmpty(t, received.ID())
	require.Equal(t, "/kargo/test-component", received.Source())
	require.False(t, received.Time().IsZero())
	require.Equal(t, msg.Subject, received.Subject())
}
