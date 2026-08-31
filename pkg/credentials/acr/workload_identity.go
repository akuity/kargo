package acr

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/containers/azcontainerregistry"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"

	"github.com/akuity/kargo/pkg/cache/coalescing"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/logging"
)

const (
	// armScope is the Entra scope a token is obtained for solely to read the
	// claims it carries about the identity it was issued to.
	armScope = "https://management.azure.com/.default"
	// selfDiscoveryTimeout bounds the token acquisition that locates the
	// controller's own identity. It runs while the process is starting, so it is
	// short: failing to discover leaves Project-specific identities disabled,
	// which is a working state, whereas a hung start is not.
	selfDiscoveryTimeout = 10 * time.Second

	// cacheTTL is how long an ACR token is cached for. ACR refresh tokens expire
	// after three hours and the exchange API does not report a token's actual
	// expiry, so no TTL can be derived from one and this is used for all of them.
	cacheTTL = 150 * time.Minute
	// cleanupInterval is how often expired tokens are evicted from the cache.
	cleanupInterval = 30 * time.Minute

	// acrTokenUsername is the fixed username used for ACR token authentication
	acrTokenUsername = "00000000-0000-0000-0000-000000000000"
	// acrScope is the Azure AD scope required for ACR authentication
	acrScope = "https://containerregistry.azure.net/.default"

	// tokenAcquisitionTimeout bounds a single token acquisition. Because an
	// acquisition executes under a context detached from any caller's, this is
	// the only thing bounding its duration. It is generous because it serves only
	// as a fail-safe: exceeding it fails every caller waiting on that
	// acquisition, and no fresh acquisition for the same registry can begin until
	// it returns.
	tokenAcquisitionTimeout = 30 * time.Second
)

// acrURLRegex matches Azure Container Registry URLs.
// Pattern matches: <registry-name>.azurecr.io
var acrURLRegex = regexp.MustCompile(`^([a-zA-Z0-9-]+)\.azurecr\.io/`)

func init() {
	if !credentials.ProvidersEnabled() {
		return
	}
	if provider := NewWorkloadIdentityProvider(context.Background()); provider != nil {
		credentials.DefaultProviderRegistry.MustRegister(
			credentials.ProviderRegistration{
				Predicate: provider.Supports,
				Value:     provider,
			},
		)
	}
}

// tokenRequest identifies the token a single acquisition obtains. A Project is
// part of it because a single controller acts for many Projects against a
// single registry, and each is entitled to a different level of access.
type tokenRequest struct {
	project      string
	registryName string
}

// cacheKey returns the key under which this request's token is cached.
func (t tokenRequest) cacheKey() string {
	return t.project + "/" + t.registryName
}

// WorkloadIdentityProvider implements credentials.Provider for Azure Container
// Registry Workload Identity.
type WorkloadIdentityProvider struct {
	// tokenCache is an in-memory cache of ACR registry access tokens keyed by
	// Project and registry name. It fills its own misses, coalescing concurrent
	// loads for any given key.
	tokenCache coalescing.Cache[tokenRequest, string]

	// credential is the controller's own identity.
	credential azcore.TokenCredential

	// resourceGroup and identities are what acting as a Project-specific
	// identity requires. They are unset when the prerequisites for doing so are
	// absent, in which case the controller's own identity is the only one used.
	resourceGroup string
	identities    userAssignedIdentityGetter

	getAccessTokenFn func(context.Context, tokenRequest) (string, error)
	exchangeFn       func(
		ctx context.Context,
		cred azcore.TokenCredential,
		registryName string,
	) (string, error)
}

// NewWorkloadIdentityProvider returns a new WorkloadIdentityProvider if Azure
// workload identity credentials are available. Otherwise, it returns nil.
func NewWorkloadIdentityProvider(ctx context.Context) credentials.Provider {
	logger := logging.LoggerFromContext(ctx)

	// Try to create a DefaultAzureCredential which supports workload identity
	credential, err := azidentity.NewWorkloadIdentityCredential(nil)
	if err != nil {
		logger.Info("Azure workload identity not available", "error", err.Error())
		return nil
	}

	logger.Info("Azure workload identity credential provider initialized")

	p := &WorkloadIdentityProvider{credential: credential}
	p.getAccessTokenFn = p.getAccessToken
	p.exchangeFn = p.exchangeForACRToken
	p.initProjectIdentities(ctx)

	tokenCache, err := coalescing.NewCache(
		p.loadAccessToken,
		&coalescing.CacheOptions{
			LoadTimeout:     new(tokenAcquisitionTimeout),
			DefaultTTL:      new(cacheTTL),
			CleanupInterval: new(cleanupInterval),
		},
	)
	if err != nil {
		logger.Error(
			err, "error creating token cache; ACR credentials integration will be disabled",
		)
		return nil
	}
	p.tokenCache = tokenCache
	return p
}

