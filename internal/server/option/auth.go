package option

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"connectrpc.com/connect"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hashicorp/go-cleanhttp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	libClient "sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/internal/server/config"
	"github.com/akuity/kargo/internal/server/user"
)

const authHeaderKey = "Authorization"

var exemptProcedures = map[string]struct{}{
	"/grpc.health.v1.Health/Check":                                   {},
	"/grpc.health.v1.Health/Watch":                                   {},
	"/akuity.io.kargo.service.v1alpha1.KargoService/GetPublicConfig": {},
	"/akuity.io.kargo.service.v1alpha1.KargoService/AdminLogin":      {},
}

// authInterceptor implements connect.Interceptor and is used to retrieve the
// value of the Authorization header from inbound requests/connections and
// store it in the context.
type authInterceptor struct {
	cfg            config.ServerConfig
	internalClient libClient.Client

	parseUnverifiedJWTFn func(
		rawToken string,
		claims jwt.Claims,
	) (*jwt.Token, []string, error)
	verifyKargoIssuedTokenFn func(rawToken string) bool
	verifyIDPIssuedTokenFn   func(
		ctx context.Context,
		rawToken string,
	) (claims, error)
	oidcTokenVerifyFn     goOIDCIDTokenVerifyFn
	oidcExtractClaimsFn   func(*oidc.IDToken) (claims, error)
	listServiceAccountsFn func(
		ctx context.Context,
		c claims,
	) (map[string]map[types.NamespacedName]struct{}, error)
}

// goOIDCIDTokenVerifyFn is a github.com/coreos/go-oidc/v3/oidc/IDTokenVerifier.Verify() function
type goOIDCIDTokenVerifyFn func(ctx context.Context, rawIDToken string) (*oidc.IDToken, error)

// newAuthInterceptor returns an initialized *authInterceptor.
func newAuthInterceptor(
	ctx context.Context,
	cfg config.ServerConfig,
	client libClient.Client,
) *authInterceptor {
	a := &authInterceptor{
		cfg:            cfg,
		internalClient: client,
	}
	if cfg.OIDCConfig != nil {
		a.oidcTokenVerifyFn = newMultiClientVerifier(ctx, cfg)
	}
	a.parseUnverifiedJWTFn =
		jwt.NewParser(jwt.WithoutClaimsValidation()).ParseUnverified
	a.verifyKargoIssuedTokenFn = a.verifyKargoIssuedToken
	a.verifyIDPIssuedTokenFn = a.verifyIDPIssuedToken
	a.oidcExtractClaimsFn = oidcExtractClaims
	a.listServiceAccountsFn = a.listServiceAccounts
	return a
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
	if cfg.DexProxyConfig == nil {
		discoURL = fmt.Sprintf(
			"%s/.well-known/openid-configuration",
			cfg.OIDCConfig.IssuerURL,
		)
	} else {
		discoURL = fmt.Sprintf(
			"%s/dex/.well-known/openid-configuration",
			cfg.DexProxyConfig.ServerAddr,
		)
		var caCertPool *x509.CertPool
		if cfg.DexProxyConfig.CACertPath != "" {
			caCertBytes, err := os.ReadFile(cfg.DexProxyConfig.CACertPath)
			if err != nil {
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
			fmt.Sprintf("%s/dex", cfg.DexProxyConfig.ServerAddr),
			1,
		)
	}

	// oidc.RemoteKeySet has an internal cache and it is sometimes refreshed. It
	// uses a context-bound http.Client to make the request if one is available.
	// This next line binds our properly configured http.Client to the context.
	ctx = oidc.ClientContext(ctx, httpClient)
	return oidc.NewRemoteKeySet(ctx, keysURL), nil
}

// WrapUnary implements connect.Interceptor.
func (a *authInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(
		ctx context.Context,
		req connect.AnyRequest,
	) (connect.AnyResponse, error) {
		var err error
		if ctx, err =
			a.authenticate(ctx, req.Spec().Procedure, req.Header()); err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		return next(ctx, req)
	}
}

// WrapStreamingClient implements connect.Interceptor.
func (a *authInterceptor) WrapStreamingClient(
	next connect.StreamingClientFunc,
) connect.StreamingClientFunc {
	return func(
		ctx context.Context,
		spec connect.Spec,
	) connect.StreamingClientConn {
		// This is a no-op because this interceptor is only used with handlers.
		return next(ctx, spec)
	}
}

// WrapStreamingHandler implements connect.Interceptor.
func (a *authInterceptor) WrapStreamingHandler(
	next connect.StreamingHandlerFunc,
) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		var err error
		if ctx, err = a.authenticate(
			ctx,
			conn.Spec().Procedure,
			conn.RequestHeader(),
		); err != nil {
			return connect.NewError(connect.CodeUnauthenticated, err)
		}
		return next(ctx, conn)
	}
}

