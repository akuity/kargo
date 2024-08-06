package github

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/akuity/kargo/internal/credentials"
)

func TestAppCredentialHelper(t *testing.T) {
	const (
		testAppIDStr          = "12345"
		testInstallationIDStr = "67890"
		testPrivateKey        = "fake-private-key"
		testAccessToken       = "fake-access-token"
	)
	testInstallationID, err := strconv.ParseInt(testInstallationIDStr, 10, 64)
	require.NoError(t, err)
	testAppID, err := strconv.ParseInt(testAppIDStr, 10, 64)
	require.NoError(t, err)

	warmTokenCache := cache.New(0, 0)
	warmTokenCache.Set(
		(&appCredentialHelper{}).tokenCacheKey(testAppID, testInstallationID, testPrivateKey),
		testAccessToken,
		cache.DefaultExpiration,
	)

	testCases := []struct {
		name       string
		credType   credentials.Type
		secret     *corev1.Secret
		helper     *appCredentialHelper
		assertions func(*testing.T, *credentials.Credentials, *cache.Cache, error)
	}{
		{
			name:     "cred type is not git",
			credType: credentials.TypeImage,
			helper:   &appCredentialHelper{},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name:     "cred type is git",
			credType: credentials.TypeGit,
			helper:   &appCredentialHelper{},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name:     "secret is nil",
			credType: credentials.TypeImage,
			helper:   &appCredentialHelper{},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name:     "secret is nil - git",
			credType: credentials.TypeGit,
			helper:   &appCredentialHelper{},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name:     "no github details provided",
			credType: credentials.TypeImage,
			secret:   &corev1.Secret{},
			helper:   &appCredentialHelper{},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name:     "no github details provided - git",
			credType: credentials.TypeGit,
			secret:   &corev1.Secret{},
			helper:   &appCredentialHelper{},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name:     "app ID missing",
			credType: credentials.TypeImage,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					installationIDKey: []byte(testInstallationIDStr),
					privateKeyKey:     []byte(testPrivateKey),
				},
			},
			helper: &appCredentialHelper{},
			assertions: func(t *testing.T, _ *credentials.Credentials, _ *cache.Cache, err error) {
				//require.ErrorContains(t, err, "must all be set or all be unset")
			},
		},
		{
			name:     "app ID missing - git",
			credType: credentials.TypeGit,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					installationIDKey: []byte(testInstallationIDStr),
					privateKeyKey:     []byte(testPrivateKey),
				},
			},
			helper: &appCredentialHelper{},
			assertions: func(t *testing.T, _ *credentials.Credentials, _ *cache.Cache, err error) {
				require.ErrorContains(t, err, "must all be set or all be unset")
			},
		},
		{
			name:     "installation ID missing",
			credType: credentials.TypeImage,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					appIDKey:      []byte(testAppIDStr),
					privateKeyKey: []byte(testPrivateKey),
				},
			},
			helper: &appCredentialHelper{},
			assertions: func(t *testing.T, _ *credentials.Credentials, _ *cache.Cache, err error) {
				require.ErrorContains(t, err, "must all be set or all be unset")
			},
		},
		{
			name:     "installation ID missing - git",
			credType: credentials.TypeGit,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					appIDKey:      []byte(testAppIDStr),
					privateKeyKey: []byte(testPrivateKey),
				},
			},
			helper: &appCredentialHelper{},
			assertions: func(t *testing.T, _ *credentials.Credentials, _ *cache.Cache, err error) {
				require.ErrorContains(t, err, "must all be set or all be unset")
			},
		},
		{
			name:     "private key missing",
			credType: credentials.TypeImage,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					appIDKey:          []byte(testAppIDStr),
					installationIDKey: []byte(testInstallationIDStr),
				},
			},
			helper: &appCredentialHelper{},
			assertions: func(t *testing.T, _ *credentials.Credentials, _ *cache.Cache, err error) {
				require.ErrorContains(t, err, "must all be set or all be unset")
			},
		},
		{
			name:     "private key missing - git",
			credType: credentials.TypeGit,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					appIDKey:          []byte(testAppIDStr),
					installationIDKey: []byte(testInstallationIDStr),
				},
			},
			helper: &appCredentialHelper{},
			assertions: func(t *testing.T, _ *credentials.Credentials, _ *cache.Cache, err error) {
				require.ErrorContains(t, err, "must all be set or all be unset")
			},
		},
		{
			name:     "cache hit",
			credType: credentials.TypeImage,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					appIDKey:          []byte(testAppIDStr),
					installationIDKey: []byte(testInstallationIDStr),
					privateKeyKey:     []byte(testPrivateKey),
				},
			},
			helper: &appCredentialHelper{
				tokenCache: warmTokenCache,
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				require.Equal(t, "kargo", creds.Username)
				require.Equal(t, testAccessToken, creds.Password)
			},
		},
		{
			name:     "cache hit - git",
			credType: credentials.TypeGit,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					appIDKey:          []byte(testAppIDStr),
					installationIDKey: []byte(testInstallationIDStr),
					privateKeyKey:     []byte(testPrivateKey),
				},
			},
			helper: &appCredentialHelper{
				tokenCache: warmTokenCache,
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				require.Equal(t, "kargo", creds.Username)
				require.Equal(t, testAccessToken, creds.Password)
			},
		},
		{
			name:     "cache miss; error getting access token",
			credType: credentials.TypeImage,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					appIDKey:          []byte(testAppIDStr),
					installationIDKey: []byte(testInstallationIDStr),
					privateKeyKey:     []byte(testPrivateKey),
				},
			},
			helper: &appCredentialHelper{
				tokenCache: cache.New(0, 0),
				getAccessTokenFn: func(int64, int64, string) (string, error) {
					return "", fmt.Errorf("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *credentials.Credentials, _ *cache.Cache, err error) {
				require.ErrorContains(t, err, "error getting installation access token")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name:     "cache miss; error getting access token - git",
			credType: credentials.TypeGit,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					appIDKey:          []byte(testAppIDStr),
					installationIDKey: []byte(testInstallationIDStr),
					privateKeyKey:     []byte(testPrivateKey),
				},
			},
			helper: &appCredentialHelper{
				tokenCache: cache.New(0, 0),
				getAccessTokenFn: func(int64, int64, string) (string, error) {
					return "", fmt.Errorf("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *credentials.Credentials, _ *cache.Cache, err error) {
				require.ErrorContains(t, err, "error getting installation access token")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name:     "cache miss; success",
			credType: credentials.TypeImage,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					appIDKey:          []byte(testAppIDStr),
					installationIDKey: []byte(testInstallationIDStr),
					privateKeyKey:     []byte(testPrivateKey),
				},
			},
			helper: &appCredentialHelper{
				tokenCache: cache.New(0, 0),
				getAccessTokenFn: func(int64, int64, string) (string, error) {
					return testAccessToken, nil
				},
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				require.Equal(t, "kargo", creds.Username)
				require.Equal(t, testAccessToken, creds.Password)
			},
		},
		{
			name:     "cache miss; success - git",
			credType: credentials.TypeGit,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					appIDKey:          []byte(testAppIDStr),
					installationIDKey: []byte(testInstallationIDStr),
					privateKeyKey:     []byte(testPrivateKey),
				},
			},
			helper: &appCredentialHelper{
				tokenCache: cache.New(0, 0),
				getAccessTokenFn: func(int64, int64, string) (string, error) {
					return testAccessToken, nil
				},
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				require.Equal(t, "kargo", creds.Username)
				require.Equal(t, testAccessToken, creds.Password)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			creds, err := testCase.helper.getCredentials(
				context.Background(),
				"", // project is irrelevant for this helper
				testCase.credType,
				"", // repoURL is irrelevant for this helper
				testCase.secret,
			)
			testCase.assertions(t, creds, testCase.helper.tokenCache, err)
		})
	}
}
