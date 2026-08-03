package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hashicorp/go-cleanhttp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	libClient "sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/user"
)

const authHeaderKey = "Authorization"

// errNoToken and errInvalidToken are the only authentication failures reported
// to clients. Both carry a 401 and disclose nothing about which check rejected
// the credential. Where the underlying reason is useful, sites wrap
// errInvalidToken with it so that the detail reaches the logs while the
// client's response remains opaque.
//
// Any other failure, such as an unreachable API server or a misconfigured
// identity provider, is returned without a status code, which the
// error-handling middleware reports as an internal error. A client must not be
// able to mistake a broken control plane for a rejected token.
var (
	errNoToken      = libhttp.ErrorStr("no token provided", http.StatusUnauthorized)
	errInvalidToken = libhttp.ErrorStr("invalid token", http.StatusUnauthorized)
)

// exemptPaths are REST paths that don't require authentication
var exemptPaths = map[string]struct{}{
	"/v1beta1/system/public-server-config": {},
	"/v1beta1/login":                       {},
}

// authMiddleware is a Gin middleware that authenticates requests
type authMiddleware struct {
	cfg            config.ServerConfig
	internalClient libClient.Client
	// A set of paths that are exempt from authentication. This is used to allow certain
	// endpoints to be accessed without a token, such as the public server config
	// endpoint.
	exemptPaths map[string]struct{}

	parseUnverifiedJWTFn func(
		rawToken string,
		claims jwt.Claims,
	) (*jwt.Token, []string, error)
	verifyKargoIssuedTokenFn func(rawToken string) bool
	verifyIDPIssuedTokenFn   func(
		ctx context.Context,
		rawToken string,
	) (claims, error)
	verifyKubernetesTokenFn func(ctx context.Context, rawToken string) error
	oidcTokenVerifyFn       goOIDCIDTokenVerifyFn
	oidcExtractClaimsFn     func(*oidc.IDToken) (claims, error)
	listServiceAccountsFn   func(
		ctx context.Context,
		c claims,
	) (map[string]map[types.NamespacedName]struct{}, error)
}

// goOIDCIDTokenVerifyFn is a github.com/coreos/go-oidc/v3/oidc/IDTokenVerifier.Verify() function
type goOIDCIDTokenVerifyFn func(ctx context.Context, rawIDToken string) (*oidc.IDToken, error)

type claims map[string]any

// AuthMiddlewareOpt is a functional option for configuring the auth middleware returned by
// NewAuthMiddleware.
type AuthMiddlewareOpt func(*authMiddleware)

// WithExemptPaths adds additional exempt paths for the auth middleware. These are added to the
// default exempt paths, which are /v1beta1/system/public-server-config and /v1beta1/login.
func WithExemptPaths(paths []string) AuthMiddlewareOpt {
	return func(a *authMiddleware) {
		for _, path := range paths {
			a.exemptPaths[path] = struct{}{}
		}
	}
}

// NewAuthMiddleware returns an initialized Gin middleware handler for authentication.
func NewAuthMiddleware(
	ctx context.Context,
	cfg config.ServerConfig,
	client libClient.Client,
	opts ...AuthMiddlewareOpt,
) gin.HandlerFunc {
	a := &authMiddleware{
		cfg:            cfg,
		internalClient: client,
		exemptPaths:    maps.Clone(exemptPaths),
	}
	if cfg.OIDCConfig != nil {
		a.oidcTokenVerifyFn = newMultiClientVerifier(ctx, cfg)
	}
	a.parseUnverifiedJWTFn = jwt.NewParser(jwt.WithoutClaimsValidation()).ParseUnverified
	a.verifyKargoIssuedTokenFn = a.verifyKargoIssuedToken
	a.verifyIDPIssuedTokenFn = a.verifyIDPIssuedToken
	a.verifyKubernetesTokenFn = a.verifyKubernetesToken
	a.oidcExtractClaimsFn = oidcExtractClaims
	a.listServiceAccountsFn = a.listServiceAccounts

	for _, opt := range opts {
		opt(a)
	}

	return a.Handler
}

