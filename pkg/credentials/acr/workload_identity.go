package acr

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/containers/azcontainerregistry"

	"github.com/akuity/kargo/pkg/cache/coalescing"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/logging"
)

const (
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
	if provider := NewWorkloadIdentityProvider(context.Background()); provider != nil {
		credentials.DefaultProviderRegistry.MustRegister(
			credentials.ProviderRegistration{
				Predicate: provider.Supports,
				Value:     provider,
			},
		)
	}
}

// WorkloadIdentityProvider implements credentials.Provider for Azure Container
// Registry Workload Identity.
type WorkloadIdentityProvider struct {
	// tokenCache is an in-memory cache of ACR registry access tokens keyed by
	// registry name. It fills its own misses, coalescing concurrent loads for any
	// given registry.
	tokenCache coalescing.Cache[string, string]

	credential azcore.TokenCredential

	getAccessTokenFn func(ctx context.Context, registryName string) (string, error)
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
	registryName := matches[1]

	logger := logging.LoggerFromContext(ctx).WithValues(
		"provider", "acrWorkloadIdentity",
		"repoURL", req.RepoURL,
	)
	ctx = logging.ContextWithLogger(ctx, logger)

	accessToken, err := p.tokenCache.Get(ctx, registryName, registryName)
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

// loadAccessToken obtains an ACR access token for the given registry. It is the
// Loader for this provider's token cache.
func (p *WorkloadIdentityProvider) loadAccessToken(
	ctx context.Context,
	registryName string,
) (string, *time.Duration, error) {
	logger := logging.LoggerFromContext(ctx)

	accessToken, err := p.getAccessTokenFn(ctx, registryName)
	if err != nil {
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

// getAccessToken returns an ACR refresh token using Azure workload identity.
func (p *WorkloadIdentityProvider) getAccessToken(
	ctx context.Context,
	registryName string,
) (string, error) {
	// Get Azure AD access token with the standard ACR scope
	token, err := p.credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{acrScope},
	})
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
