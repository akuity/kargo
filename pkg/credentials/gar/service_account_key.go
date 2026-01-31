package gar

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/logging"
)

const (
	serviceAccountKeyKey = "gcpServiceAccountKey"
	scopeStorageRead     = "https://www.googleapis.com/auth/devstorage.read_only"
)

func init() {
	if provider := NewServiceAccountKeyProvider(); provider != nil {
		credentials.DefaultProviderRegistry.MustRegister(
			credentials.ProviderRegistration{
				Predicate: provider.Supports,
				Value:     provider,
			},
		)
	}
}

type ServiceAccountKeyProvider struct {
	tokenCache *cache.Cache

	getAccessTokenFn func(
		ctx context.Context,
		encodedServiceAccountKey string,
	) (*oauth2.Token, error)
}

func NewServiceAccountKeyProvider() credentials.Provider {
	p := &ServiceAccountKeyProvider{
		tokenCache: cache.New(
			// Access tokens live for one hour. We'll hang on to them for 40
			// minutes by default. When the actual token expiry is available, it
			// is used (minus a safety margin) instead of this default.
			40*time.Minute, // Default ttl for each entry
			time.Hour,      // Cleanup interval
		),
	}
	p.getAccessTokenFn = p.getAccessToken
	return p
}

func (p *ServiceAccountKeyProvider) Supports(
	_ context.Context,
	req credentials.Request,
) (bool, error) {
	if req.Type != credentials.TypeImage && req.Type != credentials.TypeHelm ||
		req.Data == nil ||
		req.Data[serviceAccountKeyKey] == nil {
		return false, nil
	}
	if !garURLRegex.MatchString(req.RepoURL) &&
		!gcrURLRegex.MatchString(req.RepoURL) {
		return false, nil
	}
	return true, nil
}

func (p *ServiceAccountKeyProvider) GetCredentials(
	ctx context.Context,
	req credentials.Request,
) (*credentials.Credentials, error) {
	encodedServiceAccountKey := string(req.Data[serviceAccountKeyKey])
	cacheKey := tokenCacheKey(encodedServiceAccountKey)

	logger := logging.LoggerFromContext(ctx).WithValues(
		"provider", "garServiceAccountKey",
		"repoURL", req.RepoURL,
	)

	// Check the cache for the token
	if entry, exists := p.tokenCache.Get(cacheKey); exists {
		logger.Debug("access token cache hit")
		return &credentials.Credentials{
			Username: accessTokenUsername,
			Password: entry.(string), // nolint: forcetypeassert
		}, nil
	}
	logger.Debug("access token cache miss")

	// Cache miss, get a new token
	token, err := p.getAccessTokenFn(ctx, encodedServiceAccountKey)
	if err != nil {
		return nil, fmt.Errorf("error getting GCP access token: %w", err)
	}

	// If we didn't get a token, we'll treat this as no credentials found
	if token == nil || token.AccessToken == "" {
		return nil, nil
	}
	logger.Debug("obtained new access token")

	// Cache the token, preferring a TTL derived from the actual token expiry
	// when available.
	ttl := cache.DefaultExpiration
	if !token.Expiry.IsZero() {
		if remaining := time.Until(token.Expiry) - tokenCacheExpiryMargin; remaining > 0 {
			ttl = remaining
		}
	}
	logger.Debug(
		"caching access token",
		"expiry", token.Expiry,
		"ttl", ttl,
	)
	p.tokenCache.Set(cacheKey, token.AccessToken, ttl)

	return &credentials.Credentials{
		Username: accessTokenUsername,
		Password: token.AccessToken,
	}, nil
}

// getAccessToken returns a GCP access token retrieved using the provided base64
// encoded service account key. The access token is valid for one hour.
func (p *ServiceAccountKeyProvider) getAccessToken(
	ctx context.Context,
	encodedServiceAccountKey string,
) (*oauth2.Token, error) {
	decodedKey, err := base64.StdEncoding.DecodeString(encodedServiceAccountKey)
	if err != nil {
		return nil, fmt.Errorf("error decoding service account key: %w", err)
	}

	config, err := google.JWTConfigFromJSON(decodedKey, scopeStorageRead)
	if err != nil {
		return nil, fmt.Errorf("error parsing service account key: %w", err)
	}

	tokenSource := config.TokenSource(ctx)
	token, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("error getting access token: %w", err)
	}
	return token, nil
}
