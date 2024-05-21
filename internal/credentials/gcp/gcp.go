package gcp

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/oauth2/google"
	corev1 "k8s.io/api/core/v1"
)

const (
	serviceAccountKeyKey = "gcpServiceAccountKey"

	accessTokenUsername = "oauth2accesstoken"
)

// CredentialHelper is an interface for components that can extract a username
// and password for accessing Google Artifact Registry from a Secret containing
// a base64 encoded GCP service account key.
type CredentialHelper interface {
	// GetUsernameAndPassword extracts username and password (an access token that
	// lives for one hour) from a Secret IF the Secret contains a base64 encoded
	// GCP service account key. If the Secret does not contain such a key, this
	// function will return empty strings and a nil error. Implementations may
	// cache the access token for efficiency.
	GetUsernameAndPassword(context.Context, *corev1.Secret) (string, string, error)
}

type credentialHelper struct {
	tokenCache *cache.Cache

	// The following behaviors are overridable for testing purposes:

	getAccessTokenFn func(context.Context, string) (string, error)
}

// NewCredentialHelper returns an implementation of the CredentialHelper
// interface that utilizes a cache to avoid unnecessary calls to GCP.
func NewCredentialHelper() CredentialHelper {
	return &credentialHelper{
		tokenCache: cache.New(
			// Access tokens live for one hour. We'll hang on to them for 40 minutes.
			40*time.Minute, // Default ttl for each entry
			time.Hour,      // Cleanup interval
		),
		getAccessTokenFn: getAccessToken,
	}
}

// GetUsernameAndPassword implements the CredentialHelper interface.
func (c *credentialHelper) GetUsernameAndPassword(
	ctx context.Context,
	secret *corev1.Secret,
) (string, string, error) {
	// This should be base64 encoded
	encodedServiceAccountKey := string(secret.Data[serviceAccountKeyKey])
	if encodedServiceAccountKey == "" {
		return "", "", nil
	}

	cacheKey := tokenCacheKey(encodedServiceAccountKey)

	if entry, exists := c.tokenCache.Get(cacheKey); exists {
		return accessTokenUsername, entry.(string), nil // nolint: forcetypeassert
	}

	accessToken, err := c.getAccessTokenFn(ctx, encodedServiceAccountKey)
	if err != nil {
		return "", "", fmt.Errorf("error getting GCP access token: %w", err)
	}

	// Cache the access token
	c.tokenCache.Set(cacheKey, accessToken, cache.DefaultExpiration)

	return accessTokenUsername, accessToken, nil
}

// tokenCacheKey returns a cache key for a GCP access token. The key is a hash
// of the provided (encoded) service account key. Using a hash ensures that a
// decodable service account key is not stored in the cache.
func tokenCacheKey(encodedServiceAccountKey string) string {
	return fmt.Sprintf(
		"%x",
		sha256.Sum256([]byte(encodedServiceAccountKey)),
	)
}

// getAccessToken returns a GCP access token retrieved using the provided base64
// encoded service account key. The access token is valid for one hour.
func getAccessToken(
	ctx context.Context,
	encodedServiceAccountKey string,
) (string, error) {
	decodedKey, err := base64.StdEncoding.DecodeString(encodedServiceAccountKey)
	if err != nil {
		return "", fmt.Errorf("error decoding service account key: %w", err)
	}
	config, err := google.JWTConfigFromJSON(decodedKey, "https://www.googleapis.com/auth/devstorage.read_write")
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
