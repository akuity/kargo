package natswatch

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	natstest "github.com/nats-io/nats-server/v2/test"
	natsgo "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/database"
	natsrec "github.com/akuity/kargo/pkg/reconciler/nats"
)

const (
	testProject  = "fake-project"
	otherProject = "other-project"
	// noKeepalive is a keepalive interval no test reaches, for tests that
	// must see only what events cause.
	noKeepalive = time.Hour
	// waitFor bounds every wait for something the watch sends.
	waitFor = 10 * time.Second
)

type testKey struct {
	Project string
}

type testRow struct {
	Value string
}

var testTopic = natsrec.NewTopic[testKey, testRow]("kargo.internal.natswatchtest")

// testObject is what the watched resource is served as.
type testObject struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// resource is the watched resource, which a test changes behind the watch's
// back, publishing an event or not.
type resource struct {
	mu sync.Mutex
	// obj is the resource, or nil once it is gone.
	obj *testObject
	// err, when set, fails every load.
	err error
	// loads counts the loads.
	loads int
}

func newResource() *resource {
	return &resource{obj: &testObject{Name: testProject, Version: "1"}}
}

func (r *resource) set(obj *testObject) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.obj = obj
}

func (r *resource) load(context.Context) (testObject, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.loads++
	if r.err != nil {
		return testObject{}, r.err
	}
	if r.obj == nil {
		return testObject{}, fmt.Errorf("resource %q: %w", testProject, database.ErrNotFound)
	}
	return *r.obj, nil
}

func version(o testObject) string { return o.Version }

// recordingSubscriber subscribes through a real connection and hands each
// subscription it makes to the test, so the test can check it is released.
type recordingSubscriber struct {
	natsrec.Subscriber
	subs chan *natsgo.Subscription
}

func (r *recordingSubscriber) Subscribe(
	subject string,
	cb natsgo.MsgHandler,
) (*natsgo.Subscription, error) {
	sub, err := r.Subscriber.Subscribe(subject, cb)
	if err == nil {
		r.subs <- sub
	}
	return sub, err
}

// failingSubscriber fails every subscription.
type failingSubscriber struct {
	natsrec.Subscriber
}

func (failingSubscriber) Subscribe(string, natsgo.MsgHandler) (*natsgo.Subscription, error) {
	return nil, errors.New("no NATS")
}

func connect(t *testing.T) *natsgo.Conn {
	t.Helper()
	srv := natstest.RunRandClientPortServer()
	t.Cleanup(srv.Shutdown)
	conn, err := natsgo.Connect(srv.ClientURL())
	require.NoError(t, err)
	t.Cleanup(conn.Close)
	return conn
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard
	os.Exit(m.Run())
}

// harness serves a watch of a resource over HTTP.
type harness struct {
	t   *testing.T
	url string
	// matched receives every event the watch asks the match function about.
	matched chan natsrec.Event[testKey, testRow]
	// done is closed when Serve returns.
	done chan struct{}
}

func newHarness(
	t *testing.T,
	conn natsrec.Subscriber,
	res *resource,
	keepalive time.Duration,
) *harness {
	h := &harness{
		t:       t,
		matched: make(chan natsrec.Event[testKey, testRow], 100),
		done:    make(chan struct{}, 1),
	}
	router := gin.New()
	// Answer errors the way the API server's middleware does for the ones
	// these tests produce.
	router.Use(func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 {
			return
		}
		if errors.Is(c.Errors.Last().Err, database.ErrNotFound) {
			c.Status(http.StatusNotFound)
			return
		}
		c.Status(http.StatusInternalServerError)
	})
	router.GET("/watch", func(c *gin.Context) {
		defer func() { h.done <- struct{}{} }()
		Serve(
			c,
			conn,
			testTopic,
			func(e natsrec.Event[testKey, testRow]) bool {
				h.matched <- e
				return e.Key.Project == testProject
			},
			res.load,
			version,
			keepalive,
		)
	})
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	h.url = srv.URL + "/watch"
	return h
}

// watchEvent is an event as it arrives on the wire.
type watchEvent struct {
	Type   string     `json:"type"`
	Object testObject `json:"object"`
}

// stream is an open watch.
type stream struct {
	t      *testing.T
	cancel context.CancelFunc
	events chan watchEvent
}

