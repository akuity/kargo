package projectconfigs

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	natstest "github.com/nats-io/nats-server/v2/test"
	natsgo "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database/projectconfigstore"
	"github.com/akuity/kargo/pkg/server/auth/can"
)

func TestGet(t *testing.T) {
	t.Parallel()
	newStore := func() *fakeStore {
		store := &fakeStore{}
		store.add(kargoapi.ProjectConfigSpec{
			FreightLinks: []kargoapi.DeepLink{{Title: "t", URL: "https://example.com"}},
		})
		return store
	}
	forbiddenStore := newStore()
	var authorizations []can.Access
	conn := connectNATS(t)

	runTestCases(t, []testCase{
		{
			name:      "not authorized",
			store:     forbiddenStore,
			authorize: forbidEverything,
			method:    http.MethodGet,
			url:       configURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Zero(t, forbiddenStore.calls)
			},
		},
		{
			name:   "ProjectConfig does not exist",
			method: http.MethodGet,
			url:    configURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			name:   "a leftover of a deleted Project of the same name is not found",
			store:  leftoverStore(),
			method: http.MethodGet,
			url:    configURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			name:   "store fails",
			store:  &fakeStore{err: errors.New("boom")},
			method: http.MethodGet,
			url:    configURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
		{
			name:      "not authorized to watch",
			store:     forbiddenStore,
			authorize: forbidEverything,
			conn:      conn,
			method:    http.MethodGet,
			url:       configURL("?watch=true"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Zero(t, forbiddenStore.calls)
			},
		},
		{
			name:   "watching a ProjectConfig that does not exist fails before streaming",
			conn:   conn,
			method: http.MethodGet,
			url:    configURL("?watch=true"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
				require.NotEqual(t, "text/event-stream", w.Header().Get("Content-Type"))
			},
		},
		{
			name:   "watching a leftover of a deleted Project of the same name fails before streaming",
			store:  leftoverStore(),
			conn:   conn,
			method: http.MethodGet,
			url:    configURL("?watch=true"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
				require.NotEqual(t, "text/event-stream", w.Header().Get("Content-Type"))
			},
		},
		{
			name:      "returns the ProjectConfig with its type",
			store:     newStore(),
			authorize: recordAuthorizations(&authorizations),
			method:    http.MethodGet,
			url:       configURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				require.Equal(t, []can.Access{can.Get().ProjectConfig(testProject)}, authorizations)
				var raw map[string]any
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &raw))
				require.Equal(t, "kargo.akuity.io/v1alpha1", raw["apiVersion"])
				require.Equal(t, "ProjectConfig", raw["kind"])
				cfg := &kargoapi.ProjectConfig{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), cfg))
				require.Equal(t, testProject, cfg.Name)
				require.Equal(t, testProject, cfg.Namespace)
				require.Len(t, cfg.Spec.FreightLinks, 1)
			},
		},
	})
}

func TestGet_watch(t *testing.T) {
	t.Parallel()
	conn := connectNATS(t)
	store := &fakeStore{}
	created := store.add(kargoapi.ProjectConfigSpec{
		FreightLinks: []kargoapi.DeepLink{{Title: "first", URL: "https://example.com"}},
	})
	var authorizations []can.Access
	srv := httptest.NewServer(newRouter(t, store, recordAuthorizations(&authorizations), conn))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+configURL("?watch=true"), nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
	require.Equal(t, []can.Access{can.Get().ProjectConfig(testProject)}, authorizations)
	events := readWatchEvents(resp.Body)

	// The ProjectConfig is sent first, as a GET serves it.
	e := nextWatchEvent(t, events)
	require.Equal(t, "ADDED", e.Type)
	require.Equal(t, testProject, e.Object.Name)
	require.Equal(t, "ProjectConfig", e.Object.Kind)
	require.Equal(t, "kargo.akuity.io/v1alpha1", e.Object.APIVersion)
	require.Equal(t, "first", e.Object.Spec.FreightLinks[0].Title)

	// A change announced on the topic is sent as the store now holds it.
	updated := store.change(func(cfg *projectconfigstore.Config) {
		cfg.Spec = specJSON(kargoapi.ProjectConfigSpec{
			FreightLinks: []kargoapi.DeepLink{{Title: "second", URL: "https://example.com"}},
		})
	})
	require.NoError(t, projectconfigstore.Events.PublishUpdated(
		conn,
		projectconfigstore.Key{Project: testProject},
		created.ProjectConfigRow,
		updated.ProjectConfigRow,
	))
	e = nextWatchEvent(t, events)
	require.Equal(t, "MODIFIED", e.Type)
	require.Equal(t, "ProjectConfig", e.Object.Kind)
	require.Equal(t, "second", e.Object.Spec.FreightLinks[0].Title)
	require.Equal(t, projectconfigstore.ToAPI(updated).ResourceVersion, e.Object.ResourceVersion)

	// A deletion ends the watch.
	require.NoError(t, store.Delete(ctx, testProject, testProjectUID))
	require.NoError(t, projectconfigstore.Events.PublishDeleted(
		conn,
		projectconfigstore.Key{Project: testProject},
		updated.ProjectConfigRow,
	))
	e = nextWatchEvent(t, events)
	require.Equal(t, "DELETED", e.Type)
	require.Equal(t, testProject, e.Object.Name)
	require.Equal(t, "ProjectConfig", e.Object.Kind)
	select {
	case e, ok := <-events:
		require.False(t, ok, "unexpected event %+v", e)
	case <-time.After(10 * time.Second):
		require.FailNow(t, "timed out waiting for the watch to end")
	}
}

// change applies fn to the test Project's ProjectConfig as a write through
// the store would, and returns the result.
func (s *fakeStore) change(fn func(*projectconfigstore.Config)) projectconfigstore.Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg := s.configs[testProject]
	fn(&cfg)
	cfg.UpdatedAt = s.tick()
	s.configs[testProject] = cfg
	return cfg
}

func connectNATS(t *testing.T) *natsgo.Conn {
	t.Helper()
	srv := natstest.RunRandClientPortServer()
	t.Cleanup(srv.Shutdown)
	conn, err := natsgo.Connect(srv.ClientURL())
	require.NoError(t, err)
	t.Cleanup(conn.Close)
	return conn
}

type watchEvent struct {
	Type   string    `json:"type"`
	Object *Resource `json:"object"`
}

// readWatchEvents decodes the events of an SSE stream until it ends.
func readWatchEvents(body io.Reader) <-chan watchEvent {
	events := make(chan watchEvent, 100)
	go func() {
		defer close(events)
		scanner := bufio.NewScanner(body)
		for scanner.Scan() {
			data, ok := strings.CutPrefix(scanner.Text(), "data: ")
			if !ok {
				continue
			}
			var e watchEvent
			if json.Unmarshal([]byte(data), &e) == nil {
				events <- e
			}
		}
	}()
	return events
}

func nextWatchEvent(t *testing.T, events <-chan watchEvent) watchEvent {
	t.Helper()
	select {
	case e, ok := <-events:
		require.True(t, ok, "the watch ended")
		return e
	case <-time.After(10 * time.Second):
		require.FailNow(t, "timed out waiting for an event")
	}
	return watchEvent{}
}
