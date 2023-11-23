package option

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/dex"
	libOIDC "github.com/akuity/kargo/internal/api/oidc"
	"github.com/akuity/kargo/internal/api/user"
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

func TestNewAuthInterceptor(t *testing.T) {
	a, err := newAuthInterceptor(context.Background(), config.ServerConfig{}, nil)
	require.NoError(t, err)
	require.NotNil(t, a)
	require.NotNil(t, a.parseUnverifiedJWTFn)
	require.NotNil(t, a.verifyKargoIssuedTokenFn)
	require.NotNil(t, a.verifyIDPIssuedTokenFn)
	require.NotNil(t, a.oidcExtractGroupsFn)
	require.NotNil(t, a.listServiceAccountsFn)
}

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
				mux.HandleFunc(discoPath, func(w http.ResponseWriter, r *http.Request) {
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
				mux.HandleFunc(
					dexDiscoPath,
					func(w http.ResponseWriter, r *http.Request) {
						_, err := w.Write([]byte(`{
						"issuer": "` + srv.URL + `",
						"jwks_uri": "` + srv.URL + `/keys"
					}`))
						require.NoError(t, err)
					},
				)
				return srv, config.ServerConfig{
					DexProxyConfig: &dex.ProxyConfig{
						ServerAddr: srv.URL,
					},
					OIDCConfig: &libOIDC.Config{
						IssuerURL: srv.URL,
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
				mux.HandleFunc(
					dexDiscoPath,
					func(w http.ResponseWriter, r *http.Request) {
						_, err := w.Write([]byte(`{
						"issuer": "` + srv.URL + `",
						"jwks_uri": "` + srv.URL + `/keys"
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
						IssuerURL: srv.URL,
					},
				}
				err :=
					os.WriteFile(cfg.DexProxyConfig.CACertPath, dummyCACertBytes, 0600)
				require.NoError(t, err)
				return srv, cfg
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			svr, cfg := testCase.setup()
			t.Cleanup(svr.Close)
			keyset, err := getKeySet(context.Background(), cfg)
			require.NoError(t, err)
			require.NotNil(t, keyset)
		})
	}
}

func TestAuthenticate(t *testing.T) {
	// The way the tests are structured, we don't need this to be valid. It just
	// needs to be non-empty.
	const (
		testProcedure   = "akuity.io.kargo.service.v1alpha1.KargoService/ListProjects"
		testIDPIssuer   = "fake-idp-issuer"
		testKargoIssuer = "fake-kargo-issuer"
		testToken       = "some-token"
	)
	testSets := map[string]struct {
		procedure       string
		authInterceptor *authInterceptor
		token           string
		assertions      func(ctx context.Context, err error)
	}{
		"exempt procedure": {
			procedure:       "/grpc.health.v1.Health/Check",
			authInterceptor: &authInterceptor{},
			// The procedure is exempt from authentication, so no user information
			// should be bound to the context.
			assertions: func(ctx context.Context, err error) {
				require.NoError(t, err)
				_, ok := user.InfoFromContext(ctx)
				require.False(t, ok)
			},
		},
		"no token provided": {
			procedure: testProcedure,
			// It's an error if no token is provided.
			assertions: func(ctx context.Context, err error) {
				require.Error(t, err)
				require.Equal(t, "no token provided", err.Error())
				_, ok := user.InfoFromContext(ctx)
				require.False(t, ok)
			},
		},
		"non-JWT token": {
			procedure: testProcedure,
			authInterceptor: &authInterceptor{
				parseUnverifiedJWTFn: func(
					string,
					jwt.Claims,
				) (*jwt.Token, []string, error) {
					return nil, nil, errors.New("this is not a JWT")
				},
			},
			token: testToken,
			// We can't parse the token as a JWT, so we assume it could be an opaque
			// bearer token for the k8s API server. We expect user info containing the
			// raw token to be bound to the context.
			assertions: func(ctx context.Context, err error) {
				require.NoError(t, err)
				u, ok := user.InfoFromContext(ctx)
				require.True(t, ok)
				require.Equal(t, testToken, u.BearerToken)
			},
		},
		"failure verifying Kargo-issued token": {
			procedure: testProcedure,
			authInterceptor: &authInterceptor{
				cfg: config.ServerConfig{
					AdminConfig: &config.AdminConfig{
						TokenIssuer: testKargoIssuer,
					},
				},
				parseUnverifiedJWTFn: func(_ string, claims jwt.Claims) (*jwt.Token, []string, error) {
					rc, ok := claims.(*jwt.RegisteredClaims)
					require.True(t, ok)
					rc.Issuer = testKargoIssuer
					return nil, nil, nil
				},
				verifyKargoIssuedTokenFn: func(rawToken string) bool {
					return false
				},
			},
			token: testToken,
			assertions: func(ctx context.Context, err error) {
				require.Error(t, err)
				require.Equal(
					t,
					"invalid token",
					err.Error(),
				)
				_, ok := user.InfoFromContext(ctx)
				require.False(t, ok)
			},
		},
		"success verifying Kargo-issued token": {
			procedure: testProcedure,
			authInterceptor: &authInterceptor{
				cfg: config.ServerConfig{
					AdminConfig: &config.AdminConfig{
						TokenIssuer: testKargoIssuer,
					},
				},
				parseUnverifiedJWTFn: func(_ string, claims jwt.Claims) (*jwt.Token, []string, error) {
					rc, ok := claims.(*jwt.RegisteredClaims)
					require.True(t, ok)
					rc.Issuer = testKargoIssuer
					return nil, nil, nil
				},
				verifyKargoIssuedTokenFn: func(rawToken string) bool {
					return true
				},
			},
			token: testToken,
			// If this is successful, we expect that user info for the admin user
			// is bound to the context.
			assertions: func(ctx context.Context, err error) {
				require.NoError(t, err)
				u, ok := user.InfoFromContext(ctx)
				require.True(t, ok)
				require.True(t, u.IsAdmin)
				require.Empty(t, u.Username)
				require.Empty(t, u.Groups)
				require.Empty(t, u.BearerToken)
			},
		},
		"failure verifying IDP-issued token": {
			procedure: testProcedure,
			authInterceptor: &authInterceptor{
				cfg: config.ServerConfig{
					OIDCConfig: &libOIDC.Config{
						IssuerURL: testIDPIssuer,
					},
				},
				parseUnverifiedJWTFn: func(_ string, claims jwt.Claims) (*jwt.Token, []string, error) {
					rc, ok := claims.(*jwt.RegisteredClaims)
					require.True(t, ok)
					rc.Issuer = testIDPIssuer
					return nil, nil, nil
				},
				verifyIDPIssuedTokenFn: func(
					context.Context,
					string,
				) (string, []string, bool) {
					return "", nil, false
				},
			},
			token: testToken,
			assertions: func(ctx context.Context, err error) {
				require.Error(t, err)
				require.Equal(
					t,
					"invalid token",
					err.Error(),
				)
				_, ok := user.InfoFromContext(ctx)
				require.False(t, ok)
			},
		},
		"success verifying IDP-issued token": {
			procedure: testProcedure,
			authInterceptor: &authInterceptor{
				cfg: config.ServerConfig{
					OIDCConfig: &libOIDC.Config{
						IssuerURL: testIDPIssuer,
					},
				},
				parseUnverifiedJWTFn: func(_ string, claims jwt.Claims) (*jwt.Token, []string, error) {
					rc, ok := claims.(*jwt.RegisteredClaims)
					require.True(t, ok)
					rc.Issuer = testIDPIssuer
					return nil, nil, nil
				},
				verifyIDPIssuedTokenFn: func(
					context.Context,
					string,
				) (string, []string, bool) {
					return "tony@starkindustries.com", []string{"avengers"}, true
				},
				listServiceAccountsFn: func(context.Context, string, []string) (map[string][]string, error) {
					return nil, nil
				},
			},
			token: testToken,
			// On success, we expect user info containing username and groups to be
			// bound to the context.
			assertions: func(ctx context.Context, err error) {
				require.NoError(t, err)
				u, ok := user.InfoFromContext(ctx)
				require.True(t, ok)
				require.False(t, u.IsAdmin)
				require.Equal(t, "tony@starkindustries.com", u.Username)
				require.Equal(t, []string{"avengers"}, u.Groups)
				require.Empty(t, u.BearerToken)
			},
		},
		"unrecognized JWT": {
			procedure: testProcedure,
			authInterceptor: &authInterceptor{
				parseUnverifiedJWTFn: func(_ string, claims jwt.Claims) (*jwt.Token, []string, error) {
					rc, ok := claims.(*jwt.RegisteredClaims)
					require.True(t, ok)
					rc.Issuer = "unrecognized-issuer"
					return nil, nil, nil
				},
			},
			token: testToken,
			// We can't verify this token, so we assume it could be an an identity
			// token from the k8s API server's identity provider. We expect user info
			// containing the raw token to be bound to the context.
			assertions: func(ctx context.Context, err error) {
				require.NoError(t, err)
				u, ok := user.InfoFromContext(ctx)
				require.True(t, ok)
				require.Equal(t, testToken, u.BearerToken)
			},
		},
	}
	for name, ts := range testSets {
		ts := ts
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			header := http.Header{}
			if ts.token != "" {
				header.Set("Authorization", ts.token)
			}
			ctx, err := ts.authInterceptor.authenticate(
				context.Background(),
				ts.procedure,
				header,
			)
			ts.assertions(ctx, err)
		})
	}
}

func TestVerifyIDPIssuedTokenFn(t *testing.T) {
	testCases := []struct {
		name            string
		authInterceptor *authInterceptor
		assertions      func(username string, groups []string, ok bool)
	}{
		{
			name:            "OIDC not supported",
			authInterceptor: &authInterceptor{},
			assertions: func(_ string, _ []string, ok bool) {
				require.False(t, ok)
			},
		},
		{
			name: "token cannot be verified",
			authInterceptor: &authInterceptor{
				oidcTokenVerifyFn: func(
					context.Context,
					string,
				) (*oidc.IDToken, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(_ string, _ []string, ok bool) {
				require.False(t, ok)
			},
		},
		{
			name: "error getting claims from token",
			authInterceptor: &authInterceptor{
				oidcTokenVerifyFn: func(
					context.Context,
					string,
				) (*oidc.IDToken, error) {
					return &oidc.IDToken{}, nil
				},
				oidcExtractGroupsFn: func(*oidc.IDToken) ([]string, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(_ string, _ []string, ok bool) {
				require.False(t, ok)
			},
		},
		{
			name: "token is successfully verified",
			authInterceptor: &authInterceptor{
				oidcTokenVerifyFn: func(
					context.Context,
					string,
				) (*oidc.IDToken, error) {
					return &oidc.IDToken{
						Subject: "tony@starkindustries.io",
					}, nil
				},
				oidcExtractGroupsFn: func(*oidc.IDToken) ([]string, error) {
					return []string{"avengers"}, nil
				},
			},
			assertions: func(username string, groups []string, ok bool) {
				require.True(t, ok)
				require.Equal(t, "tony@starkindustries.io", username)
				require.Equal(t, []string{"avengers"}, groups)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.authInterceptor.verifyIDPIssuedToken(
					context.Background(),
					// With the way these tests are constructed, this doesn't have to
					// be valid.
					"some-token",
				),
			)
		})
	}
}

func TestVerifyKargoIssuedToken(t *testing.T) {
	const testNonJWTToken = "some-token"
	testTokenSigningKey := []byte("iwishtowashmyirishwristwatch")
	testCases := []struct {
		name            string
		tokenFn         func() string // Returns a raw token
		authInterceptor *authInterceptor
		valid           bool
	}{
		{
			name:            "admin user not supported",
			authInterceptor: &authInterceptor{},
			tokenFn: func() string {
				return testNonJWTToken
			},
			valid: false,
		},
		{
			name: "token is not a JWT",
			authInterceptor: &authInterceptor{
				cfg: config.ServerConfig{
					AdminConfig: &config.AdminConfig{
						TokenSigningKey: testTokenSigningKey,
					},
				},
			},
			tokenFn: func() string {
				return testNonJWTToken
			},
			valid: false,
		},
		{
			name: "token was not issued by Kargo",
			authInterceptor: &authInterceptor{
				cfg: config.ServerConfig{
					AdminConfig: &config.AdminConfig{
						TokenSigningKey: testTokenSigningKey,
					},
				},
			},
			tokenFn: func() string {
				token, err := jwt.NewWithClaims(
					jwt.SigningMethodHS256,
					jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
					},
				).SignedString([]byte("wrong key")) // Not testTokenSigningKey
				require.NoError(t, err)
				return token
			},
			valid: false,
		},
		{
			name: "token was issued by Kargo, but is expired",
			authInterceptor: &authInterceptor{
				cfg: config.ServerConfig{
					AdminConfig: &config.AdminConfig{
						TokenSigningKey: testTokenSigningKey,
					},
				},
			},
			tokenFn: func() string {
				token, err := jwt.NewWithClaims(
					jwt.SigningMethodHS256,
					jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
					},
				).SignedString(testTokenSigningKey)
				require.NoError(t, err)
				return token
			},
			valid: false,
		},
		{
			name: "success",
			authInterceptor: &authInterceptor{
				cfg: config.ServerConfig{
					AdminConfig: &config.AdminConfig{
						TokenSigningKey: testTokenSigningKey,
					},
				},
			},
			tokenFn: func() string {
				token, err := jwt.NewWithClaims(
					jwt.SigningMethodHS256,
					jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
					},
				).SignedString(testTokenSigningKey)
				require.NoError(t, err)
				return token
			},
			valid: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.valid,
				testCase.authInterceptor.verifyKargoIssuedToken(testCase.tokenFn()),
			)
		})
	}
}