// initProjectIdentities equips the provider to act as Project-specific
// identities. Those are looked up in the subscription and resource group
// holding the controller's own identity, which a token issued to that identity
// names, so an operator states nothing. Failing to discover any of it leaves
// the provider able to act only as the controller's own identity, which is what
// it falls back to in any case.
func (p *WorkloadIdentityProvider) initProjectIdentities(ctx context.Context) {
	logger := logging.LoggerFromContext(ctx)

	subscriptionID, resourceGroup, err := p.discoverIdentityLocation(ctx)
	if err != nil {
		logger.Info(
			"Azure Project-specific identities are unavailable; the controller's "+
				"own identity will be used for all Projects",
			"error", err.Error(),
		)
		return
	}

	identities, err := armmsi.NewUserAssignedIdentitiesClient(
		subscriptionID, p.credential, nil,
	)
	if err != nil {
		logger.Error(err, "error creating managed identity client")
		return
	}

	p.resourceGroup = resourceGroup
	p.identities = identities

	logger.Info(
		"Azure Project-specific identities enabled",
		"resourceGroup", resourceGroup,
	)
}

func (p *WorkloadIdentityProvider) discoverIdentityLocation(
	ctx context.Context,
) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, selfDiscoveryTimeout)
	defer cancel()

	token, err := p.credential.GetToken(
		ctx,
		policy.TokenRequestOptions{Scopes: []string{armScope}},
	)
	if err != nil {
		return "", "", fmt.Errorf("error obtaining a token for the controller's own identity: %w", err)
	}
	return parseIdentityResourceID(token.Token)
}

// parseIdentityResourceID returns the subscription and resource group named by
// the xms_mirid claim of the given token. The token is not verified; Entra
// issued it moments earlier and it is read only for what it says about its own
// subject.
func parseIdentityResourceID(token string) (string, string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", "", errors.New("token is not a JWT")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", fmt.Errorf("error decoding token payload: %w", err)
	}
	claims := struct {
		ResourceID string `json:"xms_mirid"`
	}{}
	if err = json.Unmarshal(payload, &claims); err != nil {
		return "", "", fmt.Errorf("error unmarshaling token payload: %w", err)
	}
	if claims.ResourceID == "" {
		return "", "", errors.New(
			"token carries no xms_mirid claim; the controller does not appear to " +
				"run as a user-assigned managed identity",
		)
	}

	// ARM resource IDs are of the form /<type>/<name>/<type>/<name>/..., and ARM
	// is not consistent about the case of the type segments.
	var subscriptionID, resourceGroup string
	segments := strings.Split(strings.Trim(claims.ResourceID, "/"), "/")
	for i := 0; i+1 < len(segments); i += 2 {
		switch strings.ToLower(segments[i]) {
		case "subscriptions":
			subscriptionID = segments[i+1]
		case "resourcegroups":
			resourceGroup = segments[i+1]
		}
	}
	if subscriptionID == "" || resourceGroup == "" {
		return "", "", fmt.Errorf(
			"could not read a subscription and resource group from %q",
			claims.ResourceID,
		)
	}
	return subscriptionID, resourceGroup, nil
}

func (p *WorkloadIdentityProvider) Supports(
	_ context.Context,
	req credentials.Request,
) (bool, error) {
	if req.Type != credentials.TypeImage && req.Type != credentials.TypeHelm {
		return false, nil
	}
	// Check if this is an ACR URL
	return acrURLRegex.MatchString(req.RepoURL), nil
}

func (p *WorkloadIdentityProvider) GetCredentials(
	ctx context.Context,
	req credentials.Request,
) (*credentials.Credentials, error) {
	// Extract the registry name from the ACR URL
	matches := acrURLRegex.FindStringSubmatch(req.RepoURL)
	if len(matches) != 2 { // This doesn't look like an ACR URL
		return nil, nil
	}
	tokenReq := tokenRequest{
		project:      req.Project,
		registryName: matches[1],
	}

	logger := logging.LoggerFromContext(ctx).WithValues(
		"provider", "acrWorkloadIdentity",
		"project", req.Project,
		"repoURL", req.RepoURL,
	)
	ctx = logging.ContextWithLogger(ctx, logger)

	accessToken, err := p.tokenCache.Get(ctx, tokenReq.cacheKey(), tokenReq)
	if err != nil {
		return nil, err
	}

	// If we didn't get a token, we'll treat this as no credentials found
	if accessToken == "" {
		return nil, nil
	}

	return &credentials.Credentials{
		Username: acrTokenUsername,
		Password: accessToken,
	}, nil
}

