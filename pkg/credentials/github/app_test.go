package github

import (
	"encoding/base64"
	"errors"
	"fmt"
	"maps"
	"testing"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
)

func TestNewAppCredentialProvider(t *testing.T) {
	provider := NewAppCredentialProvider().(*AppCredentialProvider) // nolint:forcetypeassert
	assert.NotNil(t, provider)

	assert.NotNil(t, provider.tokenCache)
	assert.NotNil(t, provider.getAccessTokenFn)
}

func TestAppCredentialProvider_Supports(t *testing.T) {
	p := NewAppCredentialProvider()

	const testRepoURL = "https://github.com/example/repo"
	// This is a control. Each test case will tweak a clone of this supported map.
	supportedDataMap := map[string][]byte{
		clientIDKey:       []byte("client"),
		appIDKey:          []byte("123"),
		installationIDKey: []byte("456"),
		privateKeyKey:     []byte("private-key"),
	}
	supports, err := p.Supports(
		t.Context(),
		credentials.Request{
			Type:    credentials.TypeGit,
			RepoURL: testRepoURL,
			Data:    supportedDataMap,
		},
	)
	require.NoError(t, err)
	require.True(t, supports)

	testCases := []struct {
		name       string
		credType   credentials.Type
		repoURL    string
		getDataMap func() map[string][]byte
		expected   bool
	}{
		{
			name:     "non-Git credential type",
			credType: credentials.Type("other"),
			repoURL:  testRepoURL,
			getDataMap: func() map[string][]byte {
				return supportedDataMap
			},
			expected: false,
		},
		{
			name:     "nil data map",
			credType: credentials.TypeGit,
			repoURL:  testRepoURL,
			getDataMap: func() map[string][]byte {
				return nil
			},
			expected: false,
		},
		{
			name:     "empty data map",
			credType: credentials.TypeGit,
			repoURL:  testRepoURL,
			getDataMap: func() map[string][]byte {
				return map[string][]byte{}
			},
			expected: false,
		},
		{
			name:     "not an http/s URL",
			credType: credentials.TypeGit,
			repoURL:  "git@github.com:example/repo.git",
			getDataMap: func() map[string][]byte {
				return supportedDataMap
			},
			expected: false,
		},
		{
			name:     "no client ID or app ID in data map",
			credType: credentials.TypeGit,
			repoURL:  testRepoURL,
			getDataMap: func() map[string][]byte {
				dm := maps.Clone(supportedDataMap)
				delete(dm, appIDKey)
				delete(dm, clientIDKey)
				return dm
			},
			expected: false,
		},
		{
			name:     "client ID and app ID are empty",
			credType: credentials.TypeGit,
			repoURL:  testRepoURL,
			getDataMap: func() map[string][]byte {
				dm := maps.Clone(supportedDataMap)
				dm[appIDKey] = []byte("")
				dm[clientIDKey] = []byte("")
				return dm
			},
			expected: false,
		},
		{
			name:     "no installation ID in data map",
			credType: credentials.TypeGit,
			repoURL:  testRepoURL,
			getDataMap: func() map[string][]byte {
				dm := maps.Clone(supportedDataMap)
				delete(dm, installationIDKey)
				return dm
			},
			expected: false,
		},
		{
			name:     "installation ID is empty",
			credType: credentials.TypeGit,
			repoURL:  testRepoURL,
			getDataMap: func() map[string][]byte {
				dm := maps.Clone(supportedDataMap)
				dm[installationIDKey] = []byte("")
				return dm
			},
			expected: false,
		},
		{
			name:     "no private key in data map",
			credType: credentials.TypeGit,
			repoURL:  testRepoURL,
			getDataMap: func() map[string][]byte {
				dm := maps.Clone(supportedDataMap)
				delete(dm, privateKeyKey)
				return dm
			},
			expected: false,
		},
		{
			name:     "private key is empty",
			credType: credentials.TypeGit,
			repoURL:  testRepoURL,
			getDataMap: func() map[string][]byte {
				dm := maps.Clone(supportedDataMap)
				dm[privateKeyKey] = []byte("")
				return dm
			},
			expected: false,
		},
		{
			name:     "valid with client ID",
			credType: credentials.TypeGit,
			repoURL:  testRepoURL,
			getDataMap: func() map[string][]byte {
				dm := maps.Clone(supportedDataMap)
				delete(dm, appIDKey)
				return dm
			},
			expected: true,
		},
		{
			name:     "valid with App ID",
			credType: credentials.TypeGit,
			repoURL:  testRepoURL,
			getDataMap: func() map[string][]byte {
				dm := maps.Clone(supportedDataMap)
				delete(dm, clientIDKey)
				return dm
			},
			expected: true,
		},
		{
			name:     "valid with .git suffix",
			credType: credentials.TypeGit,
			repoURL:  testRepoURL + ".git",
			getDataMap: func() map[string][]byte {
				dm := maps.Clone(supportedDataMap)
				delete(dm, clientIDKey)
				return dm
			},
			expected: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			supports, err := p.Supports(
				t.Context(),
				credentials.Request{
					Type:    testCase.credType,
					RepoURL: testCase.repoURL,
					Data:    testCase.getDataMap(),
				},
			)
			require.NoError(t, err)
			require.Equal(t, testCase.expected, supports)
		})
	}
}

