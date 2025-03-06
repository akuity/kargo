package gar

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/oauth2/google"

	"github.com/akuity/kargo/internal/credentials"
)

const (
	serviceAccountKeyKey = "gcpServiceAccountKey"

	scopeStorageRead = "https://www.googleapis.com/auth/devstorage.read_only"
)

type ServiceAccountKeyProvider struct {
	tokenCache *cache.Cache

	getAccessTokenFn func(ctx context.Context, encodedServiceAccountKey string) (string, error)
}

func NewServiceAccountKeyProvider() *ServiceAccountKeyProvider {
	p := &ServiceAccountKeyProvider{
		tokenCache: cache.New(
			// Access tokens live for one hour. We'll hang on to them for 40 minutes.
			40*time.Minute, // Default ttl for each entry
			time.Hour,      // Cleanup interval
		),
	}
	p.getAccessTokenFn = p.getAccessToken
	return p
}

func (p *ServiceAccountKeyProvider) Supports(credType credentials.Type, repoURL string, data map[string][]byte) bool {
	if credType != credentials.TypeImage || data == nil || data[serviceAccountKeyKey] == nil {
		return false
	}

	if !garURLRegex.MatchString(repoURL) && !gcrURLRegex.MatchString(repoURL) {
		return false
	}

	return true
}

func (p *ServiceAccountKeyProvider) GetCredentials(
	ctx context.Context,
	_ string,
	credType credentials.Type,
	repoURL string,
	data map[string][]byte,
) (*credentials.Credentials, error) {
	if !p.Supports(credType, repoURL, data) {
		return nil, nil
	}

	var (
		encodedServiceAccountKey = string(data[serviceAccountKeyKey])
		cacheKey                 = tokenCacheKey(encodedServiceAccountKey)
	)

	// Check the cache for the token
	if entry, exists := p.tokenCache.Get(cacheKey); exists {
		return &credentials.Credentials{
			Username: accessTokenUsername,
			Password: entry.(string), // nolint: forcetypeassert
		}, nil
	}

	// Cache miss, get a new token
	accessToken, err := p.getAccessTokenFn(ctx, encodedServiceAccountKey)
	if err != nil {
		return nil, fmt.Errorf("error getting GCP access token: %w", err)
	}

	// If we didn't get a token, we'll treat this as no credentials found
	if accessToken == "" {
		return nil, nil
	}

	// Cache the token
	p.tokenCache.Set(cacheKey, accessToken, cache.DefaultExpiration)

	return &credentials.Credentials{
		Username: accessTokenUsername,
		Password: accessToken,
	}, nil
}

// getAccessToken returns a GCP access token retrieved using the provided base64
// encoded service account key. The access token is valid for one hour.
func (p *ServiceAccountKeyProvider) getAccessToken(
	ctx context.Context,
	encodedServiceAccountKey string,
) (string, error) {
	decodedKey, err := base64.StdEncoding.DecodeString(encodedServiceAccountKey)
	if err != nil {
		return "", fmt.Errorf("error decoding service account key: %w", err)
	}

	config, err := google.JWTConfigFromJSON(decodedKey, scopeStorageRead)
	if err != nil {
		return "", fmt.Errorf("error parsing service account key: %w", err)
	}

	tokenSource := config.TokenSource(ctx)
	token, err := tokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("error getting access token: %w", err)
	}
	return token.AccessToken, nil
}
