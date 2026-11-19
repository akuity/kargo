package nats

import (
	"context"
	"errors"
	"testing"
	"time"

	natstest "github.com/nats-io/nats-server/v2/test"
	natsgo "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/event"
	"github.com/akuity/kargo/pkg/event/fake"
)

type testKey struct {
	Project string
	Name    string
}

type testRow struct {
	Name  string
	Phase string
}

func TestNewTopic(t *testing.T) {
	t.Parallel()
	topic := NewTopic[testKey, testRow]("PromotionRollout")
	require.Equal(t, "akuity.platform.internal.promotion.rollout.*", topic.Subject())
	require.Panics(t, func() { NewTopic[testKey, testRow]("Promotion.Rollout") })
}

func TestTopicPublish(t *testing.T) {
	t.Parallel()
	topic := NewTopic[testKey, testRow]("Test")
	key := testKey{Project: "p", Name: "r"}
	old := testRow{Name: "r", Phase: "Pending"}
	updated := testRow{Name: "r", Phase: "Running"}

	testCases := []struct {
		name    string
		topic   Topic[testKey, testRow]
		sender  *fake.Sender
		publish func(Topic[testKey, testRow], *Publisher) error
		assert  func(*testing.T, *fake.Sender, error)
	}{
		{
			name:   "created",
			topic:  topic,
			sender: &fake.Sender{SubjectPrefix: InternalSubjectPrefix},
			publish: func(topic Topic[testKey, testRow], pub *Publisher) error {
				return topic.PublishCreated(context.Background(), pub, key, updated)
			},
			assert: func(t *testing.T, sender *fake.Sender, err error) {
				require.NoError(t, err)
				sent := sender.Sent()
				require.Len(t, sent, 1)
				require.Equal(t, "TestCreated", sent[0].Type())
				require.Equal(t, "akuity.platform.internal.test.created", sent[0].Subject())
				require.Equal(t, "Test", event.KindOf(sent[0]))
				e, err := topic.Decode(sent[0])
				require.NoError(t, err)
				require.Equal(t, Created, e.Kind)
				require.Equal(t, key, e.Key)
				require.Nil(t, e.Old)
				require.Equal(t, &updated, e.New)
			},
		},
		{
			name:   "updated",
			topic:  topic,
			sender: &fake.Sender{SubjectPrefix: InternalSubjectPrefix},
			publish: func(topic Topic[testKey, testRow], pub *Publisher) error {
				return topic.PublishUpdated(context.Background(), pub, key, old, updated)
			},
			assert: func(t *testing.T, sender *fake.Sender, err error) {
				require.NoError(t, err)
				sent := sender.Sent()
				require.Len(t, sent, 1)
				require.Equal(t, "TestUpdated", sent[0].Type())
				e, err := topic.Decode(sent[0])
				require.NoError(t, err)
				require.Equal(t, Updated, e.Kind)
				require.Equal(t, key, e.Key)
				require.Equal(t, &old, e.Old)
				require.Equal(t, &updated, e.New)
			},
		},
		{
			name:   "deleted",
			topic:  topic,
			sender: &fake.Sender{SubjectPrefix: InternalSubjectPrefix},
			publish: func(topic Topic[testKey, testRow], pub *Publisher) error {
				return topic.PublishDeleted(context.Background(), pub, key, old)
			},
			assert: func(t *testing.T, sender *fake.Sender, err error) {
				require.NoError(t, err)
				sent := sender.Sent()
				require.Len(t, sent, 1)
				require.Equal(t, "TestDeleted", sent[0].Type())
				e, err := topic.Decode(sent[0])
				require.NoError(t, err)
				require.Equal(t, Deleted, e.Kind)
				require.Equal(t, &old, e.Old)
				require.Nil(t, e.New)
			},
		},
		{
			name:  "a nil publisher publishes nothing",
			topic: topic,
			publish: func(topic Topic[testKey, testRow], _ *Publisher) error {
				return topic.PublishCreated(context.Background(), nil, key, updated)
			},
			assert: func(t *testing.T, _ *fake.Sender, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:   "a topic without a resource cannot publish",
			topic:  Topic[testKey, testRow]{},
			sender: &fake.Sender{SubjectPrefix: InternalSubjectPrefix},
			publish: func(topic Topic[testKey, testRow], pub *Publisher) error {
				return topic.PublishCreated(context.Background(), pub, key, updated)
			},
			assert: func(t *testing.T, sender *fake.Sender, err error) {
				require.ErrorContains(t, err, "no resource")
				require.Empty(t, sender.Sent())
			},
		},
		{
			name:  "a sender failure is returned with the event type",
			topic: topic,
			sender: &fake.Sender{
				SubjectPrefix: InternalSubjectPrefix,
				SendErr:       errors.New("disconnected"),
			},
			publish: func(topic Topic[testKey, testRow], pub *Publisher) error {
				return topic.PublishUpdated(context.Background(), pub, key, old, updated)
			},
			assert: func(t *testing.T, _ *fake.Sender, err error) {
				require.ErrorContains(t, err, "TestUpdated")
				require.ErrorContains(t, err, "disconnected")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var pub *Publisher
			if testCase.sender != nil {
				pub = NewPublisherWithSender(testCase.sender)
			}
			err := testCase.publish(testCase.topic, pub)
			testCase.assert(t, testCase.sender, err)
		})
	}
}

func TestTopicDecode(t *testing.T) {
	t.Parallel()
	topic := NewTopic[testKey, testRow]("Test")
	testCases := []struct {
		name      string
		eventType string
		data      any
		assert    func(*testing.T, Event[testKey, testRow], error)
	}{
		{
			name:      "an event of the topic",
			eventType: "TestCreated",
			data:      map[string]any{"key": testKey{Name: "r"}, "new": testRow{Name: "r"}},
			assert: func(t *testing.T, e Event[testKey, testRow], err error) {
				require.NoError(t, err)
				require.Equal(t, Created, e.Kind)
				require.Equal(t, testKey{Name: "r"}, e.Key)
			},
		},
		{
			name:      "another resource's event",
			eventType: "OtherCreated",
			data:      map[string]any{"new": testRow{}},
			assert: func(t *testing.T, _ Event[testKey, testRow], err error) {
				require.ErrorContains(t, err, "is not one of topic Test's")
			},
		},
		{
			name:      "an event of a resource whose name extends the topic's",
			eventType: "TestThingCreated",
			data:      map[string]any{"new": testRow{}},
			assert: func(t *testing.T, _ Event[testKey, testRow], err error) {
				require.ErrorContains(t, err, "unknown event kind")
			},
		},
		{
			name:      "an event missing what its kind requires",
			eventType: "TestUpdated",
			data:      map[string]any{"new": testRow{}},
			assert: func(t *testing.T, _ Event[testKey, testRow], err error) {
				require.ErrorContains(t, err, "requires both the old and the new")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			evt, err := event.NewCloudEvent(testCase.eventType, testCase.data)
			require.NoError(t, err)
			e, err := topic.Decode(evt)
			testCase.assert(t, e, err)
		})
	}
}

// TestTopicRoundTrip checks that what a topic publishes through a real NATS
// server reaches a source subscribed to it, and that a source does not
// receive the events of a resource whose subjects share the topic's prefix.
func TestTopicRoundTrip(t *testing.T) {
	t.Parallel()
	srv := natstest.RunRandClientPortServer()
	t.Cleanup(srv.Shutdown)
	conn, err := natsgo.Connect(srv.ClientURL())
	require.NoError(t, err)
	t.Cleanup(conn.Close)

	topic := NewTopic[testKey, testRow]("Test")
	// "TestThing" publishes to akuity.platform.internal.test.thing.*, which
	// a subscription to all of Test's subjects must not match.
	extended := NewTopic[testKey, testRow]("TestThing")

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	out := make(chan testKey, 2)
	require.NoError(t, Subject(conn, topic).Start(ctx, out))
	require.NoError(t, conn.Flush())

	pub := NewPublisher(conn, "test")
	require.NoError(t, extended.PublishCreated(ctx, pub, testKey{Name: "other"}, testRow{}))
	require.NoError(t, topic.PublishUpdated(
		ctx,
		pub,
		testKey{Project: "p", Name: "r"},
		testRow{Phase: "Pending"},
		testRow{Phase: "Running"},
	))
	pub.Shutdown()

	select {
	case key := <-out:
		require.Equal(t, testKey{Project: "p", Name: "r"}, key)
	case <-time.After(5 * time.Second):
		require.Fail(t, "no event received")
	}
	select {
	case key := <-out:
		require.Failf(t, "unexpected event", "received %v", key)
	case <-time.After(100 * time.Millisecond):
	}
}