func TestAppCredentialProvider_GetCredentials(t *testing.T) {
	const testProject = "fake-project"
	const testRepoName = "repo"
	testRepoURL := fmt.Sprintf("https://github.com/example/%s", testRepoName)
	testData := map[string][]byte{
		appIDKey:          []byte("123"),
		installationIDKey: []byte("456"),
		privateKeyKey:     []byte("private-key"),
	}

	testCases := []struct {
		name             string
		credType         credentials.Type
		repoURL          string
		data             map[string][]byte
		metadata         map[string]string
		getAccessTokenFn func(
			appOrClientID string,
			installationID int64,
			encodedPrivateKey string,
			repoURL string,
		) (string, error)
		assertions func(t *testing.T, creds *credentials.Credentials, err error)
	}{
		{
			name:     "cannot extract repo name from URL",
			credType: credentials.TypeGit,
			repoURL:  "https://github.com/example", // Looks like an org; not a repo
			data:     testData,
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.Nil(t, creds)
			},
		},
		{
			name:     "error unmarshaling scope map",
			credType: credentials.TypeGit,
			repoURL:  testRepoURL,
			data:     testData,
			metadata: map[string]string{
				kargoapi.AnnotationKeyGitHubTokenScope: "invalid json",
			},
			assertions: func(t *testing.T, _ *credentials.Credentials, err error) {
				assert.ErrorContains(t, err, "error unmarshaling scope map")
			},
		},
		{
			name:     "project has no entry in scope map",
			credType: credentials.TypeGit,
			repoURL:  testRepoURL,
			data:     testData,
			metadata: map[string]string{
				kargoapi.AnnotationKeyGitHubTokenScope: `{}`,
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.Nil(t, creds)
			},
		},
		{
			name:     "project has nil entry in scope map",
			credType: credentials.TypeGit,
			repoURL:  testRepoURL,
			data:     testData,
			metadata: map[string]string{
				kargoapi.AnnotationKeyGitHubTokenScope: fmt.Sprintf(`{%q: null}`, testProject),
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.Nil(t, creds)
			},
		},
		{
			name:     "project has empty entry in scope map",
			credType: credentials.TypeGit,
			repoURL:  testRepoURL,
			data:     testData,
			metadata: map[string]string{
				kargoapi.AnnotationKeyGitHubTokenScope: fmt.Sprintf(`{%q: []}`, testProject),
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.Nil(t, creds)
			},
		},
		{
			name:     "invalid installation ID",
			credType: credentials.TypeGit,
			repoURL:  testRepoURL,
			data: map[string][]byte{
				appIDKey:          []byte("123"),
				installationIDKey: []byte("invalid"),
				privateKeyKey:     []byte("private-key"),
			},
			// We'll limit the test project to accessing only the test repo. If we
			// get as far as the error parsing the installation token, we'll know
			// the check that the scope is allowed is working.
			metadata: map[string]string{
				kargoapi.AnnotationKeyGitHubTokenScope: fmt.Sprintf(
					`{%q: [%q]}`, testProject, testRepoName,
				),
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.Error(t, err)
				assert.ErrorContains(t, err, "error parsing installation ID")
			},
		},
		// From here on out, we won't include any scope map...
		{
			name:     "error getting token",
			credType: credentials.TypeGit,
			repoURL:  testRepoURL,
			data:     testData,
			getAccessTokenFn: func(_ string, _ int64, _, _ string) (string, error) {
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
			repoURL:  testRepoURL,
			data:     testData,
			getAccessTokenFn: func(_ string, _ int64, _, _ string) (string, error) {
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

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			provider := NewAppCredentialProvider().(*AppCredentialProvider) // nolint:forcetypeassert

			if testCase.getAccessTokenFn != nil {
				provider.getAccessTokenFn = testCase.getAccessTokenFn
			}

			creds, err := provider.GetCredentials(
				t.Context(),
				credentials.Request{
					Type:     testCase.credType,
					Project:  testProject,
					RepoURL:  testCase.repoURL,
					Data:     testCase.data,
					Metadata: testCase.metadata,
				},
			)
			testCase.assertions(t, creds, err)
		})
	}
}

