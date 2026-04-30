package dex

import (
	"crypto/x509"
	"net/http/httputil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// This is self-signed and completely useless CA cert just for testing purposes.
var dummyCACertBytes = []byte(`-----BEGIN CERTIFICATE-----
MIIDvzCCAqcCFExIS2KGsSnWD7a8V0zmqhQD+XZ8MA0GCSqGSIb3DQEBCwUAMIGb
MQswCQYDVQQGEwJVUzEUMBIGA1UECAwLQ29ubmVjdGljdXQxEzARBgNVBAcMClBs
YWludmlsbGUxEjAQBgNVBAoMCUtyYW5jb3ZpYTEUMBIGA1UECwwLRW5naW5lZXJp
bmcxGDAWBgNVBAMMD2NhLmtyYW5jb3ZpYS5pbzEdMBsGCSqGSIb3DQEJARYOa2Vu
dEBha3VpdHkuaW8wHhcNMjMwNzMxMjEzMTM1WhcNMjQwNzMwMjEzMTM1WjCBmzEL
MAkGA1UEBhMCVVMxFDASBgNVBAgMC0Nvbm5lY3RpY3V0MRMwEQYDVQQHDApQbGFp
bnZpbGxlMRIwEAYDVQQKDAlLcmFuY292aWExFDASBgNVBAsMC0VuZ2luZWVyaW5n
MRgwFgYDVQQDDA9jYS5rcmFuY292aWEuaW8xHTAbBgkqhkiG9w0BCQEWDmtlbnRA
YWt1aXR5LmlvMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAwycyalcg
p7jSBkekhPakfJYYyu8/p5J+kY75Yj7Z+9ed7xTYy3bNJ09OkkUHGUyO39pK1oe/
dUgsxUC9N0Wqpo2t4+UHyc12rmX8Yi1v4G4mZj5XdV4fGh7CjqFwc3497eVqwLXJ
qDCDuvT2n5+zcgmt9f8+BUhZJh+lFPywLC62+sD74nT3oE6niREi95O3/SQT79SR
IeMWNXiZmoTETEX3Jhs1dhkVw/KhrjCXraMKK1Og9FnmLRR3JPYpl76za2MC7i9K
rzZfU7YW8Aj1sqZrLYuvxnVz4LiB1BaG0Aniz1gGfFDkaP/WvCYeDkyW19kmOyPC
LHF+4K4dAmXsQwIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQBSA3qk72RbsIjKvFGy
fwg1vpnq00y8ILRKdSYYA2+HifX9R4WyqaYSdo2S9qp+dU1iz4gFgokiut9C+kEc
zosRma12jmuMum8RfUEGUl/V9KHWjXKoJPbCKijql4InlDN5hFh32bigtgRcj9yE
1Ya4+nHHtLnUJOHLSRycBQ8BbK6o/fKz/RN4kDPBehWe7hlLmzdlSRfG6GT2tVUq
pqwF8ujOBXbmjfPqZK8rlFcGtfVotldmaFsnQuEVyO132MDyfHnyDrgqT3Ytsq8d
EZv4FqnG2KDTlXoV/Ku1ib5vzgQK5fTFfqO5dm5sLM4qQFmLadULaTcNOldyH3KG
c1e3
-----END CERTIFICATE-----`)

func TestNewProxy(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func() ProxyConfig
		assertions func(*testing.T, *httputil.ReverseProxy, error)
	}{
		{
			name: "no CA cert file specified",
			setup: func() ProxyConfig {
				return ProxyConfig{}
			},
			assertions: func(t *testing.T, proxy *httputil.ReverseProxy, err error) {
				require.NoError(t, err)
				require.NotNil(t, proxy)
			},
		},
		{
			name: "CA cert file does not exist",
			setup: func() ProxyConfig {
				return ProxyConfig{
					CACertPath: "/bogus/path",
				}
			},
			assertions: func(t *testing.T, _ *httputil.ReverseProxy, err error) {
				require.ErrorContains(t, err, "error reading CA cert file")
				require.ErrorContains(t, err, "/bogus/path")
			},
		},
		{
			name: "CA cert file has invalid contents",
			setup: func() ProxyConfig {
				cfg := ProxyConfig{
					CACertPath: filepath.Join(t.TempDir(), "ca.crt"),
				}
				err := os.WriteFile(cfg.CACertPath, []byte("invalid"), 0600)
				require.NoError(t, err)
				return cfg
			},
			assertions: func(t *testing.T, _ *httputil.ReverseProxy, err error) {
				require.ErrorContains(t, err, "error building CA cert pool")
			},
		},
		{
			name: "CA cert file has valid contents",
			setup: func() ProxyConfig {
				cfg := ProxyConfig{
					CACertPath: filepath.Join(t.TempDir(), "ca.crt"),
				}
				err := os.WriteFile(cfg.CACertPath, dummyCACertBytes, 0600)
				require.NoError(t, err)
				return cfg
			},
			assertions: func(t *testing.T, proxy *httputil.ReverseProxy, err error) {
				require.NoError(t, err)
				require.NotNil(t, proxy)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			proxy, err := NewProxy(testCase.setup())
			testCase.assertions(t, proxy, err)
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
