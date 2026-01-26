package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

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

func TestNewAuthMiddleware(t *testing.T) {
	a := &authMiddleware{}
	middleware := newAuthMiddleware(context.Background(), config.ServerConfig{}, nil)
	require.NotNil(t, middleware)
	// Call the middleware to get the initialized authMiddleware
	// We can't directly inspect it, but we can verify it doesn't panic
	require.NotPanics(t, func() {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(middleware)
	})
	_ = a // Use the variable to avoid unused error
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
				mux.HandleFunc(
					dexDiscoPath,
					func(w http.ResponseWriter, _ *http.Request) {
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
					func(w http.ResponseWriter, _ *http.Request) {
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
		testPath        = "/v1beta1/projects"
		testIDPIssuer   = "fake-idp-issuer"
		testKargoIssuer = "fake-kargo-issuer"
		testToken       = "some-token"
	)
	testSets := map[string]struct {
		path           string
		authMiddleware *authMiddleware
		token          string
		assertions     func(ctx context.Context, err error)
	}{
		"exempt path": {
			path:           "/v1beta1/system/public-server-config",
			authMiddleware: &authMiddleware{},
			// The path is exempt from authentication, so no user information
			// should be bound to the context.
			assertions: func(ctx context.Context, err error) {
				require.NoError(t, err)
				_, ok := user.InfoFromContext(ctx)
				require.False(t, ok)
			},
		},
		"no token provided": {
			path: testPath,
			// It's an error if no token is provided.
			assertions: func(ctx context.Context, err error) {
				require.Error(t, err)
				require.Equal(t, "no token provided", err.Error())
				_, ok := user.InfoFromContext(ctx)
				require.False(t, ok)
			},
		},
		"non-JWT token": {
			path: testPath,
			authMiddleware: &authMiddleware{
				parseUnverifiedJWTFn: func(
					string,
					jwt.Claims,
				) (*jwt.Token, []string, error) {
					return nil, nil, errors.New("this is not a JWT")
				},
			},
			token: testToken,
			assertions: func(ctx context.Context, err error) {
				require.Equal(t, "invalid token", err.Error())
				_, ok := user.InfoFromContext(ctx)
				require.False(t, ok)
			},
		},
		"failure verifying Kargo-issued token": {
			path: testPath,
			authMiddleware: &authMiddleware{
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
				verifyKargoIssuedTokenFn: func(_ string) bool {
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
			path: testPath,
			authMiddleware: &authMiddleware{
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
				verifyKargoIssuedTokenFn: func(_ string) bool {
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
				require.Empty(t, u.Claims["sub"])
				require.Empty(t, u.Claims["groups"])
				require.Equal(t, testToken, u.BearerToken)
			},
		},
		"failure verifying IDP-issued token": {
			path: testPath,
			authMiddleware: &authMiddleware{
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
				) (claims, error) {
					return claims{}, errors.New("invalid token")
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
			path: testPath,
			authMiddleware: &authMiddleware{
				cfg: config.ServerConfig{
					OIDCConfig: &libOIDC.Config{
						IssuerURL:     testIDPIssuer,
						UsernameClaim: "preferred_username",
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
				) (claims, error) {
					return claims{
						"preferred_username": "foo",
						"sub":                "ironman",
						"email":              "tony@starkindustries.com",
						"groups": []string{
							"avengers",
							"shield",
						},
					}, nil
				},
				listServiceAccountsFn: func(
					context.Context,
					claims,
				) (map[string]map[types.NamespacedName]struct{}, error) {
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
				require.Equal(t, u.Username, "foo")
				require.Equal(t, "ironman", u.Claims["sub"])
				require.Equal(t, "tony@starkindustries.com", u.Claims["email"])
				require.Equal(t, []string{"avengers", "shield"}, u.Claims["groups"])
				require.Equal(t, testToken, u.BearerToken)
			},
		},
		"unrecognized JWT recognized by Kubernetes": {
			path: testPath,
			authMiddleware: &authMiddleware{
				parseUnverifiedJWTFn: func(_ string, claims jwt.Claims) (*jwt.Token, []string, error) {
					rc, ok := claims.(*jwt.RegisteredClaims)
					require.True(t, ok)
					rc.Issuer = "unrecognized-issuer"
					return nil, nil, nil
				},
				verifyKubernetesTokenFn: func(context.Context, string) error {
					return nil // Token is recognized by Kubernetes
				},
			},
			token: testToken,
			// We can't verify this token, so we check if Kubernetes recognizes it.
			// In this case it does, so we expect user info containing the raw token
			// to be bound to the context.
			assertions: func(ctx context.Context, err error) {
				require.NoError(t, err)
				u, ok := user.InfoFromContext(ctx)
				require.True(t, ok)
				require.Equal(t, testToken, u.BearerToken)
			},
		},
		"unrecognized JWT not recognized by Kubernetes": {
			path: testPath,
			authMiddleware: &authMiddleware{
				parseUnverifiedJWTFn: func(_ string, claims jwt.Claims) (*jwt.Token, []string, error) {
					rc, ok := claims.(*jwt.RegisteredClaims)
					require.True(t, ok)
					rc.Issuer = "unrecognized-issuer"
					return nil, nil, nil
				},
				verifyKubernetesTokenFn: func(context.Context, string) error {
					return errors.New("token not recognized")
				},
			},
			token: testToken,
			// We can't verify this token and Kubernetes doesn't recognize it either.
			// This should result in an authentication error.
			assertions: func(ctx context.Context, err error) {
				require.Error(t, err)
				require.Equal(t, "invalid token", err.Error())
				_, ok := user.InfoFromContext(ctx)
				require.False(t, ok)
			},
		},
	}
	for name, ts := range testSets {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// For exempt paths test, we need to handle the nil middleware case
			if ts.authMiddleware == nil {
				ts.authMiddleware = &authMiddleware{}
			}
			ctx, err := ts.authMiddleware.authenticate(
				context.Background(),
				ts.path,
				ts.token,
			)
			ts.assertions(ctx, err)
		})
	}
}

func TestVerifyIDPIssuedTokenFn(t *testing.T) {
	testCases := []struct {
		name           string
		authMiddleware *authMiddleware
		assertions     func(t *testing.T, c claims, err error)
	}{
		{
			name:           "OIDC not supported",
			authMiddleware: &authMiddleware{},
			assertions: func(t *testing.T, _ claims, err error) {
				require.ErrorContains(t, err, "OpenID Connect is not supported")
			},
		},
		{
			name: "token cannot be verified",
			authMiddleware: &authMiddleware{
				cfg: config.ServerConfig{
					OIDCConfig: &libOIDC.Config{},
				},
				oidcTokenVerifyFn: func(
					context.Context,
					string,
				) (*oidc.IDToken, error) {
					return nil, errors.New("invalid token")
				},
			},
			assertions: func(t *testing.T, _ claims, err error) {
				require.ErrorContains(t, err, "invalid token")
			},
		},
		{
			name: "error getting claims from token",
			authMiddleware: &authMiddleware{
				cfg: config.ServerConfig{
					OIDCConfig: &libOIDC.Config{},
				},
				oidcTokenVerifyFn: func(
					context.Context,
					string,
				) (*oidc.IDToken, error) {
					return &oidc.IDToken{}, nil
				},
				oidcExtractClaimsFn: func(*oidc.IDToken) (claims, error) {
					return claims{}, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ claims, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "token is successfully verified",
			authMiddleware: &authMiddleware{
				cfg: config.ServerConfig{
					OIDCConfig: &libOIDC.Config{},
				},
				oidcTokenVerifyFn: func(
					context.Context,
					string,
				) (*oidc.IDToken, error) {
					return &oidc.IDToken{
						Subject: "ironman",
					}, nil
				},
				oidcExtractClaimsFn: func(*oidc.IDToken) (claims, error) {
					return claims{
						"sub":   "ironman",
						"email": "tony@starkindustries.io",
						"groups": []string{
							"avengers",
							"shield",
						},
					}, nil
				},
			},
			assertions: func(t *testing.T, c claims, err error) {
				require.NoError(t, err)
				require.Equal(t, "ironman", c["sub"])
				require.Equal(t, "tony@starkindustries.io", c["email"])
				require.Equal(t, []string{"avengers", "shield"}, c["groups"])
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			c, err := testCase.authMiddleware.verifyIDPIssuedToken(
				context.Background(),
				// With the way these tests are constructed, this doesn't have to
				// be valid.
				"some-token",
			)
			testCase.assertions(t, c, err)
		})
	}
}

func TestVerifyKargoIssuedToken(t *testing.T) {
	const testNonJWTToken = "some-token"
	testTokenSigningKey := []byte("iwishtowashmyirishwristwatch")
	testCases := []struct {
		name           string
		tokenFn        func() string // Returns a raw token
		authMiddleware *authMiddleware
		valid          bool
	}{
		{
			name:           "admin user not supported",
			authMiddleware: &authMiddleware{},
			tokenFn: func() string {
				return testNonJWTToken
			},
			valid: false,
		},
		{
			name: "token is not a JWT",
			authMiddleware: &authMiddleware{
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
			authMiddleware: &authMiddleware{
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
			authMiddleware: &authMiddleware{
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
			authMiddleware: &authMiddleware{
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
				testCase.authMiddleware.verifyKargoIssuedToken(testCase.tokenFn()),
			)
		})
	}
}

func TestAuthMiddlewareHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		path           string
		token          string
		authMiddleware *authMiddleware
		expectedStatus int
		expectUserInfo bool
	}{
		{
			name:           "exempt path - no auth required",
			path:           "/v1beta1/system/public-server-config",
			authMiddleware: &authMiddleware{},
			expectedStatus: http.StatusOK,
			expectUserInfo: false,
		},
		{
			name:           "no token provided",
			path:           "/v1beta1/projects",
			authMiddleware: &authMiddleware{},
			expectedStatus: http.StatusUnauthorized,
			expectUserInfo: false,
		},
		{
			name: "valid admin token",
			path: "/v1beta1/projects",
			authMiddleware: &authMiddleware{
				cfg: config.ServerConfig{
					AdminConfig: &config.AdminConfig{
						TokenIssuer: "kargo",
					},
				},
				parseUnverifiedJWTFn: func(_ string, claims jwt.Claims) (*jwt.Token, []string, error) {
					rc, ok := claims.(*jwt.RegisteredClaims)
					require.True(t, ok)
					rc.Issuer = "kargo"
					return nil, nil, nil
				},
				verifyKargoIssuedTokenFn: func(_ string) bool {
					return true
				},
			},
			token:          "valid-admin-token",
			expectedStatus: http.StatusOK,
			expectUserInfo: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(tc.authMiddleware.Handler)
			router.GET("/v1beta1/*path", func(c *gin.Context) {
				_, hasUser := user.InfoFromContext(c.Request.Context())
				require.Equal(t, tc.expectUserInfo, hasUser)
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestVerifyKubernetesToken(t *testing.T) {
	const testToken = "test-bearer-token"
	testCases := []struct {
		name              string
		mockK8sAPIHandler http.HandlerFunc
		assertions        func(t *testing.T, err error)
	}{
		{
			name: "Kubernetes API returns 200",
			mockK8sAPIHandler: func(w http.ResponseWriter, r *http.Request) {
				// Verify the token was passed correctly
				require.Equal(t, "/api", r.URL.Path)
				require.Equal(t, "Bearer "+testToken, r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusOK)
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "Kubernetes API returns non-200",
			mockK8sAPIHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(
					t, err, "unexpected response from Kubernetes API server",
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			srv := httptest.NewServer(testCase.mockK8sAPIHandler)
			t.Cleanup(srv.Close)
			authenticator := &authMiddleware{
				cfg: config.ServerConfig{RestConfig: &rest.Config{Host: srv.URL}},
			}
			err := authenticator.verifyKubernetesToken(t.Context(), testToken)
			testCase.assertions(t, err)
		})
	}
}