func TestAppCredentialProvider_getUsernameAndPassword(t *testing.T) {
	const (
		fakeAppOrClientID  = "fake-id"
		fakeInstallationID = int64(456)
		fakePrivateKey     = "private-key"
		fakeRepoURL        = "https://github.com/example/repo"
		fakeRepoName       = "repo"
		fakeAccessToken    = "test-token"
	)

	p := &AppCredentialProvider{}

	testTokenCacheKey := p.tokenCacheKey(
		fakeAppOrClientID,
		fakeInstallationID,
		fakePrivateKey,
		fakeRepoURL,
	)

	testCases := []struct {
		name             string
		setupCache       func(c *cache.Cache)
		getAccessTokenFn func(
			appOrClientID string,
			installationID int64,
			encodedPrivateKey string,
			repoURL string,
		) (string, error)
		assertions func(*testing.T, *cache.Cache, *credentials.Credentials, error)
	}{
		{
			name: "cache hit",
			setupCache: func(c *cache.Cache) {
				c.Set(testTokenCacheKey, fakeAccessToken, cache.DefaultExpiration)
			},
			assertions: func(
				t *testing.T,
				_ *cache.Cache,
				creds *credentials.Credentials,
				err error,
			) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, accessTokenUsername, creds.Username)
				assert.Equal(t, fakeAccessToken, creds.Password)
			},
		},
		{
			name: "cache miss, successful token fetch",
			getAccessTokenFn: func(_ string, _ int64, _, _ string) (string, error) {
				return fakeAccessToken, nil
			},
			assertions: func(
				t *testing.T,
				c *cache.Cache,
				creds *credentials.Credentials,
				err error,
			) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, accessTokenUsername, creds.Username)
				assert.Equal(t, fakeAccessToken, creds.Password)

				// Verify the token was cached
				cachedToken, found := c.Get(testTokenCacheKey)
				assert.True(t, found)
				assert.Equal(t, fakeAccessToken, cachedToken)
			},
		},
		{
			name: "error in getAccessToken",
			getAccessTokenFn: func(_ string, _ int64, _, _ string) (string, error) {
				return "", errors.New("token error")
			},
			assertions: func(
				t *testing.T,
				c *cache.Cache,
				creds *credentials.Credentials,
				err error,
			) {
				assert.ErrorContains(t, err, "token error")
				assert.Nil(t, creds)

				// Verify the token was not cached
				_, found := c.Get(testTokenCacheKey)
				assert.False(t, found)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			provider := NewAppCredentialProvider().(*AppCredentialProvider) // nolint:forcetypeassert

			if testCase.setupCache != nil {
				testCase.setupCache(provider.tokenCache)
			}

			if testCase.getAccessTokenFn != nil {
				provider.getAccessTokenFn = testCase.getAccessTokenFn
			}

			creds, err := provider.getUsernameAndPassword(
				fakeAppOrClientID,
				fakeInstallationID,
				fakePrivateKey,
				fakeRepoURL,
			)
			testCase.assertions(t, provider.tokenCache, creds, err)
		})
	}
}

