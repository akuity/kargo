package authn

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/hashicorp/go-cleanhttp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/user"
)

// Claims are the verified claims of an OpenID Connect token.
type Claims map[string]any

// verifyFunc verifies a raw ID token. It is the shape of go-oidc's
// IDTokenVerifier.Verify.
type verifyFunc func(ctx context.Context, rawIDToken string) (*oidc.IDToken, error)

// oidcAuthenticator recognizes tokens issued by Kargo's OpenID Connect
// identity provider and maps the users behind them to ServiceAccounts.
type oidcAuthenticator struct {
	cfg  config.ServerConfig
	kube client.Client

	// verify is built from the identity provider's published keys. Building
	// it needs the provider to be reachable, so it is retried on first use
	// if it could not be built at startup.
	mu     sync.Mutex
	verify verifyFunc

	// The following behaviors are overridable for testing purposes:
	extractClaimsFn       func(*oidc.IDToken) (Claims, error)
	listServiceAccountsFn func(context.Context, Claims) (map[string]map[types.NamespacedName]struct{}, error)
}

var _ Authenticator = (*oidcAuthenticator)(nil)

// NewOIDC returns an Authenticator for tokens issued by the configured OpenID
// Connect identity provider. cfg.OIDCConfig must be set.
func NewOIDC(ctx context.Context, cfg config.ServerConfig, kube client.Client) Authenticator {
	a := &oidcAuthenticator{cfg: cfg, kube: kube}
	a.verify = newMultiClientVerifier(ctx, cfg)
	a.extractClaimsFn = extractClaims
	a.listServiceAccountsFn = a.listServiceAccounts
	return a
}

// Authenticate implements Authenticator. A token claiming the identity
// provider's issuer URL is of this kind.
func (a *oidcAuthenticator) Authenticate(
	ctx context.Context,
	rawToken string,
	hint Hint,
) (user.Identity, bool, error) {
	if hint.Issuer != a.cfg.OIDCConfig.IssuerURL {
		return nil, false, nil
	}
	claims, err := a.verifyToken(ctx, rawToken)
	if err != nil {
		return nil, true, err
	}
	serviceAccounts, err := a.listServiceAccountsFn(ctx, claims)
	if err != nil {
		// Stated explicitly because the underlying Kubernetes error carries a
		// status code of its own, which the error-handling middleware would
		// otherwise report to the client as if it described their request.
		return nil, true, fmt.Errorf("list service accounts for user: %w", err)
	}
	var username string
	if raw, ok := claims[a.cfg.OIDCConfig.UsernameClaim]; ok {
		if username, ok = raw.(string); !ok {
			// The token verified, so this is a mismatch between the identity
			// provider and this server's configuration, not a bad credential.
			// Left untyped so it is reported as an internal error.
			return nil, true, fmt.Errorf(
				"claim %q must be a string; got %T",
				a.cfg.OIDCConfig.UsernameClaim, raw,
			)
		}
	}
	return user.OIDCUser{
		Claims:                         claims,
		UsernameClaim:                  a.cfg.OIDCConfig.UsernameClaim,
		Username:                       username,
		ServiceAccountsByNamespace:     serviceAccounts,
		GlobalServiceAccountNamespaces: a.cfg.OIDCConfig.GlobalServiceAccountNamespaces,
	}, true, nil
}

// verifyToken verifies that the identity provider issued the token and
// returns its claims.
func (a *oidcAuthenticator) verifyToken(ctx context.Context, rawToken string) (Claims, error) {
	verify, err := a.verifier(ctx)
	if err != nil {
		return nil, err
	}
	token, err := verify(ctx, rawToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}
	claims, err := a.extractClaimsFn(token)
	if err != nil {
		// The token verified, so failing to read its claims is our problem,
		// not the client's. Left untyped so it is reported as an internal
		// error.
		return nil, fmt.Errorf("extract claims from verified token: %w", err)
	}
	return claims, nil
}

// verifier returns the token verifier, building it if it could not be built
// when the Authenticator was.
func (a *oidcAuthenticator) verifier(ctx context.Context) (verifyFunc, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.verify == nil {
		a.verify = newMultiClientVerifier(ctx, a.cfg)
	}
	if a.verify == nil {
		return nil, errors.New(
			"could not validate token, possibly due to a transient network " +
				"error; if the problem persists, check your OpenID Connect " +
				"configuration",
		)
	}
	return a.verify, nil
}

func extractClaims(token *oidc.IDToken) (Claims, error) {
	c := Claims{}
	err := token.Claims(&c)
	return c, err
}

