package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type polledItem struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

var polledItems = polledWatch[polledItem]{
	name:    func(item polledItem) string { return item.Name },
	version: func(item polledItem) string { return item.Version },
}

func newPolledWatchContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder, context.CancelFunc) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	ctx, cancel := context.WithCancel(t.Context())
	c.Request = httptest.NewRequest(http.MethodGet, "/watch", nil).WithContext(ctx)
	return c, w, cancel
}

func TestSendPolledChanges(t *testing.T) {
	t.Parallel()
	c, w, cancel := newPolledWatchContext(t)
	defer cancel()
	known := map[string]polledItem{
		"a": {Name: "a", Version: "1"},
		"b": {Name: "b", Version: "1"},
		"d": {Name: "d", Version: "1"},
	}
	items := []polledItem{
		{Name: "c", Version: "1"}, // new
		{Name: "a", Version: "2"}, // changed
		{Name: "d", Version: "1"}, // unchanged
	}
	known, ok := sendPolledChanges(c, polledItems, known, items)
	require.True(t, ok)
	require.Equal(
		t,
		[]string{"ADDED c", "MODIFIED a", "DELETED b"},
		sseEvents(t, w.Body.String(), polledItems.name),
	)
	require.Equal(t, map[string]polledItem{
		"a": {Name: "a", Version: "2"},
		"c": {Name: "c", Version: "1"},
		"d": {Name: "d", Version: "1"},
	}, known)
}

func TestServePolledWatch(t *testing.T) {
	t.Parallel()

	t.Run("snapshot failure before streaming is an ordinary error", func(t *testing.T) {
		t.Parallel()
		c, w, cancel := newPolledWatchContext(t)
		defer cancel()
		source := polledItems
		source.snapshot = func(context.Context) ([]polledItem, error) { return nil, errors.New("offline") }
		servePolledWatch(c, time.Millisecond, "", source)
		require.Len(t, c.Errors, 1)
		require.ErrorContains(t, c.Errors[0], "offline")
		require.Empty(t, w.Header().Get("Content-Type"))
	})

	t.Run("unseeded watch replays everything as added", func(t *testing.T) {
		t.Parallel()
		c, w, cancel := newPolledWatchContext(t)
		cancel() // Stop right after the replay.
		source := polledItems
		source.snapshot = func(context.Context) ([]polledItem, error) {
			return []polledItem{{Name: "a", Version: "5"}, {Name: "b", Version: "7"}}, nil
		}
		servePolledWatch(c, time.Millisecond, "", source)
		require.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
		require.Equal(t, []string{"ADDED a", "ADDED b"}, sseEvents(t, w.Body.String(), polledItems.name))
	})

	t.Run("seeded watch sends only what is newer, as modified", func(t *testing.T) {
		t.Parallel()
		c, w, cancel := newPolledWatchContext(t)
		cancel()
		source := polledItems
		source.snapshot = func(context.Context) ([]polledItem, error) {
			return []polledItem{{Name: "a", Version: "5"}, {Name: "b", Version: "7"}}, nil
		}
		servePolledWatch(c, time.Millisecond, "5", source)
		require.Equal(t, []string{"MODIFIED b"}, sseEvents(t, w.Body.String(), polledItems.name))
	})

	t.Run("polls for changes and reports a failed poll as a watch error", func(t *testing.T) {
		t.Parallel()
		c, w, cancel := newPolledWatchContext(t)
		defer cancel()
		var polls atomic.Int32
		source := polledItems
		source.snapshot = func(context.Context) ([]polledItem, error) {
			switch polls.Add(1) {
			case 1:
				return []polledItem{{Name: "a", Version: "1"}}, nil
			case 2:
				return []polledItem{{Name: "a", Version: "2"}, {Name: "b", Version: "1"}}, nil
			case 3:
				return []polledItem{{Name: "b", Version: "1"}}, nil
			default:
				return nil, errors.New("offline")
			}
		}
		servePolledWatch(c, time.Millisecond, "", source)
		require.Equal(
			t,
			[]string{"ADDED a", "MODIFIED a", "ADDED b", "DELETED a"},
			sseEvents(t, w.Body.String(), polledItems.name),
		)
		require.Contains(t, w.Body.String(), "event: error")
		require.Contains(t, w.Body.String(), "offline")
	})

	t.Run("stops when the client goes away", func(t *testing.T) {
		t.Parallel()
		c, w, cancel := newPolledWatchContext(t)
		source := polledItems
		source.snapshot = func(context.Context) ([]polledItem, error) { return nil, nil }
		done := make(chan struct{})
		go func() {
			servePolledWatch(c, time.Millisecond, "", source)
			close(done)
		}()
		time.Sleep(10 * time.Millisecond)
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("watch did not stop")
		}
		require.False(t, strings.Contains(w.Body.String(), "data:"))
	})
}