// Handler is the actual Gin middleware handler function
func (a *authMiddleware) Handler(c *gin.Context) {
	ctx := c.Request.Context()
	path := c.Request.URL.Path
	logger := logging.LoggerFromContext(ctx).WithValues("path", path)

	logger.Debug("authenticating request")

	// Extract token
	rawToken := strings.TrimPrefix(c.GetHeader(authHeaderKey), "Bearer ")

	// Authenticate and get user info
	newCtx, err := a.authenticate(ctx, path, rawToken)
	if err != nil {
		logger.Debug("authentication failed", "error", err.Error())
		// Leave the status code and response body to the error-handling
		// middleware, which honors the code carried by the error.
		_ = c.Error(err)
		c.Abort()
		return
	}

	// Update the request context with authenticated user info
	c.Request = c.Request.WithContext(newCtx)

	logger.Debug("authentication successful")
	c.Next()
}

// authenticate validates the token and extracts user information
func (a *authMiddleware) authenticate(
	ctx context.Context,
	path string,
	rawToken string,
) (context.Context, error) {
	logger := logging.LoggerFromContext(ctx).WithValues("path", path)

	// Check if this path is exempt from authentication
	if _, ok := a.exemptPaths[path]; ok {
		logger.Debug("skipping authentication for exempt path")
		return ctx, nil
	}

	if rawToken == "" {
		return ctx, errNoToken
	}

	// Are we dealing with a JWT?
	//
	// If not, we no longer assume this is potentially some other form of token
	// that the Kubernetes API server might recognize, as that is an increasingly
	// unlikely scenario.
	//
	// If this IS a JWT, we cannot trust these claims yet because we're not
	// verifying the token just yet. We use untrustedClaims.Issuer only as a hint
	// as to HOW we might be able to verify the token further.
	untrustedClaims := jwt.RegisteredClaims{}
	if _, _, err := a.parseUnverifiedJWTFn(rawToken, &untrustedClaims); err != nil {
		return ctx, errInvalidToken
	}
	logger.Debug("found untrusted claims in token", "claims", untrustedClaims)

	// If we get to here, we're dealing with a JWT. It could have been issued:
	//
	//   1. Directly by the Kargo API server (in the case of admin)
	//   2. By Kargo's OpenID Connect identity provider
	//   3. By the Kubernetes cluster's identity provider
	//   4. By Kubernetes itself (a service account token, perhaps)

	if a.cfg.AdminConfig != nil &&
		untrustedClaims.Issuer == a.cfg.AdminConfig.TokenIssuer {
		// Case 1: This token was allegedly issued directly by the Kargo API server.
		logger.Debug("admin token allegedly issued by Kargo API server")
		if a.verifyKargoIssuedTokenFn(rawToken) {
			logger.Debug("admin token verified as issued by Kargo API server")
			return user.ContextWithInfo(
				ctx,
				user.Info{
					IsAdmin:     true,
					BearerToken: rawToken,
				},
			), nil
		}
		return ctx, errInvalidToken
	}

	if a.cfg.OIDCConfig != nil &&
		untrustedClaims.Issuer == a.cfg.OIDCConfig.IssuerURL {
		// Case 2: This token was allegedly issued by Kargo's OpenID Connect
		// identity provider.
		logger.Debug(
			"token allegedly issued by Kargo's OpenID Connect identity provider",
		)
		c, err := a.verifyIDPIssuedTokenFn(ctx, rawToken)
		if err != nil {
			return ctx, err
		}
		logger.Debug(
			"token verified as issued by Kargo's OpenID Connect identity provider",
		)
		sa, err := a.listServiceAccountsFn(ctx, c)
		if err != nil {
			// Stated explicitly because the underlying Kubernetes error carries a
			// status code of its own, which the error-handling middleware would
			// otherwise report to the client as if it described their request.
			return ctx, libhttp.Error(
				fmt.Errorf("list service accounts for user: %w", err),
				http.StatusInternalServerError,
			)
		}
		var username string
		if un, ok := c[a.cfg.OIDCConfig.UsernameClaim]; ok {
			if username, ok = un.(string); !ok {
				// The token verified, so this is a mismatch between the identity
				// provider and this server's configuration, not a bad credential.
				// Left untyped so it is reported as an internal error.
				return ctx, fmt.Errorf(
					"claim %q must be a string; got %T",
					a.cfg.OIDCConfig.UsernameClaim,
					un,
				)
			}
		}
		return user.ContextWithInfo(
			ctx,
			user.Info{
				Claims:                     c,
				ServiceAccountsByNamespace: sa,
				BearerToken:                rawToken,
				UsernameClaim:              a.cfg.OIDCConfig.UsernameClaim,
				Username:                   username,
			},
		), nil

	}

	// Case 3 or 4: We don't know how to verify this token. It's possibly a token
	// issued by the Kubernetes cluster's identity provider.

	// Test whether Kubernetes recognizes this token by making a request to /api
	logger.Debug("could not verify token; checking if Kubernetes recognizes it")
	if err := a.verifyKubernetesTokenFn(ctx, rawToken); err != nil {
		return ctx, err
	}
	logger.Debug("token recognized by Kubernetes")

	return user.ContextWithInfo(
		ctx,
		user.Info{
			BearerToken: rawToken,
		},
	), nil
}