// open starts a watch and returns its stream once the response headers
// arrive.
func (h *harness) open() *stream {
	ctx, cancel := context.WithCancel(context.Background())
	h.t.Cleanup(cancel)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.url, nil)
	require.NoError(h.t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(h.t, err)
	h.t.Cleanup(func() { _ = resp.Body.Close() })
	require.Equal(h.t, http.StatusOK, resp.StatusCode)
	require.Equal(h.t, "text/event-stream", resp.Header.Get("Content-Type"))
	s := &stream{t: h.t, cancel: cancel, events: make(chan watchEvent, 100)}
	go func() {
		defer close(s.events)
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			data, ok := strings.CutPrefix(scanner.Text(), "data: ")
			if !ok {
				continue
			}
			var e watchEvent
			if json.Unmarshal([]byte(data), &e) == nil {
				s.events <- e
			}
		}
	}()
	return s
}

// next returns the next event, failing the test if none arrives in time or
// the stream ends.
func (s *stream) next() watchEvent {
	select {
	case e, ok := <-s.events:
		require.True(s.t, ok, "the stream ended")
		return e
	case <-time.After(waitFor):
		require.FailNow(s.t, "timed out waiting for an event")
	}
	return watchEvent{}
}

// requireEnded fails the test unless the stream ends in time without
// another event.
func (s *stream) requireEnded() {
	select {
	case e, ok := <-s.events:
		require.False(s.t, ok, "unexpected event %+v", e)
	case <-time.After(waitFor):
		require.FailNow(s.t, "timed out waiting for the stream to end")
	}
}

func (h *harness) requireServeReturned() {
	select {
	case <-h.done:
	case <-time.After(waitFor):
		require.FailNow(h.t, "timed out waiting for Serve to return")
	}
}

