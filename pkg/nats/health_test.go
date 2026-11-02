package nats

import (
	"net"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	natstest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
)

// fakeConn is a statusSource whose status changes the test controls. It must
// be used inside a synctest bubble.
type fakeConn struct {
	mu      sync.Mutex
	status  nats.Status
	changes chan nats.Status
}

func newFakeConn(t *testing.T, initial nats.Status) *fakeConn {
	c := &fakeConn{
		status: initial,
		// Matches the buffer the library gives StatusChanged channels
		changes: make(chan nats.Status, 10),
	}
	// Lets the checker's listener exit before the bubble ends
	t.Cleanup(func() { close(c.changes) })
	return c
}

func (c *fakeConn) Status() nats.Status {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.status
}

func (c *fakeConn) StatusChanged(...nats.Status) chan nats.Status {
	return c.changes
}

// set changes the status, reports the change, and waits for the checker to
// apply it.
func (c *fakeConn) set(status nats.Status) {
	c.setSilently(status)
	c.notify(status)
}

// setSilently changes the status without reporting the change, as when the
// library drops a notification because the channel is full.
func (c *fakeConn) setSilently(status nats.Status) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status = status
}

// notify reports a status change, whether or not it is still the current
// status, and waits for the checker to handle it.
func (c *fakeConn) notify(status nats.Status) {
	c.changes <- status
	synctest.Wait()
}

func TestHealthCheckerConfigFromEnv(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		require.Equal(
			t,
			HealthCheckerConfig{LivenessGracePeriod: 10 * time.Minute},
			HealthCheckerConfigFromEnv(),
		)
	})
	t.Run("overridden", func(t *testing.T) {
		t.Setenv("NATS_LIVENESS_GRACE_PERIOD", "30s")
		require.Equal(
			t,
			HealthCheckerConfig{LivenessGracePeriod: 30 * time.Second},
			HealthCheckerConfigFromEnv(),
		)
	})
}

func TestNewHealthChecker(t *testing.T) {
	testCases := []struct {
		name   string
		cfg    HealthCheckerConfig
		assert func(*testing.T, *HealthChecker)
	}{
		{
			name: "grace period unspecified",
			assert: func(t *testing.T, h *HealthChecker) {
				require.Equal(t, defaultLivenessGracePeriod, h.gracePeriod)
			},
		},
		{
			name: "grace period specified",
			cfg:  HealthCheckerConfig{LivenessGracePeriod: time.Minute},
			assert: func(t *testing.T, h *HealthChecker) {
				require.Equal(t, time.Minute, h.gracePeriod)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				testCase.assert(
					t,
					newHealthChecker(newFakeConn(t, nats.CONNECTED), testCase.cfg),
				)
			})
		})
	}
}

func TestHealthCheckerReady(t *testing.T) {
	testCases := []struct {
		name    string
		initial nats.Status
		changes []nats.Status
		assert  func(*testing.T, error)
	}{
		{
			name:    "connected",
			initial: nats.CONNECTED,
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:    "reconnecting",
			initial: nats.CONNECTED,
			changes: []nats.Status{nats.RECONNECTING},
			assert: func(t *testing.T, err error) {
				require.EqualError(t, err, "NATS connection is RECONNECTING")
			},
		},
		{
			name:    "reconnected",
			initial: nats.CONNECTED,
			changes: []nats.Status{nats.RECONNECTING, nats.CONNECTED},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:    "closed",
			initial: nats.CONNECTED,
			changes: []nats.Status{nats.CLOSED},
			assert: func(t *testing.T, err error) {
				require.EqualError(t, err, "NATS connection is CLOSED")
			},
		},
		{
			name:    "not connected when the checker is created",
			initial: nats.RECONNECTING,
			assert: func(t *testing.T, err error) {
				require.EqualError(t, err, "NATS connection is RECONNECTING")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				conn := newFakeConn(t, testCase.initial)
				h := newHealthChecker(conn, HealthCheckerConfig{})
				for _, status := range testCase.changes {
					conn.set(status)
				}
				testCase.assert(t, h.Ready())
			})
		})
	}
}

func TestHealthCheckerStaleNotification(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		conn := newFakeConn(t, nats.CONNECTED)
		h := newHealthChecker(conn, HealthCheckerConfig{})
		conn.set(nats.RECONNECTING)
		// The connection reconnects, but the notification for it is dropped
		// while an earlier RECONNECTING notification is still waiting to be
		// read.
		conn.setSilently(nats.CONNECTED)
		conn.notify(nats.RECONNECTING)
		require.NoError(t, h.Ready())
		require.NoError(t, h.Live())
	})
}

