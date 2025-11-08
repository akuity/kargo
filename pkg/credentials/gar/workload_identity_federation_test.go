package gar

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/akuity/kargo/pkg/credentials"
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
			name: "rejects unsupported credentials",
			provider: &WorkloadIdentityFederationProvider{
				projectID: fakeProjectID,
			},
			credType: credentials.TypeGit,
			repoURL:  fakeGARRepoURL,
			assert: func(t *testing.T, result bool) {
				assert.False(t, result, "should not support unsupported credentials")
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
		// Helm chart test cases
		{
			name: "supports Helm credentials for GAR chart URL",
			provider: &WorkloadIdentityFederationProvider{
				projectID: fakeProjectID,
			},
			credType: credentials.TypeHelm,
			repoURL:  fakeGARRepoURL,
			assert: func(t *testing.T, result bool) {
				assert.True(t, result, "should support GAR chart URL with Helm credentials")
			},
		},
		{
			name: "supports Helm credentials for GCR chart URL",
			provider: &WorkloadIdentityFederationProvider{
				projectID: fakeProjectID,
			},
			credType: credentials.TypeHelm,
			repoURL:  fakeGCRRepoURL,
			assert: func(t *testing.T, result bool) {
				assert.True(t, result, "should support GCR chart URL with Helm credentials")
			},
		},
		{
			name: "rejects Helm credentials for non-GAR/GCR URL",
			provider: &WorkloadIdentityFederationProvider{
				projectID: fakeProjectID,
			},
			credType: credentials.TypeHelm,
			repoURL:  "docker.io/library/alpine",
			assert: func(t *testing.T, result bool) {
				assert.False(t, result, "should not support non-GAR/GCR URL with Helm credentials")
			},
		},
		{
			name:     "rejects Helm credentials with empty project ID",
			provider: &WorkloadIdentityFederationProvider{},
			credType: credentials.TypeHelm,
			repoURL:  fakeGARRepoURL,
			assert: func(t *testing.T, result bool) {
				assert.False(t, result, "should not support when project ID is empty")
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			supports, err := testCase.provider.Supports(
				t.Context(),
				credentials.Request{
					Type:    testCase.credType,
					RepoURL: testCase.repoURL,
				},
			)
			require.NoError(t, err)
			testCase.assert(t, supports)
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
		name                  string
		provider              *WorkloadIdentityFederationProvider
		setupTokenCache       func(c *cache.Cache)
		setupTokenSourceCache func(c *cache.Cache)
		project               string
		credType              credentials.Type
		repoURL               string
		assert                func(
			t *testing.T,
			tokenCache *cache.Cache,
			tokenSourceCache *cache.Cache,
			creds *credentials.Credentials,
			err error,
		)
	}{
		{
			name: "token cache hit",
			provider: &WorkloadIdentityFederationProvider{
				projectID:  fakeProjectID,
				tokenCache: cache.New(10*time.Hour, time.Hour),
			},
			setupTokenCache: func(c *cache.Cache) {
				c.Set(tokenCacheKey(fakeProject), fakeToken, cache.DefaultExpiration)
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeGCRRepoURL,
			assert: func(t *testing.T, _, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, accessTokenUsername, creds.Username)
				assert.Equal(t, fakeToken, creds.Password)
			},
		},
		{
			name: "token cache miss, token source cache hit",
			provider: &WorkloadIdentityFederationProvider{
				projectID:        fakeProjectID,
				tokenCache:       cache.New(10*time.Hour, time.Hour),
				tokenSourceCache: cache.New(10*time.Hour, time.Hour),
				getAccessTokenFn: func(_ context.Context, _ string) (string, error) {
					return fakeToken, nil
				},
			},
			setupTokenSourceCache: func(c *cache.Cache) {
				c.Set(tokenCacheKey(fakeProject), newFakeTokenSource(fakeToken), cache.DefaultExpiration)
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeGCRRepoURL,
			assert: func(t *testing.T, _, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, accessTokenUsername, creds.Username)
				assert.Equal(t, fakeToken, creds.Password)
			},
		},
		{
			name: "miss in both caches, successful token fetch",
			provider: &WorkloadIdentityFederationProvider{
				projectID:        fakeProjectID,
				tokenCache:       cache.New(10*time.Hour, time.Hour),
				tokenSourceCache: cache.New(10*time.Hour, time.Hour),
				getAccessTokenFn: func(_ context.Context, _ string) (string, error) {
					return fakeToken, nil
				},
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeGCRRepoURL,
			assert: func(t *testing.T, tokenCache, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, accessTokenUsername, creds.Username)
				assert.Equal(t, fakeToken, creds.Password)

				// Verify the token was cached
				token, found := tokenCache.Get(tokenCacheKey(fakeProject))
				assert.True(t, found)
				assert.Equal(t, fakeToken, token)
			},
		},
		{
			name: "error in getAccessToken",
			provider: &WorkloadIdentityFederationProvider{
				projectID:        fakeProjectID,
				tokenCache:       cache.New(10*time.Hour, time.Hour),
				tokenSourceCache: cache.New(10*time.Hour, time.Hour),
				getAccessTokenFn: func(_ context.Context, _ string) (string, error) {
					return "", fmt.Errorf("token fetch error")
				},
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeGCRRepoURL,
			assert: func(t *testing.T, _, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.ErrorContains(t, err, "token fetch error")
				assert.Nil(t, creds)
			},
		},
		{
			name: "empty token from getAccessToken",
			provider: &WorkloadIdentityFederationProvider{
				projectID:        fakeProjectID,
				tokenCache:       cache.New(10*time.Hour, time.Hour),
				tokenSourceCache: cache.New(10*time.Hour, time.Hour),
				getAccessTokenFn: func(_ context.Context, _ string) (string, error) {
					return "", nil
				},
				tokenSource: newFakeTokenSource(fakeToken),
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeGCRRepoURL,
			assert: func(t *testing.T, _, tokenSourceCache *cache.Cache, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, accessTokenUsername, creds.Username)
				assert.Equal(t, fakeToken, creds.Password)

				// Verify the token source was cached
				tokenSource, found := tokenSourceCache.Get(tokenCacheKey(fakeProject))
				assert.True(t, found)
				ts, ok := tokenSource.(oauth2.TokenSource)
				assert.True(t, ok)
				token, err := ts.Token()
				assert.NoError(t, err)
				assert.Equal(t, fakeToken, token.AccessToken)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.setupTokenCache != nil {
				testCase.setupTokenCache(testCase.provider.tokenCache)
			}
			if testCase.setupTokenSourceCache != nil {
				testCase.setupTokenSourceCache(testCase.provider.tokenSourceCache)
			}
			creds, err := testCase.provider.GetCredentials(
				t.Context(),
				credentials.Request{
					Type:    testCase.credType,
					Project: testCase.project,
					RepoURL: testCase.repoURL,
				},
			)
			testCase.assert(
				t,
				testCase.provider.tokenCache,
				testCase.provider.tokenSourceCache,
				creds,
				err,
			)
		})
	}
}

type fakeTokenSource struct {
	// The token to be returned by the Token() method
	token string
}

func newFakeTokenSource(token string) oauth2.TokenSource {
	return &fakeTokenSource{token: token}
}

func (f *fakeTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken: f.token,
	}, nil
}
