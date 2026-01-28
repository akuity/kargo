package acr

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/credentials"
)

func TestNewWorkloadIdentityProvider(t *testing.T) {
	const azFederatedTokenFile = "AZURE_FEDERATED_TOKEN_FILE"
	const azClientID = "AZURE_CLIENT_ID"
	const azTenantID = "AZURE_TENANT_ID"
	t.Run("workload identity not available", func(t *testing.T) {
		// Make it look unavailable by ensuring key env vars are unset
		t.Setenv(azFederatedTokenFile, "") // Ensures cleanup
		os.Unsetenv(azFederatedTokenFile)  // Actually unsets
		t.Setenv(azClientID, "")           // Ensures cleanup
		os.Unsetenv(azClientID)            // Actually unsets
		t.Setenv(azTenantID, "")           // Ensures cleanup
		os.Unsetenv(azTenantID)            // Actually unsets
		require.Nil(t, NewWorkloadIdentityProvider(t.Context()))
	})
	t.Run("workload identity available", func(t *testing.T) {
		// Make it look available by ensuring key env vars are set, albeit with
		// nonsense values.
		const nonsense = "nonsense"
		t.Setenv(azFederatedTokenFile, nonsense)
		t.Setenv(azClientID, nonsense)
		t.Setenv(azTenantID, nonsense)
		require.NotNil(t, NewWorkloadIdentityProvider(t.Context()))
	})
}

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

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			supports, err := p.Supports(
				t.Context(),
				credentials.Request{
					Type:    testCase.credType,
					RepoURL: testCase.repoURL,
				},
			)
			require.NoError(t, err)
			require.Equal(t, testCase.expected, supports)
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
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.provider.credential = &mockCredential{}
			testCase.provider.tokenCache = cache.New(10*time.Hour, time.Hour)
			if testCase.setupCache != nil {
				testCase.setupCache(testCase.provider.tokenCache)
			}
			creds, err := testCase.provider.GetCredentials(
				t.Context(),
				credentials.Request{
					Type:    testCase.credType,
					RepoURL: testCase.repoURL,
				},
			)
			testCase.assertions(t, testCase.provider.tokenCache, creds, err)
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

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			matches := acrURLRegex.FindStringSubmatch(testCase.url)
			if testCase.expected {
				assert.Len(t, matches, 2)
				assert.Equal(t, testCase.registry, matches[1])
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
