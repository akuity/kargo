package gar

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/akuity/kargo/pkg/cache/coalescing"
	"github.com/akuity/kargo/pkg/credentials"
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
			name:     "valid GCR image repo with service account key",
			credType: credentials.TypeImage,
			repoURL:  fakeGCRRepoURL,
			data: map[string][]byte{
				serviceAccountKeyKey: []byte(fakeServiceAccountKey),
			},
			expected: true,
		},
		{
			name:     "valid GAR image repo with service account key",
			credType: credentials.TypeImage,
			repoURL:  fakeGARRepoURL,
			data: map[string][]byte{
				serviceAccountKeyKey: []byte(fakeServiceAccountKey),
			},
			expected: true,
		},
		{
			name:     "unsupported credential type",
			credType: credentials.TypeGit,
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
		// Helm chart test cases
		{
			name:     "valid GAR chart repo with service account key",
			credType: credentials.TypeHelm,
			repoURL:  fakeGARRepoURL,
			data: map[string][]byte{
				serviceAccountKeyKey: []byte(fakeServiceAccountKey),
			},
			expected: true,
		},
		{
			name:     "valid GCR chart repo with service account key",
			credType: credentials.TypeHelm,
			repoURL:  fakeGCRRepoURL,
			data: map[string][]byte{
				serviceAccountKeyKey: []byte(fakeServiceAccountKey),
			},
			expected: true,
		},
		{
			name:     "Helm chart repo with non-GAR/GCR URL",
			credType: credentials.TypeHelm,
			repoURL:  "docker.io/library/nginx",
			data: map[string][]byte{
				serviceAccountKeyKey: []byte(fakeServiceAccountKey),
			},
			expected: false,
		},
		{
			name:     "Helm chart repo missing service account key",
			credType: credentials.TypeHelm,
			repoURL:  fakeGARRepoURL,
			data:     map[string][]byte{},
			expected: false,
		},
	}

	p := NewServiceAccountKeyProvider()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			supports, err := p.Supports(
				t.Context(),
				credentials.Request{
					Type:    testCase.credType,
					RepoURL: testCase.repoURL,
					Data:    testCase.data,
				},
			)
			require.NoError(t, err)
			require.Equal(t, testCase.expected, supports)
		})
	}
}

func TestServiceAccountKeyProvider_GetCredentials(t *testing.T) {
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
		getAccessTokenFn func(
			ctx context.Context,
			encodedServiceAccountKey string,
		) (*oauth2.Token, error)
		assertions func(t *testing.T, creds *credentials.Credentials, err error)
	}{
		{
			name:     "token obtained",
			credType: credentials.TypeImage,
			repoURL:  fakeGARRepoURL,
			data: map[string][]byte{
				serviceAccountKeyKey: []byte(fakeServiceAccountKey),
			},
			getAccessTokenFn: func(context.Context, string) (*oauth2.Token, error) {
				return &oauth2.Token{
					AccessToken: fakeAccessToken,
					Expiry:      time.Now().Add(time.Hour),
				}, nil
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, accessTokenUsername, creds.Username)
				assert.Equal(t, fakeAccessToken, creds.Password)
			},
		},
		{
			name:     "token obtained, but too near expiry to cache",
			credType: credentials.TypeImage,
			repoURL:  fakeGARRepoURL,
			data: map[string][]byte{
				serviceAccountKeyKey: []byte(fakeServiceAccountKey),
			},
			getAccessTokenFn: func(context.Context, string) (*oauth2.Token, error) {
				return &oauth2.Token{
					AccessToken: fakeAccessToken,
					Expiry:      time.Now().Add(-time.Hour),
				}, nil
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, fakeAccessToken, creds.Password)
			},
		},
		{
			name:     "error in getAccessToken",
			credType: credentials.TypeImage,
			repoURL:  fakeGARRepoURL,
			data: map[string][]byte{
				serviceAccountKeyKey: []byte(fakeServiceAccountKey),
			},
			getAccessTokenFn: func(context.Context, string) (*oauth2.Token, error) {
				return nil, errors.New("access token error")
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
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
			getAccessTokenFn: func(context.Context, string) (*oauth2.Token, error) {
				return nil, nil
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.NoError(t, err)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			provider := NewServiceAccountKeyProvider().(*ServiceAccountKeyProvider) // nolint:forcetypeassert
			provider.getAccessTokenFn = testCase.getAccessTokenFn

			// A cache that caches nothing leaves loading as the only way a caller
			// can obtain a token, so every case exercises the load. Whether a hit is
			// served from the cache instead is the cache's concern, not this
			// provider's.
			tokenCache, err := coalescing.NewCache(
				provider.loadAccessToken,
				&coalescing.CacheOptions{
					LoadTimeout:  new(tokenAcquisitionTimeout),
					CacheNothing: true,
				},
			)
			require.NoError(t, err)
			provider.tokenCache = tokenCache

			creds, err := provider.GetCredentials(
				t.Context(),
				credentials.Request{
					Type:    testCase.credType,
					RepoURL: testCase.repoURL,
					Data:    testCase.data,
				},
			)
			testCase.assertions(t, creds, err)
		})
	}
}

func TestServiceAccountKeyProvider_GetCredentials_perKey(t *testing.T) {
	t.Parallel()

	// Coalescing, cancellation, and the load deadline all belong to the cache this
	// provider delegates to, and are covered by that package's tests. What remains
	// this provider's own responsibility is that the key it caches under and the
	// input it loads from describe the same service account key, so that no set of
	// credentials is ever served another's token.

	const fakeRepoURL = "us-central1-docker.pkg.dev/my-project/my-repo"

	provider := &ServiceAccountKeyProvider{
		getAccessTokenFn: func(
			_ context.Context,
			encodedServiceAccountKey string,
		) (*oauth2.Token, error) {
			// The token identifies the key it was obtained with.
			return &oauth2.Token{
				AccessToken: "token-for-" + encodedServiceAccountKey,
				Expiry:      time.Now().Add(time.Hour),
			}, nil
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

	for _, key := range []string{"key-a", "key-b", "key-a"} {
		creds, err := provider.GetCredentials(
			t.Context(),
			credentials.Request{
				Type:    credentials.TypeImage,
				RepoURL: fakeRepoURL,
				Data:    map[string][]byte{serviceAccountKeyKey: []byte(key)},
			},
		)
		require.NoError(t, err)
		require.NotNil(t, creds)
		require.Equal(t, accessTokenUsername, creds.Username)
		require.Equal(t, "token-for-"+key, creds.Password)
	}
}
