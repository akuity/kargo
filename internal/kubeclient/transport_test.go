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
	for name, testSet := range testSets {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					_, err := w.Write([]byte(req.Header.Get("Authorization")))
					require.NoError(t, err)
				}),
			)
			defer srv.Close()

			hc := http.Client{
				Transport: newAuthorizationHeaderHook(http.DefaultTransport),
			}
			req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
			if testSet.credential != "" {
				req.Header.Set("Authorization", testSet.credential)
			}
			require.NoError(t, err)
			res, err := hc.Do(req.WithContext(context.TODO()))
			defer func() {
				_ = res.Body.Close()
			}()
			require.NoError(t, err)
			defer res.Body.Close()
			data, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			require.Equal(t, ts.expected, string(data))
		})
	}
}
