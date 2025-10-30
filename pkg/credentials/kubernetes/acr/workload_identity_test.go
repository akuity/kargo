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

	"github.com/akuity/kargo/pkg/credentials"
)

func TestWorkloadIdentityProvider_Supports(t *testing.T) {
	const testOCIRepoURL = "myregistry.azurecr.io/my-repo"
	const testHTTPSRepoURL = "https://myregistry.azurecr.io/my-repo"

	testCases := []struct {
		name     string
		credType credentials.Type
		repoURL  string
		expected bool
	}{
		{
			name:     "image credential type supported",
			credType: credentials.TypeImage,
			repoURL:  testOCIRepoURL,
			expected: true,
		},
		{
			name:     "helm credential type supported",
			credType: credentials.TypeHelm,
			repoURL:  testOCIRepoURL,
			expected: true,
		},
		{
			name:     "helm HTTP/S repo URLs not supported",
			credType: credentials.TypeHelm,
			repoURL:  testHTTPSRepoURL,
			expected: false,
		},
		{
			name:     "git credential type not supported",
			credType: credentials.TypeGit,
			repoURL:  testOCIRepoURL,
			expected: false,
		},
		{
			name: "non-ACR repo URL not supported",

			credType: credentials.TypeImage,
			repoURL:  "docker.io/library/nginx",
			expected: false,
		},
	}

	p := &WorkloadIdentityProvider{credential: &mockCredential{}}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result := p.Supports(tt.credType, tt.repoURL, nil, nil)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWorkloadIdentityProvider_GetCredentials(t *testing.T) {
	const testRepoURL = "myregistry.azurecr.io/repo"
	const testRegistryName = "myregistry"
	const testToken = "fake-access-token"

	testCases := []struct {
		name       string
		provider   *WorkloadIdentityProvider
		credType   credentials.Type
		repoURL    string
		setupCache func(cache *cache.Cache)
		assertions func(*testing.T, *cache.Cache, *credentials.Credentials, error)
	}{
		{
			name:     "not supported",
			provider: &WorkloadIdentityProvider{},
			credType: credentials.TypeGit,
			repoURL:  "git://repo",
			assertions: func(
				t *testing.T,
				_ *cache.Cache,
				creds *credentials.Credentials,
				err error,
			) {
				assert.NoError(t, err)
				assert.Nil(t, creds)
			},
		},
		{
			name:     "non-ACR URL",
			provider: &WorkloadIdentityProvider{},
			credType: credentials.TypeImage,
			repoURL:  "not-an-acr-url",
			assertions: func(
				t *testing.T,
				_ *cache.Cache,
				creds *credentials.Credentials,
				err error,
			) {
				assert.NoError(t, err)
				assert.Nil(t, creds)
			},
		},
		{
			name:     "cache hit",
			provider: &WorkloadIdentityProvider{},
			credType: credentials.TypeImage,
			repoURL:  testRepoURL,
			setupCache: func(c *cache.Cache) {
				c.Set(testRegistryName, testToken, cache.DefaultExpiration)
			},
			assertions: func(
				t *testing.T,
				_ *cache.Cache,
				creds *credentials.Credentials,
				err error,
			) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, acrTokenUsername, creds.Username)
				assert.Equal(t, testToken, creds.Password)
			},
		},
		{
			name: "cache miss, successful token fetch",
			provider: &WorkloadIdentityProvider{
				getAccessTokenFn: func(_ context.Context, _ string) (string, error) {
					return testToken, nil
				},
			},
			credType: credentials.TypeImage,
			repoURL:  testRepoURL,
			assertions: func(
				t *testing.T,
				c *cache.Cache,
				creds *credentials.Credentials,
				err error,
			) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, acrTokenUsername, creds.Username)
				assert.Equal(t, testToken, creds.Password)

				// Verify the token was cached
				cachedToken, found := c.Get(testRegistryName)
				assert.True(t, found)
				assert.Equal(t, testToken, cachedToken)
			},
		},
		{
			name: "error in getAccessToken",
			provider: &WorkloadIdentityProvider{
				getAccessTokenFn: func(_ context.Context, _ string) (string, error) {
					return "", errors.New("access token error")
				},
			},
			credType: credentials.TypeImage,
			repoURL:  testRepoURL,
			assertions: func(
				t *testing.T,
				_ *cache.Cache,
				creds *credentials.Credentials,
				err error,
			) {
				assert.ErrorContains(t, err, "error getting ACR access token")
				assert.Nil(t, creds)
			},
		},
		{
			name: "empty token from getAccessToken",
			provider: &WorkloadIdentityProvider{
				getAccessTokenFn: func(_ context.Context, _ string) (string, error) {
					return "", nil
				},
			},
			credType: credentials.TypeImage,
			repoURL:  testRepoURL,
			assertions: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.Nil(t, creds)
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			tt.provider.credential = &mockCredential{}
			tt.provider.tokenCache = cache.New(10*time.Hour, time.Hour)
			if tt.setupCache != nil {
				tt.setupCache(tt.provider.tokenCache)
			}
			creds, err := tt.provider.GetCredentials(
				context.Background(),
				"",
				tt.credType,
				tt.repoURL,
				nil,
				nil,
			)
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
			name:     "ACR URL",
			url:      "myregistry.azurecr.io/repo",
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
