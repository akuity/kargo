package sse

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/watch"
)

func newContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c, w
}

func TestSetHeaders(t *testing.T) {
	t.Parallel()
	c, w := newContext()
	SetHeaders(c)
	require.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	require.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	require.Equal(t, "keep-alive", w.Header().Get("Connection"))
	require.Equal(t, "no", w.Header().Get("X-Accel-Buffering"))
	require.True(t, w.Flushed)
}

func TestSendWatchEvent(t *testing.T) {
	t.Parallel()

	t.Run("writes the event as an SSE data line", func(t *testing.T) {
		t.Parallel()
		c, w := newContext()
		require.True(t, SendWatchEvent(c, watch.Modified, map[string]string{"name": "x"}))
		require.Equal(t, "data: {\"type\":\"MODIFIED\",\"object\":{\"name\":\"x\"}}\n\n", w.Body.String())
		require.True(t, w.Flushed)
	})

	t.Run("an object that cannot be encoded is skipped", func(t *testing.T) {
		t.Parallel()
		c, w := newContext()
		require.True(t, SendWatchEvent(c, watch.Added, func() {}))
		require.Empty(t, w.Body.String())
	})
}

func TestWriteKeepalive(t *testing.T) {
	t.Parallel()
	c, w := newContext()
	require.True(t, WriteKeepalive(c))
	require.Equal(t, ": keepalive\n\n", w.Body.String())
	require.True(t, w.Flushed)
}
