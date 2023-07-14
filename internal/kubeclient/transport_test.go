package kubeclient

import (
	"context"
	"io"
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
		ts := ts
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

func TestCredentialInjector(t *testing.T) {
	testSets := map[string]struct {
		ctx          context.Context
		baseCred     string
		expectedCred string
	}{
		"no credential": {
			ctx: context.Background(),
		},
		"context without credential": {
			ctx:          context.Background(),
			baseCred:     "base-token",
			expectedCred: "base-token",
		},
		"override base credential": {
			ctx:          SetCredentialToContext(context.Background(), "user-token"),
			baseCred:     "base-token",
			expectedCred: "user-token",
		},
	}
	for name, ts := range testSets {
		ts := ts
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					_, _ = w.Write([]byte(req.Header.Get("Authorization")))
				}))
			t.Cleanup(srv.Close)

			hc := http.Client{
				Transport: NewCredentialInjector(http.DefaultTransport),
			}
			req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
			require.NoError(t, err)
			if ts.baseCred != "" {
				req.Header.Set("Authorization", ts.baseCred)
			}
			res, err := hc.Do(req.WithContext(ts.ctx))
			require.NoError(t, err)
			defer func() {
				_ = res.Body.Close()
			}()
			rawCred, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			require.Equal(t, ts.expectedCred, string(rawCred))
		})
	}
}
