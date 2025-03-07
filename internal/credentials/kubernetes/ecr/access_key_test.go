package ecr

import (
	"context"
	"errors"
	"testing"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"

	"github.com/akuity/kargo/internal/credentials"
)

func TestNewAccessKeyProvider(t *testing.T) {
	provider := NewAccessKeyProvider().(*AccessKeyProvider) // nolint:forcetypeassert

	assert.NotNil(t, provider)
	assert.NotNil(t, provider.tokenCache)
	assert.NotNil(t, provider.getAuthTokenFn)
}

func TestAccessKeyProvider_Supports(t *testing.T) {
	const (
		fakeRepoURL    = "123456789012.dkr.ecr.us-west-2.amazonaws.com/my-repo"
		fakeOCIRepoURL = "oci://123456789012.dkr.ecr.us-west-2.amazonaws.com/my-repo"

		fakeRegion = "us-west-2"
		fakeID     = "AKIAIOSFODNN7EXAMPLE"                     // nolint:gosec
		fakeSecret = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" // nolint:gosec
	)

	testCases := []struct {
		name     string
		credType credentials.Type
		repoURL  string
		data     map[string][]byte
		expected bool
	}{
		{
			name:     "valid image credentials",
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			data: map[string][]byte{
				regionKey: []byte(fakeRegion),
				idKey:     []byte(fakeID),
				secretKey: []byte(fakeSecret),
			},
			expected: true,
		},
		{
			name:     "valid helm oci credentials",
			credType: credentials.TypeHelm,
			repoURL:  fakeOCIRepoURL,
			data: map[string][]byte{
				regionKey: []byte(fakeRegion),
				idKey:     []byte(fakeID),
				secretKey: []byte(fakeSecret),
			},
			expected: true,
		},
		{
			name:     "helm but not oci",
			credType: credentials.TypeHelm,
			repoURL:  "https://123456789012.dkr.ecr.us-west-2.amazonaws.com/my-repo",
			data: map[string][]byte{
				regionKey: []byte(fakeRegion),
				idKey:     []byte(fakeID),
				secretKey: []byte(fakeSecret),
			},
			expected: false,
		},
		{
			name:     "missing region",
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			data: map[string][]byte{
				idKey:     []byte(fakeID),
				secretKey: []byte(fakeSecret),
			},
			expected: false,
		},
		{
			name:     "missing access key ID",
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			data: map[string][]byte{
				regionKey: []byte(fakeRegion),
				secretKey: []byte(fakeSecret),
			},
			expected: false,
		},
		{
			name:     "missing secret key",
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			data: map[string][]byte{
				regionKey: []byte(fakeRegion),
				idKey:     []byte(fakeID),
			},
			expected: false,
		},
		{
			name:     "invalid URL format",
			credType: credentials.TypeImage,
			repoURL:  "not-an-ecr-url",
			data: map[string][]byte{
				regionKey: []byte(fakeRegion),
				idKey:     []byte(fakeID),
				secretKey: []byte(fakeSecret),
			},
			expected: false,
		},
		{
			name:     "unsupported credential type",
			credType: credentials.TypeGit,
			repoURL:  fakeRepoURL,
			data: map[string][]byte{
				regionKey: []byte(fakeRegion),
				idKey:     []byte(fakeID),
				secretKey: []byte(fakeSecret),
			},
			expected: false,
		},
		{
			name:     "empty data",
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			data:     map[string][]byte{},
			expected: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewAccessKeyProvider()
			result := provider.Supports(tt.credType, tt.repoURL, tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAccessKeyProvider_GetCredentials(t *testing.T) {
	ctx := context.Background()

	const (
		fakeRepoURL = "123456789012.dkr.ecr.us-west-2.amazonaws.com/my-repo"
		fakeRegion  = "us-west-2"
		fakeID      = "AKIAIOSFODNN7EXAMPLE"                     // nolint:gosec
		fakeSecret  = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" // nolint:gosec
		// base64 of "AWS:password"
		fakeToken = "QVdTOnBhc3N3b3Jk" // nolint:gosec
	)

	testCases := []struct {
		name           string
		credType       credentials.Type
		repoURL        string
		data           map[string][]byte
		getAuthTokenFn func(ctx context.Context, region, accessKeyID, secretAccessKey string) (string, error)
		setupCache     func(cache *cache.Cache)
		assertions     func(t *testing.T, c *cache.Cache, creds *credentials.Credentials, err error)
	}{
		{
			name:     "unsupported credentials",
			credType: credentials.TypeGit,
			repoURL:  "not-an-ecr-url",
			data:     map[string][]byte{},
			getAuthTokenFn: func(_ context.Context, _, _, _ string) (string, error) {
				return "", nil
			},
			assertions: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.NoError(t, err)
			},
		},
		{
			name:     "cache hit",
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			data: map[string][]byte{
				regionKey: []byte(fakeRegion),
				idKey:     []byte(fakeID),
				secretKey: []byte(fakeSecret),
			},
			setupCache: func(c *cache.Cache) {
				c.Set(
					tokenCacheKey(fakeRegion, fakeID, fakeSecret),
					fakeToken, // base64 of "AWS:password"
					cache.DefaultExpiration,
				)
			},
			assertions: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, "AWS", creds.Username)
				assert.Equal(t, "password", creds.Password)
			},
		},
		{
			name:     "cache miss, successful token fetch",
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			data: map[string][]byte{
				regionKey: []byte(fakeRegion),
				idKey:     []byte(fakeID),
				secretKey: []byte(fakeSecret),
			},
			getAuthTokenFn: func(_ context.Context, _, _, _ string) (string, error) {
				return fakeToken, nil
			},
			assertions: func(t *testing.T, c *cache.Cache, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, "AWS", creds.Username)
				assert.Equal(t, "password", creds.Password)

				// Verify the token was cached
				token, found := c.Get(tokenCacheKey(fakeRegion, fakeID, fakeSecret))
				assert.True(t, found)
				assert.Equal(t, fakeToken, token)
			},
		},
		{
			name:     "error in getAuthToken",
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			data: map[string][]byte{
				regionKey: []byte(fakeRegion),
				idKey:     []byte(fakeID),
				secretKey: []byte(fakeSecret),
			},
			getAuthTokenFn: func(_ context.Context, _, _, _ string) (string, error) {
				return "", errors.New("auth token error")
			},
			assertions: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.ErrorContains(t, err, "error getting ECR auth token")
				assert.Nil(t, creds)
			},
		},
		{
			name:     "empty token from getAuthToken",
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			data: map[string][]byte{
				regionKey: []byte(fakeRegion),
				idKey:     []byte(fakeID),
				secretKey: []byte(fakeSecret),
			},
			getAuthTokenFn: func(_ context.Context, _, _, _ string) (string, error) {
				return "", nil
			},
			assertions: func(t *testing.T, c *cache.Cache, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.NoError(t, err)

				// Verify the token was not cached
				_, found := c.Get(tokenCacheKey(fakeRegion, fakeID, fakeSecret))
				assert.False(t, found)
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewAccessKeyProvider().(*AccessKeyProvider) // nolint:forcetypeassert
			provider.getAuthTokenFn = tt.getAuthTokenFn

			if tt.setupCache != nil {
				tt.setupCache(provider.tokenCache)
			}

			creds, err := provider.GetCredentials(ctx, "", tt.credType, tt.repoURL, tt.data)
			tt.assertions(t, provider.tokenCache, creds, err)
		})
	}
}

func Test_decodeAuthToken(t *testing.T) {
	testCases := []struct {
		name       string
		token      string
		assertions func(t *testing.T, creds *credentials.Credentials, err error)
	}{
		{
			name:  "valid token",
			token: "QVdTOnBhc3N3b3Jk", // base64 of "AWS:password"
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, "AWS", creds.Username)
				assert.Equal(t, "password", creds.Password)
			},
		},
		{
			name:  "invalid base64",
			token: "invalid-base64",
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.ErrorContains(t, err, "error decoding token")
				assert.Nil(t, creds)
			},
		},
		{
			name:  "valid base64 but invalid format",
			token: "bm90LWEtdmFsaWQtdG9rZW4=", // base64 of "not-a-valid-token"
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.ErrorContains(t, err, "invalid token format")
				assert.Nil(t, creds)
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			creds, err := decodeAuthToken(tt.token)
			tt.assertions(t, creds, err)
		})
	}
}