func (a *authInterceptor) listServiceAccounts(
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
		kargoapi.ProjectLabelKey: kargoapi.LabelTrueValue,
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

// authenticate retrieves the value of the Authorization header from inbound
// requests/connections, attempts to validate it and extract meaningful user
// information from it. If successful, it stores the user information in the
// context. If unsuccessful for any reason, it returns an error. It is the
// caller's responsibility to wrap the error appropriately to convey to the
// client that an authentication failure has occurred.
func (a *authInterceptor) authenticate(
	ctx context.Context,
	procedure string,
	header http.Header,
) (context.Context, error) {
	if _, ok := exemptProcedures[procedure]; ok {
		return ctx, nil
	}

	rawToken := strings.TrimPrefix(header.Get(authHeaderKey), "Bearer ")
	if rawToken == "" {
		return ctx, errors.New("no token provided")
	}

	// Are we dealing with a JWT?
	//
	// Note: If this is a JWT, we cannot trust these claims yet because we're not
	// verifying the token yet. We use untrustedClaims.Issuer only as a hint as to
	// HOW we might be able to verify the token further.
	untrustedClaims := jwt.RegisteredClaims{}
	if _, _, err := a.parseUnverifiedJWTFn(rawToken, &untrustedClaims); err != nil {
		// This token isn't a JWT, so it's probably an opaque bearer token for the
		// Kubernetes API server. Just run with it. If we're wrong, Kubernetes API
		// calls will simply have auth errors that will bubble back to the client.
		return user.ContextWithInfo(
			ctx,
			user.Info{
				BearerToken: rawToken,
			},
		), nil
	}

	// If we get to here, we're dealing with a JWT. It could have been issued:
	//
	//   1. Directly by the Kargo API server (in the case of admin)
	//   2. By Kargo's OpenID Connect identity provider
	//   3. By the Kubernetes cluster's identity provider
	//   4. By Kubernetes itself (a service account token, perhaps)

	if a.cfg.AdminConfig != nil &&
		untrustedClaims.Issuer == a.cfg.AdminConfig.TokenIssuer {
		// Case 1: This token was allegedly issued directly by the Kargo API server.
		if a.verifyKargoIssuedTokenFn(rawToken) {
			return user.ContextWithInfo(
				ctx,
				user.Info{
					IsAdmin:     true,
					BearerToken: rawToken,
				},
			), nil
		}
		return ctx, errors.New("invalid token")
	}

	if a.cfg.OIDCConfig != nil &&
		untrustedClaims.Issuer == a.cfg.OIDCConfig.IssuerURL {
		// Case 2: This token was allegedly issued by Kargo's OpenID Connect
		// identity provider.
		c, err := a.verifyIDPIssuedTokenFn(ctx, rawToken)
		if err != nil {
			return ctx, err
		}
		sa, err := a.listServiceAccountsFn(ctx, c)
		if err != nil {
			return ctx, fmt.Errorf("list service accounts for user: %w", err)
		}
		return user.ContextWithInfo(
			ctx,
			user.Info{
				Claims:                     c,
				ServiceAccountsByNamespace: sa,
				BearerToken:                rawToken,
				Username:                   c[a.cfg.UsernameClaim],
			},
		), nil

	}

	// Case 3 or 4: We don't know how to verify this token. It's probably a token
	// issued by the Kubernetes cluster's identity provider. Just run with it. If
	// we're wrong, Kubernetes API calls will simply have auth errors that will
	// bubble back to the client.
	return user.ContextWithInfo(
		ctx,
		user.Info{
			BearerToken: rawToken,
		},
	), nil
}

var verifierMu = sync.Mutex{}

// verifyIDPIssuedToken attempts to verify that the provided raw token was
// issued by Kargo's OpenID Connect identity provider. On success, select claims
// are extracted and returned.
func (a *authInterceptor) verifyIDPIssuedToken(
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
		return c, err
	}
	return a.oidcExtractClaimsFn(token)
}

// verifyKargoIssuedToken attempts to verify that the provided raw token was
// issued directly by the Kargo API server and returns a boolean value
// indicating success (true) or failure (false).
func (a *authInterceptor) verifyKargoIssuedToken(rawToken string) bool {
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

type claims map[string]any

func oidcExtractClaims(token *oidc.IDToken) (claims, error) {
	c := claims{}
	err := token.Claims(&c)
	return c, err
}