// loadAccessToken obtains an ACR access token for the given request. It is the
// Loader for this provider's token cache.
func (p *WorkloadIdentityProvider) loadAccessToken(
	ctx context.Context,
	req tokenRequest,
) (string, *time.Duration, error) {
	logger := logging.LoggerFromContext(ctx)

	accessToken, err := p.getAccessTokenFn(ctx, req)
	if err != nil {
		// TODO(krancour): Failures are not cached, so a Project that can never
		// obtain a token re-attempts as often as its callers ask rather than at the
		// interval a cached token would imply. Attempting the Project-specific
		// identity first makes each of those attempts cost more than it used to.
		// This would be nice to solve.
		return "", nil, fmt.Errorf("error getting ACR access token: %w", err)
	}

	// If we didn't get a token, we'll treat this as no credentials found
	if accessToken == "" {
		return "", nil, nil
	}
	logger.Debug("obtained new access token")

	// The ACR refresh token exchange API does not expose token expiry, so no TTL
	// of the token's own can be computed. A TTL of zero defers to the cache's
	// default.
	logger.Debug(
		"caching access token",
		"ttl", cacheTTL, // This is the default TTL the cache is actually using
	)
	return accessToken,
		new(time.Duration), // zero TTL defers to the cache's default TTL
		nil
}

// getAccessToken returns an ACR refresh token for the given request. It first
// attempts to obtain one as the Project's own Azure identity, which is what
// confines a Project to the registries it has been granted. Failing that, it
// obtains one as the controller's own identity, which forgoes Project-level
// isolation and is therefore the last resort.
func (p *WorkloadIdentityProvider) getAccessToken(
	ctx context.Context,
	req tokenRequest,
) (string, error) {
	logger := logging.LoggerFromContext(ctx)

	if cred := p.projectCredential(ctx, req.project); cred != nil {
		accessToken, err := p.exchangeFn(ctx, cred, req.registryName)
		if err == nil {
			logger.Debug("obtained token as Project-specific identity")
			return accessToken, nil
		}
		logger.Debug(
			"error obtaining token as Project-specific identity; "+
				"falling back to the controller's own identity",
			"error", err.Error(),
		)
	}

	return p.exchangeFn(ctx, p.credential, req.registryName)
}

// projectCredential returns a credential for the given Project's own Azure
// identity, or nil if one cannot be obtained. A nil return is not an error
// condition: a Project without an identity of its own is expected, and the
// caller falls back to the controller's own identity.
func (p *WorkloadIdentityProvider) projectCredential(
	ctx context.Context,
	project string,
) azcore.TokenCredential {
	logger := logging.LoggerFromContext(ctx)

	if p.identities == nil || project == "" {
		return nil
	}

	// IMPORTANT: The identity is named by the Project being acted on and is
	// never taken from a request. Every Project-specific identity trusts the
	// same ServiceAccount, so Entra cannot tell one Project from another and
	// this derivation is the only thing confining a Project to its own
	// identity. An identity a Project could influence would make this provider
	// a confused deputy.
	clientID, err := p.resolveClientID(ctx, project)
	if err != nil {
		if errors.Is(err, errNoProjectIdentity) {
			logger.Debug("no Project-specific identity exists")
		} else {
			logger.Error(err, "error resolving Project-specific identity")
		}
		return nil
	}

	// The assertion is the controller's own projected ServiceAccount token, and
	// the tenant is the controller's own tenant. Only the identity the token is
	// redeemed for differs, so only that is specified here; the rest is read
	// from the environment Azure's workload identity webhook established.
	cred, err := azidentity.NewWorkloadIdentityCredential(
		&azidentity.WorkloadIdentityCredentialOptions{ClientID: clientID},
	)
	if err != nil {
		logger.Error(err, "error creating Project-specific credential")
		return nil
	}
	return cred
}

// exchangeForACRToken returns an ACR refresh token for the given registry,
// obtained as the given identity.
func (p *WorkloadIdentityProvider) exchangeForACRToken(
	ctx context.Context,
	cred azcore.TokenCredential,
	registryName string,
) (string, error) {
	// Get Azure AD access token with the standard ACR scope
	token, err := cred.GetToken(
		ctx,
		policy.TokenRequestOptions{Scopes: []string{acrScope}},
	)
	if err != nil {
		return "", fmt.Errorf("failed to get Azure AD access token for ACR: %w", err)
	}

	// Create authentication client for token exchange
	serviceURL := fmt.Sprintf("https://%s.azurecr.io", registryName)
	authClient, err := azcontainerregistry.NewAuthenticationClient(serviceURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create ACR authentication client: %w", err)
	}

	// Exchange Azure AD token for ACR refresh token.
	//
	// Note: Despite Azure's naming, this "refresh token" is actually used as an
	// access token. i.e. It's what's provide as the password when authenticating
	// using any OCI client.
	refreshTokenResp, err := authClient.ExchangeAADAccessTokenForACRRefreshToken(
		ctx,
		azcontainerregistry.PostContentSchemaGrantTypeAccessToken,
		fmt.Sprintf("%s.azurecr.io", registryName),
		&azcontainerregistry.AuthenticationClientExchangeAADAccessTokenForACRRefreshTokenOptions{
			AccessToken: &token.Token,
		},
	)
	if err != nil {
		return "", fmt.Errorf(
			"failed to exchange Azure AD token for ACR refresh token: %w", err,
		)
	}

	if refreshTokenResp.RefreshToken == nil {
		return "", errors.New("received empty ACR refresh token")
	}

	return *refreshTokenResp.RefreshToken, nil
}