// listServiceAccounts finds the ServiceAccounts, in Project namespaces and
// the configured global namespaces, that the claims map the user to.
func (a *oidcAuthenticator) listServiceAccounts(
	ctx context.Context,
	c Claims,
) (map[string]map[types.NamespacedName]struct{}, error) {
	queries := []client.MatchingFields{}
	for claimName, claimValue := range c {
		if claimValuesString, ok := claimValue.(string); ok {
			queries = append(queries, client.MatchingFields{
				indexer.ServiceAccountsByOIDCClaimsField: indexer.FormatClaim(claimName, claimValuesString),
			})
		}
		if claimValueSlice, ok := claimValue.([]any); ok {
			for _, claimValueSliceItem := range claimValueSlice {
				if claimValueSliceItemString, ok := claimValueSliceItem.(string); ok {
					queries = append(queries, client.MatchingFields{
						indexer.ServiceAccountsByOIDCClaimsField: indexer.FormatClaim(
							claimName, claimValueSliceItemString,
						),
					})
				}
			}
		}
	}
	// allowedNamespaces is the set of namespaces in which to search for
	// ServiceAccounts the user may be mapped to: every Project namespace and
	// any additional namespaces the Kargo admin has designated.
	allowedNamespaces := make(map[string]struct{})
	for _, ns := range a.cfg.OIDCConfig.GlobalServiceAccountNamespaces {
		allowedNamespaces[ns] = struct{}{}
	}
	nsList := &corev1.NamespaceList{}
	if err := a.kube.List(ctx, nsList, client.MatchingLabels{
		kargoapi.LabelKeyProject: kargoapi.LabelValueTrue,
	}); err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}
	for _, ns := range nsList.Items {
		allowedNamespaces[ns.GetName()] = struct{}{}
	}
	accounts := make(map[string]map[types.NamespacedName]struct{})
	for _, query := range queries {
		list := &corev1.ServiceAccountList{}
		if err := a.kube.List(ctx, list, query); err != nil {
			return nil, fmt.Errorf("list service accounts: %w", err)
		}
		for _, sa := range list.Items {
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

// newMultiClientVerifier returns a verifier that accepts a token issued to
// any of the configured OpenID Connect clients. There are commonly two, one
// for the web UI and one for the CLI, each needing a verifier of its own. It
// returns nil if the identity provider's keys could not be fetched, which the
// caller treats as a transient condition to retry.
func newMultiClientVerifier(ctx context.Context, cfg config.ServerConfig) verifyFunc {
	keyset, err := getKeySet(ctx, cfg)
	if err != nil {
		// The likely cause is a misconfigured issuer URL. In case it is a
		// transient network error instead, log it and let the first
		// authentication attempt retry.
		logging.LoggerFromContext(ctx).Error(
			err,
			"error getting keys from OpenID Connect provider; will try again on first authn attempt",
		)
		return nil
	}
	verifiers := []verifyFunc{
		oidc.NewVerifier(
			cfg.OIDCConfig.IssuerURL,
			keyset,
			&oidc.Config{ClientID: cfg.OIDCConfig.ClientID},
		).Verify,
	}
	if cfg.OIDCConfig.CLIClientID != "" {
		verifiers = append(verifiers, oidc.NewVerifier(
			cfg.OIDCConfig.IssuerURL,
			keyset,
			&oidc.Config{ClientID: cfg.OIDCConfig.CLIClientID},
		).Verify)
	}
	return func(ctx context.Context, rawIDToken string) (*oidc.IDToken, error) {
		errs := make([]error, 0, len(verifiers))
		for _, verify := range verifiers {
			token, err := verify(ctx, rawIDToken)
			if err == nil {
				return token, nil
			}
			errs = append(errs, err)
		}
		return nil, errors.Join(errs...)
	}
}

// getKeySet retrieves the key set from the OpenID Connect identity provider.
//
// This purposefully does not use oidc.NewProvider() and provider.Verifier()
// because they are not flexible enough to handle the Dex proxy case.
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
		if cfg.DexProxyConfig.CACertPath != "" {
			var caCertBytes []byte
			// #nosec G703 -- Contextually, this was an operator-specified path;
			// typically having been specified by the chart, and without even a
			// an option for specifying an alternate path.
			if caCertBytes, err = os.ReadFile(cfg.DexProxyConfig.CACertPath); err != nil {
				return nil, fmt.Errorf("error reading CA cert file %q: %w", cfg.DexProxyConfig.CACertPath, err)
			}
			caCertPool := x509.NewCertPool()
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

	// oidc.RemoteKeySet has an internal cache that is sometimes refreshed. It
	// uses a context-bound http.Client to make the request if one is
	// available, so the properly configured client is bound here.
	ctx = oidc.ClientContext(ctx, httpClient)
	return oidc.NewRemoteKeySet(ctx, keysURL), nil
}
