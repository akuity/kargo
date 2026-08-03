package gar

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/akuity/kargo/pkg/cache/coalescing"
	"github.com/akuity/kargo/pkg/credentials"
)

func TestWorkloadIdentityFederationProvider_Supports(t *testing.T) {
	t.Parallel()

	const (
		fakeProjectID  = "test-project"
		fakeGCRRepoURL = "gcr.io/my-project/my-repo"
		fakeGARRepoURL = "us-central1-docker.pkg.dev/my-project/my-repo"
	)

	testCases := []struct {
		name       string
		provider   *WorkloadIdentityFederationProvider
		credType   credentials.Type
		repoURL    string
		assertions func(t *testing.T, result bool)
	}{
		{
			name:     "supports image credentials for GAR URL",
			provider: &WorkloadIdentityFederationProvider{projectID: fakeProjectID},
			credType: credentials.TypeImage,
			repoURL:  fakeGARRepoURL,
			assertions: func(t *testing.T, result bool) {
				assert.True(t, result)
			},
		},
		{
			name:     "supports image credentials for GCR URL",
			provider: &WorkloadIdentityFederationProvider{projectID: fakeProjectID},
			credType: credentials.TypeImage,
			repoURL:  fakeGCRRepoURL,
			assertions: func(t *testing.T, result bool) {
				assert.True(t, result)
			},
		},
		{
			name:     "supports Helm credentials for GAR URL",
			provider: &WorkloadIdentityFederationProvider{projectID: fakeProjectID},
			credType: credentials.TypeHelm,
			repoURL:  fakeGARRepoURL,
			assertions: func(t *testing.T, result bool) {
				assert.True(t, result)
			},
		},
		{
			name:     "supports Helm credentials for GCR URL",
			provider: &WorkloadIdentityFederationProvider{projectID: fakeProjectID},
			credType: credentials.TypeHelm,
			repoURL:  fakeGCRRepoURL,
			assertions: func(t *testing.T, result bool) {
				assert.True(t, result)
			},
		},
		{
			name:     "rejects unsupported credential type",
			provider: &WorkloadIdentityFederationProvider{projectID: fakeProjectID},
			credType: credentials.TypeGit,
			repoURL:  fakeGARRepoURL,
			assertions: func(t *testing.T, result bool) {
				assert.False(t, result)
			},
		},
		{
			name:     "rejects non-GAR/GCR URL",
			provider: &WorkloadIdentityFederationProvider{projectID: fakeProjectID},
			credType: credentials.TypeImage,
			repoURL:  "docker.io/library/alpine",
			assertions: func(t *testing.T, result bool) {
				assert.False(t, result)
			},
		},
		{
			name:     "rejects Helm credentials for non-GAR/GCR URL",
			provider: &WorkloadIdentityFederationProvider{projectID: fakeProjectID},
			credType: credentials.TypeHelm,
			repoURL:  "docker.io/library/alpine",
			assertions: func(t *testing.T, result bool) {
				assert.False(t, result)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			supports, err := testCase.provider.Supports(
				t.Context(),
				credentials.Request{
					Type:    testCase.credType,
					RepoURL: testCase.repoURL,
				},
			)
			require.NoError(t, err)
			testCase.assertions(t, supports)
		})
	}
}

func TestWorkloadIdentityFederationProvider_GetCredentials(t *testing.T) {
	t.Parallel()

	const (
		fakeProjectID  = "test-project"
		fakeProject    = "kargo-project"
		fakeGCRRepoURL = "gcr.io/my-project/my-repo"
		fakeToken      = "fake-token"
	)

	testCases := []struct {
		name       string
		provider   *WorkloadIdentityFederationProvider
		project    string
		credType   credentials.Type
		repoURL    string
		assertions func(
			t *testing.T,
			creds *credentials.Credentials,
			err error,
		)
	}{
		{
			name: "token obtained",
			provider: &WorkloadIdentityFederationProvider{
				projectID: fakeProjectID,
				getAccessTokenFn: func(context.Context, string) (string, time.Time, error) {
					return fakeToken, time.Now().Add(time.Hour), nil
				},
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeGCRRepoURL,
			assertions: func(
				t *testing.T,
				creds *credentials.Credentials,
				err error,
			) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, accessTokenUsername, creds.Username)
				assert.Equal(t, fakeToken, creds.Password)
			},
		},
		{
			name: "token obtained, but too near expiry to cache",
			provider: &WorkloadIdentityFederationProvider{
				projectID: fakeProjectID,
				getAccessTokenFn: func(context.Context, string) (string, time.Time, error) {
					return fakeToken, time.Now().Add(-time.Hour), nil
				},
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeGCRRepoURL,
			assertions: func(
				t *testing.T,
				creds *credentials.Credentials,
				err error,
			) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, fakeToken, creds.Password)
			},
		},
		{
			name: "error in getAccessToken",
			provider: &WorkloadIdentityFederationProvider{
				projectID: fakeProjectID,
				getAccessTokenFn: func(context.Context, string) (string, time.Time, error) {
					return "", time.Time{}, fmt.Errorf("token fetch error")
				},
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeGCRRepoURL,
			assertions: func(
				t *testing.T,
				creds *credentials.Credentials,
				err error,
			) {
				assert.ErrorContains(t, err, "error getting GCP access token")
				assert.ErrorContains(t, err, "token fetch error")
				assert.Nil(t, creds)
			},
		},
		{
			name: "empty token from getAccessToken falls back to default token source",
			provider: &WorkloadIdentityFederationProvider{
				projectID: fakeProjectID,
				getAccessTokenFn: func(context.Context, string) (string, time.Time, error) {
					return "", time.Time{}, nil
				},
				tokenSource: newFakeTokenSource(fakeToken),
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeGCRRepoURL,
			assertions: func(
				t *testing.T,
				creds *credentials.Credentials,
				err error,
			) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, accessTokenUsername, creds.Username)
				assert.Equal(t, fakeToken, creds.Password)
			},
		},
		{
			name: "error from default token source",
			provider: &WorkloadIdentityFederationProvider{
				projectID: fakeProjectID,
				getAccessTokenFn: func(context.Context, string) (string, time.Time, error) {
					return "", time.Time{}, nil
				},
				tokenSource: newFailingTokenSource(fmt.Errorf("token source error")),
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeGCRRepoURL,
			assertions: func(
				t *testing.T,
				creds *credentials.Credentials,
				err error,
			) {
				assert.ErrorContains(t, err, "error getting GCP access token")
				assert.ErrorContains(t, err, "token source error")
				assert.Nil(t, creds)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// A cache that caches nothing leaves loading as the only way a caller
			// can obtain a token, so every case exercises the load. Whether a hit is
			// served from the cache instead is the cache's concern, not this
			// provider's.
			tokenCache, err := coalescing.NewCache(
				testCase.provider.loadAccessToken,
				&coalescing.CacheOptions{
					LoadTimeout:  new(tokenAcquisitionTimeout),
					CacheNothing: true,
				},
			)
			require.NoError(t, err)
			testCase.provider.tokenCache = tokenCache

			creds, err := testCase.provider.GetCredentials(
				t.Context(),
				credentials.Request{
					Type:    testCase.credType,
					Project: testCase.project,
					RepoURL: testCase.repoURL,
				},
			)
			testCase.assertions(t, creds, err)
		})
	}
}

