package gar

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/oauth2/google"
	corev1 "k8s.io/api/core/v1"

	"github.com/akuity/kargo/internal/credentials"
)

const serviceAccountKeyKey = "gcpServiceAccountKey"

type serviceAccountKeyCredentialHelper struct {
	tokenCache *cache.Cache

	// The following behaviors are overridable for testing purposes:

	getAccessTokenFn func(context.Context, string) (string, error)
}

// NewServiceAccountKeyCredentialHelper returns an implementation
// credentials.Helper that utilizes a cache to avoid unnecessary calls to GCP.
func NewServiceAccountKeyCredentialHelper() credentials.Helper {
	s := &serviceAccountKeyCredentialHelper{
		tokenCache: cache.New(
			// Access tokens live for one hour. We'll hang on to them for 40 minutes.
			40*time.Minute, // Default ttl for each entry
			time.Hour,      // Cleanup interval
		),
	}
	s.getAccessTokenFn = s.getAccessToken
	return s.getCredentials
}

func (s *serviceAccountKeyCredentialHelper) getCredentials(
	ctx context.Context,
	_ string,
	credType credentials.Type,
	repoURL string,
	secret *corev1.Secret,
) (*credentials.Credentials, error) {
	if credType != credentials.TypeImage || secret == nil {
		// This helper can't handle this
		return nil, nil
	}

	if !garURLRegex.MatchString(repoURL) && !gcrURLRegex.MatchString(repoURL) {
		// This doesn't look like a Google Artifact Registry URL
		return nil, nil
	}

	// This should be base64 encoded
	encodedServiceAccountKey := string(secret.Data[serviceAccountKeyKey])
	if encodedServiceAccountKey == "" {
		return nil, nil
	}

	cacheKey := s.tokenCacheKey(encodedServiceAccountKey)

	if entry, exists := s.tokenCache.Get(cacheKey); exists {
		return &credentials.Credentials{
			Username: accessTokenUsername,
			Password: entry.(string), // nolint: forcetypeassert
		}, nil
	}

	accessToken, err := s.getAccessTokenFn(ctx, encodedServiceAccountKey)
	if err != nil {
		return nil, fmt.Errorf("error getting GCP access token: %w", err)
	}

	if accessToken == "" {
		return nil, nil
	}

	// Cache the access token
	s.tokenCache.Set(cacheKey, accessToken, cache.DefaultExpiration)

	return &credentials.Credentials{
		Username: accessTokenUsername,
		Password: accessToken,
	}, nil
}

// tokenCacheKey returns a cache key for a GCP access token. The key is a hash
// of the provided (encoded) service account key. Using a hash ensures that a
// decodable service account key is not stored in the cache.
func (s *serviceAccountKeyCredentialHelper) tokenCacheKey(encodedServiceAccountKey string) string {
	return fmt.Sprintf(
		"%x",
		sha256.Sum256([]byte(encodedServiceAccountKey)),
	)
}

// getAccessToken returns a GCP access token retrieved using the provided base64
// encoded service account key. The access token is valid for one hour.
func (s *serviceAccountKeyCredentialHelper) getAccessToken(
	ctx context.Context,
	encodedServiceAccountKey string,
) (string, error) {
	decodedKey, err := base64.StdEncoding.DecodeString(encodedServiceAccountKey)
	if err != nil {
		return "", fmt.Errorf("error decoding service account key: %w", err)
	}
	config, err := google.JWTConfigFromJSON(decodedKey, "https://www.googleapis.com/auth/devstorage.read_only")
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
