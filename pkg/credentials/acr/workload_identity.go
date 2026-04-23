package acr

import (
	"context"
	"fmt"
	"regexp"
	"strings"
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
	// cacheTTLMinutes is how long we cache tokens before refreshing them.
	// Set to 2.5 hours to ensure we refresh before the 3-hour ACR token expiry.
	cacheTTLMinutes = 150
	// cleanupIntervalMinutes is how often the cache cleanup runs
	cleanupIntervalMinutes = 30
	// acrScope is the Azure AD scope required for ACR authentication
	acrScope = "https://containerregistry.azure.net/.default"
	// adoScope is the Azure AD scope for Azure DevOps
	adoScope = "499b84ac-1321-427f-aa17-267ca6975798/.default"
	// azTokenUsername is the fixed username used for token authentication
	azTokenUsername = "00000000-0000-0000-0000-000000000000"
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
// Registry and Azure DevOps via Azure Workload Identity.
type WorkloadIdentityProvider struct {
	// tokenCache is an in-memory cache of access tokens keyed by ACR registry
	// name or (static) ADO scope.
	tokenCache *cache.Cache
	credential azcore.TokenCredential

	getAccessTokenFn func(ctx context.Context, credentialsType credentials.Type, registryName string) (string, time.Duration, error)
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
			// ACR refresh tokens expire in 3 hours. We'll hang on to them for
			// 2.5 hours. The ACR refresh token exchange API does not expose
			// actual token expiry, so a dynamic TTL is not possible here.
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
	switch req.Type {
	case credentials.TypeImage, credentials.TypeHelm:
		// Check if this is an ACR URL
		return acrURLRegex.MatchString(req.RepoURL), nil
	case credentials.TypeGit:
		return strings.HasPrefix(req.RepoURL, "http://") || strings.HasPrefix(req.RepoURL, "https://"), nil
	default:
		return false, nil
	}
}

func (p *WorkloadIdentityProvider) GetCredentials(
	ctx context.Context,
	req credentials.Request,
) (*credentials.Credentials, error) {
	var cacheKey string
	switch req.Type {
	case credentials.TypeImage, credentials.TypeHelm:
		// Extract the registry name from the ACR URL
		matches := acrURLRegex.FindStringSubmatch(req.RepoURL)
		if len(matches) != 2 { // This doesn't look like an ACR URL
			return nil, nil
		}
		cacheKey = matches[1]
	case credentials.TypeGit:
		// Use the Azure DevOps scope as the cache key
		cacheKey = adoScope
	default:
		return nil, fmt.Errorf("invalid credentials type: %s", req.Type)
	}

	logger := logging.LoggerFromContext(ctx).WithValues(
		"provider", "acrWorkloadIdentity",
		"repoURL", req.RepoURL,
	)

	// Check the cache for the token
	if entry, exists := p.tokenCache.Get(cacheKey); exists {
		logger.Debug("access token cache hit")
		return &credentials.Credentials{
			Username: azTokenUsername,
			Password: entry.(string), // nolint: forcetypeassert
		}, nil
	}
	logger.Debug("access token cache miss")

	// Cache miss, get a new token
	accessToken, ttl, err := p.getAccessTokenFn(ctx, req.Type, cacheKey)
	if err != nil {
		return nil, fmt.Errorf("error getting access token: %w", err)
	}

	// If we didn't get a token, we'll treat this as no credentials found
	if accessToken == "" {
		return nil, nil
	}
	logger.Debug("obtained new access token")

	// Use the token TTL if available, otherwise fall back to the default.
	// In general, Azure AD exposes token expiry, but the ACR refresh token
	// exchange API does not.
	if ttl == 0 {
		ttl = cache.DefaultExpiration
	}
	logger.Debug(
		"caching access token",
		"ttl", ttl,
	)
	p.tokenCache.Set(cacheKey, accessToken, ttl)

	return &credentials.Credentials{
		Username: azTokenUsername,
		Password: accessToken,
	}, nil
}

func (p *WorkloadIdentityProvider) getAccessToken(
	ctx context.Context,
	credentialsType credentials.Type,
	registryName string,
) (string, time.Duration, error) {
	switch credentialsType {
	case credentials.TypeImage, credentials.TypeHelm:
		token, err := p.getAcrAccessToken(ctx, registryName)
		return token, 0, err
	case credentials.TypeGit:
		return p.getAdoAccessToken(ctx)
	default:
		return "", 0, fmt.Errorf("invalid credentials type: %s", credentialsType)
	}
}

// getAcrAccessToken returns an ACR refresh token using Azure workload identity.
func (p *WorkloadIdentityProvider) getAcrAccessToken(
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

// getAdoAccessToken returns an ADO access token using Azure workload identity.
func (p *WorkloadIdentityProvider) getAdoAccessToken(ctx context.Context) (string, time.Duration, error) {
	// Get Azure AD access token with the standard ADO scope
	token, err := p.credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{adoScope},
	})
	if err != nil {
		return "", 0, fmt.Errorf("failed to get Azure AD access token for ADO: %w", err)
	}
	// Return the time until the explicit refresh-on if provided, otherwise fall
	// back to the expires-on minus 5 minutes (to be safe).
	duration := time.Duration(0)
	if !token.RefreshOn.IsZero() {
		duration = time.Until(token.RefreshOn)
	} else if !token.ExpiresOn.IsZero() {
		duration = time.Until(token.ExpiresOn) - time.Minute*5
	}
	return token.Token, duration, nil
}