func (a *authMiddleware) listServiceAccounts(
	ctx context.Context,
	c claims,
) (map[string]map[types.NamespacedName]struct{}, error) {
	queries := []libClient.MatchingFields{}
	for claimName, claimValue := range c {
		if claimValuesString, ok := claimValue.(string); ok {
			queries = append(queries, libClient.MatchingFields{
				indexer.ServiceAccountsByOIDCClaimsField: indexer.FormatClaim(claimName, claimValuesString),
			})
		}
		if claimValueSlice, ok := claimValue.([]any); ok {
			for _, claimValueSliceItem := range claimValueSlice {
				if claimValueSliceItemString, ok := claimValueSliceItem.(string); ok {
					queries = append(queries, libClient.MatchingFields{
						indexer.ServiceAccountsByOIDCClaimsField: indexer.FormatClaim(
							claimName, claimValueSliceItemString,
						),
					})
				}
			}
		}
	}
	// allowedNamespaces is a set of all namespaces in which to search for
	// ServiceAccounts the user may be mapped to. These will includes all project
	// namespaces and any additional namespaces that the Kargo admin has
	// designated.
	allowedNamespaces := make(map[string]struct{})
	if a.cfg.OIDCConfig != nil {
		// Add namespaces designated by the Kargo admin to the set.
		for _, ns := range a.cfg.OIDCConfig.GlobalServiceAccountNamespaces {
			allowedNamespaces[ns] = struct{}{}
		}
	}
	// Find all project namespaces.
	nsList := &corev1.NamespaceList{}
	if err := a.internalClient.List(ctx, nsList, libClient.MatchingLabels{
		kargoapi.LabelKeyProject: kargoapi.LabelValueTrue,
	}); err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}
	// Add all project namespaces to the set.
	for _, ns := range nsList.Items {
		allowedNamespaces[ns.GetName()] = struct{}{}
	}
	// Now search all identified namespaces for ServiceAccounts that the user may
	// be mapped to.
	accounts := make(map[string]map[types.NamespacedName]struct{})
	for _, query := range queries {
		// List ALL ServiceAccounts matching the query.
		list := &corev1.ServiceAccountList{}
		if err := a.internalClient.List(ctx, list, query); err != nil {
			return nil, fmt.Errorf("list service accounts: %w", err)
		}
		for _, sa := range list.Items {
			// Skip if it's not in a namespace we care about.
			if _, ok := allowedNamespaces[sa.GetNamespace()]; !ok {
				continue
			}
			key := types.NamespacedName{
				Namespace: sa.GetNamespace(),
				Name:      sa.GetName(),
			}
			if _, ok := accounts[key.Namespace]; !ok {
				accounts[key.Namespace] = make(map[types.NamespacedName]struct{})
			}
			accounts[key.Namespace][key] = struct{}{}
		}
	}
	return accounts, nil
}

var verifierMu = sync.Mutex{}

// verifyIDPIssuedToken attempts to verify that the provided raw token was
// issued by Kargo's OpenID Connect identity provider. On success, select claims
// are extracted and returned.
func (a *authMiddleware) verifyIDPIssuedToken(
	ctx context.Context,
	rawToken string,
) (claims, error) {
	if a.cfg.OIDCConfig == nil {
		// Really, this method never should have been called under these
		// circumstances.
		return claims{}, errors.New("OpenID Connect is not supported")
	}
	c := claims{}
	if a.oidcTokenVerifyFn == nil {
		verifierMu.Lock()
		if a.oidcTokenVerifyFn == nil {
			a.oidcTokenVerifyFn = newMultiClientVerifier(ctx, a.cfg)
		}
		verifier := a.oidcTokenVerifyFn
		verifierMu.Unlock()
		if verifier == nil {
			return c, errors.New(
				"could not validate token, possibly due to a transient network " +
					"error; if the problem persists, check your OpenID Connect " +
					"configuration",
			)
		}
	}
	token, err := a.oidcTokenVerifyFn(ctx, rawToken)
	if err != nil {
		return c, fmt.Errorf("%w: %w", errInvalidToken, err)
	}
	c, err = a.oidcExtractClaimsFn(token)
	if err != nil {
		// The token verified, so failing to read its claims is our problem, not
		// the client's. Left untyped so it is reported as an internal error.
		return c, fmt.Errorf("extract claims from verified token: %w", err)
	}
	return c, nil
}

