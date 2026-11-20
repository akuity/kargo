package dex

import (
	"crypto/x509"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/server/dex/dextest"
)

func TestNewTransport(t *testing.T) {
	testCases := []struct {
		name   string
		setup  func(*testing.T) ProxyConfig
		assert func(*testing.T, http.RoundTripper, error)
	}{
		{
			name: "no CA cert file specified",
			setup: func(*testing.T) ProxyConfig {
				return ProxyConfig{}
			},
			assert: func(t *testing.T, rt http.RoundTripper, err error) {
				require.NoError(t, err)
				require.NotNil(t, rt)
			},
		},
		{
			name: "CA cert file does not exist",
			setup: func(*testing.T) ProxyConfig {
				return ProxyConfig{CACertPath: "/bogus/path"}
			},
			assert: func(t *testing.T, _ http.RoundTripper, err error) {
				require.ErrorContains(t, err, "error reading CA cert file")
				require.ErrorContains(t, err, "/bogus/path")
			},
		},
		{
			name: "CA cert file has invalid contents",
			setup: func(t *testing.T) ProxyConfig {
				path := filepath.Join(t.TempDir(), "ca.crt")
				require.NoError(t, os.WriteFile(path, []byte("invalid"), 0o600))
				return ProxyConfig{CACertPath: path}
			},
			assert: func(t *testing.T, _ http.RoundTripper, err error) {
				require.ErrorContains(t, err, "invalid CA cert data")
			},
		},
		{
			name: "CA cert file has valid contents",
			setup: func(t *testing.T) ProxyConfig {
				path := filepath.Join(t.TempDir(), "ca.crt")
				require.NoError(t, os.WriteFile(path, dummyCACertBytes, 0o600))
				return ProxyConfig{CACertPath: path}
			},
			assert: func(t *testing.T, rt http.RoundTripper, err error) {
				require.NoError(t, err)
				require.NotNil(t, rt)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			rt, err := NewTransport(testCase.setup(t))
			testCase.assert(t, rt, err)
		})
	}
}

func TestTransportCACertReload(t *testing.T) {
	testCases := []struct {
		name     string
		interval time.Duration
		// run receives a server whose current certificate is already trusted
		// via the CA cert file at caCertPath, and a transport built from it.
		run func(
			t *testing.T,
			srv *dextest.RotatingTLSServer,
			caCertPath string,
			rt http.RoundTripper,
		)
	}{
		{
			name: "trusts the server's new cert once the CA cert file is updated",
			run: func(
				t *testing.T,
				srv *dextest.RotatingTLSServer,
				caCertPath string,
				rt http.RoundTripper,
			) {
				newCertPEM := srv.Rotate(t)
				requireUnknownAuthority(t, get(t, rt, srv.URL))
				dextest.ReplaceFile(t, caCertPath, newCertPEM)
				require.NoError(t, get(t, rt, srv.URL))
			},
		},
		{
			name:     "does not reread the CA cert file before the interval elapses",
			interval: time.Hour,
			run: func(
				t *testing.T,
				srv *dextest.RotatingTLSServer,
				caCertPath string,
				rt http.RoundTripper,
			) {
				dextest.ReplaceFile(t, caCertPath, srv.Rotate(t))
				requireUnknownAuthority(t, get(t, rt, srv.URL))
			},
		},
		{
			name: "keeps the current CA cert when the file becomes invalid",
			run: func(
				t *testing.T,
				srv *dextest.RotatingTLSServer,
				caCertPath string,
				rt http.RoundTripper,
			) {
				dextest.ReplaceFile(t, caCertPath, []byte("invalid"))
				require.NoError(t, get(t, rt, srv.URL))
			},
		},
		{
			name: "keeps the current CA cert when the file disappears",
			run: func(
				t *testing.T,
				srv *dextest.RotatingTLSServer,
				caCertPath string,
				rt http.RoundTripper,
			) {
				require.NoError(t, os.Remove(caCertPath))
				require.NoError(t, get(t, rt, srv.URL))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			srv, certPEM := dextest.NewRotatingTLSServer(t, okHandler())
			caCertPath := filepath.Join(t.TempDir(), "ca.crt")
			dextest.ReplaceFile(t, caCertPath, certPEM)
			rt, err := NewTransport(ProxyConfig{
				CACertPath:           caCertPath,
				CACertReloadInterval: testCase.interval,
			})
			require.NoError(t, err)
			require.NoError(t, get(t, rt, srv.URL))
			testCase.run(t, srv, caCertPath, rt)
		})
	}
}

func TestBuildCACertPool(t *testing.T) {
	testCases := []struct {
		name        string
		caCertBytes []byte
		assertions  func(*testing.T, *x509.CertPool, error)
	}{
		{
			name:        "invalid cert bytes provided",
			caCertBytes: []byte("junk"),
			assertions: func(t *testing.T, _ *x509.CertPool, err error) {
				require.Error(t, err)
				require.Equal(t, "invalid CA cert data", err.Error())
			},
		},
		{
			name:        "valid cert bytes provided",
			caCertBytes: dummyCACertBytes,
			assertions: func(t *testing.T, caCertPool *x509.CertPool, err error) {
				require.NoError(t, err)
				require.NotNil(t, caCertPool)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			caCertPool, err := buildCACertPool(testCase.caCertBytes)
			testCase.assertions(t, caCertPool, err)
		})
	}
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

// get makes a GET request to url using rt and returns any transport error.
func get(t *testing.T, rt http.RoundTripper, url string) error {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	require.NoError(t, err)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	return nil
}

func requireUnknownAuthority(t *testing.T, err error) {
	t.Helper()
	var unknownAuthorityErr x509.UnknownAuthorityError
	require.ErrorAs(t, err, &unknownAuthorityErr)
}
