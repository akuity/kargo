package server

import (
	"context"
	"maps"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	libClient "sigs.k8s.io/controller-runtime/pkg/client"

	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/auth/authn"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/user"
)

const authHeaderKey = "Authorization"

// errNoToken is reported when a request carries no bearer token at all. See
// authn.ErrInvalidToken for the only other authentication failure clients
// hear about.
var errNoToken = libhttp.ErrorStr("no token provided", http.StatusUnauthorized)

// exemptPaths are REST paths that don't require authentication. The UI's fetch
// wrapper keeps its own copy of the entries the UI calls (authExemptPaths in
// ui/src/lib/api/custom-fetch.ts); a path added here that the UI calls should
// be added there as well.
var exemptPaths = map[string]struct{}{
	"/v1beta1/system/public-server-config": {},
	"/v1beta1/login":                       {},
}

// authMiddleware is a Gin middleware that authenticates requests: it resolves
// each request's bearer token to a user.Identity and binds it to the request
// context for handlers and the authorizing Kubernetes client to read back.
type authMiddleware struct {
	authenticator authn.Authenticator
	// exemptPaths are paths that are served without a token, such as the
	// public server config endpoint.
	exemptPaths map[string]struct{}
}

// AuthMiddlewareOpt is a functional option for configuring the auth middleware
// returned by NewAuthMiddleware.
type AuthMiddlewareOpt func(*authMiddleware)

// WithExemptPaths returns an AuthMiddlewareOpt that adds the given paths to
// the set of paths that are exempt from authentication.
func WithExemptPaths(paths []string) AuthMiddlewareOpt {
	return func(a *authMiddleware) {
		for _, path := range paths {
			a.exemptPaths[path] = struct{}{}
		}
	}
}

// WithAuthenticator returns an AuthMiddlewareOpt that replaces the
// Authenticator the middleware resolves tokens with.
func WithAuthenticator(authenticator authn.Authenticator) AuthMiddlewareOpt {
	return func(a *authMiddleware) {
		a.authenticator = authenticator
	}
}

// NewAuthMiddleware returns a Gin middleware that authenticates requests with
// the Authenticators the server configuration calls for (see authn.New),
// remembering verified tokens for as long as it allows.
func NewAuthMiddleware(
	ctx context.Context,
	cfg config.ServerConfig,
	client libClient.Client,
	opts ...AuthMiddlewareOpt,
) gin.HandlerFunc {
	a := &authMiddleware{
		authenticator: authn.WithTokenCache(
			authn.New(ctx, cfg, client),
			cfg.AuthCacheConfig.TokenTTL,
		),
		exemptPaths: maps.Clone(exemptPaths),
	}
	for _, opt := range opts {
		opt(a)
	}
	return a.Handler
}

// Handler is the actual Gin middleware handler function.
func (a *authMiddleware) Handler(c *gin.Context) {
	ctx := c.Request.Context()
	path := c.Request.URL.Path
	logger := logging.LoggerFromContext(ctx).WithValues("path", path)

	logger.Debug("authenticating request")

	rawToken := strings.TrimPrefix(c.GetHeader(authHeaderKey), "Bearer ")

	newCtx, err := a.authenticate(ctx, path, rawToken)
	if err != nil {
		logger.Debug("authentication failed", "error", err.Error())
		// Leave the status code and response body to the error-handling
		// middleware, which honors the code carried by the error.
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.Request = c.Request.WithContext(newCtx)

	logger.Debug("authentication successful")
	c.Next()
}

// authenticate resolves the token to an identity and returns a context
// carrying it. An exempt path yields the context unchanged.
func (a *authMiddleware) authenticate(
	ctx context.Context,
	path string,
	rawToken string,
) (context.Context, error) {
	logger := logging.LoggerFromContext(ctx).WithValues("path", path)

	if _, ok := a.exemptPaths[path]; ok {
		logger.Debug("skipping authentication for exempt path")
		return ctx, nil
	}

	if rawToken == "" {
		return ctx, errNoToken
	}

	id, ok, err := a.authenticator.Authenticate(ctx, rawToken)
	if err != nil {
		return ctx, err
	}
	if !ok {
		return ctx, authn.ErrInvalidToken
	}
	logger.Debug("token verified", "actor", id.Actor())
	ctx = user.ContextWithBearerToken(ctx, rawToken)
	return user.ContextWithIdentity(ctx, id), nil
}
