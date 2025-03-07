package github

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/internal/credentials"
)

func TestNewAppCredentialProvider(t *testing.T) {
	provider := NewAppCredentialProvider()

	assert.NotNil(t, provider)
	assert.NotNil(t, provider.tokenCache)
	assert.NotNil(t, provider.getAccessTokenFn)
}

func TestAppCredentialProvider_Supports(t *testing.T) {
	p := NewAppCredentialProvider()

	testCases := []struct {
		name     string
		credType credentials.Type
		repoURL  string
		data     map[string][]byte
		expected bool
	}{
		{
			name:     "supports valid Git credential with all required fields",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/akuity/kargo",
			data: map[string][]byte{
				appIDKey:          []byte("123"),
				installationIDKey: []byte("456"),
				privateKeyKey:     []byte("private-key"),
			},
			expected: true,
		},
		{
			name:     "does not support non-Git credential type",
			credType: credentials.Type("other"),
			repoURL:  "https://github.com/akuity/kargo",
			data: map[string][]byte{
				appIDKey:          []byte("123"),
				installationIDKey: []byte("456"),
				privateKeyKey:     []byte("private-key"),
			},
			expected: false,
		},
		{
			name:     "does not support empty data",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/akuity/kargo",
			data:     map[string][]byte{},
			expected: false,
		},
		{
			name:     "does not support missing appID",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/akuity/kargo",
			data: map[string][]byte{
				installationIDKey: []byte("456"),
				privateKeyKey:     []byte("private-key"),
			},
			expected: false,
		},
		{
			name:     "does not support missing installationID",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/akuity/kargo",
			data: map[string][]byte{
				appIDKey:      []byte("123"),
				privateKeyKey: []byte("private-key"),
			},
			expected: false,
		},
		{
			name:     "does not support missing privateKey",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/akuity/kargo",
			data: map[string][]byte{
				appIDKey:          []byte("123"),
				installationIDKey: []byte("456"),
			},
			expected: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result := p.Supports(tt.credType, tt.repoURL, tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAppCredentialProvider_GetCredentials(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name             string
		credType         credentials.Type
		repoURL          string
		data             map[string][]byte
		getAccessTokenFn func(appID, installationID int64, encodedPrivateKey, baseURL string) (string, error)
		assertions       func(t *testing.T, creds *credentials.Credentials, err error)
	}{
		{
			name:     "unsupported credential type",
			credType: credentials.Type("other"),
			repoURL:  "https://github.com/akuity/kargo",
			data: map[string][]byte{
				appIDKey:          []byte("123"),
				installationIDKey: []byte("456"),
				privateKeyKey:     []byte("private-key"),
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.NoError(t, err)
			},
		},
		{
			name:     "invalid app ID",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/akuity/kargo",
			data: map[string][]byte{
				appIDKey:          []byte("invalid"),
				installationIDKey: []byte("456"),
				privateKeyKey:     []byte("private-key"),
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.Error(t, err)
				assert.ErrorContains(t, err, "error parsing app ID")
			},
		},
		{
			name:     "invalid installation ID",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/akuity/kargo",
			data: map[string][]byte{
				appIDKey:          []byte("123"),
				installationIDKey: []byte("invalid"),
				privateKeyKey:     []byte("private-key"),
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.Error(t, err)
				assert.ErrorContains(t, err, "error parsing installation ID")
			},
		},
		{
			name:     "invalid repo URL",
			credType: credentials.TypeGit,
			repoURL:  "://invalid",
			data: map[string][]byte{
				appIDKey:          []byte("123"),
				installationIDKey: []byte("456"),
				privateKeyKey:     []byte("private-key"),
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.Error(t, err)
				assert.ErrorContains(t, err, "error extracting base URL")
			},
		},
		{
			name:     "error getting token",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/akuity/kargo",
			data: map[string][]byte{
				appIDKey:          []byte("123"),
				installationIDKey: []byte("456"),
				privateKeyKey:     []byte("private-key"),
			},
			getAccessTokenFn: func(_, _ int64, _, _ string) (string, error) {
				return "", errors.New("token error")
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.Error(t, err)
				assert.ErrorContains(t, err, "token error")
			},
		},
		{
			name:     "successful token retrieval",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/akuity/kargo",
			data: map[string][]byte{
				appIDKey:          []byte("123"),
				installationIDKey: []byte("456"),
				privateKeyKey:     []byte("private-key"),
			},
			getAccessTokenFn: func(_, _ int64, _, _ string) (string, error) {
				return "test-token", nil
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, accessTokenUsername, creds.Username)
				assert.Equal(t, "test-token", creds.Password)
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewAppCredentialProvider()
			if tt.getAccessTokenFn != nil {
				provider.getAccessTokenFn = tt.getAccessTokenFn
			}

			creds, err := provider.GetCredentials(ctx, "", tt.credType, tt.repoURL, tt.data)
			tt.assertions(t, creds, err)
		})
	}
}

func TestAppCredentialProvider_getUsernameAndPassword(t *testing.T) {
	const (
		fakeAppID          = int64(123)
		fakeInstallationID = int64(456)
		fakePrivateKey     = "private-key"
		fakeBaseURL        = "https://github.com"
		fakeAccessToken    = "test-token"
	)

	testCases := []struct {
		name              string
		appID             int64
		installationID    int64
		encodedPrivateKey string
		baseURL           string
		setupCache        func(c *cache.Cache)
		getAccessTokenFn  func(appID, installationID int64, encodedPrivateKey, baseURL string) (string, error)
		assertions        func(t *testing.T, c *cache.Cache, creds *credentials.Credentials, err error)
	}{
		{
			name:              "cache hit",
			appID:             fakeAppID,
			installationID:    fakeInstallationID,
			encodedPrivateKey: fakePrivateKey,
			baseURL:           fakeBaseURL,
			setupCache: func(c *cache.Cache) {
				cacheKey := tokenCacheKey(fakeBaseURL, fakeAppID, fakeInstallationID, fakePrivateKey)
				c.Set(cacheKey, fakeAccessToken, cache.DefaultExpiration)
			},
			assertions: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, accessTokenUsername, creds.Username)
				assert.Equal(t, fakeAccessToken, creds.Password)
			},
		},
		{
			name:              "cache miss, successful token fetch",
			appID:             fakeAppID,
			installationID:    fakeInstallationID,
			encodedPrivateKey: fakePrivateKey,
			baseURL:           fakeBaseURL,
			getAccessTokenFn: func(_, _ int64, _, _ string) (string, error) {
				return fakeAccessToken, nil
			},
			assertions: func(t *testing.T, c *cache.Cache, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, accessTokenUsername, creds.Username)
				assert.Equal(t, fakeAccessToken, creds.Password)

				// Verify the token was cached
				cacheKey := tokenCacheKey(fakeBaseURL, fakeAppID, fakeInstallationID, fakePrivateKey)
				cachedToken, found := c.Get(cacheKey)
				assert.True(t, found)
				assert.Equal(t, fakeAccessToken, cachedToken)
			},
		},
		{
			name:              "error in getAccessToken",
			appID:             fakeAppID,
			installationID:    fakeInstallationID,
			encodedPrivateKey: fakePrivateKey,
			baseURL:           fakeBaseURL,
			getAccessTokenFn: func(_, _ int64, _, _ string) (string, error) {
				return "", errors.New("token error")
			},
			assertions: func(t *testing.T, c *cache.Cache, creds *credentials.Credentials, err error) {
				assert.ErrorContains(t, err, "token error")
				assert.Nil(t, creds)

				// Verify the token was not cached
				cacheKey := tokenCacheKey(fakeBaseURL, fakeAppID, fakeInstallationID, fakePrivateKey)
				_, found := c.Get(cacheKey)
				assert.False(t, found)
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewAppCredentialProvider()

			if tt.setupCache != nil {
				tt.setupCache(provider.tokenCache)
			}

			if tt.getAccessTokenFn != nil {
				provider.getAccessTokenFn = tt.getAccessTokenFn
			}

			creds, err := provider.getUsernameAndPassword(tt.appID, tt.installationID, tt.encodedPrivateKey, tt.baseURL)
			tt.assertions(t, provider.tokenCache, creds, err)
		})
	}
}

