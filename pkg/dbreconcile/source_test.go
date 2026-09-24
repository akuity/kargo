package dbreconcile

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSubject(t *testing.T) {
	t.Parallel()

	t.Run("a connection is required", func(t *testing.T) {
		t.Parallel()
		err := Subject[string, item](nil, "kargo.items.>").
			Start(context.Background(), func(context.Context, Event[string, item]) {}, "")
		require.EqualError(t, err, "a NATS connection is required")
	})

	t.Run("a subject is required", func(t *testing.T) {
		t.Parallel()
		err := Subject[string, item](connect(t), "").
			Start(context.Background(), func(context.Context, Event[string, item]) {}, "")
		require.EqualError(t, err, "a subject is required")
	})

	t.Run("a failed subscription is reported", func(t *testing.T) {
		t.Parallel()
		conn := connect(t)
		conn.Close()
		err := Subject[string, item](conn, "kargo.items.>").
			Start(context.Background(), func(context.Context, Event[string, item]) {}, "")
		require.ErrorContains(t, err, `error subscribing to "kargo.items.>"`)
	})

	t.Run("delivers valid events and drops everything else", func(t *testing.T) {
		t.Parallel()
		conn := connect(t)
		emitted := make(chan Event[string, item], 10)
		require.NoError(t, Subject[string, item](conn, "kargo.items.>").Start(
			t.Context(),
			func(_ context.Context, e Event[string, item]) { emitted <- e },
			"",
		))

		// Neither of these is an event it can deliver.
		require.NoError(t, conn.Publish("kargo.items.created", []byte("not json")))
		require.NoError(t, conn.Publish("kargo.items.created", []byte(`{"kind":"Created","key":"a"}`)))
		// A different resource's subject is not delivered at all.
		require.NoError(t, Publish(conn, "kargo.others.created", created("other")))
		require.NoError(t, Publish(conn, "kargo.items.updated", updated("a", "Pending", "Running")))

		// Messages from one publisher arrive in order, so had anything before
		// the valid event been delivered, it would have arrived first.
		select {
		case e := <-emitted:
			require.Equal(t, updated("a", "Pending", "Running"), e)
		case <-time.After(time.Second):
			t.Fatal("event was not delivered")
		}
		require.Empty(t, emitted)
	})

	t.Run("a queue group splits events between its members", func(t *testing.T) {
		t.Parallel()
		conn := connect(t)
		var inGroup1, inGroup2, alone atomic.Int32
		start := func(counter *atomic.Int32, group string) {
			require.NoError(t, Subject[string, item](conn, "kargo.items.>").Start(
				t.Context(),
				func(context.Context, Event[string, item]) { counter.Add(1) },
				group,
			))
		}
		start(&inGroup1, "items.0")
		start(&inGroup2, "items.0")
		start(&alone, "")
		for i := range 10 {
			require.NoError(t, Publish(conn, "kargo.items.created", created(strconv.Itoa(i))))
		}
		// The group as a whole receives each event exactly once...
		require.Eventually(t, func() bool {
			return inGroup1.Load()+inGroup2.Load() == 10
		}, time.Second, time.Millisecond)
		// ...while a subscriber outside it receives all of them.
		require.Eventually(t, func() bool { return alone.Load() == 10 }, time.Second, time.Millisecond)
		require.Never(t, func() bool {
			return inGroup1.Load()+inGroup2.Load() > 10
		}, 50*time.Millisecond, 5*time.Millisecond)
	})

	t.Run("WithoutQueueGroup delivers every event to every subscriber", func(t *testing.T) {
		t.Parallel()
		conn := connect(t)
		var first, second atomic.Int32
		start := func(counter *atomic.Int32) {
			require.NoError(t, Subject[string, item](conn, "kargo.items.>", WithoutQueueGroup()).Start(
				t.Context(),
				func(context.Context, Event[string, item]) { counter.Add(1) },
				"items.0",
			))
		}
		start(&first)
		start(&second)
		for i := range 10 {
			require.NoError(t, Publish(conn, "kargo.items.created", created(strconv.Itoa(i))))
		}
		require.Eventually(t, func() bool {
			return first.Load() == 10 && second.Load() == 10
		}, time.Second, time.Millisecond)
	})

	t.Run("stops delivering once its context is done", func(t *testing.T) {
		t.Parallel()
		conn := connect(t)
		emitted := make(chan Event[string, item], 10)
		ctx, cancel := context.WithCancel(t.Context())
		require.NoError(t, Subject[string, item](conn, "kargo.items.>").Start(
			ctx,
			func(_ context.Context, e Event[string, item]) { emitted <- e },
			"",
		))
		require.Equal(t, 1, conn.NumSubscriptions())

		cancel()
		require.Eventually(t, func() bool { return conn.NumSubscriptions() == 0 }, time.Second, time.Millisecond)
		require.NoError(t, Publish(conn, "kargo.items.created", created("a")))
		require.NoError(t, conn.Flush())
		require.Never(t, func() bool { return len(emitted) > 0 }, 50*time.Millisecond, 5*time.Millisecond)
	})
}