func TestAppCredentialProvider_decodeKey(t *testing.T) {
	const testKey = "-----BEGIN PRIVATE KEY-----\nfakekey\n-----END PRIVATE KEY-----"
	testCases := []struct {
		name       string
		key        string
		assertions func(t *testing.T, key []byte, err error)
	}{
		{
			name: "key is not base64 encoded",
			key:  testKey,
			assertions: func(t *testing.T, key []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, []byte(testKey), key)
			},
		},
		{
			name: "key is base64 encoded",
			key:  base64.StdEncoding.EncodeToString([]byte(testKey)),
			assertions: func(t *testing.T, key []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, []byte(testKey), key)
			},
		},
		{
			name: "key is a corrupted base64 encoding",
			key:  "corrupted", // These are all base64 digits. :)
			assertions: func(t *testing.T, _ []byte, err error) {
				require.ErrorContains(t, err, "probable corrupt base64 encoding")
			},
		},
	}
	p := &AppCredentialProvider{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			key, err := p.decodeKey(testCase.key)
			testCase.assertions(t, key, err)
		})
	}
}

func TestAppCredentialProvider_extractRepoName(t *testing.T) {
	testCases := []struct {
		name     string
		repoURL  string
		expected string
	}{
		{
			name:     "invalid repo URL",
			repoURL:  "https://github.com/akuity",
			expected: "",
		},
		{
			name:     "GitHub URL",
			repoURL:  "https://github.com/example/repo",
			expected: "repo",
		},
		{
			name:     "GitHub URL with .git suffix",
			repoURL:  "https://github.com/example/repo.git",
			expected: "repo",
		},
		{
			name:     "GitHub Enterprise URL",
			repoURL:  "https://github.example.com/example/repo",
			expected: "repo",
		},
		{
			name:     "GitHub Enterprise URL with extra path components", // Possible?
			repoURL:  "https://example.com/github/example/repo",
			expected: "repo",
		},
	}
	p := &AppCredentialProvider{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expected, p.extractRepoName(testCase.repoURL))
		})
	}
}

func TestAppCredentialProvider_extractBaseURL(t *testing.T) {
	testCases := []struct {
		name       string
		repoURL    string
		assertions func(t *testing.T, baseURL string, err error)
	}{
		{
			name:    "invalid URL",
			repoURL: "://invalid",
			assertions: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "error parsing URL")
			},
		},
		{
			name:    "valid HTTPS URL",
			repoURL: "https://github.com/example/repo",
			assertions: func(t *testing.T, baseURL string, err error) {
				require.NoError(t, err)
				require.Equal(t, "https://github.com", baseURL)
			},
		},
		{
			name:    "valid HTTP URL",
			repoURL: "http://github.example.com/example/repo",
			assertions: func(t *testing.T, baseURL string, err error) {
				require.NoError(t, err)
				require.Equal(t, "http://github.example.com", baseURL)
			},
		},
		{
			name:    "URL with port number",
			repoURL: "https://github.example.com:8443/example/repo",
			assertions: func(t *testing.T, baseURL string, err error) {
				require.NoError(t, err)
				require.Equal(t, "https://github.example.com:8443", baseURL)
			},
		},
	}
	p := &AppCredentialProvider{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			baseURL, err := p.extractBaseURL(testCase.repoURL)
			testCase.assertions(t, baseURL, err)
		})
	}
}
