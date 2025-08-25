package acr

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"

	"github.com/akuity/kargo/internal/credentials"
)

func TestNewWorkloadIdentityProvider(t *testing.T) {
	provider := NewWorkloadIdentityProvider(context.Background())
	// Provider may be nil if Azure credentials are not available in the test environment
	if provider != nil {
		providerImpl := provider.(*WorkloadIdentityProvider) // nolint:forcetypeassert
		assert.NotNil(t, providerImpl.tokenCache)
		assert.NotNil(t, providerImpl.credential)
		assert.NotNil(t, providerImpl.getAccessTokenFn)
	}
}

func TestWorkloadIdentityProvider_Supports(t *testing.T) {
	const (
		fakeRepoURL      = "myregistry.azurecr.io/my-repo"
		fakeOCIRepoURL   = "oci://myregistry.azurecr.io/my-repo"
		fakeHTTPSRepoURL = "https://myregistry.azurecr.io/my-repo"
	)

	testCases := []struct {
		name     string
		provider *WorkloadIdentityProvider
		credType credentials.Type
		repoURL  string
		expected bool
	}{
		{
			name: "no credential configured",
			provider: &WorkloadIdentityProvider{
				credential: nil,
			},
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			expected: false,
		},
		{
			name: "image credentials supported",
			provider: &WorkloadIdentityProvider{
				credential: &mockCredential{},
			},
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			expected: true,
		},
		{
			name: "OCI helm credentials supported",
			provider: &WorkloadIdentityProvider{
				credential: &mockCredential{},
			},
			credType: credentials.TypeHelm,
			repoURL:  fakeOCIRepoURL,
			expected: true,
		},
		{
			name: "non-OCI helm credentials not supported",
			provider: &WorkloadIdentityProvider{
				credential: &mockCredential{},
			},
			credType: credentials.TypeHelm,
			repoURL:  fakeHTTPSRepoURL,
			expected: false,
		},
		{
			name: "git credentials not supported",
			provider: &WorkloadIdentityProvider{
				credential: &mockCredential{},
			},
			credType: credentials.TypeGit,
			repoURL:  fakeRepoURL,
			expected: false,
		},
		{
			name: "non-ACR URL not supported",
			provider: &WorkloadIdentityProvider{
				credential: &mockCredential{},
			},
			credType: credentials.TypeImage,
			repoURL:  "docker.io/library/nginx",
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

func TestWorkloadIdentityProvider_GetCredentials(t *testing.T) {
	ctx := context.Background()

	const (
		fakeProject      = "fake-project"
		fakeRepoURL      = "myregistry.azurecr.io/repo"
		fakeRegistryName = "myregistry"
		fakeToken        = "fake-access-token"
		expectedUsername = "00000000-0000-0000-0000-000000000000"
	)

	testCases := []struct {
		name       string
		provider   *WorkloadIdentityProvider
		project    string
		credType   credentials.Type
		repoURL    string
		setupCache func(cache *cache.Cache)
		assertions func(t *testing.T, c *cache.Cache, creds *credentials.Credentials, err error)
	}{
		{
			name: "not supported",
			provider: &WorkloadIdentityProvider{
				credential: &mockCredential{},
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
			name: "non-ACR URL",
			provider: &WorkloadIdentityProvider{
				credential: &mockCredential{},
				tokenCache: cache.New(10*time.Hour, time.Hour),
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  "not-an-acr-url",
			assertions: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.NoError(t, err)
			},
		},
		{
			name: "cache hit",
			provider: &WorkloadIdentityProvider{
				credential: &mockCredential{},
				tokenCache: cache.New(10*time.Hour, time.Hour),
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			setupCache: func(c *cache.Cache) {
				cacheKey := tokenCacheKey(fakeRegistryName, fakeProject)
				c.Set(cacheKey, fakeToken, cache.DefaultExpiration)
			},
			assertions: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, expectedUsername, creds.Username)
				assert.Equal(t, fakeToken, creds.Password)
			},
		},
		{
			name: "cache miss, successful token fetch",
			provider: &WorkloadIdentityProvider{
				credential: &mockCredential{},
				tokenCache: cache.New(10*time.Hour, time.Hour),
				getAccessTokenFn: func(_ context.Context, _ string) (string, error) {
					return fakeToken, nil
				},
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			assertions: func(t *testing.T, c *cache.Cache, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, expectedUsername, creds.Username)
				assert.Equal(t, fakeToken, creds.Password)

				// Verify the token was cached
				cachedToken, found := c.Get(tokenCacheKey(fakeRegistryName, fakeProject))
				assert.True(t, found)
				assert.Equal(t, fakeToken, cachedToken)
			},
		},
		{
			name: "error in getAccessToken",
			provider: &WorkloadIdentityProvider{
				credential: &mockCredential{},
				tokenCache: cache.New(10*time.Hour, time.Hour),
				getAccessTokenFn: func(_ context.Context, _ string) (string, error) {
					return "", errors.New("access token error")
				},
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			assertions: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.ErrorContains(t, err, "error getting ACR access token")
				assert.Nil(t, creds)
			},
		},
		{
			name: "empty token from getAccessToken",
			provider: &WorkloadIdentityProvider{
				credential: &mockCredential{},
				tokenCache: cache.New(10*time.Hour, time.Hour),
				getAccessTokenFn: func(_ context.Context, _ string) (string, error) {
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

func TestACRURLRegex(t *testing.T) {
	testCases := []struct {
		name     string
		url      string
		expected bool
		registry string
	}{
		{
			name:     "standard ACR URL",
			url:      "myregistry.azurecr.io/repo",
			expected: true,
			registry: "myregistry",
		},
		{
			name:     "OCI ACR URL",
			url:      "oci://myregistry.azurecr.io/repo",
			expected: true,
			registry: "myregistry",
		},
		{
			name:     "Docker Hub URL",
			url:      "docker.io/library/nginx",
			expected: false,
		},
		{
			name:     "ECR URL",
			url:      "123456789012.dkr.ecr.us-west-2.amazonaws.com/repo",
			expected: false,
		},
		{
			name:     "Google Artifact Registry URL",
			url:      "us-central1-docker.pkg.dev/project/repo",
			expected: false,
		},
		{
			name:     "ACR URL with complex registry name",
			url:      "my-registry-123.azurecr.io/namespace/repo",
			expected: true,
			registry: "my-registry-123",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			matches := acrURLRegex.FindStringSubmatch(tt.url)
			if tt.expected {
				assert.Len(t, matches, 2, "Expected regex to match and capture registry name")
				assert.Equal(t, tt.registry, matches[1], "Registry name should be captured correctly")
			} else {
				assert.Nil(t, matches, "Expected regex not to match")
			}
		})
	}
}

// mockCredential is a mock implementation of azcore.TokenCredential for testing
type mockCredential struct{}

func (m *mockCredential) GetToken(_ context.Context, _ policy.TokenRequestOptions) (azcore.AccessToken, error) {
	// Return a mock token for testing
	return azcore.AccessToken{
		Token:     "mock-access-token",
		ExpiresOn: time.Now().Add(time.Hour),
	}, nil
}
