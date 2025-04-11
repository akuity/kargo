package gar

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"

	"github.com/akuity/kargo/internal/credentials"
)

func TestWorkloadIdentityFederationProvider_Supports(t *testing.T) {
	const (
		fakeProjectID  = "test-project"
		fakeGCRRepoURL = "gcr.io/my-project/my-repo"
		fakeGARRepoURL = "us-central1-docker.pkg.dev/my-project/my-repo"
	)

	testCases := []struct {
		name     string
		provider *WorkloadIdentityFederationProvider
		credType credentials.Type
		repoURL  string
		assert   func(t *testing.T, result bool)
	}{
		{
			name: "supports image credentials for GAR URL",
			provider: &WorkloadIdentityFederationProvider{
				projectID: fakeProjectID,
			},
			credType: credentials.TypeImage,
			repoURL:  fakeGARRepoURL,
			assert: func(t *testing.T, result bool) {
				assert.True(t, result, "should support GAR URL with image credentials")
			},
		},
		{
			name: "supports image credentials for GCR URL",
			provider: &WorkloadIdentityFederationProvider{
				projectID: fakeProjectID,
			},
			credType: credentials.TypeImage,
			repoURL:  fakeGCRRepoURL,
			assert: func(t *testing.T, result bool) {
				assert.True(t, result, "should support GCR URL with image credentials")
			},
		},
		{
			name: "rejects non-image credentials",
			provider: &WorkloadIdentityFederationProvider{
				projectID: fakeProjectID,
			},
			credType: credentials.TypeGit,
			repoURL:  fakeGARRepoURL,
			assert: func(t *testing.T, result bool) {
				assert.False(t, result, "should not support non-image credentials")
			},
		},
		{
			name: "rejects non-GAR/GCR URL",
			provider: &WorkloadIdentityFederationProvider{
				projectID: fakeProjectID,
			},
			credType: credentials.TypeImage,
			repoURL:  "docker.io/library/alpine",
			assert: func(t *testing.T, result bool) {
				assert.False(t, result, "should not support non-GAR/GCR URL")
			},
		},
		{
			name:     "rejects empty project ID",
			provider: &WorkloadIdentityFederationProvider{},
			credType: credentials.TypeImage,
			repoURL:  fakeGARRepoURL,
			assert: func(t *testing.T, result bool) {
				assert.False(t, result, "should not support when project ID is empty")
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.provider.Supports(tt.credType, tt.repoURL, nil)
			tt.assert(t, result)
		})
	}
}

func TestWorkloadIdentityFederationProvider_GetCredentials(t *testing.T) {
	const (
		fakeProjectID  = "test-project"
		fakeProject    = "kargo-project"
		fakeGCRRepoURL = "gcr.io/my-project/my-repo"
		fakeToken      = "fake-token"
	)

	testCases := []struct {
		name       string
		provider   *WorkloadIdentityFederationProvider
		setupCache func(c *cache.Cache)
		project    string
		credType   credentials.Type
		repoURL    string
		assert     func(t *testing.T, c *cache.Cache, creds *credentials.Credentials, err error)
	}{
		{
			name: "returns nil when not supported",
			provider: &WorkloadIdentityFederationProvider{
				projectID: fakeProjectID,
			},
			project:  fakeProject,
			credType: credentials.TypeGit, // Not supported
			repoURL:  fakeGCRRepoURL,
			assert: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.Nil(t, creds)
			},
		},
		{
			name: "cache hit",
			provider: &WorkloadIdentityFederationProvider{
				projectID:  fakeProjectID,
				tokenCache: cache.New(10*time.Hour, time.Hour),
			},
			setupCache: func(c *cache.Cache) {
				c.Set(tokenCacheKey(fakeProject), fakeToken, cache.DefaultExpiration)
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeGCRRepoURL,
			assert: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, accessTokenUsername, creds.Username)
				assert.Equal(t, fakeToken, creds.Password)
			},
		},
		{
			name: "cache miss, successful token fetch",
			provider: &WorkloadIdentityFederationProvider{
				projectID:  fakeProjectID,
				tokenCache: cache.New(10*time.Hour, time.Hour),
				getAccessTokenFn: func(_ context.Context, _ string) (string, error) {
					return fakeToken, nil
				},
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeGCRRepoURL,
			assert: func(t *testing.T, c *cache.Cache, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, accessTokenUsername, creds.Username)
				assert.Equal(t, fakeToken, creds.Password)

				// Verify the token was cached
				token, found := c.Get(tokenCacheKey(fakeProject))
				assert.True(t, found)
				assert.Equal(t, fakeToken, token)
			},
		},
		{
			name: "error in getAccessToken",
			provider: &WorkloadIdentityFederationProvider{
				projectID:  fakeProjectID,
				tokenCache: cache.New(10*time.Hour, time.Hour),
				getAccessTokenFn: func(_ context.Context, _ string) (string, error) {
					return "", fmt.Errorf("token fetch error")
				},
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeGCRRepoURL,
			assert: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.ErrorContains(t, err, "token fetch error")
				assert.Nil(t, creds)
			},
		},
		{
			name: "empty token from getAccessToken",
			provider: &WorkloadIdentityFederationProvider{
				projectID:  fakeProjectID,
				tokenCache: cache.New(10*time.Hour, time.Hour),
				getAccessTokenFn: func(_ context.Context, _ string) (string, error) {
					return "", nil
				},
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeGCRRepoURL,
			assert: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.Nil(t, creds)
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupCache != nil {
				tt.setupCache(tt.provider.tokenCache)
			}

			creds, err := tt.provider.GetCredentials(context.Background(), tt.project, tt.credType, tt.repoURL, nil)
			tt.assert(t, tt.provider.tokenCache, creds, err)
		})
	}
}
