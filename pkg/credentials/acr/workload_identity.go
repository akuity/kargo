package acr

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/containers/azcontainerregistry"
	"github.com/patrickmn/go-cache"

	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/logging"
)

const (
	// cacheTTLMinutes is how long we cache ACR tokens before refreshing them.
	// Set to 2.5 hours to ensure we refresh before the 3-hour token expiry.
	cacheTTLMinutes = 150
	// cleanupIntervalMinutes is how often the cache cleanup runs
	cleanupIntervalMinutes = 30
	// acrTokenUsername is the fixed username used for ACR token authentication
	acrTokenUsername = "00000000-0000-0000-0000-000000000000"
	// acrScope is the Azure AD scope required for ACR authentication
	acrScope = "https://containerregistry.azure.net/.default"
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
	// registry name.
	tokenCache *cache.Cache
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

	p := &WorkloadIdentityProvider{
		tokenCache: cache.New(
			cacheTTLMinutes*time.Minute,        // Default ttl for each entry
			cleanupIntervalMinutes*time.Minute, // Cleanup interval
		),
		credential: credential,
	}
	p.getAccessTokenFn = p.getAccessToken
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

	// Check the cache for the token
	if entry, exists := p.tokenCache.Get(registryName); exists {
		return &credentials.Credentials{
			Username: acrTokenUsername,
			Password: entry.(string), // nolint: forcetypeassert
		}, nil
	}

	// Cache miss, get a new token
	accessToken, err := p.getAccessTokenFn(ctx, registryName)
	if err != nil {
		return nil, fmt.Errorf("error getting ACR access token: %w", err)
	}

	// If we didn't get a token, we'll treat this as no credentials found
	if accessToken == "" {
		return nil, nil
	}

	// Cache the token
	p.tokenCache.Set(registryName, accessToken, cache.DefaultExpiration)

	return &credentials.Credentials{
		Username: acrTokenUsername,
		Password: accessToken,
	}, nil
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
		return "", fmt.Errorf("received empty ACR refresh token")
	}

	return *refreshTokenResp.RefreshToken, nil
}