// verifyKargoIssuedToken attempts to verify that the provided raw token was
// issued directly by the Kargo API server and returns a boolean value
// indicating success (true) or failure (false).
func (a *authMiddleware) verifyKargoIssuedToken(rawToken string) bool {
	if a.cfg.AdminConfig == nil {
		return false
	}
	_, err := jwt.NewParser().Parse(
		rawToken,
		func(*jwt.Token) (any, error) {
			return a.cfg.AdminConfig.TokenSigningKey, nil
		},
	)
	return err == nil
}

// verifyKubernetesToken tests whether the Kubernetes API server recognizes the
// provided token by making a GET request to the /api endpoint. This is a
// lightweight check that doesn't require any specific permissions.
func (a *authMiddleware) verifyKubernetesToken(
	ctx context.Context,
	rawToken string,
) error {
	if a.cfg.RestConfig == nil { // This shouldn't happen, but just in case...
		return errors.New("Kubernetes REST config is not available") // nolint: staticcheck
	}

	transport, err := rest.TransportFor(a.cfg.RestConfig)
	if err != nil {
		return fmt.Errorf("create transport: %w", err)
	}

	apiURL := strings.TrimSuffix(a.cfg.RestConfig.Host, "/") + "/api"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+rawToken)

	// #nosec G704 -- This request is not for a user-specified URL, so there is
	// virtually no risk of SSRF here.
	resp, err := (&http.Client{Transport: transport}).Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	// Only Kubernetes declining to recognize the token says anything about the
	// token. Every other failure above is ours, and is left untyped so that it is
	// reported as an internal error rather than as a rejected credential.
	//
	// TODO(krancour/fuskovic): This isn't perfect, but the strategy of verifying
	// that Kubernetes recognized the token is already set to change in
	// https://github.com/akuity/kargo/pull/6754 so we will not obsess over this
	// for now.
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"%w: unexpected response from Kubernetes API server: %d",
			errInvalidToken,
			resp.StatusCode,
		)
	}

	return nil
}

func oidcExtractClaims(token *oidc.IDToken) (claims, error) {
	c := claims{}
	err := token.Claims(&c)
	return c, err
}

// newMultiClientVerifier returns a function that implements go-oidc IDTokenVerifier.Verify()
// but iterates through multiple verifiers. We commonly have both a CLI and Web OIDC client,
// each needing it's own OIDC verification.
func newMultiClientVerifier(ctx context.Context, cfg config.ServerConfig) goOIDCIDTokenVerifyFn {
	keyset, err := getKeySet(ctx, cfg)
	if err != nil {
		// The likely cause of this error is misconfiguration of the issuer URL.
		// In case it's actually a transient network error, we'll log the error and
		// return nil. Each authn attempt will retry this operation until it
		// succeeds.
		logger := logging.LoggerFromContext(ctx)
		logger.Error(
			err,
			"error getting keys from OpenID Connect provider; will try again on first authn attempt",
		)
		return nil
	}
	// verifyFuncs might have two verify funcs: the web and cli verifier
	var verifyFuncs []goOIDCIDTokenVerifyFn
	verifyFuncs = append(verifyFuncs, oidc.NewVerifier(
		cfg.OIDCConfig.IssuerURL,
		keyset,
		&oidc.Config{
			ClientID: cfg.OIDCConfig.ClientID,
		},
	).Verify)
	if cfg.OIDCConfig.CLIClientID != "" {
		verifyFuncs = append(verifyFuncs, oidc.NewVerifier(
			cfg.OIDCConfig.IssuerURL,
			keyset,
			&oidc.Config{
				ClientID: cfg.OIDCConfig.CLIClientID,
			},
		).Verify)
	}
	multiVerifyFunc := func(ctx context.Context, rawIDToken string) (*oidc.IDToken, error) {
		errs := make([]error, 0, len(verifyFuncs))
		for _, fn := range verifyFuncs {
			t, err := fn(ctx, rawIDToken)
			if err == nil {
				// we found one that worked
				return t, nil
			}
			errs = append(errs, err)
		}
		// if we get here, we've iterated all our verifiers and none of them worked.
		return nil, errors.Join(errs...)
	}
	return multiVerifyFunc
}

