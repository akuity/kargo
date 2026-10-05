package authn

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/dex"
	libOIDC "github.com/akuity/kargo/pkg/server/oidc"
	"github.com/akuity/kargo/pkg/server/user"
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

func TestGetKeySet(t *testing.T) {
	const discoPath = "/.well-known/openid-configuration"
	const dexDiscoPath = "/dex/.well-known/openid-configuration"
	testCases := []struct {
		name  string
		setup func() (*httptest.Server, config.ServerConfig)
	}{
		{
			name: "basic case",
			setup: func() (*httptest.Server, config.ServerConfig) {
				mux := http.NewServeMux()
				srv := httptest.NewServer(mux)
				t.Cleanup(srv.Close)
				mux.HandleFunc(discoPath, func(w http.ResponseWriter, _ *http.Request) {
					_, err := w.Write([]byte(`{
						"issuer": "` + srv.URL + `",
						"jwks_uri": "` + srv.URL + `/keys"
					}`))
					require.NoError(t, err)
				})
				return srv, config.ServerConfig{
					OIDCConfig: &libOIDC.Config{
						IssuerURL: srv.URL,
					},
				}
			},
		},
		{
			name: "with Dex proxy",
			setup: func() (*httptest.Server, config.ServerConfig) {
				mux := http.NewServeMux()
				srv := httptest.NewServer(mux)
				t.Cleanup(srv.Close)
				issuerURL := srv.URL + "/dex"
				mux.HandleFunc(
					dexDiscoPath,
					func(w http.ResponseWriter, _ *http.Request) {
						_, err := w.Write([]byte(`{
						"issuer": "` + issuerURL + `",
						"jwks_uri": "` + issuerURL + `/keys"
					}`))
						require.NoError(t, err)
					},
				)
				return srv, config.ServerConfig{
					DexProxyConfig: &dex.ProxyConfig{
						ServerAddr: srv.URL,
					},
					OIDCConfig: &libOIDC.Config{
						IssuerURL: issuerURL,
					},
				}
			},
		},
		{
			name: "with Dex proxy under basePath",
			setup: func() (*httptest.Server, config.ServerConfig) {
				mux := http.NewServeMux()
				srv := httptest.NewServer(mux)
				t.Cleanup(srv.Close)
				issuerURL := srv.URL + "/kargo/dex"
				mux.HandleFunc(
					"/kargo/dex/.well-known/openid-configuration",
					func(w http.ResponseWriter, _ *http.Request) {
						_, err := w.Write([]byte(`{
						"issuer": "` + issuerURL + `",
						"jwks_uri": "` + issuerURL + `/keys"
					}`))
						require.NoError(t, err)
					},
				)
				return srv, config.ServerConfig{
					DexProxyConfig: &dex.ProxyConfig{
						ServerAddr: srv.URL,
					},
					OIDCConfig: &libOIDC.Config{
						IssuerURL: issuerURL,
					},
				}
			},
		},
		{
			name: "with Dex proxy and CA cert",
			setup: func() (*httptest.Server, config.ServerConfig) {
				mux := http.NewServeMux()
				srv := httptest.NewServer(mux)
				t.Cleanup(srv.Close)
				issuerURL := srv.URL + "/dex"
				mux.HandleFunc(
					dexDiscoPath,
					func(w http.ResponseWriter, _ *http.Request) {
						_, err := w.Write([]byte(`{
						"issuer": "` + issuerURL + `",
						"jwks_uri": "` + issuerURL + `/keys"
					}`))
						require.NoError(t, err)
					},
				)
				cfg := config.ServerConfig{
					DexProxyConfig: &dex.ProxyConfig{
						ServerAddr: srv.URL,
						CACertPath: filepath.Join(t.TempDir(), "ca.crt"),
					},
					OIDCConfig: &libOIDC.Config{
						IssuerURL: issuerURL,
					},
				}
				err := os.WriteFile(cfg.DexProxyConfig.CACertPath, dummyCACertBytes, 0o600)
				require.NoError(t, err)
				return srv, cfg
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			svr, cfg := testCase.setup()
			t.Cleanup(svr.Close)
			keyset, err := getKeySet(t.Context(), cfg)
			require.NoError(t, err)
			require.NotNil(t, keyset)
		})
	}
}

