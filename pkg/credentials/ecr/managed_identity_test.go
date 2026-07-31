package ecr

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/cache/coalescing"
	"github.com/akuity/kargo/pkg/credentials"
)

func TestManagedIdentityProvider_Supports(t *testing.T) {
	const (
		fakeAccountID = "123456789012"
		fakeRepoURL   = "123456789012.dkr.ecr.us-west-2.amazonaws.com/my-repo"
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
			name: "helm credentials supported",
			provider: &ManagedIdentityProvider{
				accountID: fakeAccountID,
			},
			credType: credentials.TypeHelm,
			repoURL:  fakeRepoURL,
			expected: true,
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
		{
			name: "non-ECR image URL not supported",
			provider: &ManagedIdentityProvider{
				accountID: fakeAccountID,
			},
			credType: credentials.TypeImage,
			repoURL:  "us-docker.pkg.dev/project/repo/image",
			expected: false,
		},
		{
			name: "non-ECR helm URL not supported",
			provider: &ManagedIdentityProvider{
				accountID: fakeAccountID,
			},
			credType: credentials.TypeHelm,
			repoURL:  "us-docker.pkg.dev/project/repo/chart",
			expected: false,
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
			require.Equal(t, testCase.expected, supports)
		})
	}
}

func TestManagedIdentityProvider_GetCredentials(t *testing.T) {
	const (
		fakeAccountID = "123456789012"
		fakeProject   = "fake-project"
		fakeRepoURL   = "123456789012.dkr.ecr.us-west-2.amazonaws.com/repo"
		// base64 of "AWS:password"
		fakeToken = "QVdTOnBhc3N3b3Jk" // nolint:gosec
	)

	testCases := []struct {
		name       string
		provider   *ManagedIdentityProvider
		project    string
		credType   credentials.Type
		repoURL    string
		assertions func(t *testing.T, creds *credentials.Credentials, err error)
	}{
		{
			name: "not supported",
			provider: &ManagedIdentityProvider{
				accountID: fakeAccountID,
			},
			project:  fakeProject,
			credType: credentials.TypeGit,
			repoURL:  "git://repo",
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.NoError(t, err)
			},
		},
		{
			name: "non-ECR URL",
			provider: &ManagedIdentityProvider{
				accountID: fakeAccountID,
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  "not-an-ecr-url",
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.NoError(t, err)
			},
		},
		{
			name: "token obtained",
			provider: &ManagedIdentityProvider{
				accountID: fakeAccountID,
				getAuthTokenFn: func(
					context.Context,
					string,
					string,
				) (string, time.Time, error) {
					return fakeToken, time.Now().Add(12 * time.Hour), nil
				},
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, "AWS", creds.Username)
				assert.Equal(t, "password", creds.Password)
			},
		},
		{
			name: "token obtained, but too near expiry to cache",
			provider: &ManagedIdentityProvider{
				accountID: fakeAccountID,
				getAuthTokenFn: func(
					context.Context,
					string,
					string,
				) (string, time.Time, error) {
					return fakeToken, time.Now().Add(-time.Hour), nil
				},
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, "password", creds.Password)
			},
		},
		{
			name: "error in getAuthToken",
			provider: &ManagedIdentityProvider{
				accountID: fakeAccountID,
				getAuthTokenFn: func(
					context.Context,
					string,
					string,
				) (string, time.Time, error) {
					return "", time.Time{}, errors.New("auth token error")
				},
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.ErrorContains(t, err, "error getting ECR auth token")
				assert.Nil(t, creds)
			},
		},
		{
			name: "empty token from getAuthToken",
			provider: &ManagedIdentityProvider{
				accountID: fakeAccountID,
				getAuthTokenFn: func(
					context.Context,
					string,
					string,
				) (string, time.Time, error) {
					return "", time.Time{}, nil
				},
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.NoError(t, err)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// A cache that caches nothing leaves loading as the only way a caller
			// can obtain a token, so every case exercises the load. Whether a hit is
			// served from the cache instead is the cache's concern, not this
			// provider's.
			tokenCache, err := coalescing.NewCache(
				testCase.provider.loadAuthToken,
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

func TestManagedIdentityProvider_GetCredentials_perProject(t *testing.T) {
	t.Parallel()

	// Coalescing, cancellation, and the load deadline all belong to the cache this
	// provider delegates to, and are covered by that package's tests. What remains
	// this provider's own responsibility is that the key it caches under and the
	// input it loads from describe the same region and Kargo Project, so that no
	// Project is ever served another's token.

	const fakeAccountID = "123456789012"

	provider := &ManagedIdentityProvider{
		accountID: fakeAccountID,
		getAuthTokenFn: func(
			_ context.Context,
			region string,
			project string,
		) (string, time.Time, error) {
			// The token identifies the region and Project it was obtained for.
			return base64.StdEncoding.EncodeToString(
				[]byte("AWS:" + region + "/" + project),
			), time.Now().Add(12 * time.Hour), nil
		},
	}
	tokenCache, err := coalescing.NewCache(
		provider.loadAuthToken,
		&coalescing.CacheOptions{
			LoadTimeout: new(tokenAcquisitionTimeout),
			DefaultTTL:  new(time.Hour),
		},
	)
	require.NoError(t, err)
	provider.tokenCache = tokenCache

	// Each dimension of the key varies in turn, and the first pair repeats at the
	// end to confirm it is still served its own token.
	for _, request := range []struct {
		region  string
		project string
	}{
		{"us-west-2", "project-a"},
		{"us-west-2", "project-b"},
		{"eu-west-1", "project-a"},
		{"us-west-2", "project-a"},
	} {
		creds, err := provider.GetCredentials(
			t.Context(),
			credentials.Request{
				Type:    credentials.TypeImage,
				Project: request.project,
				RepoURL: fakeAccountID + ".dkr.ecr." + request.region +
					".amazonaws.com/repo",
			},
		)
		require.NoError(t, err)
		require.NotNil(t, creds)
		require.Equal(t, "AWS", creds.Username)
		require.Equal(t, request.region+"/"+request.project, creds.Password)
	}
}
