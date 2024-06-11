package github

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestNewAppCredentialHelper(t *testing.T) {
	h := NewAppCredentialHelper()
	require.NotNil(t, h)
	a, ok := h.(*appCredentialHelper)
	require.True(t, ok)
	require.NotNil(t, a.getAccessTokenFn)
}

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
		secret     *corev1.Secret
		helper     AppCredentialHelper
		assertions func(t *testing.T, username, password string, c *cache.Cache, err error)
	}{
		{
			name:   "no github details provided",
			secret: &corev1.Secret{},
			helper: NewAppCredentialHelper(),
			assertions: func(t *testing.T, username, password string, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Empty(t, username)
				require.Empty(t, password)
			},
		},
		{
			name: "app ID missing",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					installationIDKey: []byte(testInstallationIDStr),
					privateKeyKey:     []byte(testPrivateKey),
				},
			},
			helper: NewAppCredentialHelper(),
			assertions: func(t *testing.T, _, _ string, _ *cache.Cache, err error) {
				require.ErrorContains(t, err, "must all be set or all be unset")
			},
		},
		{
			name: "installation ID missing",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					appIDKey:      []byte(testAppIDStr),
					privateKeyKey: []byte(testPrivateKey),
				},
			},
			helper: NewAppCredentialHelper(),
			assertions: func(t *testing.T, _, _ string, _ *cache.Cache, err error) {
				require.ErrorContains(t, err, "must all be set or all be unset")
			},
		},
		{
			name: "private key missing",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					appIDKey:          []byte(testAppIDStr),
					installationIDKey: []byte(testInstallationIDStr),
				},
			},
			helper: NewAppCredentialHelper(),
			assertions: func(t *testing.T, _, _ string, _ *cache.Cache, err error) {
				require.ErrorContains(t, err, "must all be set or all be unset")
			},
		},
		{
			name: "cache hit",
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
			assertions: func(t *testing.T, username, password string, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Equal(t, "kargo", username)
				require.Equal(t, testAccessToken, password)
			},
		},
		{
			name: "cache miss; error getting access token",
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
			assertions: func(t *testing.T, _, _ string, _ *cache.Cache, err error) {
				require.ErrorContains(t, err, "error getting installation access token")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "cache miss; success",
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
			assertions: func(t *testing.T, username, password string, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Equal(t, "kargo", username)
				require.Equal(t, testAccessToken, password)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			username, password, err :=
				testCase.helper.GetUsernameAndPassword(testCase.secret)
			cache := testCase.helper.(*appCredentialHelper).tokenCache // nolint: forcetypeassert
			testCase.assertions(t, username, password, cache, err)
		})
	}
}