func TestOIDC_Authenticate(t *testing.T) {
	t.Parallel()
	const issuer = "https://idp.example.com"
	cfg := config.ServerConfig{OIDCConfig: &libOIDC.Config{
		IssuerURL:                      issuer,
		UsernameClaim:                  "preferred_username",
		GlobalServiceAccountNamespaces: []string{"kargo"},
	}}
	verified := func(context.Context, string) (*oidc.IDToken, error) {
		return &oidc.IDToken{Subject: "ironman"}, nil
	}
	fullClaims := Claims{
		"preferred_username": "foo",
		"sub":                "ironman",
		"email":              "tony@starkindustries.com",
		"groups":             []string{"avengers", "shield"},
	}
	noAccounts := func(context.Context, Claims) (map[string]map[types.NamespacedName]struct{}, error) {
		return nil, nil
	}
	testCases := []struct {
		name          string
		authenticator *oidcAuthenticator
		token         string
		assert        func(*testing.T, user.Identity, bool, error)
	}{
		{
			name:          "another issuer is not this kind of token",
			authenticator: &oidcAuthenticator{cfg: cfg},
			token:         tokenFrom(t, "someone-else"),
			assert: func(t *testing.T, _ user.Identity, ok bool, err error) {
				require.NoError(t, err)
				require.False(t, ok)
			},
		},
		{
			// The identity provider's keys could not be fetched at startup and
			// still cannot be.
			name:          "verifier unavailable",
			authenticator: &oidcAuthenticator{cfg: cfg},
			token:         tokenFrom(t, issuer),
			assert: func(t *testing.T, _ user.Identity, ok bool, err error) {
				require.True(t, ok)
				require.ErrorContains(t, err, "could not validate token")
				require.NotErrorIs(t, err, ErrInvalidToken)
			},
		},
		{
			name: "token cannot be verified",
			authenticator: &oidcAuthenticator{
				cfg: cfg,
				verify: func(context.Context, string) (*oidc.IDToken, error) {
					return nil, errors.New("bad signature")
				},
			},
			token: tokenFrom(t, issuer),
			assert: func(t *testing.T, _ user.Identity, ok bool, err error) {
				require.True(t, ok)
				require.ErrorIs(t, err, ErrInvalidToken)
				require.ErrorContains(t, err, "bad signature")
			},
		},
		{
			name: "claims cannot be read",
			authenticator: &oidcAuthenticator{
				cfg:    cfg,
				verify: verified,
				extractClaimsFn: func(*oidc.IDToken) (Claims, error) {
					return nil, errors.New("something went wrong")
				},
			},
			token: tokenFrom(t, issuer),
			assert: func(t *testing.T, _ user.Identity, ok bool, err error) {
				require.True(t, ok)
				require.ErrorContains(t, err, "something went wrong")
				require.NotErrorIs(t, err, ErrInvalidToken)
			},
		},
		{
			// The API server's own request was refused. The refusal's 403 must
			// not reach the client as if it described their request.
			name: "ServiceAccounts cannot be listed",
			authenticator: &oidcAuthenticator{
				cfg:             cfg,
				verify:          verified,
				extractClaimsFn: func(*oidc.IDToken) (Claims, error) { return fullClaims, nil },
				listServiceAccountsFn: func(context.Context, Claims) (map[string]map[types.NamespacedName]struct{}, error) {
					return nil, apierrors.NewForbidden(
						schema.GroupResource{Resource: "serviceaccounts"},
						"",
						errors.New("not permitted"),
					)
				},
			},
			token: tokenFrom(t, issuer),
			assert: func(t *testing.T, _ user.Identity, ok bool, err error) {
				require.True(t, ok)
				require.ErrorContains(t, err, "list service accounts for user")
				require.NotErrorIs(t, err, ErrInvalidToken)
				requireErrorStatus(t, err, http.StatusInternalServerError)
			},
		},
		{
			name: "username claim is not a string",
			authenticator: &oidcAuthenticator{
				cfg:    cfg,
				verify: verified,
				extractClaimsFn: func(*oidc.IDToken) (Claims, error) {
					return Claims{"preferred_username": 42}, nil
				},
				listServiceAccountsFn: noAccounts,
			},
			token: tokenFrom(t, issuer),
			assert: func(t *testing.T, _ user.Identity, ok bool, err error) {
				require.True(t, ok)
				require.ErrorContains(t, err, `claim "preferred_username" must be a string`)
			},
		},
		{
			name: "token verified",
			authenticator: &oidcAuthenticator{
				cfg:             cfg,
				verify:          verified,
				extractClaimsFn: func(*oidc.IDToken) (Claims, error) { return fullClaims, nil },
				listServiceAccountsFn: func(context.Context, Claims) (map[string]map[types.NamespacedName]struct{}, error) {
					return map[string]map[types.NamespacedName]struct{}{
						"demo": {{Namespace: "demo", Name: "viewer"}: {}},
					}, nil
				},
			},
			token: tokenFrom(t, issuer),
			assert: func(t *testing.T, id user.Identity, ok bool, err error) {
				require.NoError(t, err)
				require.True(t, ok)
				require.Equal(t, user.OIDCUser{
					Claims:        fullClaims,
					UsernameClaim: "preferred_username",
					Username:      "foo",
					ServiceAccountsByNamespace: map[string]map[types.NamespacedName]struct{}{
						"demo": {{Namespace: "demo", Name: "viewer"}: {}},
					},
					GlobalServiceAccountNamespaces: []string{"kargo"},
				}, id)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			id, ok, err := testCase.authenticator.Authenticate(t.Context(), testCase.token)
			testCase.assert(t, id, ok, err)
		})
	}
}
