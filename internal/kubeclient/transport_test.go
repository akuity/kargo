package kubeclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_credentialHook(t *testing.T) {
	testSets := map[string]struct {
		credential string
		expected   string
	}{
		"empty authorization header": {
			expected: "",
		},
		"non-empty authorization header": {
			credential: "Bearer token",
			expected:   "Bearer token",
		},
	}
	for name, ts := range testSets {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				}))
			t.Cleanup(srv.Close)

			hc := http.Client{
				Transport: newAuthorizationHeaderHook(http.DefaultTransport),
			}
			req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
			require.NoError(t, err)
			if ts.credential != "" {
				req.Header.Set("Authorization", ts.credential)
			}
			res, err := hc.Do(req.WithContext(context.Background()))
			defer func() {
				_ = res.Body.Close()
			}()
			require.NoError(t, err)
			got := res.Header.Get(xKargoUserCredentialHeader)
			require.Equal(t, ts.credential, got)
		})
	}
}
