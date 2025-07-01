package ecr

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"

	"github.com/akuity/kargo/internal/credentials"
)

func TestManagedIdentityProvider_Supports(t *testing.T) {
	const (
		fakeAccountID  = "123456789012"
		fakeRepoURL    = "123456789012.dkr.ecr.us-west-2.amazonaws.com/my-repo"
		fakeOCIRepoURL = "oci://123456789012.dkr.ecr.us-west-2.amazonaws.com/my-repo"
	)

	testCases := []struct {
		name     string
		provider *ManagedIdentityProvider
		credType credentials.Type
		repoURL  string
		expected bool
	}{
		{
			name: "no account ID configured",
			provider: &ManagedIdentityProvider{
				accountID: "",
			},
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			expected: false,
		},
		{
			name: "image credentials supported",
			provider: &ManagedIdentityProvider{
				accountID: fakeAccountID,
			},
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			expected: true,
		},
		{
			name: "OCI helm credentials supported",
			provider: &ManagedIdentityProvider{
				accountID: fakeAccountID,
			},
			credType: credentials.TypeHelm,
			repoURL:  fakeOCIRepoURL,
			expected: true,
		},
		{
			name: "non-OCI helm credentials not supported",
			provider: &ManagedIdentityProvider{
				accountID: fakeAccountID,
			},
			credType: credentials.TypeHelm,
			repoURL:  "https://123456789012.dkr.ecr.us-west-2.amazonaws.com/repo",
			expected: false,
		},
		{
			name: "git credentials not supported",
			provider: &ManagedIdentityProvider{
				accountID: fakeAccountID,
			},
			credType: credentials.TypeGit,
			repoURL:  fakeRepoURL,
			expected: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.provider.Supports(tt.credType, tt.repoURL, nil)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestManagedIdentityProvider_GetCredentials(t *testing.T) {
	ctx := context.Background()

	const (
		fakeAccountID = "123456789012"
		fakeProject   = "fake-project"
		fakeRepoURL   = "123456789012.dkr.ecr.us-west-2.amazonaws.com/repo"
		fakeRegion    = "us-west-2"
		// base64 of "AWS:password"
		fakeToken = "QVdTOnBhc3N3b3Jk" // nolint:gosec
	)

	testCases := []struct {
		name       string
		provider   *ManagedIdentityProvider
		project    string
		credType   credentials.Type
		repoURL    string
		setupCache func(cache *cache.Cache)
		assertions func(t *testing.T, c *cache.Cache, creds *credentials.Credentials, err error)
	}{
		{
			name: "not supported",
			provider: &ManagedIdentityProvider{
				accountID:  fakeAccountID,
				tokenCache: cache.New(10*time.Hour, time.Hour),
			},
			project:  fakeProject,
			credType: credentials.TypeGit,
			repoURL:  "git://repo",
			assertions: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.NoError(t, err)
			},
		},
		{
			name: "non-ECR URL",
			provider: &ManagedIdentityProvider{
				accountID:  fakeAccountID,
				tokenCache: cache.New(10*time.Hour, time.Hour),
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  "not-an-ecr-url",
			assertions: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.NoError(t, err)
			},
		},
		{
			name: "cache hit",
			provider: &ManagedIdentityProvider{
				accountID:  fakeAccountID,
				tokenCache: cache.New(10*time.Hour, time.Hour),
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			setupCache: func(c *cache.Cache) {
				cacheKey := tokenCacheKey(fakeRegion, fakeProject)
				c.Set(cacheKey, fakeToken, cache.DefaultExpiration)
			},
			assertions: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, "AWS", creds.Username)
				assert.Equal(t, "password", creds.Password)
			},
		},
		{
			name: "cache miss, successful token fetch",
			provider: &ManagedIdentityProvider{
				accountID:  fakeAccountID,
				tokenCache: cache.New(10*time.Hour, time.Hour),
				getAuthTokenFn: func(_ context.Context, _, _ string) (string, error) {
					return fakeToken, nil
				},
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			assertions: func(t *testing.T, c *cache.Cache, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, "AWS", creds.Username)
				assert.Equal(t, "password", creds.Password)

				// Verify the token was cached
				cachedToken, found := c.Get(tokenCacheKey(fakeRegion, fakeProject))
				assert.True(t, found)
				assert.Equal(t, fakeToken, cachedToken)
			},
		},
		{
			name: "error in getAuthToken",
			provider: &ManagedIdentityProvider{
				accountID:  fakeAccountID,
				tokenCache: cache.New(10*time.Hour, time.Hour),
				getAuthTokenFn: func(_ context.Context, _, _ string) (string, error) {
					return "", errors.New("auth token error")
				},
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			assertions: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.ErrorContains(t, err, "error getting ECR auth token")
				assert.Nil(t, creds)
			},
		},
		{
			name: "empty token from getAuthToken",
			provider: &ManagedIdentityProvider{
				accountID:  fakeAccountID,
				tokenCache: cache.New(10*time.Hour, time.Hour),
				getAuthTokenFn: func(_ context.Context, _, _ string) (string, error) {
					return "", nil
				},
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			assertions: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupCache != nil {
				tt.setupCache(tt.provider.tokenCache)
			}

			creds, err := tt.provider.GetCredentials(ctx, tt.project, tt.credType, tt.repoURL, nil)
			tt.assertions(t, tt.provider.tokenCache, creds, err)
		})
	}
}