func TestServe(t *testing.T) {
	t.Parallel()

	t.Run("sends the object, then its state after each matching event", func(t *testing.T) {
		t.Parallel()
		conn := connect(t)
		res := newResource()
		h := newHarness(t, conn, res, noKeepalive)
		s := h.open()
		require.Equal(t, watchEvent{Type: "ADDED", Object: testObject{Name: testProject, Version: "1"}}, s.next())

		// An event about another resource is not about this one, so the
		// change made alongside it goes unseen.
		res.set(&testObject{Name: testProject, Version: "2"})
		require.NoError(t, testTopic.PublishCreated(conn, testKey{Project: otherProject}, testRow{}))
		select {
		case e := <-h.matched:
			require.Equal(t, otherProject, e.Key.Project)
		case <-time.After(waitFor):
			require.FailNow(t, "timed out waiting for the event to be matched")
		}
		require.Never(t, func() bool { return len(s.events) > 0 }, 200*time.Millisecond, 10*time.Millisecond)

		// An event about this resource sends it as it is now, not as the
		// event carries it.
		require.NoError(t, testTopic.PublishUpdated(
			conn,
			testKey{Project: testProject},
			testRow{Value: "a"},
			testRow{Value: "b"},
		))
		require.Equal(t, watchEvent{Type: "MODIFIED", Object: testObject{Name: testProject, Version: "2"}}, s.next())

		// So does the next one.
		res.set(&testObject{Name: testProject, Version: "3"})
		require.NoError(t, testTopic.PublishUpdated(
			conn,
			testKey{Project: testProject},
			testRow{Value: "b"},
			testRow{Value: "c"},
		))
		require.Equal(t, watchEvent{Type: "MODIFIED", Object: testObject{Name: testProject, Version: "3"}}, s.next())
	})

	t.Run("sends DELETED and ends when the resource is gone", func(t *testing.T) {
		t.Parallel()
		conn := connect(t)
		res := newResource()
		h := newHarness(t, conn, res, noKeepalive)
		s := h.open()
		require.Equal(t, "ADDED", s.next().Type)

		res.set(nil)
		require.NoError(t, testTopic.PublishDeleted(conn, testKey{Project: testProject}, testRow{}))
		// The resource is sent as it was last seen.
		require.Equal(t, watchEvent{Type: "DELETED", Object: testObject{Name: testProject, Version: "1"}}, s.next())
		s.requireEnded()
		h.requireServeReturned()
	})

	t.Run("a keepalive tick catches a change whose event was lost", func(t *testing.T) {
		t.Parallel()
		conn := connect(t)
		res := newResource()
		h := newHarness(t, conn, res, 50*time.Millisecond)
		s := h.open()
		require.Equal(t, "ADDED", s.next().Type)

		// No event is published for this change.
		res.set(&testObject{Name: testProject, Version: "2"})
		require.Equal(t, watchEvent{Type: "MODIFIED", Object: testObject{Name: testProject, Version: "2"}}, s.next())
	})

	t.Run("a keepalive tick catches a deletion whose event was lost", func(t *testing.T) {
		t.Parallel()
		conn := connect(t)
		res := newResource()
		h := newHarness(t, conn, res, 50*time.Millisecond)
		s := h.open()
		require.Equal(t, "ADDED", s.next().Type)

		res.set(nil)
		require.Equal(t, "DELETED", s.next().Type)
		s.requireEnded()
		h.requireServeReturned()
	})

	t.Run("a failed re-read keeps the watch open", func(t *testing.T) {
		t.Parallel()
		conn := connect(t)
		res := newResource()
		h := newHarness(t, conn, res, noKeepalive)
		s := h.open()
		require.Equal(t, "ADDED", s.next().Type)

		res.mu.Lock()
		res.err = errors.New("database unavailable")
		res.mu.Unlock()
		require.NoError(t, testTopic.PublishCreated(conn, testKey{Project: testProject}, testRow{}))
		require.Eventually(t, func() bool {
			res.mu.Lock()
			defer res.mu.Unlock()
			return res.loads == 2
		}, waitFor, 10*time.Millisecond)

		res.mu.Lock()
		res.err = nil
		res.obj = &testObject{Name: testProject, Version: "2"}
		res.mu.Unlock()
		require.NoError(t, testTopic.PublishCreated(conn, testKey{Project: testProject}, testRow{}))
		require.Equal(t, watchEvent{Type: "MODIFIED", Object: testObject{Name: testProject, Version: "2"}}, s.next())
	})

	t.Run("unsubscribes when the client goes away", func(t *testing.T) {
		t.Parallel()
		sub := &recordingSubscriber{Subscriber: connect(t), subs: make(chan *natsgo.Subscription, 1)}
		h := newHarness(t, sub, newResource(), noKeepalive)
		s := h.open()
		require.Equal(t, "ADDED", s.next().Type)
		natsSub := <-sub.subs
		require.True(t, natsSub.IsValid())

		s.cancel()
		h.requireServeReturned()
		require.False(t, natsSub.IsValid())
	})

	t.Run("a resource that does not exist fails before streaming", func(t *testing.T) {
		t.Parallel()
		sub := &recordingSubscriber{Subscriber: connect(t), subs: make(chan *natsgo.Subscription, 1)}
		res := newResource()
		res.set(nil)
		h := newHarness(t, sub, res, noKeepalive)
		status, contentType := h.get()
		require.Equal(t, http.StatusNotFound, status)
		require.NotEqual(t, "text/event-stream", contentType)
		h.requireServeReturned()
		require.False(t, (<-sub.subs).IsValid())
	})

	t.Run("a failed first read fails before streaming", func(t *testing.T) {
		t.Parallel()
		res := newResource()
		res.err = errors.New("database unavailable")
		sub := &recordingSubscriber{Subscriber: connect(t), subs: make(chan *natsgo.Subscription, 1)}
		h := newHarness(t, sub, res, noKeepalive)
		status, contentType := h.get()
		require.Equal(t, http.StatusInternalServerError, status)
		require.NotEqual(t, "text/event-stream", contentType)
		h.requireServeReturned()
		require.False(t, (<-sub.subs).IsValid())
	})

	t.Run("a failed subscription fails before streaming", func(t *testing.T) {
		t.Parallel()
		res := newResource()
		h := newHarness(t, failingSubscriber{}, res, noKeepalive)
		status, _ := h.get()
		require.Equal(t, http.StatusInternalServerError, status)
		h.requireServeReturned()
		require.Zero(t, res.loads)
	})
}

// get requests a watch and returns the response's status and content type,
// for tests that expect it to fail before streaming.
func (h *harness) get() (int, string) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, h.url, nil)
	require.NoError(h.t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(h.t, err)
	defer resp.Body.Close()
	return resp.StatusCode, resp.Header.Get("Content-Type")
}
