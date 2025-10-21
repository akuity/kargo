package acr

import (
	"context"
	"fmt"
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

// WorkloadIdentityProvider implements credentials.Provider for Azure Container Registry
// workload identity authentication.
type WorkloadIdentityProvider struct {
	tokenCache *cache.Cache
	credential azcore.TokenCredential

	getAccessTokenFn func(ctx context.Context, registryName string) (string, error)
}

// NewWorkloadIdentityProvider returns a new WorkloadIdentityProvider
// if Azure workload identity credentials are available.
func NewWorkloadIdentityProvider(ctx context.Context) credentials.Provider {
	logger := logging.LoggerFromContext(ctx)

	// Try to create a DefaultAzureCredential which supports workload identity
	credential, err := azidentity.NewDefaultAzureCredential(nil)
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
	credType credentials.Type,
	repoURL string,
	_ map[string][]byte,
	_ map[string]string,
) bool {
	if p.credential == nil {
		return false
	}

	if credType != credentials.TypeImage && credType != credentials.TypeHelm {
		return false
	}

	// Check if this is an ACR URL
	return acrURLRegex.MatchString(repoURL)
}

func (p *WorkloadIdentityProvider) GetCredentials(
	ctx context.Context,
	project string,
	credType credentials.Type,
	repoURL string,
	_ map[string][]byte,
	_ map[string]string,
) (*credentials.Credentials, error) {
	if !p.Supports(credType, repoURL, nil, nil) {
		return nil, nil
	}

	logger := logging.LoggerFromContext(ctx)

	// Extract the registry name from the ACR URL
	matches := acrURLRegex.FindStringSubmatch(repoURL)
	if len(matches) != 2 { // This doesn't look like an ACR URL
		return nil, nil
	}

	var (
		registryName = matches[1]
		cacheKey     = tokenCacheKey(registryName, project)
	)

	// Check the cache for the token
	if entry, exists := p.tokenCache.Get(cacheKey); exists {
		token := entry.(string) // nolint: forcetypeassert
		logger.Debug(
			"using cached ACR token from workload identity provider",
			"registry", registryName,
			"project", project,
		)
		return &credentials.Credentials{
			Username: acrTokenUsername,
			Password: token,
		}, nil
	}

	// Cache miss, get a new token
	accessToken, err := p.getAccessTokenFn(ctx, registryName)
	if err != nil {
		// Log the error but don't fail hard - this allows fallback to other
		// credential providers if ACR authentication fails
		logger.Error(
			err, "error getting ACR access token",
			"registry", registryName,
			"project", project,
		)
		return nil, fmt.Errorf("error getting ACR access token: %w", err)
	}

	// If we didn't get a token, we'll treat this as no credentials found
	if accessToken == "" {
		return nil, nil
	}

	// Cache the token
	p.tokenCache.Set(cacheKey, accessToken, cache.DefaultExpiration)

	logger.Debug(
		"obtained new ACR token from workload identity provider",
		"registry", registryName,
		"project", project,
	)

	return &credentials.Credentials{
		Username: acrTokenUsername,
		Password: accessToken,
	}, nil
}

// getAccessToken returns an ACR refresh token using Azure workload identity.
func (p *WorkloadIdentityProvider) getAccessToken(ctx context.Context, registryName string) (string, error) {
	logger := logging.LoggerFromContext(ctx).WithValues("registryName", registryName)

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

	// Exchange Azure AD token for ACR refresh token
	// Key fix: use registry hostname as service parameter (not full URL or just name)
	registryHostname := fmt.Sprintf("%s.azurecr.io", registryName)
	refreshTokenResp, err := authClient.ExchangeAADAccessTokenForACRRefreshToken(
		ctx,
		azcontainerregistry.PostContentSchemaGrantTypeAccessToken,
		registryHostname, // Use hostname format: "registryname.azurecr.io"
		&azcontainerregistry.AuthenticationClientExchangeAADAccessTokenForACRRefreshTokenOptions{
			AccessToken: &token.Token,
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to exchange Azure AD token for ACR refresh token: %w", err)
	}

	if refreshTokenResp.RefreshToken == nil {
		return "", fmt.Errorf("received empty ACR refresh token")
	}

	logger.Debug("successfully obtained ACR refresh token")
	return *refreshTokenResp.RefreshToken, nil
}