// getKeySet retrieves the key set from the an OpenID Connect identify provider.
//
// Note: This function purposefully does not use oidc.NewProvider() and
// provider.Verifier() because they're not flexible enough to handle the Dex
// proxy case.
func getKeySet(ctx context.Context, cfg config.ServerConfig) (oidc.KeySet, error) {
	httpClient := cleanhttp.DefaultClient()

	var discoURL string
	// dexBaseAddr is the in-cluster URL of Dex, with the path component Dex
	// serves its endpoints under (derived from its configured issuer URL).
	// Only set when DexProxyConfig is non-nil.
	var dexBaseAddr string
	var err error
	if cfg.DexProxyConfig == nil {
		if discoURL, err = url.JoinPath(
			cfg.OIDCConfig.IssuerURL,
			".well-known",
			"openid-configuration",
		); err != nil {
			return nil, fmt.Errorf(
				"error constructing discovery URL from issuer URL %q: %w",
				cfg.OIDCConfig.IssuerURL,
				err,
			)
		}
	} else {
		var issuerURL *url.URL
		if issuerURL, err = url.Parse(cfg.OIDCConfig.IssuerURL); err != nil {
			return nil, fmt.Errorf(
				"error parsing OIDC issuer URL %q: %w",
				cfg.OIDCConfig.IssuerURL,
				err,
			)
		}
		// Dex routes its endpoints based on the path of its configured issuer
		// URL, which includes the API server's basePath when one is set. Use
		// that same path with the in-cluster Dex address so paths line up.
		dexBaseAddr = cfg.DexProxyConfig.ServerAddr + issuerURL.Path
		if discoURL, err = url.JoinPath(
			dexBaseAddr,
			".well-known",
			"openid-configuration",
		); err != nil {
			return nil, fmt.Errorf(
				"error constructing discovery URL from issuer URL %q: %w",
				cfg.OIDCConfig.IssuerURL,
				err,
			)
		}
		var caCertPool *x509.CertPool
		if cfg.DexProxyConfig.CACertPath != "" {
			var caCertBytes []byte
			// #nosec G703 -- Contextually, this was an operator-specified path;
			// typically having been specified by the chart, and without even a
			// an option for specifying an alternate path.
			if caCertBytes, err = os.ReadFile(cfg.DexProxyConfig.CACertPath); err != nil {
				return nil, fmt.Errorf("error reading CA cert file %q: %w", cfg.DexProxyConfig.CACertPath, err)
			}
			caCertPool = x509.NewCertPool()
			if ok := caCertPool.AppendCertsFromPEM(caCertBytes); !ok {
				return nil, errors.New("invalid CA cert data")
			}
			transport := cleanhttp.DefaultTransport()
			transport.TLSClientConfig = &tls.Config{
				MinVersion: tls.VersionTLS12,
				RootCAs:    caCertPool,
			}
			httpClient.Transport = transport
		}
	}

	// #nosec G704 -- Contextually, this URL is specified by an operator and not
	// by an end user.
	discoResp, err := httpClient.Get(discoURL)
	if err != nil {
		return nil, fmt.Errorf("error making discovery request to OpenID Connect identity provider: %w", err)
	}
	defer discoResp.Body.Close()
	bodyBytes, err := io.ReadAll(discoResp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading discovery request response body: %w", err)
	}
	providerCfg := struct {
		KeysURL string `json:"jwks_uri"`
	}{}
	if err = json.Unmarshal(bodyBytes, &providerCfg); err != nil {
		fmt.Println(string(bodyBytes))
		return nil, fmt.Errorf("error unmarshaling discovery request response body: %w", err)
	}

	keysURL := providerCfg.KeysURL
	if cfg.DexProxyConfig != nil {
		keysURL = strings.Replace(
			keysURL,
			cfg.OIDCConfig.IssuerURL,
			dexBaseAddr,
			1,
		)
	}

	// oidc.RemoteKeySet has an internal cache and it is sometimes refreshed. It
	// uses a context-bound http.Client to make the request if one is available.
	// This next line binds our properly configured http.Client to the context.
	ctx = oidc.ClientContext(ctx, httpClient)
	return oidc.NewRemoteKeySet(ctx, keysURL), nil
}