func TestWorkloadIdentityFederationProvider_GetCredentials_noProjectToken(t *testing.T) {
	t.Parallel()

	// A Project with no service account of its own has that fact cached in place
	// of a token, so the lookup that established it is not repeated. The token
	// itself cannot be cached in its stead, the controller's own token source
	// being what refreshes it.

	const fakeRepoURL = "us-central1-docker.pkg.dev/my-project/my-repo"

	var lookups int
	provider := &WorkloadIdentityFederationProvider{
		tokenSource: newFakeTokenSource("token-from-default-source"),
		getAccessTokenFn: func(context.Context, string) (string, time.Time, error) {
			lookups++
			return "", time.Time{}, nil
		},
	}
	tokenCache, err := coalescing.NewCache(
		provider.loadAccessToken,
		&coalescing.CacheOptions{
			LoadTimeout:     new(tokenAcquisitionTimeout),
			CleanupInterval: new(time.Hour),
		},
	)
	require.NoError(t, err)
	provider.tokenCache = tokenCache

	for range 2 {
		creds, credsErr := provider.GetCredentials(
			t.Context(),
			credentials.Request{
				Type:    credentials.TypeImage,
				Project: "project-without-a-service-account",
				RepoURL: fakeRepoURL,
			},
		)
		require.NoError(t, credsErr)
		require.NotNil(t, creds)
		require.Equal(t, accessTokenUsername, creds.Username)
		require.Equal(t, "token-from-default-source", creds.Password)
	}
	require.Equal(t, 1, lookups)
}

func TestWorkloadIdentityFederationProvider_GetCredentials_perProject(t *testing.T) {
	t.Parallel()

	// Coalescing, cancellation, and the load deadline all belong to the cache this
	// provider delegates to, and are covered by that package's tests. What remains
	// this provider's own responsibility is that the key it caches under and the
	// input it loads from describe the same Project, so that no Project is ever
	// served another's token.

	const fakeRepoURL = "us-central1-docker.pkg.dev/my-project/my-repo"

	provider := &WorkloadIdentityFederationProvider{
		getAccessTokenFn: func(
			_ context.Context,
			project string,
		) (string, time.Time, error) {
			// The token identifies the Project it was obtained for.
			return "token-for-" + project, time.Now().Add(time.Hour), nil
		},
	}
	tokenCache, err := coalescing.NewCache(
		provider.loadAccessToken,
		&coalescing.CacheOptions{
			LoadTimeout: new(tokenAcquisitionTimeout),
			DefaultTTL:  new(time.Hour),
		},
	)
	require.NoError(t, err)
	provider.tokenCache = tokenCache

	for _, project := range []string{"project-a", "project-b", "project-a"} {
		creds, err := provider.GetCredentials(
			t.Context(),
			credentials.Request{
				Type:    credentials.TypeImage,
				Project: project,
				RepoURL: fakeRepoURL,
			},
		)
		require.NoError(t, err)
		require.NotNil(t, creds)
		require.Equal(t, accessTokenUsername, creds.Username)
		require.Equal(t, "token-for-"+project, creds.Password)
	}
}

type fakeTokenSource struct {
	token string
}

func newFakeTokenSource(token string) oauth2.TokenSource {
	return &fakeTokenSource{token: token}
}

func (f *fakeTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: f.token}, nil
}

type failingTokenSource struct {
	err error
}

func newFailingTokenSource(err error) oauth2.TokenSource {
	return &failingTokenSource{err: err}
}

func (f *failingTokenSource) Token() (*oauth2.Token, error) {
	return nil, f.err
}
