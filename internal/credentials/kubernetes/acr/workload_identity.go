package acr

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/containers/azcontainerregistry"
	"github.com/patrickmn/go-cache"

	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/logging"
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
			// ACR refresh tokens are typically valid for 3 hours. We'll cache them for 2.5 hours.
			150*time.Minute, // Default ttl for each entry
			30*time.Minute,  // Cleanup interval
		),
		credential: credential,
	}
	p.getAccessTokenFn = p.getAccessToken
	return p
}

func (p *WorkloadIdentityProvider) Supports(credType credentials.Type, repoURL string, _ map[string][]byte) bool {
	if p.credential == nil {
		return false
	}

	if credType != credentials.TypeImage && credType != credentials.TypeHelm {
		return false
	}

	if credType == credentials.TypeHelm && !strings.HasPrefix(repoURL, "oci://") {
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
) (*credentials.Credentials, error) {
	if !p.Supports(credType, repoURL, nil) {
		return nil, nil
	}

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
		return &credentials.Credentials{
			Username: "00000000-0000-0000-0000-000000000000", // ACR username for token auth
			Password: token,
		}, nil
	}

	// Cache miss, get a new token
	accessToken, err := p.getAccessTokenFn(ctx, registryName)
	if err != nil {
		// Log the error but don't fail hard - this allows fallback to other
		// credential providers if ACR authentication fails
		logging.LoggerFromContext(ctx).Error(
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

	return &credentials.Credentials{
		Username: "00000000-0000-0000-0000-000000000000", // ACR username for token auth
		Password: accessToken,
	}, nil
}

// getAccessToken returns an ACR refresh token using Azure workload identity.
func (p *WorkloadIdentityProvider) getAccessToken(ctx context.Context, registryName string) (string, error) {
	logger := logging.LoggerFromContext(ctx).WithValues("registryName", registryName)

	// Get Azure AD access token with the standard ACR scope
	scope := "https://containerregistry.azure.net/.default"
	token, err := p.credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{scope},
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
