package dbreconcile

import (
	"encoding/json"
	"testing"
	"time"

	natstest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
)

// item is the resource type the tests use.
type item struct {
	Name  string `json:"name"`
	Phase string `json:"phase,omitempty"`
}

func created(key string) Event[string, item] {
	return Event[string, item]{Kind: Created, Key: key, New: &item{Name: key}}
}

func updated(key, oldPhase, newPhase string) Event[string, item] {
	return Event[string, item]{
		Kind: Updated,
		Key:  key,
		Old:  &item{Name: key, Phase: oldPhase},
		New:  &item{Name: key, Phase: newPhase},
	}
}

func deleted(key string) Event[string, item] {
	return Event[string, item]{Kind: Deleted, Key: key, Old: &item{Name: key}}
}

// connect starts an in-process NATS server and connects to it.
func connect(t *testing.T) *nats.Conn {
	t.Helper()
	srv := natstest.RunRandClientPortServer()
	t.Cleanup(srv.Shutdown)
	conn, err := nats.Connect(srv.ClientURL())
	require.NoError(t, err)
	t.Cleanup(conn.Close)
	return conn
}

func TestEventValidate(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		event   Event[string, item]
		wantErr string
	}{
		{name: "created", event: created("a")},
		{name: "updated", event: updated("a", "Pending", "Running")},
		{name: "deleted", event: deleted("a")},
		{
			name:    "created without the new resource",
			event:   Event[string, item]{Kind: Created, Key: "a"},
			wantErr: "a Created event requires the new resource",
		},
		{
			name:    "updated without the old resource",
			event:   Event[string, item]{Kind: Updated, Key: "a", New: &item{}},
			wantErr: "an Updated event requires both the old and the new resource",
		},
		{
			name:    "updated without the new resource",
			event:   Event[string, item]{Kind: Updated, Key: "a", Old: &item{}},
			wantErr: "an Updated event requires both the old and the new resource",
		},
		{
			name:    "deleted without the old resource",
			event:   Event[string, item]{Kind: Deleted, Key: "a"},
			wantErr: "a Deleted event requires the old resource",
		},
		{
			name:    "unknown kind",
			event:   Event[string, item]{Kind: "Renamed", Key: "a", New: &item{}},
			wantErr: `unknown event kind "Renamed"`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			err := testCase.event.validate()
			if testCase.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, testCase.wantErr)
		})
	}
}

func TestEventJSON(t *testing.T) {
	t.Parallel()
	data, err := json.Marshal(created("a"))
	require.NoError(t, err)
	// Absent resources are left out of the message altogether.
	require.JSONEq(t, `{"kind":"Created","key":"a","new":{"name":"a"}}`, string(data))

	decoded := Event[string, item]{}
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, created("a"), decoded)
}

func TestPublish(t *testing.T) {
	t.Parallel()

	t.Run("no connection is a no-op", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, Publish(nil, "kargo.items.created", created("a")))
	})

	t.Run("subject is required", func(t *testing.T) {
		t.Parallel()
		require.EqualError(t, Publish(connect(t), "", created("a")), "subject is required")
	})

	t.Run("invalid events are refused", func(t *testing.T) {
		t.Parallel()
		err := Publish(connect(t), "kargo.items.created", Event[string, item]{Kind: Created, Key: "a"})
		require.ErrorContains(t, err, "invalid event: a Created event requires the new resource")
	})

	t.Run("publishes the event as JSON", func(t *testing.T) {
		t.Parallel()
		conn := connect(t)
		sub, err := conn.SubscribeSync("kargo.items.>")
		require.NoError(t, err)
		require.NoError(t, Publish(conn, "kargo.items.updated", updated("a", "Pending", "Running")))
		msg, err := sub.NextMsg(time.Second)
		require.NoError(t, err)
		require.Equal(t, "kargo.items.updated", msg.Subject)
		require.JSONEq(
			t,
			`{"kind":"Updated","key":"a","old":{"name":"a","phase":"Pending"},"new":{"name":"a","phase":"Running"}}`,
			string(msg.Data),
		)
	})

	t.Run("a closed connection is reported", func(t *testing.T) {
		t.Parallel()
		conn := connect(t)
		conn.Close()
		require.ErrorContains(
			t,
			Publish(conn, "kargo.items.created", created("a")),
			`error publishing event to "kargo.items.created"`,
		)
	})
}