func TestHealthCheckerLive(t *testing.T) {
	const gracePeriod = time.Minute
	testCases := []struct {
		name string
		// run drives the connection through a scenario and returns the result
		// of a liveness check at its end.
		run    func(*fakeConn, *HealthChecker) error
		assert func(*testing.T, error)
	}{
		{
			name: "connected",
			run: func(_ *fakeConn, h *HealthChecker) error {
				return h.Live()
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "closed",
			run: func(conn *fakeConn, h *HealthChecker) error {
				conn.set(nats.CLOSED)
				return h.Live()
			},
			assert: func(t *testing.T, err error) {
				require.EqualError(t, err, "NATS connection is closed")
			},
		},
		{
			name: "disconnected within grace period",
			run: func(conn *fakeConn, h *HealthChecker) error {
				conn.set(nats.RECONNECTING)
				time.Sleep(gracePeriod)
				return h.Live()
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			// The grace period runs from the disconnect itself, not from the
			// first check that finds the connection disconnected.
			name: "disconnected beyond grace period",
			run: func(conn *fakeConn, h *HealthChecker) error {
				conn.set(nats.RECONNECTING)
				time.Sleep(gracePeriod + time.Second)
				return h.Live()
			},
			assert: func(t *testing.T, err error) {
				require.EqualError(
					t,
					err,
					"NATS connection has been disconnected for 1m1s",
				)
			},
		},
		{
			// The library reports RECONNECTING again after every failed
			// attempt.
			name: "repeated reconnect attempts do not reset the grace period",
			run: func(conn *fakeConn, h *HealthChecker) error {
				conn.set(nats.RECONNECTING)
				time.Sleep(gracePeriod / 2)
				conn.set(nats.RECONNECTING)
				time.Sleep(gracePeriod/2 + time.Second)
				return h.Live()
			},
			assert: func(t *testing.T, err error) {
				require.EqualError(
					t,
					err,
					"NATS connection has been disconnected for 1m1s",
				)
			},
		},
		{
			name: "reconnecting resets the grace period",
			run: func(conn *fakeConn, h *HealthChecker) error {
				conn.set(nats.RECONNECTING)
				time.Sleep(gracePeriod - time.Second)
				conn.set(nats.CONNECTED)
				conn.set(nats.RECONNECTING)
				time.Sleep(gracePeriod - time.Second)
				return h.Live()
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				conn := newFakeConn(t, nats.CONNECTED)
				h := newHealthChecker(
					conn,
					HealthCheckerConfig{LivenessGracePeriod: gracePeriod},
				)
				testCase.assert(t, testCase.run(conn, h))
			})
		})
	}
}

func TestHealthCheckerDoesNotBlockDuringReconnect(t *testing.T) {
	t.Parallel()

	srv := natstest.RunRandClientPortServer()
	addr := srv.Addr().String()
	conn, err := nats.Connect(
		srv.ClientURL(),
		// Long enough that a check blocked on the connection would be obvious
		nats.Timeout(5*time.Second),
		nats.ReconnectWait(10*time.Millisecond),
		nats.MaxReconnects(-1),
	)
	require.NoError(t, err)
	t.Cleanup(conn.Close)
	h := NewHealthChecker(conn, HealthCheckerConfig{})
	require.NoError(t, h.Ready())

	// Take the server's place with a listener that accepts connections but
	// never completes the handshake. The client holds its lock while it waits
	// for the handshake, until it times out.
	srv.Shutdown()
	l, err := net.Listen("tcp", addr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })
	accepted := make(chan net.Conn, 1)
	go func() {
		c, acceptErr := l.Accept()
		if acceptErr == nil {
			accepted <- c
		}
	}()
	select {
	case c := <-accepted:
		// Closing this ends the stalled handshake, which lets conn.Close
		// return promptly. Cleanups run last-in, first-out.
		t.Cleanup(func() { _ = c.Close() })
	case <-time.After(5 * time.Second):
		require.FailNow(t, "client never attempted to reconnect")
	}

	start := time.Now()
	require.EqualError(t, h.Ready(), "NATS connection is RECONNECTING")
	require.NoError(t, h.Live())
	require.Less(t, time.Since(start), time.Second)
}
