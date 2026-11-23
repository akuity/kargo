package nats

import (
	"testing"
	"time"

	natstest "github.com/nats-io/nats-server/v2/test"
	natsgo "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
)

type testKey struct {
	Name string
}

type testRow struct {
	Value string
}

var testTopic = NewTopic[testKey, testRow]("kargo.internal.test")

func connect(t *testing.T) *natsgo.Conn {
	t.Helper()
	srv := natstest.RunRandClientPortServer()
	t.Cleanup(srv.Shutdown)
	conn, err := natsgo.Connect(srv.ClientURL())
	require.NoError(t, err)
	t.Cleanup(conn.Close)
	return conn
}

func TestTopic_Subscribe(t *testing.T) {
	t.Parallel()

	t.Run("a connection is required", func(t *testing.T) {
		t.Parallel()
		_, err := testTopic.Subscribe(nil, func(Event[testKey, testRow]) {})
		require.ErrorContains(t, err, "a NATS connection is required")
	})

	t.Run("a topic is required", func(t *testing.T) {
		t.Parallel()
		_, err := Topic[testKey, testRow]{}.Subscribe(
			connect(t),
			func(Event[testKey, testRow]) {},
		)
		require.ErrorContains(t, err, "a topic is required")
	})

	t.Run("a callback is required", func(t *testing.T) {
		t.Parallel()
		_, err := testTopic.Subscribe(connect(t), nil)
		require.ErrorContains(t, err, "a callback is required")
	})

	t.Run("delivers valid events and drops the rest", func(t *testing.T) {
		t.Parallel()
		conn := connect(t)
		events := make(chan Event[testKey, testRow], 10)
		sub, err := testTopic.Subscribe(conn, func(e Event[testKey, testRow]) {
			events <- e
		})
		require.NoError(t, err)
		require.True(t, sub.IsValid())
		require.NoError(t, conn.Flush())

		// Neither a message that is not an event nor an event lacking what
		// its kind requires reaches the callback.
		require.NoError(t, conn.Publish("kargo.internal.test.updated", []byte("not json")))
		require.NoError(t, conn.Publish(
			"kargo.internal.test.updated",
			[]byte(`{"kind":"Updated","key":{"Name":"a"}}`),
		))
		require.NoError(t, testTopic.PublishUpdated(
			conn,
			testKey{Name: "a"},
			testRow{Value: "old"},
			testRow{Value: "new"},
		))
		require.NoError(t, testTopic.PublishDeleted(conn, testKey{Name: "b"}, testRow{Value: "gone"}))

		// Messages from one connection arrive in the order it published
		// them, so the first event received is the first valid one.
		var got []Event[testKey, testRow]
		for range 2 {
			select {
			case e := <-events:
				got = append(got, e)
			case <-time.After(10 * time.Second):
				require.FailNow(t, "timed out waiting for an event")
			}
		}
		require.Equal(t, Updated, got[0].Kind)
		require.Equal(t, testKey{Name: "a"}, got[0].Key)
		require.Equal(t, &testRow{Value: "old"}, got[0].Old)
		require.Equal(t, &testRow{Value: "new"}, got[0].New)
		require.Equal(t, Deleted, got[1].Kind)
		require.Equal(t, testKey{Name: "b"}, got[1].Key)
		require.Equal(t, &testRow{Value: "gone"}, got[1].Old)

		require.NoError(t, sub.Unsubscribe())
		require.False(t, sub.IsValid())
	})

	t.Run("ignores other topics", func(t *testing.T) {
		t.Parallel()
		conn := connect(t)
		events := make(chan Event[testKey, testRow], 10)
		_, err := testTopic.Subscribe(conn, func(e Event[testKey, testRow]) {
			events <- e
		})
		require.NoError(t, err)
		require.NoError(t, conn.Flush())

		other := NewTopic[testKey, testRow]("kargo.internal.other")
		require.NoError(t, other.PublishCreated(conn, testKey{Name: "x"}, testRow{}))
		require.NoError(t, testTopic.PublishCreated(conn, testKey{Name: "y"}, testRow{}))

		select {
		case e := <-events:
			require.Equal(t, testKey{Name: "y"}, e.Key)
		case <-time.After(10 * time.Second):
			require.FailNow(t, "timed out waiting for an event")
		}
	})
}