func Test_decodeKey(t *testing.T) {
	const key = "-----BEGIN PRIVATE KEY-----\nfakekey\n-----END PRIVATE KEY-----"
	testCases := []struct {
		name        string
		key         string
		expectedKey string
		expectsErr  bool
	}{
		{
			name:        "key is not base64 encoded",
			key:         key,
			expectedKey: key,
		},
		{
			name:        "key is base64 encoded",
			key:         base64.StdEncoding.EncodeToString([]byte(key)),
			expectedKey: key,
		},
		{
			name:       "key is a corrupted base64 encoding",
			key:        "corrupted", // These are all base64 digits. :)
			expectsErr: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			key, err := decodeKey(testCase.key)
			if testCase.expectsErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, []byte(testCase.expectedKey), key)
		})
	}
}

func Test_extractBaseURL(t *testing.T) {
	testCases := []struct {
		name        string
		url         string
		expected    string
		shouldError bool
	}{
		{
			name:        "valid HTTPS URL",
			url:         "https://github.com/akuity/kargo",
			expected:    "https://github.com",
			shouldError: false,
		},
		{
			name:        "valid HTTP URL",
			url:         "http://github.com/akuity/kargo",
			expected:    "http://github.com",
			shouldError: false,
		},
		{
			name:        "invalid URL",
			url:         "://invalid",
			expected:    "",
			shouldError: true,
		},
		{
			name:        "URL with port",
			url:         "https://github.com:8080/akuity/kargo",
			expected:    "https://github.com:8080",
			shouldError: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractBaseURL(tt.url)

			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
