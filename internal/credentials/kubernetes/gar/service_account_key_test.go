package gar

import (
	"context"
	"errors"
	"testing"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"

	"github.com/akuity/kargo/internal/credentials"
)

func TestNewServiceAccountKeyProvider(t *testing.T) {
	provider := NewServiceAccountKeyProvider().(*ServiceAccountKeyProvider) // nolint:forcetypeassert
	assert.NotNil(t, provider)

	assert.NotNil(t, provider.tokenCache)
	assert.NotNil(t, provider.getAccessTokenFn)
}

func TestServiceAccountKeyProvider_Supports(t *testing.T) {
	const (
		fakeGCRRepoURL        = "gcr.io/my-project/my-repo"
		fakeGARRepoURL        = "us-central1-docker.pkg.dev/my-project/my-repo"
		fakeServiceAccountKey = "base64-encoded-service-account-key"
	)

	testCases := []struct {
		name     string
		credType credentials.Type
		repoURL  string
		data     map[string][]byte
		expected bool
	}{
		{
			name:     "valid GAR repo with service account key",
			credType: credentials.TypeImage,
			repoURL:  fakeGARRepoURL,
			data: map[string][]byte{
				serviceAccountKeyKey: []byte(fakeServiceAccountKey),
			},
			expected: true,
		},
		{
			name:     "valid GCR repo with service account key",
			credType: credentials.TypeImage,
			repoURL:  fakeGCRRepoURL,
			data: map[string][]byte{
				serviceAccountKeyKey: []byte(fakeServiceAccountKey),
			},
			expected: true,
		},
		{
			name:     "wrong credential type",
			credType: credentials.TypeHelm,
			repoURL:  fakeGARRepoURL,
			data: map[string][]byte{
				serviceAccountKeyKey: []byte(fakeServiceAccountKey),
			},
			expected: false,
		},
		{
			name:     "missing service account key",
			credType: credentials.TypeImage,
			repoURL:  fakeGARRepoURL,
			data:     map[string][]byte{},
			expected: false,
		},
		{
			name:     "nil data",
			credType: credentials.TypeImage,
			repoURL:  fakeGARRepoURL,
			data:     nil,
			expected: false,
		},
		{
			name:     "non-GAR/GCR URL",
			credType: credentials.TypeImage,
			repoURL:  "docker.io/library/nginx",
			data: map[string][]byte{
				serviceAccountKeyKey: []byte(fakeServiceAccountKey),
			},
			expected: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewServiceAccountKeyProvider()
			result := provider.Supports(tt.credType, tt.repoURL, tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestServiceAccountKeyProvider_GetCredentials(t *testing.T) {
	ctx := context.Background()

	const (
		fakeGARRepoURL        = "us-central1-docker.pkg.dev/my-project/my-repo"
		fakeServiceAccountKey = "base64-encoded-service-account-key"
		fakeAccessToken       = "fake-access-token"
	)

	testCases := []struct {
		name             string
		credType         credentials.Type
		repoURL          string
		data             map[string][]byte
		getAccessTokenFn func(ctx context.Context, encodedServiceAccountKey string) (string, error)
		setupCache       func(c *cache.Cache)
		assertions       func(t *testing.T, c *cache.Cache, creds *credentials.Credentials, err error)
	}{
		{
			name:     "unsupported credentials",
			credType: credentials.TypeHelm,
			repoURL:  fakeGARRepoURL,
			data:     map[string][]byte{},
			assertions: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.NoError(t, err)
			},
		},
		{
			name:     "cache hit",
			credType: credentials.TypeImage,
			repoURL:  fakeGARRepoURL,
			data: map[string][]byte{
				serviceAccountKeyKey: []byte(fakeServiceAccountKey),
			},
			setupCache: func(c *cache.Cache) {
				cacheKey := tokenCacheKey(fakeServiceAccountKey)
				c.Set(cacheKey, fakeAccessToken, cache.DefaultExpiration)
			},
			assertions: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, accessTokenUsername, creds.Username)
				assert.Equal(t, fakeAccessToken, creds.Password)
			},
		},
		{
			name:     "cache miss, successful token fetch",
			credType: credentials.TypeImage,
			repoURL:  fakeGARRepoURL,
			data: map[string][]byte{
				serviceAccountKeyKey: []byte(fakeServiceAccountKey),
			},
			getAccessTokenFn: func(_ context.Context, _ string) (string, error) {
				return fakeAccessToken, nil
			},
			assertions: func(t *testing.T, c *cache.Cache, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, accessTokenUsername, creds.Username)
				assert.Equal(t, fakeAccessToken, creds.Password)

				// Verify the token was cached
				cachedToken, found := c.Get(tokenCacheKey(fakeServiceAccountKey))
				assert.True(t, found)
				assert.Equal(t, fakeAccessToken, cachedToken)
			},
		},
		{
			name:     "error in getAccessToken",
			credType: credentials.TypeImage,
			repoURL:  fakeGARRepoURL,
			data: map[string][]byte{
				serviceAccountKeyKey: []byte(fakeServiceAccountKey),
			},
			getAccessTokenFn: func(_ context.Context, _ string) (string, error) {
				return "", errors.New("access token error")
			},
			assertions: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.ErrorContains(t, err, "error getting GCP access token")
				assert.Nil(t, creds)
			},
		},
		{
			name:     "empty token from getAccessToken",
			credType: credentials.TypeImage,
			repoURL:  fakeGARRepoURL,
			data: map[string][]byte{
				serviceAccountKeyKey: []byte(fakeServiceAccountKey),
			},
			getAccessTokenFn: func(_ context.Context, _ string) (string, error) {
				return "", nil
			},
			assertions: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewServiceAccountKeyProvider().(*ServiceAccountKeyProvider) // nolint:forcetypeassert
			provider.getAccessTokenFn = tt.getAccessTokenFn

			if tt.setupCache != nil {
				tt.setupCache(provider.tokenCache)
			}

			creds, err := provider.GetCredentials(ctx, "", tt.credType, tt.repoURL, tt.data)
			tt.assertions(t, provider.tokenCache, creds, err)
		})
	}
}
