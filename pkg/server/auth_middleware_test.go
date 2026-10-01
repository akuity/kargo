package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/auth/authn"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/user"
)

// fakeAuthenticator answers every token with a fixed result.
type fakeAuthenticator struct {
	id  user.Identity
	ok  bool
	err error
}

func (f fakeAuthenticator) Authenticate(
	context.Context,
	string,
) (user.Identity, bool, error) {
	return f.id, f.ok, f.err
}

// testJWT is a syntactically valid JWT; the fake authenticators never verify
// it.
func testJWT(t *testing.T) string {
	t.Helper()
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer: "fake-issuer",
	}).SignedString([]byte("any key"))
	require.NoError(t, err)
	return raw
}

func TestNewAuthMiddleware(t *testing.T) {
	middleware := NewAuthMiddleware(t.Context(), config.ServerConfig{}, nil)
	require.NotNil(t, middleware)
	require.NotPanics(t, func() {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(middleware)
	})
}

func TestWithExemptPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const (
		testExemptPath = "/a/fake/path"
		loginPath      = "/v1beta1/login"
		protectedPath  = "/v1beta1/protected"
	)

	middleware := NewAuthMiddleware(
		t.Context(),
		config.ServerConfig{},
		nil,
		WithExemptPaths([]string{testExemptPath}),
	)

	router := gin.New()
	srv := &server{}
	router.Use(srv.handleError)
	router.Use(middleware)
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }
	router.GET(testExemptPath, ok)
	router.GET(loginPath, ok)
	router.GET(protectedPath, ok)

	testCases := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "custom exempt path skips authentication",
			path:           testExemptPath,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "default exempt paths are preserved",
			path:           loginPath,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "other paths still require authentication",
			path:           protectedPath,
			expectedStatus: http.StatusUnauthorized,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, testCase.path, nil)
			router.ServeHTTP(w, req)
			require.Equal(t, testCase.expectedStatus, w.Code)
		})
	}
}

func TestAuthenticate(t *testing.T) {
	const testPath = "/v1beta1/projects"
	validToken := testJWT(t)
	testCases := []struct {
		name          string
		path          string
		token         string
		authenticator authn.Authenticator
		assertions    func(*testing.T, context.Context, error)
	}{
		{
			name: "exempt path",
			path: "/v1beta1/system/public-server-config",
			// The path is exempt from authentication, so no identity should
			// be bound to the context, and the Authenticator is never asked.
			authenticator: fakeAuthenticator{err: errors.New("must not be called")},
			assertions: func(t *testing.T, ctx context.Context, err error) {
				require.NoError(t, err)
				_, ok := user.IdentityFromContext(ctx)
				require.False(t, ok)
			},
		},
		{
			name: "no token provided",
			path: testPath,
			assertions: func(t *testing.T, ctx context.Context, err error) {
				requireErrorStatus(t, err, http.StatusUnauthorized)
				require.Equal(t, "no token provided", err.Error())
				_, ok := user.IdentityFromContext(ctx)
				require.False(t, ok)
			},
		},
		{
			name:          "token rejected",
			path:          testPath,
			token:         validToken,
			authenticator: fakeAuthenticator{ok: true, err: authn.ErrInvalidToken},
			assertions: func(t *testing.T, ctx context.Context, err error) {
				require.ErrorIs(t, err, authn.ErrInvalidToken)
				_, ok := user.IdentityFromContext(ctx)
				require.False(t, ok)
			},
		},
		{
			// A failure to verify, as opposed to a failed verification, is
			// passed through untouched so that it is reported as an internal
			// error rather than a rejected credential.
			name:          "token could not be verified",
			path:          testPath,
			token:         validToken,
			authenticator: fakeAuthenticator{ok: true, err: errors.New("connection refused")},
			assertions: func(t *testing.T, _ context.Context, err error) {
				require.ErrorContains(t, err, "connection refused")
				var httpErr *libhttp.HTTPError
				require.False(t, errors.As(err, &httpErr))
			},
		},
		{
			name:          "token recognized by nobody",
			path:          testPath,
			token:         validToken,
			authenticator: fakeAuthenticator{ok: false},
			assertions: func(t *testing.T, _ context.Context, err error) {
				require.ErrorIs(t, err, authn.ErrInvalidToken)
			},
		},
		{
			name:          "token verified",
			path:          testPath,
			token:         validToken,
			authenticator: fakeAuthenticator{ok: true, id: user.Admin{}},
			assertions: func(t *testing.T, ctx context.Context, err error) {
				require.NoError(t, err)
				id, ok := user.IdentityFromContext(ctx)
				require.True(t, ok)
				require.Equal(t, user.Admin{}, id)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			a := &authMiddleware{
				authenticator: testCase.authenticator,
				exemptPaths:   exemptPaths,
			}
			ctx, err := a.authenticate(t.Context(), testCase.path, testCase.token)
			testCase.assertions(t, ctx, err)
		})
	}
}

func TestAuthMiddlewareHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	validToken := testJWT(t)

	testCases := []struct {
		name           string
		path           string
		token          string
		authenticator  authn.Authenticator
		expectedStatus int
		expectedBody   string
		expectIdentity bool
	}{
		{
			name:           "exempt path - no auth required",
			path:           "/v1beta1/system/public-server-config",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "no token provided",
			path:           "/v1beta1/projects",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"no token provided"}`,
		},
		{
			name:           "token rejected",
			path:           "/v1beta1/projects",
			token:          validToken,
			authenticator:  fakeAuthenticator{ok: true, err: authn.ErrInvalidToken},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"invalid token"}`,
		},
		{
			// A failure to verify the token, as opposed to a failed verification,
			// must not be reported to the client as a rejected credential.
			name:           "token verification fails",
			path:           "/v1beta1/projects",
			token:          validToken,
			authenticator:  fakeAuthenticator{ok: true, err: errors.New("create transport: no credentials")},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"internal server error"}`,
		},
		{
			name:           "valid token",
			path:           "/v1beta1/projects",
			token:          validToken,
			authenticator:  fakeAuthenticator{ok: true, id: user.Admin{}},
			expectedStatus: http.StatusOK,
			expectIdentity: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			// The auth middleware delegates status codes and response bodies to
			// the error-handling middleware, so both are needed to observe what
			// a client actually receives.
			srv := &server{}
			router.Use(srv.handleError)
			a := &authMiddleware{authenticator: tc.authenticator, exemptPaths: exemptPaths}
			router.Use(a.Handler)
			router.GET("/v1beta1/*path", func(c *gin.Context) {
				_, hasIdentity := user.IdentityFromContext(c.Request.Context())
				require.Equal(t, tc.expectIdentity, hasIdentity)
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.Equal(t, tc.expectedStatus, w.Code)
			if tc.expectedBody != "" {
				require.JSONEq(t, tc.expectedBody, w.Body.String())
			}
		})
	}
}

// requireErrorStatus asserts that err carries the expected HTTP status code for
// the error-handling middleware to act on.
func requireErrorStatus(t *testing.T, err error, code int) {
	t.Helper()
	var httpErr *libhttp.HTTPError
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, code, httpErr.Code())
}
