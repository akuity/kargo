package kubeclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	ctxutil "github.com/akuityio/kargo/internal/context"
)

func TestAuthRoundTripper(t *testing.T) {
	testSets := map[string]struct {
		newContext func() context.Context
		expected   string
	}{
		"context without auth credential": {
			newContext: context.Background,
		},
		"context with auth credential": {
			newContext: func() context.Context {
				return ctxutil.SetAuthCredential(context.Background(), "Bearer token")
			},
			expected: "Bearer token",
		},
	}
	for name, ts := range testSets {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Write([]byte(req.Header.Get("Authorization")))
			}))
			defer srv.Close()

			hc := http.Client{
				Transport: newAuthRoundTripper(http.DefaultTransport),
			}
			req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
			require.NoError(t, err)
			res, err := hc.Do(req.WithContext(ts.newContext()))
			require.NoError(t, err)
			data, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			require.Equal(t, ts.expected, string(data))
		})
	}
}
